package dao

import (
	"app/tables/gogame"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
	"reflect"
	"time"
)

type fairProofInput struct {
	Record   map[string]any `json:"record"`
	AgentID  uint32         `json:"agentId"`
	UserID   uint32         `json:"userId"`
	RoomID   int64          `json:"roomId"`
	Inning   int64          `json:"inning"`
	RoomType int64          `json:"roomType"`
}

func fairCommitment(record map[string]any) (string, error) {
	cards, ok := record["pokerList"].([]any)
	if !ok || len(cards) < 1 || len(cards) > 416 {
		return "", status.Error(codes.InvalidArgument, "invalid fair proof deck")
	}
	immutable := map[string]any{}
	for _, key := range []string{"proofId", "agentId", "gameId", "gameRoomId", "roomType", "currentInning", "createdAt", "holeEnd", "isHoldem", "holeCount", "playerIds", "pokerList", "notarizationCode"} {
		immutable[key] = record[key]
	}
	raw, err := json.Marshal(immutable)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", sha256.Sum256(raw)), nil
}
func fairViewer(record map[string]any, uid uint32) bool {
	values, _ := record["viewers"].([]any)
	for _, value := range values {
		if gogameNumber(value) == int64(uid) {
			return true
		}
	}
	return false
}
func validateProofUpdate(old, next map[string]any) error {
	for _, key := range []string{"viewers", "revealed", "revealedCards"} {
		before, _ := old[key].([]any)
		after, _ := next[key].([]any)
		for _, value := range before {
			found := false
			for _, candidate := range after {
				if reflect.DeepEqual(value, candidate) {
					found = true
					break
				}
			}
			if !found {
				return paymentInvalid("fair proof disclosure cannot be removed")
			}
		}
	}
	if gogameNumber(next["finishedAt"]) < gogameNumber(old["finishedAt"]) || gogameNumber(next["boardCount"]) < gogameNumber(old["boardCount"]) {
		return paymentInvalid("fair proof cannot move backwards")
	}
	return nil
}
func (dd *DBDao) gogameFairProof(ctx context.Context, operation, payload string) (any, error) {
	var input fairProofInput
	if err := DecodeGogameJSON(payload, &input); err != nil {
		return nil, err
	}
	if operation == "fairProofRead" {
		if input.AgentID == 0 || input.UserID == 0 || input.RoomID <= 0 || input.Inning <= 0 {
			return nil, status.Error(codes.InvalidArgument, "invalid proof query")
		}
		query := dd.manager.WithContext(ctx).Where("agent_id=? AND room_id=? AND inning=?", input.AgentID, input.RoomID, input.Inning)
		if input.RoomType > 0 {
			query = query.Where("room_type=?", input.RoomType)
		}
		var rows []gogame.FairProof
		if err := query.Order("created_at DESC").Limit(10).Find(&rows).Error; err != nil {
			return nil, err
		}
		for _, row := range rows {
			var record map[string]any
			if err := DecodeGogameJSON(row.Payload, &record); err != nil {
				return nil, err
			}
			if fairViewer(record, input.UserID) {
				return record, nil
			}
		}
		return nil, nil
	}
	record := input.Record
	id, _ := record["proofId"].(string)
	agent, room, inning := gogameNumber(record["agentId"]), gogameNumber(record["gameRoomId"]), gogameNumber(record["currentInning"])
	if len(id) != 64 || agent <= 0 || room <= 0 || inning <= 0 || gogameNumber(record["version"]) < 1 || gogameNumber(record["createdAt"]) < time.Now().Add(-31*24*time.Hour).UnixMilli() {
		return nil, status.Error(codes.InvalidArgument, "invalid proof identity")
	}
	commitment, err := fairCommitment(record)
	if err != nil {
		return nil, err
	}
	raw, err := json.Marshal(record)
	if err != nil || len(raw) > 500000 {
		return nil, status.Error(codes.InvalidArgument, "proof exceeds size limit")
	}
	err = dd.manager.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Insert the immutable commitment once; the same identity can never acquire
		// another deck, even if the acknowledgement is lost.
		_, e := tx.Statement.ConnPool.ExecContext(ctx, "INSERT IGNORE INTO gg_fair_proof (id,agent_id,room_id,inning,room_type,version,created_at,commitment,payload) VALUES (?,?,?,?,?,?,?,?,?)", id, agent, room, inning, gogameNumber(record["roomType"]), gogameNumber(record["version"]), gogameNumber(record["createdAt"]), commitment, string(raw))
		if e != nil {
			return e
		}
		rows, e := gogameRows(ctx, tx, "SELECT commitment,version,payload FROM gg_fair_proof WHERE id=? FOR UPDATE", id)
		if e != nil {
			return e
		}
		if len(rows) != 1 || rows[0]["commitment"] != commitment {
			return paymentInvalid("fair commitment conflict")
		}
		oldVersion, nextVersion := gogameNumber(rows[0]["version"]), gogameNumber(record["version"])
		if nextVersion < oldVersion {
			return nil
		}
		if nextVersion == oldVersion {
			if rows[0]["payload"] != string(raw) {
				return paymentInvalid("fair proof version conflict")
			}
			return nil
		}
		var old map[string]any
		if e = DecodeGogameJSON(rows[0]["payload"].(string), &old); e != nil {
			return e
		}
		if e = validateProofUpdate(old, record); e != nil {
			return e
		}
		_, e = tx.Statement.ConnPool.ExecContext(ctx, "UPDATE gg_fair_proof SET version=?,payload=? WHERE id=?", nextVersion, string(raw), id)
		return e
	})
	if err != nil {
		return nil, err
	}
	return map[string]any{"stored": true, "commitment": commitment}, nil
}
