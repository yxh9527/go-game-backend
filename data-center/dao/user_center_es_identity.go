package dao

import (
	"bytes"
	"context"
	"encoding/json"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func canonicalReportJSON(raw []byte) ([]byte, error) {
	var value any
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if err := decoder.Decode(&value); err != nil {
		return nil, err
	}
	return json.Marshal(value)
}

// An existing ES document is the original receipt. Verify it directly instead
// of treating any same-ID conflict as success or creating a MySQL copy.
func (es *ESDao) confirmOriginalCenterReport(ctx context.Context, index, id string, expected UserCenterRecord) error {
	stored, err := es.es.Get().Index(index).Id(id).Do(ctx)
	if err != nil {
		return err
	}
	if stored == nil || !stored.Found || stored.Id != id || len(stored.Source) > 2*1024*1024 {
		return status.Error(codes.Unavailable, "original ES report confirmation unavailable")
	}
	want, err := json.Marshal(expected)
	if err != nil {
		return err
	}
	want, err = canonicalReportJSON(want)
	if err != nil {
		return err
	}
	actual, err := canonicalReportJSON(stored.Source)
	if err != nil {
		return err
	}
	if !bytes.Equal(want, actual) {
		return status.Error(codes.FailedPrecondition, "settled report conflicts with original ES document")
	}
	return nil
}
