package tasks

import (
	"encoding/json"
	"testing"
)

func TestClipResponse_JSONRoundTrip(t *testing.T) {
	resp := ClipResponse{
		Description: "desc",
		Contents: []ClipItem{
			{QID: "q1", QLibID: "ql1", VideoURL: "v", ImageURL: "i"},
		},
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var out ClipResponse
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if out.Description != resp.Description {
		t.Fatalf("description mismatch")
	}
	if len(out.Contents) != 1 || out.Contents[0].QID != "q1" {
		t.Fatalf("contents mismatch: %+v", out.Contents)
	}
}
