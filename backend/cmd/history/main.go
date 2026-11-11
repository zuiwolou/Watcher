package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"watcher/backend/internal/history/store"
	"watcher/backend/internal/webhttp/middleware"
	"watcher/backend/pkg/bus"
	"watcher/backend/pkg/canonical"
	"watcher/backend/pkg/config"
	"watcher/backend/pkg/database"
	"watcher/backend/pkg/logx"
	"watcher/backend/pkg/response"
)

func main() {
	cfg, _ := config.Load("history")
	log, _ := logx.New(cfg.Env)
	defer log.Sync() //nolint:errcheck

	db, err := database.OpenMySQL(cfg.HistoryDSN)
	if err != nil {
		log.Fatal("mysql", zap.Error(err))
	}
	st := store.New(db)
	if err := st.AutoMigrate(); err != nil {
		log.Fatal("migrate", zap.Error(err))
	}

	lookupd := env("WATCHER_NSQ_LOOKUPD", "127.0.0.1:4161")
	if cons, err := bus.Subscribe(lookupd, bus.TopicCanonical, "history", func(e bus.Envelope) error {
		var m canonical.Message
		if err := json.Unmarshal(e.Payload, &m); err != nil {
			return nil
		}
		return st.IngestCanonical(context.Background(), m)
	}); err != nil {
		log.Warn("nsq subscribe skip", zap.Error(err))
	} else {
		defer cons.Stop()
		log.Info("nsq subscribed", zap.String("topic", bus.TopicCanonical))
	}
	// Alarm snapshots: retain rows in hist_alarm_snapshots; TTL cleanup is a later iteration.
	if cons, err := bus.Subscribe(lookupd, bus.TopicAlarm, "history-alarm", func(e bus.Envelope) error {
		var payload map[string]any
		if err := json.Unmarshal(e.Payload, &payload); err != nil {
			return nil
		}
		alarmID := ""
		if v, ok := payload["alarmId"]; ok {
			alarmID = fmt.Sprint(v)
		}
		status := "raised"
		if v, ok := payload["status"].(string); ok && v != "" {
			status = v
		}
		msgID := e.MsgID
		if msgID == "" {
			msgID = alarmID + ":raised"
		}
		return st.SaveAlarm(context.Background(), alarmID, status, payload, msgID)
	}); err != nil {
		log.Warn("nsq alarm subscribe skip", zap.Error(err))
	} else {
		defer cons.Stop()
		log.Info("nsq subscribed", zap.String("topic", bus.TopicAlarm))
	}

	r := gin.New()
	r.Use(gin.Recovery())
	r.GET("/healthz", func(c *gin.Context) { c.JSON(200, gin.H{"status": "ok"}) })

	internal := r.Group("/internal")
	internal.Use(middleware.InternalToken())
	internal.POST("/ingest", func(c *gin.Context) {
		var m canonical.Message
		if err := c.ShouldBindJSON(&m); err != nil {
			response.Fail(c, 400, "VALIDATION_ERROR", err.Error())
			return
		}
		if err := st.IngestCanonical(c.Request.Context(), m); err != nil {
			response.Fail(c, 500, "INTERNAL_ERROR", err.Error())
			return
		}
		response.OK(c, gin.H{"ok": true})
	})

	api := r.Group("/api/v1/history")
	api.Use(middleware.InternalToken())
	api.GET("/telemetry", func(c *gin.Context) {
		from := parseTime(c.DefaultQuery("from", time.Now().Add(-24*time.Hour).Format(time.RFC3339)))
		to := parseTime(c.DefaultQuery("to", time.Now().Format(time.RFC3339)))
		list, err := st.QueryTelemetry(c.Request.Context(), c.Query("warehouseId"), c.Query("metric"), from, to, 500)
		if err != nil {
			response.Fail(c, 500, "INTERNAL_ERROR", err.Error())
			return
		}
		response.OK(c, list)
	})

	api.GET("/tracks", func(c *gin.Context) {
		from := parseTime(c.DefaultQuery("from", time.Now().Add(-24*time.Hour).Format(time.RFC3339)))
		to := parseTime(c.DefaultQuery("to", time.Now().Format(time.RFC3339)))
		list, err := st.QueryTrack(c.Request.Context(), c.Query("warehouseId"), c.Query("deviceId"), from, to, 500)
		if err != nil {
			response.Fail(c, 500, "INTERNAL_ERROR", err.Error())
			return
		}
		response.OK(c, list)
	})

	api.GET("/alarms", func(c *gin.Context) {
		from := parseTime(c.DefaultQuery("from", time.Now().Add(-24*time.Hour).Format(time.RFC3339)))
		to := parseTime(c.DefaultQuery("to", time.Now().Format(time.RFC3339)))
		list, err := st.QueryAlarms(c.Request.Context(), c.Query("alarmId"), from, to, 500)
		if err != nil {
			response.Fail(c, 500, "INTERNAL_ERROR", err.Error())
			return
		}
		response.OK(c, list)
	})

	addr := env("WATCHER_HISTORY_ADDR", ":8082")
	log.Info("history listening", zap.String("addr", addr))
	_ = r.Run(addr)
}

func parseTime(s string) time.Time {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return time.Now()
	}
	return t
}

func env(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
