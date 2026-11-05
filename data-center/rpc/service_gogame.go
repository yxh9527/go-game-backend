package rpc

import (
	"context"
	"crypto/subtle"
	"data-center/dao"
	"encoding/json"
	"errors"
	"go.uber.org/zap"
	"micro_service/services"
	"os"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type gogameRepository interface {
	GogameCall(context.Context, string, string) (any, error)
	WithGogameMatch(context.Context, int64, func(map[string]any, map[string]any) error) (bool, error)
}
type gogameStorageService struct {
	services.UnimplementedGogameStorageServiceServer
	db gogameRepository
}

func newGogameStorageService(db gogameRepository) *gogameStorageService {
	return &gogameStorageService{db: db}
}

func gogameAuthorize(ctx context.Context) error {
	expected := os.Getenv("GOGAME_DATA_CENTER_TOKEN")
	md, _ := metadata.FromIncomingContext(ctx)
	tokens := md.Get("x-gogame-storage-token")
	if expected == "" {
		return status.Error(codes.Unavailable, "gogame storage credential not configured")
	}
	if len(tokens) != 1 || subtle.ConstantTimeCompare([]byte(tokens[0]), []byte(expected)) != 1 {
		return status.Error(codes.PermissionDenied, "invalid gogame storage credential")
	}
	return nil
}

func gogameError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, context.Canceled) {
		return status.Error(codes.Canceled, "gogame request cancelled")
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return status.Error(codes.DeadlineExceeded, "gogame storage timeout")
	}
	if _, ok := status.FromError(err); ok {
		return err
	}
	// SQL/schema/connection details stay inside the data-center logs.
	zap.L().Error("gogame storage operation failed", zap.Error(err))
	return status.Error(codes.Internal, "gogame database operation failed")
}

func (d *gogameStorageService) Call(ctx context.Context, req *services.GogameStorageRequest) (*services.GogameStorageResponse, error) {
	if err := gogameAuthorize(ctx); err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	value, err := d.db.GogameCall(ctx, req.Operation, req.Payload)
	if err != nil {
		return nil, gogameError(err)
	}
	payload, err := json.Marshal(value)
	if err != nil {
		return nil, gogameError(err)
	}
	return &services.GogameStorageResponse{Phase: "result", Payload: string(payload)}, nil
}

func (d *gogameStorageService) MatchTransaction(stream services.GogameStorageService_MatchTransactionServer) error {
	if err := gogameAuthorize(stream.Context()); err != nil {
		return err
	}
	first, err := stream.Recv()
	if err != nil {
		return err
	}
	if first.Operation != "begin" {
		return status.Error(codes.InvalidArgument, "match transaction must begin")
	}
	var args struct {
		ID int64 `json:"id"`
	}
	if err = dao.DecodeGogameJSON(first.Payload, &args); err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(stream.Context(), 60*time.Second)
	defer cancel()
	found, err := d.db.WithGogameMatch(ctx, args.ID, func(row, state map[string]any) error {
		payload, e := json.Marshal(map[string]any{"row": row, "state": state})
		if e != nil {
			return e
		}
		if e = stream.Send(&services.GogameStorageResponse{Phase: "snapshot", Payload: string(payload)}); e != nil {
			return e
		}
		// Recv runs separately so a client that never commits cannot hold a MySQL
		// row lock beyond the server's transaction deadline.
		type reply struct {
			message *services.GogameStorageRequest
			err     error
		}
		received := make(chan reply, 1)
		go func() { message, e := stream.Recv(); received <- reply{message, e} }()
		var commit *services.GogameStorageRequest
		select {
		case <-ctx.Done():
			return ctx.Err()
		case result := <-received:
			commit, e = result.message, result.err
		}
		if e != nil {
			return e
		}
		if commit.Operation != "commit" {
			return status.Error(codes.InvalidArgument, "expected transaction commit")
		}
		var next struct {
			Row   map[string]any `json:"row"`
			State map[string]any `json:"state"`
		}
		if e = dao.DecodeGogameJSON(commit.Payload, &next); e != nil {
			return e
		}
		if next.Row == nil || next.State == nil {
			return status.Error(codes.InvalidArgument, "row and journal required")
		}
		// Replace contents only after receiving and validating the complete frame.
		for k := range row {
			delete(row, k)
		}
		for k, v := range next.Row {
			row[k] = v
		}
		for k := range state {
			delete(state, k)
		}
		for k, v := range next.State {
			state[k] = v
		}
		return nil
	})
	if err != nil {
		return gogameError(err)
	}
	if !found {
		return stream.Send(&services.GogameStorageResponse{Phase: "absent", Payload: "null"})
	}
	return stream.Send(&services.GogameStorageResponse{Phase: "committed", Payload: "null"})
}
