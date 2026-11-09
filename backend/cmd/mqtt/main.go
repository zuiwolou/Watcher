package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"os"
	"sync"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/google/uuid"
	"go.uber.org/zap"

	bizsvc "watcher/backend/internal/biz/service"
	"watcher/backend/pkg/bus"
	"watcher/backend/pkg/config"
	"watcher/backend/pkg/database"
	"watcher/backend/pkg/logx"
	"watcher/backend/pkg/mqttmap"
)

type profileView struct {
	ID           uint64
	TopicPattern string
	MappingJSON  string
	Enabled      bool
}

type schemaStore struct {
	mu       sync.RWMutex
	profiles []profileView
}

func (s *schemaStore) reload(biz *bizsvc.Service, log *zap.Logger) {
	if biz == nil {
		return
	}
	list, err := biz.ListMqttSchemas(context.Background())
	if err != nil {
		log.Warn("reload mqtt schemas", zap.Error(err))
		return
	}
	views := make([]profileView, 0, len(list))
	for _, p := range list {
		if !p.Enabled {
			continue
		}
		views = append(views, profileView{ID: p.ID, TopicPattern: p.TopicPattern, MappingJSON: p.MappingJSON, Enabled: p.Enabled})
	}
	s.mu.Lock()
	s.profiles = views
	s.mu.Unlock()
	log.Info("mqtt schemas loaded", zap.Int("count", len(views)))
}

func (s *schemaStore) mappingForTopic(topic string) (string, uint64) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, p := range s.profiles {
		if mqttmap.TopicMatch(p.TopicPattern, topic) {
			return p.MappingJSON, p.ID
		}
	}
	return "", 0
}

func (s *schemaStore) mappingByID(id uint64) string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, p := range s.profiles {
		if p.ID == id {
			return p.MappingJSON
		}
	}
	return ""
}

func main() {
	log, _ := logx.New(env("WATCHER_ENV", "dev"))
	defer log.Sync() //nolint:errcheck

	broker := env("WATCHER_MQTT_BROKER", "tcp://127.0.0.1:1883")
	topic := env("WATCHER_MQTT_TOPIC", "watcher/+/telemetry")
	historyURL := env("WATCHER_HISTORY_URL", "http://127.0.0.1:8082/internal/ingest")
	webEvalURL := env("WATCHER_WEB_EVAL_URL", "http://127.0.0.1:8080/internal/monitor/evaluate")
	internalToken := env("WATCHER_INTERNAL_TOKEN", "dev-internal-token")
	nsqd := env("WATCHER_NSQD", "127.0.0.1:4150")
	nsqlookup := env("WATCHER_NSQLOOKUPD", "127.0.0.1:4161")

	store := &schemaStore{}
	var biz *bizsvc.Service
	if cfg, err := config.Load("mqtt"); err == nil {
		if d, err := database.OpenMySQL(cfg.MySQLDSN); err != nil {
			log.Warn("mysql skip (schema hot-reload disabled)", zap.Error(err))
		} else {
			biz = bizsvc.New(d)
			_ = biz.AutoMigrate()
			store.reload(biz, log)
		}
	}

	var pub *bus.Publisher
	if p, err := bus.NewPublisher(nsqd); err != nil {
		log.Warn("nsq publisher skip", zap.Error(err))
	} else {
		pub = p
		defer pub.Stop()
	}

	if biz != nil {
		if _, err := bus.Subscribe(nsqlookup, bus.TopicConfigMQTT, "mqtt-schema-reload", func(env bus.Envelope) error {
			log.Info("config.mqtt.updated received", zap.String("msgId", env.MsgID))
			store.reload(biz, log)
			return nil
		}); err != nil {
			log.Warn("subscribe config.mqtt.updated skip", zap.Error(err))
			go func() {
				t := time.NewTicker(30 * time.Second)
				defer t.Stop()
				for range t.C {
					store.reload(biz, log)
				}
			}()
		}
	}

	opts := mqtt.NewClientOptions().AddBroker(broker).SetClientID("watcher-mqtt-" + uuid.NewString()[:8])
	opts.SetAutoReconnect(true)
	opts.OnConnect = func(c mqtt.Client) {
		log.Info("mqtt connected", zap.String("broker", broker))
		token := c.Subscribe(topic, 1, func(_ mqtt.Client, msg mqtt.Message) {
			mapping, _ := store.mappingForTopic(msg.Topic())
			m, err := mqttmap.Apply(mapping, msg.Payload())
			if err != nil {
				log.Warn("map payload", zap.Error(err), zap.String("topic", msg.Topic()))
				return
			}
			if biz != nil && m.DeviceID != "" {
				if dev, err := biz.FindDeviceByCode(context.Background(), m.DeviceID); err == nil {
					if m.WarehouseID == "" && dev.WarehouseID != "" {
						m.WarehouseID = dev.WarehouseID
					}
					if mapping == "" && dev.SchemaProfileID != nil {
						if mj := store.mappingByID(*dev.SchemaProfileID); mj != "" {
							if remapped, err2 := mqttmap.Apply(mj, msg.Payload()); err2 == nil {
								m = remapped
								if m.WarehouseID == "" {
									m.WarehouseID = dev.WarehouseID
								}
							}
						}
					}
				}
			}
			if pub != nil {
				_ = pub.Publish(bus.TopicCanonical, "svc-mqtt", m)
			}
			b, _ := json.Marshal(m)
			resp, err := http.Post(historyURL, "application/json", bytes.NewReader(b))
			if err == nil {
				_ = resp.Body.Close()
			}
			if webEvalURL != "" && m.MsgType != "location" && len(m.Metrics) > 0 {
				body, _ := json.Marshal(map[string]any{
					"warehouseId": m.WarehouseID, "deviceId": m.DeviceID, "metrics": m.Metrics,
				})
				req, _ := http.NewRequest(http.MethodPost, webEvalURL, bytes.NewReader(body))
				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("X-Internal-Token", internalToken)
				r2, err := http.DefaultClient.Do(req)
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

func env(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
