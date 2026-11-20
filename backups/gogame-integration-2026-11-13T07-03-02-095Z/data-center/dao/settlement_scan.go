package dao

import (
	"app/esindex"
	"context"
	"encoding/json"
	"fmt"
	"github.com/olivere/elastic/v7"
	"time"
)

const settlementScanBytes = 24 * 1024 * 1024

// Read the same standard financial documents for every game. SearchAfter
// avoids the ES result-window limit without building another index or cache.
// The original time range is fixed by the caller for the whole request.
func (es *ESDao) scanSettlementDocuments(parent context.Context, filter elastic.Query, visit func(*elastic.SearchHit) (bool, error)) error {
	ctx, cancel := context.WithTimeout(parent, 8*time.Second)
	defer cancel()
	var cursor []interface{}
	seen := map[string]bool{}
	bytes := 0
	source := elastic.NewFetchSourceContext(true).Include("userId", "agentId", "gameId", "currency", "complete", "playedDate", "hash", "bet", "win", "balance", "log")
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		search := es.es.Search().Index(esindex.Settlement()).Query(filter).Size(100).
			Sort("playedDate", false).Sort("hash.keyword", false).FetchSourceContext(source)
		if len(cursor) > 0 {
			search = search.SearchAfter(cursor...)
		}
		response, err := search.Do(ctx)
		if err != nil {
			return err
		}
		if response == nil || response.Hits == nil {
			return fmt.Errorf("invalid settlement search response")
		}
		hits := response.Hits.Hits
		for _, hit := range hits {
			if hit == nil {
				return fmt.Errorf("nil original settlement document")
			}
			bytes += len(hit.Source)
			if bytes > settlementScanBytes {
				return fmt.Errorf("settlement history exceeds bounded response size")
			}
			done, err := visit(hit)
			if err != nil {
				return err
			}
			if done {
				return nil
			}
		}
		if len(hits) < 100 {
			return nil
		}
		last := hits[len(hits)-1]
		if len(last.Sort) != 2 {
			return fmt.Errorf("settlement history sort cursor missing")
		}
		encoded, err := json.Marshal(last.Sort)
		if err != nil {
			return err
		}
		key := string(encoded)
		if seen[key] {
			return fmt.Errorf("settlement history cursor did not advance")
		}
		seen[key] = true
		cursor = append([]interface{}(nil), last.Sort...)
	}
}
