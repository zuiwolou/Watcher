package main

import (
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"watcher/backend/pkg/auth"
	"watcher/backend/pkg/authz"
	"watcher/backend/pkg/config"
	"watcher/backend/pkg/database"
	"watcher/backend/pkg/ldapx"
	"watcher/backend/pkg/logx"
	"watcher/backend/pkg/storage"
	bizsvc "watcher/backend/internal/biz/service"
	"watcher/backend/internal/webhttp/handler"
	"watcher/backend/internal/webhttp/iam"
	"watcher/backend/internal/webhttp/middleware"
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
	bizH := handler.NewBizHandler(biz)
	apprH := handler.NewApprovalHandler(biz, enforcer)
	mediaH := handler.NewMediaHandler(minioClient)

	r := gin.New()
	r.Use(gin.Recovery(), middleware.RequestID())
	r.GET("/healthz", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "ok"}) })

	v1 := r.Group("/api/v1")
	{
		v1.POST("/auth/login", authH.Login)
		v1.POST("/auth/refresh", authH.Refresh)

		authed := v1.Group("")
		authed.Use(middleware.JWT(jwtMgr))
		authed.GET("/me", authH.Me)
		authed.GET("/me/permissions", authH.Me)
		authed.POST("/auth/change-password", authH.ChangePassword)

		authed.GET("/admin/permissions", middleware.RequirePerm(enforcer, "role", "view"), adminH.ListPermissions)
		authed.GET("/admin/users", middleware.RequirePerm(enforcer, "user", "view"), adminH.ListUsers)
		authed.POST("/admin/users", middleware.RequirePerm(enforcer, "user", "edit"), adminH.CreateUser)
		authed.PUT("/admin/users/:id", middleware.RequirePerm(enforcer, "user", "edit"), adminH.UpdateUser)
		authed.GET("/admin/roles", middleware.RequirePerm(enforcer, "role", "view"), adminH.ListRoles)
		authed.POST("/admin/roles", middleware.RequirePerm(enforcer, "role", "edit"), adminH.CreateRole)
		authed.PUT("/admin/roles/:id", middleware.RequirePerm(enforcer, "role", "edit"), adminH.UpdateRole)

		authed.POST("/media/upload-url", mediaH.UploadURL)

		authed.GET("/materials", middleware.RequirePerm(enforcer, "material", "view"), bizH.ListMaterials)
		authed.POST("/materials", middleware.RequirePerm(enforcer, "material", "edit"), bizH.CreateMaterial)
		authed.GET("/stocks", middleware.RequirePerm(enforcer, "material", "view"), bizH.ListStocks)
		authed.GET("/unique-codes/:code", middleware.RequirePerm(enforcer, "material", "view"), bizH.GetUniqueCode)

		authed.GET("/inbound/orders", middleware.RequirePerm(enforcer, "warehouse", "view"), bizH.ListInbound)
		authed.POST("/inbound/orders", middleware.RequirePerm(enforcer, "inbound", "edit"), bizH.CreateInbound)
		authed.POST("/inbound/orders/:id/issue-codes", middleware.RequirePerm(enforcer, "code", "issue"), bizH.IssueCodes)
		authed.POST("/inbound/orders/:id/confirm", middleware.RequirePerm(enforcer, "inbound", "confirm"), bizH.ConfirmInbound)

		authed.POST("/outbound/orders", middleware.RequirePerm(enforcer, "outbound", "edit"), bizH.CreateOutbound)
		authed.POST("/outbound/orders/:id/confirm", middleware.RequirePerm(enforcer, "outbound", "confirm"), bizH.ConfirmOutbound)

		authed.GET("/transport/units", middleware.RequirePerm(enforcer, "transport", "view"), bizH.ListUnits)
		authed.POST("/transport/units", middleware.RequirePerm(enforcer, "unit", "edit"), bizH.CreateUnit)
		authed.GET("/transport/tasks", middleware.RequirePerm(enforcer, "transport", "view"), bizH.ListTransportTasks)
		authed.POST("/transport/tasks", middleware.RequirePerm(enforcer, "transport", "edit"), bizH.CreateTransportTask)
		authed.GET("/transport/tasks/:id/checkins", middleware.RequirePerm(enforcer, "transport", "view"), bizH.ListCheckins)
		authed.POST("/transport/checkins", middleware.RequirePerm(enforcer, "checkin", "exec"), bizH.SubmitCheckin)

		// inventory
		authed.GET("/inventory/tasks", middleware.RequirePerm(enforcer, "inventory", "view"), bizH.ListInventoryTasks)
		authed.POST("/inventory/tasks", middleware.RequirePerm(enforcer, "inventory", "create"), bizH.CreateInventoryTask)
		authed.POST("/inventory/tasks/:id/scan", middleware.RequirePerm(enforcer, "inventory", "scan"), bizH.ScanInventory)
		authed.POST("/inventory/tasks/:id/submit", middleware.RequirePerm(enforcer, "inventory", "scan"), bizH.SubmitInventory)
		authed.POST("/inventory/tasks/:id/post", middleware.RequirePerm(enforcer, "inventory", "adjust"), bizH.PostInventory)
		authed.GET("/inventory/tasks/:id/lines", middleware.RequirePerm(enforcer, "inventory", "view"), bizH.ListInventoryLines)

		// approval & stock adjust
		authed.GET("/approvals/templates", middleware.RequirePerm(enforcer, "approval", "view"), apprH.ListTemplates)
		authed.PUT("/approvals/templates", middleware.RequirePerm(enforcer, "approval", "act"), apprH.UpsertTemplate)
		authed.GET("/approvals/todo", middleware.RequirePerm(enforcer, "approval", "view"), apprH.ListTodo)
		authed.POST("/approvals/:id/act", middleware.RequirePerm(enforcer, "approval", "act"), apprH.Act)
		authed.POST("/stocks/adjustments", middleware.RequirePerm(enforcer, "stock", "adjust"), apprH.CreateAdjustment)
		authed.GET("/stocks/adjustments", middleware.RequirePerm(enforcer, "stock", "adjust"), apprH.ListAdjustments)

		// inspection
		authed.GET("/inspection/plans", middleware.RequirePerm(enforcer, "inspection", "view"), bizH.ListInspPlans)
		authed.POST("/inspection/plans", middleware.RequirePerm(enforcer, "inspection", "create"), bizH.CreateInspPlan)
		authed.POST("/inspection/routes", middleware.RequirePerm(enforcer, "inspection", "create"), bizH.CreateInspRoute)
		authed.POST("/inspection/points", middleware.RequirePerm(enforcer, "inspection", "create"), bizH.AddInspPoint)
		authed.GET("/inspection/routes/:id/points", middleware.RequirePerm(enforcer, "inspection", "view"), bizH.ListInspPoints)
		authed.GET("/inspection/tasks", middleware.RequirePerm(enforcer, "inspection", "view"), bizH.ListInspTasks)
		authed.POST("/inspection/tasks", middleware.RequirePerm(enforcer, "inspection", "create"), bizH.CreateInspTask)
		authed.POST("/inspection/tasks/:id/checkin", middleware.RequirePerm(enforcer, "inspection", "exec"), bizH.InspCheckin)

		// alarm / threshold / geofence
		authed.GET("/alarms", middleware.RequirePerm(enforcer, "alarm", "view"), bizH.ListAlarms)
		authed.POST("/alarms/:id/ack", middleware.RequirePerm(enforcer, "alarm", "ack"), bizH.AckAlarm)
		authed.POST("/alarms/:id/close", middleware.RequirePerm(enforcer, "alarm", "ack"), bizH.CloseAlarm)
		authed.GET("/thresholds", middleware.RequirePerm(enforcer, "threshold", "edit"), bizH.ListThresholds)
		authed.PUT("/thresholds/:warehouseId", middleware.RequirePerm(enforcer, "threshold", "edit"), bizH.UpsertThreshold)
		authed.POST("/monitor/evaluate", middleware.RequirePerm(enforcer, "threshold", "edit"), bizH.EvaluateTelemetry)
		authed.GET("/geofences", middleware.RequirePerm(enforcer, "geofence", "view"), bizH.ListGeofences)
		authed.PUT("/geofences", middleware.RequirePerm(enforcer, "geofence", "edit"), bizH.UpsertGeofence)
		authed.POST("/geofences/evaluate", middleware.RequirePerm(enforcer, "geofence", "edit"), bizH.EvaluateLocation)

		// history proxy (A3)
		historyBase := envOr("WATCHER_HISTORY_URL", "http://127.0.0.1:8082")
		authed.GET("/history/telemetry", middleware.RequirePerm(enforcer, "monitor", "view"), handler.HistoryProxy(historyBase, "/api/v1/history/telemetry"))
		authed.GET("/history/tracks", middleware.RequirePerm(enforcer, "track", "view"), handler.HistoryProxy(historyBase, "/api/v1/history/tracks"))
	}

	// WSS: JWT then reverse proxy to push
	pushBase := envOr("WATCHER_PUSH_URL", "http://127.0.0.1:8083")
	r.GET("/v1/stream", middleware.JWT(jwtMgr), handler.StreamProxy(pushBase))

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
