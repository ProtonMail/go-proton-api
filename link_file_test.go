package proton

import (
	"encoding/json"
	"testing"
)

func ptr(s string) *string { return &s }

func TestCreateRevisionReq_MarshalJSON_Empty(t *testing.T) {
	req := CreateRevisionReq{}
	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if string(data) != "{}" {
		t.Fatalf("expected {}, got %s", data)
	}
}

func TestCreateRevisionReq_MarshalJSON_WithCurrentRevisionID(t *testing.T) {
	req := CreateRevisionReq{CurrentRevisionID: ptr("rev-abc-123")}
	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	expected := `{"CurrentRevisionID":"rev-abc-123"}`
	if string(data) != expected {
		t.Fatalf("expected %s, got %s", expected, data)
	}
}

func TestCreateRevisionReq_MarshalJSON_NilCurrentRevisionID(t *testing.T) {
	req := CreateRevisionReq{CurrentRevisionID: nil}
	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if string(data) != "{}" {
		t.Fatalf("expected {}, got %s", data)
	}
}
