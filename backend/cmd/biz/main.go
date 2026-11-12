package main

import (
	"context"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"google.golang.org/grpc"

	"watcher/backend/internal/biz/grpcserver"
	bizsvc "watcher/backend/internal/biz/service"
	"watcher/backend/pkg/config"
	"watcher/backend/pkg/database"
	"watcher/backend/pkg/etcdx"
	"watcher/backend/pkg/logx"
	"watcher/backend/pkg/metricsx"
	bizv1 "watcher/contracts/gen/biz/v1"
)

// svc-biz 进程入口（BE-06）：HTTP 健康检查 + Outbox worker + gRPC BizService。
// webhttp 仍可同进程嵌入 *bizsvc.Service；拆分时拨号本进程 gRPC。
func main() {
	cfg, err := config.Load("biz")
	if err != nil {
		panic(err)
	}
	log, _ := logx.New(cfg.Env)
	defer log.Sync() //nolint:errcheck

	db, err := database.OpenMySQL(cfg.BizDSN)
	if err != nil {
		log.Fatal("mysql", zap.Error(err))
	}
	biz := bizsvc.New(db)
	_ = biz.AutoMigrate()

	if etcdCli, err := etcdx.Dial(); err != nil {
		log.Warn("etcd dial", zap.Error(err))
	} else if etcdCli != nil {
		defer etcdCli.Close()
		if v, err := etcdCli.GetString(context.Background(), etcdx.KeyGeofenceCooldown); err == nil && v != "" {
			if d, err := time.ParseDuration(v); err == nil && d > 0 {
				biz.SetAlarmSuppressWindow(d)
				log.Info("geofence cooldown from etcd", zap.Duration("cooldown", d))
			}
		}
	}

	go func() {
		t := time.NewTicker(3 * time.Second)
		defer t.Stop()
		for range t.C {
			n, err := biz.ProcessOutboxBatch(context.Background(), 20)
			if err != nil {
				log.Warn("outbox", zap.Error(err))
				continue
			}
			if n > 0 {
				log.Info("outbox delivered", zap.Int("n", n))
			}
		}
	}()

	r := gin.New()
	r.Use(gin.Recovery())
	r.GET("/healthz", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "ok", "service": "biz"}) })
	metricsx.Register(r)

	grpcAddr := cfg.GRPCAddr
	if grpcAddr == "" {
		grpcAddr = ":9091"
	}
	go func() {
		ln, err := net.Listen("tcp", grpcAddr)
		if err != nil {
			log.Warn("grpc listen skip", zap.Error(err))
			return
		}
		gs := grpc.NewServer()
		bizv1.RegisterBizServiceServer(gs, grpcserver.New(biz))
		log.Info("biz gRPC listening", zap.String("addr", grpcAddr))
		if err := gs.Serve(ln); err != nil {
			log.Warn("grpc serve", zap.Error(err))
		}
	}()

	addr := env("WATCHER_BIZ_HTTP_ADDR", ":8085")
	log.Info("biz http listening", zap.String("addr", addr))
	if err := r.Run(addr); err != nil {
		log.Fatal("run", zap.Error(err))
	}
}

func env(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
