package main

import (
	"encoding/json"
	"net/http"
	"os"
	"sync"

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

var upgrader = websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}

func main() {
	log, _ := logx.New(env("WATCHER_ENV", "dev"))
	defer log.Sync() //nolint:errcheck
	hub := NewHub()

	lookupd := env("WATCHER_NSQ_LOOKUPD", "127.0.0.1:4161")
	if cons, err := bus.Subscribe(lookupd, bus.TopicCanonical, "push", func(e bus.Envelope) error {
		var m canonical.Message
		if err := json.Unmarshal(e.Payload, &m); err != nil {
			return nil
		}
		env := toEnvelope(m)
		b, err := proto.Marshal(env)
		if err != nil {
			return nil
		}
		hub.Broadcast(b)
		return nil
	}); err != nil {
		log.Warn("nsq skip", zap.Error(err))
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
