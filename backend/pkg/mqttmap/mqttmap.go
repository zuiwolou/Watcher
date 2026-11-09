package mqttmap

import (
	"encoding/json"
	"fmt"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"watcher/backend/pkg/canonical"
)

// Mapping describes how a raw MQTT JSON payload maps into Canonical.
// Example:
//
//	{
//	  "warehouseId": "whId",
//	  "deviceId": "devId",
//	  "msgType": "type",
//	  "metrics": {"temperature":"temp","humidity":"hum"},
//	  "location": {"lat":"lat","lng":"lng"}
//	}
type Mapping struct {
	WarehouseID string            `json:"warehouseId"`
	DeviceID    string            `json:"deviceId"`
	MsgType     string            `json:"msgType"`
	Metrics     map[string]string `json:"metrics"`
	Location    map[string]string `json:"location"`
	Defaults    map[string]any    `json:"defaults"`
}

func ParseMapping(mappingJSON string) (Mapping, error) {
	var m Mapping
	if strings.TrimSpace(mappingJSON) == "" {
		return DefaultMapping(), nil
	}
	if err := json.Unmarshal([]byte(mappingJSON), &m); err != nil {
		return m, fmt.Errorf("mappingJson: %w", err)
	}
	if m.WarehouseID == "" {
		m.WarehouseID = "warehouseId"
	}
	if m.DeviceID == "" {
		m.DeviceID = "deviceId"
	}
	if len(m.Metrics) == 0 {
		m.Metrics = map[string]string{
			"temperature": "temperature", "humidity": "humidity",
			"waterLeak": "waterLeak", "intrusion": "intrusion",
		}
	}
	return m, nil
}

func DefaultMapping() Mapping {
	return Mapping{
		WarehouseID: "warehouseId",
		DeviceID:    "deviceId",
		MsgType:     "msgType",
		Metrics: map[string]string{
			"temperature": "temperature", "humidity": "humidity",
			"waterLeak": "waterLeak", "intrusion": "intrusion",
		},
		Location: map[string]string{"lat": "lat", "lng": "lng"},
	}
}

// Apply converts raw payload bytes into a Canonical message using mapping.
func Apply(mappingJSON string, raw []byte) (canonical.Message, error) {
	var m canonical.Message
	if err := json.Unmarshal(raw, &m); err == nil && m.MsgID != "" {
		if m.Ver == 0 {
			m.Ver = 1
		}
		return m, nil
	}
	var flat map[string]any
	if err := json.Unmarshal(raw, &flat); err != nil {
		return m, err
	}
	mp, err := ParseMapping(mappingJSON)
	if err != nil {
		return m, err
	}
	for k, v := range mp.Defaults {
		if _, ok := flat[k]; !ok {
			flat[k] = v
		}
	}
	m = canonical.Message{
		Ver: 1, MsgID: uuid.NewString(), MsgType: "telemetry",
		TsUnixMs: time.Now().UnixMilli(), Metrics: map[string]float64{},
	}
	if v, ok := asString(lookup(flat, mp.WarehouseID)); ok {
		m.WarehouseID = v
	}
	if v, ok := asString(lookup(flat, mp.DeviceID)); ok {
		m.DeviceID = v
	}
	if mp.MsgType != "" {
		if v, ok := asString(lookup(flat, mp.MsgType)); ok && v != "" {
			m.MsgType = v
		}
	}
	for canonKey, src := range mp.Metrics {
		if num, ok := asFloat(lookup(flat, src)); ok {
			m.Metrics[canonKey] = num
		}
	}
	latKey, lngKey := "lat", "lng"
	if mp.Location != nil {
		if v := mp.Location["lat"]; v != "" {
			latKey = v
		}
		if v := mp.Location["lng"]; v != "" {
			lngKey = v
		}
	}
	if lat, ok := asFloat(lookup(flat, latKey)); ok {
		lng, _ := asFloat(lookup(flat, lngKey))
		m.MsgType = "location"
		m.Location = &canonical.Location{Lat: lat, Lng: lng}
	}
	if m.WarehouseID == "" || m.DeviceID == "" {
		return m, fmt.Errorf("warehouseId and deviceId required")
	}
	return m, nil
}

// TopicMatch supports MQTT-style + and # wildcards, and Go path.Match patterns.
func TopicMatch(pattern, topic string) bool {
	if pattern == "" || pattern == "#" {
		return true
	}
	if pattern == topic {
		return true
	}
	if ok, _ := path.Match(pattern, topic); ok {
		return true
	}
	pp := strings.Split(pattern, "/")
	tt := strings.Split(topic, "/")
	for i := 0; i < len(pp); i++ {
		if i >= len(tt) {
			return false
		}
		if pp[i] == "#" {
			return true
		}
		if pp[i] == "+" {
			continue
		}
		if pp[i] != tt[i] {
			return false
		}
	}
	return len(pp) == len(tt)
}

func lookup(m map[string]any, key string) any {
	if key == "" {
		return nil
	}
	if v, ok := m[key]; ok {
		return v
	}
	// dotted path a.b
	parts := strings.Split(key, ".")
	var cur any = m
	for _, p := range parts {
		mm, ok := cur.(map[string]any)
		if !ok {
			return nil
		}
		cur, ok = mm[p]
		if !ok {
			return nil
		}
	}
	return cur
}

func asString(v any) (string, bool) {
	switch t := v.(type) {
	case string:
		return t, true
	case float64:
		return strconv.FormatFloat(t, 'f', -1, 64), true
	case json.Number:
		return t.String(), true
	default:
		return "", false
	}
}

func asFloat(v any) (float64, bool) {
	switch t := v.(type) {
	case float64:
		return t, true
	case json.Number:
		f, err := t.Float64()
		return f, err == nil
	case string:
		f, err := strconv.ParseFloat(t, 64)
		return f, err == nil
	default:
		return 0, false
	}
}
