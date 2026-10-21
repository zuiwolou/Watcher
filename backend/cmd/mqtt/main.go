package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"watcher/backend/pkg/bus"
	"watcher/backend/pkg/canonical"
	"watcher/backend/pkg/logx"
)

func main() {
	log, _ := logx.New(env("WATCHER_ENV", "dev"))
	defer log.Sync() //nolint:errcheck

	broker := env("WATCHER_MQTT_BROKER", "tcp://127.0.0.1:1883")
	topic := env("WATCHER_MQTT_TOPIC", "watcher/+/telemetry")
	historyURL := env("WATCHER_HISTORY_URL", "http://127.0.0.1:8082/internal/ingest")
	webEvalURL := env("WATCHER_WEB_EVAL_URL", "") // optional: full evaluate URL with token not ideal; skip
	nsqd := env("WATCHER_NSQD", "127.0.0.1:4150")

	var pub *bus.Publisher
	if p, err := bus.NewPublisher(nsqd); err != nil {
		log.Warn("nsq publisher skip", zap.Error(err))
	} else {
		pub = p
		defer pub.Stop()
	}

	opts := mqtt.NewClientOptions().AddBroker(broker).SetClientID("watcher-mqtt-" + uuid.NewString()[:8])
	opts.SetAutoReconnect(true)
	opts.OnConnect = func(c mqtt.Client) {
		log.Info("mqtt connected", zap.String("broker", broker))
		token := c.Subscribe(topic, 1, func(_ mqtt.Client, msg mqtt.Message) {
			m, err := mapPayload(msg.Payload())
			if err != nil {
				log.Warn("map payload", zap.Error(err))
				return
			}
			if pub != nil {
				_ = pub.Publish(bus.TopicCanonical, "svc-mqtt", m)
			}
			b, _ := json.Marshal(m)
			resp, err := http.Post(historyURL, "application/json", bytes.NewReader(b))
			if err == nil {
				_ = resp.Body.Close()
			}
			if webEvalURL != "" && m.MsgType != "location" {
				body, _ := json.Marshal(map[string]any{
					"warehouseId": m.WarehouseID, "deviceId": m.DeviceID, "metrics": m.Metrics,
				})
				r2, err := http.Post(webEvalURL, "application/json", bytes.NewReader(body))
				if err == nil {
					_ = r2.Body.Close()
				}
			}
			log.Info("ingested", zap.String("msgId", m.MsgID), zap.String("type", m.MsgType))
		})
		token.Wait()
	}
	client := mqtt.NewClient(opts)
	if t := client.Connect(); t.Wait() && t.Error() != nil {
		log.Fatal("mqtt connect", zap.Error(t.Error()))
	}

	select {}
}

// mapPayload accepts either Canonical JSON or simple {warehouseId,deviceId,temperature,...}
func mapPayload(raw []byte) (canonical.Message, error) {
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
	m = canonical.Message{
		Ver: 1, MsgID: uuid.NewString(), MsgType: "telemetry",
		TsUnixMs: time.Now().UnixMilli(), Metrics: map[string]float64{},
	}
	if v, ok := flat["warehouseId"].(string); ok {
		m.WarehouseID = v
	}
	if v, ok := flat["deviceId"].(string); ok {
		m.DeviceID = v
	}
	if v, ok := flat["msgType"].(string); ok {
		m.MsgType = v
	}
	for _, k := range []string{"temperature", "humidity", "waterLeak", "intrusion"} {
		if num, ok := asFloat(flat[k]); ok {
			m.Metrics[k] = num
		}
	}
	if lat, ok1 := asFloat(flat["lat"]); ok1 {
		lng, _ := asFloat(flat["lng"])
		m.MsgType = "location"
		m.Location = &canonical.Location{Lat: lat, Lng: lng}
	}
	if m.WarehouseID == "" {
		m.WarehouseID = "WH01"
	}
	if m.DeviceID == "" {
		m.DeviceID = "DEV01"
	}
	return m, nil
}

func asFloat(v any) (float64, bool) {
	switch t := v.(type) {
	case float64:
		return t, true
	case json.Number:
		f, err := t.Float64()
		return f, err == nil
	default:
		return 0, false
	}
}

func env(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
