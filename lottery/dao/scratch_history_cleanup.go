package dao

import (
	"context"
	"strings"
	"time"
)

// Scan only the retired materialized history namespace. The new three-minute
// query cache, outstanding tickets, ownership and finance keys are excluded.
func (rd *RedisDao) RemoveLegacyScratchHistory(cursor uint64) (uint64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	keys, next, err := rd.cli.Scan(ctx, cursor, "gogame:scratch:history*", 100).Result()
	if err != nil {
		return cursor, err
	}
	remove := make([]string, 0, len(keys))
	for _, key := range keys {
		if strings.HasPrefix(key, "gogame:scratch:history:") || strings.HasPrefix(key, "gogame:scratch:history-order:") {
			remove = append(remove, key)
		}
	}
	if len(remove) > 0 {
		if err := rd.cli.Unlink(ctx, remove...).Err(); err != nil {
			return cursor, err
		}
	}
	return next, nil
}
