package main

import (
	"context"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	bizsvc "watcher/backend/internal/biz/service"
	"watcher/backend/internal/webhttp/handler"
	"watcher/backend/internal/webhttp/iam"
	"watcher/backend/internal/webhttp/middleware"
	"watcher/backend/pkg/auth"
	"watcher/backend/pkg/authz"
	"watcher/backend/pkg/bus"
	"watcher/backend/pkg/config"
	"watcher/backend/pkg/database"
	"watcher/backend/pkg/ldapx"
	"watcher/backend/pkg/logx"
	"watcher/backend/pkg/storage"
)

func main() {
	cfg, err := config.Load("webhttp")
	if err != nil {
		panic(err)
	}
	log, err := logx.New(cfg.Env)
	if err != nil {
		panic(err)
	}
	defer log.Sync() //nolint:errcheck

	db, err := database.OpenMySQL(cfg.MySQLDSN)
	if err != nil {
		log.Fatal("mysql", zap.Error(err))
	}

	if err := iam.AutoMigrate(db); err != nil {
		log.Fatal("iam migrate", zap.Error(err))
	}
	enforcer, err := authz.NewEnforcer(db)
	if err != nil {
		log.Fatal("casbin", zap.Error(err))
	}
	if err := iam.SeedAdmin(db, enforcer, cfg.SeedAdminUser, cfg.SeedAdminPassword, cfg.DefaultWarehouseID); err != nil {
		log.Fatal("seed admin", zap.Error(err))
	}

	biz := bizsvc.New(db)
	if err := biz.AutoMigrate(); err != nil {
		log.Fatal("biz migrate", zap.Error(err))
	}
	if err := biz.SeedWarehouse(cfg.DefaultWarehouseID, cfg.DefaultWarehouseID, "默认暂存仓"); err != nil {
		log.Fatal("seed warehouse", zap.Error(err))
	}
	if err := biz.SeedDefaults(); err != nil {
		log.Fatal("seed defaults", zap.Error(err))
	}

	nsqd := envOr("WATCHER_NSQD", "127.0.0.1:4150")
	if pub, err := bus.NewPublisher(nsqd); err != nil {
		log.Warn("nsq publisher skip", zap.Error(err))
		biz.SetPublisher(bus.NoopPublisher{})
	} else {
		biz.SetPublisher(pub)
		defer pub.Stop()
	}

	// Outbox worker: true delivery of regulatory integration queue
	go func() {
		t := time.NewTicker(3 * time.Second)
		defer t.Stop()
		for range t.C {
			n, err := biz.ProcessOutboxBatch(context.Background(), 20)
			if err != nil {
				log.Warn("outbox batch", zap.Error(err))
				continue
			}
			if n > 0 {
				log.Info("outbox delivered", zap.Int("count", n))
			}
		}
	}()

	var minioClient *storage.MinIO
	minioClient, err = storage.NewMinIO(storage.Config{
		Endpoint: cfg.MinIOEndpoint, AccessKey: cfg.MinIOAccessKey, SecretKey: cfg.MinIOSecretKey,
		Bucket: cfg.MinIOBucket, UseSSL: cfg.MinIOUseSSL, PublicURL: cfg.MinIOPublicURL,
	})
	if err != nil {
		log.Warn("minio unavailable", zap.Error(err))
		minioClient = nil
	}

	ldapClient := ldapx.New(ldapx.Config{
		Enabled: cfg.LDAPEnabled, URL: cfg.LDAPURL, BindDN: cfg.LDAPBindDN, BindPass: cfg.LDAPBindPass,
		BaseDN: cfg.LDAPBaseDN, UserFilter: cfg.LDAPFilter, StartTLS: cfg.LDAPStartTLS,
	})

	jwtMgr := auth.NewManager(cfg.JWTSecret, cfg.AccessTokenTTL, cfg.RefreshTokenTTL)
	authH := handler.NewAuthHandler(db, jwtMgr, enforcer, ldapClient, cfg.DefaultWarehouseID)
	adminH := handler.NewAdminHandler(db, enforcer)
	auditH := handler.NewAuditHandler(db)
	bizH := handler.NewBizHandler(biz, db)
	apprH := handler.NewApprovalHandler(biz, enforcer, db)
	mediaH := handler.NewMediaHandler(minioClient)

	r := gin.New()
	r.Use(gin.Recovery(), middleware.RequestID())
	r.GET("/healthz", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "ok"}) })

	internal := r.Group("/internal")
	internal.Use(middleware.InternalToken())
	internal.POST("/monitor/evaluate", bizH.InternalEvaluateTelemetry)

	v1 := r.Group("/api/v1")
	{
		v1.POST("/auth/login", authH.Login)
		v1.POST("/auth/refresh", authH.Refresh)

		authed := v1.Group("")
		authed.Use(middleware.JWT(jwtMgr))
		authed.GET("/me", authH.Me)
		authed.GET("/me/permissions", authH.Me)
		authed.POST("/auth/change-password", authH.ChangePassword)

		// Admin-only: IAM / schema / integrations / devices / outbox
		admin := authed.Group("")
		admin.Use(middleware.RequireClient("admin"))
		admin.GET("/admin/permissions", middleware.RequirePerm(enforcer, "role", "view"), adminH.ListPermissions)
		admin.GET("/admin/users", middleware.RequirePerm(enforcer, "user", "view"), adminH.ListUsers)
		admin.POST("/admin/users", middleware.RequirePerm(enforcer, "user", "edit"), adminH.CreateUser)
		admin.PUT("/admin/users/:id", middleware.RequirePerm(enforcer, "user", "edit"), adminH.UpdateUser)
		admin.GET("/admin/roles", middleware.RequirePerm(enforcer, "role", "view"), adminH.ListRoles)
		admin.POST("/admin/roles", middleware.RequirePerm(enforcer, "role", "edit"), adminH.CreateRole)
		admin.PUT("/admin/roles/:id", middleware.RequirePerm(enforcer, "role", "edit"), adminH.UpdateRole)
		admin.GET("/admin/audit", middleware.RequirePerm(enforcer, "audit", "view"), auditH.List)
		admin.GET("/mqtt/schemas", middleware.RequirePerm(enforcer, "mqtt", "schema:edit"), bizH.ListMqttSchemas)
		admin.PUT("/mqtt/schemas", middleware.RequirePerm(enforcer, "mqtt", "schema:edit"), bizH.UpsertMqttSchema)
		admin.POST("/mqtt/schemas/dry-run", middleware.RequirePerm(enforcer, "mqtt", "schema:edit"), bizH.DryRunMqttSchema)
		admin.GET("/integrations", middleware.RequirePerm(enforcer, "openapi", "config"), bizH.ListIntegrations)
		admin.PUT("/integrations", middleware.RequirePerm(enforcer, "openapi", "config"), bizH.UpsertIntegration)
		admin.GET("/openapi/outbox", middleware.RequirePerm(enforcer, "openapi", "config"), bizH.ListOutbox)
		admin.POST("/openapi/outbox/:id/retry", middleware.RequirePerm(enforcer, "openapi", "config"), bizH.RetryOutbox)
		admin.GET("/devices", middleware.RequirePerm(enforcer, "device", "edit"), bizH.ListDevices)
		admin.POST("/devices", middleware.RequirePerm(enforcer, "device", "edit"), bizH.CreateDevice)
		admin.PUT("/devices/:id", middleware.RequirePerm(enforcer, "device", "edit"), bizH.UpdateDevice)
		admin.PUT("/devices/:id/mqtt-binding", middleware.RequirePerm(enforcer, "device", "edit"), bizH.BindDeviceMqtt)

		// Business surface: portal / app / admin clients
		bizAPI := authed.Group("")
		bizAPI.Use(middleware.RequireAnyClient("admin", "portal", "app"))
		bizAPI.POST("/media/upload-url", mediaH.UploadURL)

		bizAPI.GET("/warehouses", middleware.RequirePerm(enforcer, "warehouse", "view"), bizH.ListWarehouses)
		bizAPI.GET("/materials", middleware.RequirePerm(enforcer, "material", "view"), bizH.ListMaterials)
		bizAPI.POST("/materials", middleware.RequirePerm(enforcer, "material", "edit"), bizH.CreateMaterial)
		bizAPI.GET("/stocks", middleware.RequirePerm(enforcer, "material", "view"), bizH.ListStocks)
		bizAPI.GET("/unique-codes/:code", middleware.RequirePerm(enforcer, "material", "view"), bizH.GetUniqueCode)

		bizAPI.GET("/inbound/orders", middleware.RequirePerm(enforcer, "warehouse", "view"), bizH.ListInbound)
		bizAPI.POST("/inbound/orders", middleware.RequirePerm(enforcer, "inbound", "edit"), bizH.CreateInbound)
		bizAPI.POST("/inbound/orders/:id/issue-codes", middleware.RequirePerm(enforcer, "code", "issue"), bizH.IssueCodes)
		bizAPI.POST("/inbound/orders/:id/confirm", middleware.RequirePerm(enforcer, "inbound", "confirm"), bizH.ConfirmInbound)

		bizAPI.GET("/outbound/orders", middleware.RequirePerm(enforcer, "warehouse", "view"), bizH.ListOutbound)
		bizAPI.POST("/outbound/orders", middleware.RequirePerm(enforcer, "outbound", "edit"), bizH.CreateOutbound)
		bizAPI.POST("/outbound/orders/:id/confirm", middleware.RequirePerm(enforcer, "outbound", "confirm"), bizH.ConfirmOutbound)

		bizAPI.GET("/transport/units", middleware.RequirePerm(enforcer, "transport", "view"), bizH.ListUnits)
		bizAPI.POST("/transport/units", middleware.RequirePerm(enforcer, "unit", "edit"), bizH.CreateUnit)
		bizAPI.GET("/transport/tasks", middleware.RequirePerm(enforcer, "transport", "view"), bizH.ListTransportTasks)
		bizAPI.POST("/transport/tasks", middleware.RequirePerm(enforcer, "transport", "edit"), bizH.CreateTransportTask)
		bizAPI.GET("/transport/tasks/:id/checkins", middleware.RequirePerm(enforcer, "transport", "view"), bizH.ListCheckins)
		bizAPI.POST("/transport/checkins", middleware.RequirePerm(enforcer, "checkin", "exec"), bizH.SubmitCheckin)

		bizAPI.GET("/inventory/tasks", middleware.RequirePerm(enforcer, "inventory", "view"), bizH.ListInventoryTasks)
		bizAPI.POST("/inventory/tasks", middleware.RequirePerm(enforcer, "inventory", "create"), bizH.CreateInventoryTask)
		bizAPI.POST("/inventory/tasks/:id/scan", middleware.RequirePerm(enforcer, "inventory", "scan"), bizH.ScanInventory)
		bizAPI.POST("/inventory/tasks/:id/submit", middleware.RequirePerm(enforcer, "inventory", "scan"), bizH.SubmitInventory)
		bizAPI.POST("/inventory/tasks/:id/post", middleware.RequirePerm(enforcer, "inventory", "adjust"), bizH.PostInventory)
		bizAPI.GET("/inventory/tasks/:id/lines", middleware.RequirePerm(enforcer, "inventory", "view"), bizH.ListInventoryLines)

		bizAPI.GET("/approvals/templates", middleware.RequirePerm(enforcer, "approval", "view"), apprH.ListTemplates)
		bizAPI.PUT("/approvals/templates", middleware.RequirePerm(enforcer, "approval", "act"), apprH.UpsertTemplate)
		bizAPI.GET("/approvals/todo", middleware.RequirePerm(enforcer, "approval", "view"), apprH.ListTodo)
		bizAPI.GET("/approvals/orders", middleware.RequirePerm(enforcer, "approval", "view"), apprH.ListOrders)
		bizAPI.GET("/approvals/orders/:id", middleware.RequirePerm(enforcer, "approval", "view"), apprH.GetOrder)
		bizAPI.POST("/approvals/:id/act", middleware.RequirePerm(enforcer, "approval", "act"), apprH.Act)
		bizAPI.POST("/stocks/adjustments", middleware.RequirePerm(enforcer, "stock", "adjust"), apprH.CreateAdjustment)
		bizAPI.GET("/stocks/adjustments", middleware.RequirePerm(enforcer, "stock", "adjust"), apprH.ListAdjustments)

		bizAPI.GET("/inspection/plans", middleware.RequirePerm(enforcer, "inspection", "view"), bizH.ListInspPlans)
		bizAPI.POST("/inspection/plans", middleware.RequirePerm(enforcer, "inspection", "create"), bizH.CreateInspPlan)
		bizAPI.POST("/inspection/routes", middleware.RequirePerm(enforcer, "inspection", "create"), bizH.CreateInspRoute)
		bizAPI.POST("/inspection/points", middleware.RequirePerm(enforcer, "inspection", "create"), bizH.AddInspPoint)
		bizAPI.GET("/inspection/routes/:id/points", middleware.RequirePerm(enforcer, "inspection", "view"), bizH.ListInspPoints)
		bizAPI.GET("/inspection/tasks", middleware.RequirePerm(enforcer, "inspection", "view"), bizH.ListInspTasks)
		bizAPI.POST("/inspection/tasks", middleware.RequirePerm(enforcer, "inspection", "create"), bizH.CreateInspTask)
		bizAPI.POST("/inspection/tasks/:id/checkin", middleware.RequirePerm(enforcer, "inspection", "exec"), bizH.InspCheckin)

		bizAPI.GET("/alarms", middleware.RequirePerm(enforcer, "alarm", "view"), bizH.ListAlarms)
		bizAPI.POST("/alarms/:id/ack", middleware.RequirePerm(enforcer, "alarm", "ack"), bizH.AckAlarm)
		bizAPI.POST("/alarms/:id/close", middleware.RequirePerm(enforcer, "alarm", "ack"), bizH.CloseAlarm)
		bizAPI.GET("/thresholds", middleware.RequirePerm(enforcer, "threshold", "edit"), bizH.ListThresholds)
		bizAPI.PUT("/thresholds/:warehouseId", middleware.RequirePerm(enforcer, "threshold", "edit"), bizH.UpsertThreshold)
		bizAPI.POST("/monitor/evaluate", middleware.RequirePerm(enforcer, "threshold", "edit"), bizH.EvaluateTelemetry)
		bizAPI.GET("/geofences", middleware.RequirePerm(enforcer, "geofence", "view"), bizH.ListGeofences)
		bizAPI.PUT("/geofences", middleware.RequirePerm(enforcer, "geofence", "edit"), bizH.UpsertGeofence)
		bizAPI.POST("/geofences/evaluate", middleware.RequirePerm(enforcer, "geofence", "edit"), bizH.EvaluateLocation)

		historyBase := envOr("WATCHER_HISTORY_URL", "http://127.0.0.1:8082")
		bizAPI.GET("/history/telemetry", middleware.RequirePerm(enforcer, "monitor", "view"), handler.HistoryProxy(historyBase, "/api/v1/history/telemetry"))
		bizAPI.GET("/history/tracks", middleware.RequirePerm(enforcer, "track", "view"), handler.HistoryProxy(historyBase, "/api/v1/history/tracks"))
	}

	pushBase := envOr("WATCHER_PUSH_URL", "http://127.0.0.1:8083")
	r.GET("/v1/stream", middleware.JWT(jwtMgr), middleware.RequireAnyClient("admin", "portal", "app"), handler.StreamProxy(pushBase))

	log.Info("webhttp listening", zap.String("addr", cfg.HTTPAddr))
	if err := r.Run(cfg.HTTPAddr); err != nil {
		log.Fatal("run", zap.Error(err))
	}
}

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
