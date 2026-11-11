package service

import (
	"encoding/json"
	"testing"
)

func TestApplyFieldMap(t *testing.T) {
	raw := `{"alarmId":12,"warehouseId":"WH01","title":"越界"}`
	b, err := applyFieldMap(raw, map[string]string{
		"alarmId":     "alarm_id",
		"warehouseId": "wh_code",
	})
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatal(err)
	}
	if _, ok := m["alarmId"]; ok {
		t.Fatal("alarmId should be renamed")
	}
	if m["alarm_id"] != float64(12) {
		t.Fatalf("alarm_id=%v", m["alarm_id"])
	}
	if m["wh_code"] != "WH01" {
		t.Fatalf("wh_code=%v", m["wh_code"])
	}
	if m["title"] != "越界" {
		t.Fatalf("unmapped title lost: %v", m["title"])
	}
}
