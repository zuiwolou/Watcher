package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"

	pushv1 "watcher/contracts/gen/push/v1"
	"watcher/backend/pkg/bus"
	"watcher/backend/pkg/canonical"
	"watcher/backend/pkg/logx"
	"watcher/backend/pkg/response"
)

type Hub struct {
	mu   sync.RWMutex
	conns map[*websocket.Conn]struct{}
}

func NewHub() *Hub { return &Hub{conns: map[*websocket.Conn]struct{}{}} }

func (h *Hub) Add(c *websocket.Conn) {
	h.mu.Lock()
	h.conns[c] = struct{}{}
	h.mu.Unlock()
}

func (h *Hub) Remove(c *websocket.Conn) {
	h.mu.Lock()
	delete(h.conns, c)
	h.mu.Unlock()
}

func (h *Hub) Broadcast(b []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for c := range h.conns {
		_ = c.WriteMessage(websocket.BinaryMessage, b)
	}
}

var upgrader = websocket.Upgrader{CheckOrigin: checkOrigin}

func checkOrigin(r *http.Request) bool {
	allow := os.Getenv("WATCHER_WS_ORIGINS") // comma-separated; empty = allow all (dev)
	if allow == "" || allow == "*" {
		return true
	}
	origin := r.Header.Get("Origin")
	if origin == "" {
		return true
	}
	for _, o := range splitCSV(allow) {
		if o == origin {
			return true
		}
	}
	return false
}

func splitCSV(s string) []string {
	var out []string
	start := 0
	for i := 0; i <= len(s); i++ {
		if i == len(s) || s[i] == ',' {
			part := trimSpace(s[start:i])
			if part != "" {
				out = append(out, part)
			}
			start = i + 1
		}
	}
	return out
}

func trimSpace(s string) string {
	for len(s) > 0 && (s[0] == ' ' || s[0] == '\t') {
		s = s[1:]
	}
	for len(s) > 0 && (s[len(s)-1] == ' ' || s[len(s)-1] == '\t') {
		s = s[:len(s)-1]
	}
	return s
}

func main() {
	log, _ := logx.New(env("WATCHER_ENV", "dev"))
	defer log.Sync() //nolint:errcheck
	hub := NewHub()

	lookupd := env("WATCHER_NSQ_LOOKUPD", "127.0.0.1:4161")
	broadcastProto := func(env *pushv1.PushEnvelope) {
		b, err := proto.Marshal(env)
		if err != nil {
			return
		}
		hub.Broadcast(b)
	}
	if cons, err := bus.Subscribe(lookupd, bus.TopicCanonical, "push", func(e bus.Envelope) error {
		var m canonical.Message
		if err := json.Unmarshal(e.Payload, &m); err != nil {
			return nil
		}
		broadcastProto(toEnvelope(m))
		return nil
	}); err != nil {
		log.Warn("nsq canonical skip", zap.Error(err))
	} else {
		defer cons.Stop()
	}
	if cons, err := bus.Subscribe(lookupd, bus.TopicAlarm, "push-alarm", func(e bus.Envelope) error {
		var raw map[string]any
		if err := json.Unmarshal(e.Payload, &raw); err != nil {
			return nil
		}
		broadcastProto(alarmEnvelope(raw))
		return nil
	}); err != nil {
		log.Warn("nsq alarm skip", zap.Error(err))
	} else {
		defer cons.Stop()
	}

	r := gin.New()
	r.Use(gin.Recovery())
	r.GET("/healthz", func(c *gin.Context) { c.JSON(200, gin.H{"status": "ok"}) })

	r.POST("/internal/publish", func(c *gin.Context) {
		raw, err := c.GetRawData()
		if err != nil {
			response.Fail(c, 400, "VALIDATION_ERROR", err.Error())
			return
		}
		hub.Broadcast(raw)
		response.OK(c, gin.H{"ok": true})
	})

	r.GET("/v1/stream", func(c *gin.Context) {
		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			return
		}
		hub.Add(conn)
		defer func() { hub.Remove(conn); _ = conn.Close() }()
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	})

	addr := env("WATCHER_PUSH_ADDR", ":8083")
	log.Info("push listening", zap.String("addr", addr))
	_ = r.Run(addr)
}

func alarmEnvelope(raw map[string]any) *pushv1.PushEnvelope {
	a := &pushv1.Alarm{}
	if v, ok := raw["alarmId"]; ok {
		a.AlarmId = stringify(v)
	}
	if v, ok := raw["metric"].(string); ok {
		a.Metric = v
	}
	if v, ok := raw["title"].(string); ok {
		a.Title = v
	}
	if v, ok := raw["deviceId"].(string); ok {
		_ = v
	}
	if v, ok := asFloat64(raw["value"]); ok {
		a.Value = v
	}
	if v, ok := asFloat64(raw["threshold"]); ok {
		a.Threshold = v
	}
	a.Status = "raised"
	a.Level = "important"
	if a.Title == "" && a.Metric != "" {
		a.Title = a.Metric + " 超阈"
	}
	wh, _ := raw["warehouseId"].(string)
	dev, _ := raw["deviceId"].(string)
	return &pushv1.PushEnvelope{
		Ver: 1, MsgType: pushv1.MsgType_ALARM, MsgId: a.AlarmId,
		TsUnixMs: timeNowMs(), WarehouseId: wh, DeviceId: dev,
		Payload: &pushv1.PushEnvelope_Alarm{Alarm: a},
	}
}

func stringify(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case float64:
		return fmt.Sprintf("%.0f", t)
	case json.Number:
		return t.String()
	default:
		return fmt.Sprint(v)
	}
}

func asFloat64(v any) (float64, bool) {
	switch t := v.(type) {
	case float64:
		return t, true
	case int:
		return float64(t), true
	case json.Number:
		f, err := t.Float64()
		return f, err == nil
	default:
		return 0, false
	}
}

func timeNowMs() int64 {
	return time.Now().UnixMilli()
}

func toEnvelope(m canonical.Message) *pushv1.PushEnvelope {
	e := &pushv1.PushEnvelope{
		Ver: 1, MsgId: m.MsgID, TsUnixMs: m.TsUnixMs,
		WarehouseId: m.WarehouseID, DeviceId: m.DeviceID,
	}
	switch m.MsgType {
	case "location":
		e.MsgType = pushv1.MsgType_LOCATION
		if m.Location != nil {
			e.Payload = &pushv1.PushEnvelope_Location{Location: &pushv1.Location{
				Lat: m.Location.Lat, Lng: m.Location.Lng, Speed: m.Location.Speed,
				Heading: m.Location.Heading, Accuracy: m.Location.Accuracy, UnitId: m.Location.UnitID,
			}}
		}
	default:
		e.MsgType = pushv1.MsgType_TELEMETRY
		t := &pushv1.Telemetry{Custom: map[string]float64{}}
		for k, v := range m.Metrics {
			switch k {
			case "temperature":
				t.Temperature = v
			case "humidity":
				t.Humidity = v
			case "waterLeak":
				t.WaterLeak = v
			case "intrusion":
				t.Intrusion = v
			default:
				t.Custom[k] = v
			}
		}
		e.Payload = &pushv1.PushEnvelope_Telemetry{Telemetry: t}
	}
	return e
}

func env(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
