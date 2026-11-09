package mqttmap

import "testing"

func TestApplyDefault(t *testing.T) {
	raw := []byte(`{"warehouseId":"WH01","deviceId":"D1","temperature":12.5}`)
	m, err := Apply("", raw)
	if err != nil {
		t.Fatal(err)
	}
	if m.WarehouseID != "WH01" || m.DeviceID != "D1" {
		t.Fatalf("identity %+v", m)
	}
	if m.Metrics["temperature"] != 12.5 {
		t.Fatalf("metrics %+v", m.Metrics)
	}
}

func TestApplyCustomMapping(t *testing.T) {
	mapping := `{"warehouseId":"wh","deviceId":"dev","metrics":{"temperature":"t"}}`
	raw := []byte(`{"wh":"WH01","dev":"D2","t":9}`)
	m, err := Apply(mapping, raw)
	if err != nil {
		t.Fatal(err)
	}
	if m.DeviceID != "D2" || m.Metrics["temperature"] != 9 {
		t.Fatalf("%+v", m)
	}
}

func TestTopicMatch(t *testing.T) {
	if !TopicMatch("watcher/+/telemetry", "watcher/WH01/telemetry") {
		t.Fatal("expected match")
	}
	if TopicMatch("watcher/+/telemetry", "watcher/WH01/loc") {
		t.Fatal("should not match")
	}
}
