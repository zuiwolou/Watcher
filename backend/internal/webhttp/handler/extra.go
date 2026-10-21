package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/casbin/casbin/v2"
	"github.com/gin-gonic/gin"

	"watcher/backend/pkg/errcode"
	"watcher/backend/pkg/response"
	bizsvc "watcher/backend/internal/biz/service"
	"watcher/backend/internal/webhttp/middleware"
)

// --- inventory ---

func (h *BizHandler) CreateInventoryTask(c *gin.Context) {
	var req struct {
		Name        string `json:"name" binding:"required"`
		WarehouseID string `json:"warehouseId" binding:"required"`
		ScopeJSON   string `json:"scopeJson"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, errcode.ValidationError, err.Error())
		return
	}
	t, err := h.biz.CreateInventoryTask(c.Request.Context(), req.Name, req.WarehouseID, req.ScopeJSON, mustUserID(c))
	if err != nil {
		mapBizErr(c, err)
		return
	}
	response.OK(c, t)
}

func (h *BizHandler) ListInventoryTasks(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
	list, total, err := h.biz.ListInventoryTasks(c.Request.Context(), page, size)
	if err != nil {
		mapBizErr(c, err)
		return
	}
	response.Page(c, list, page, size, total)
}

func (h *BizHandler) ScanInventory(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	var req struct {
		UniqueCode      string `json:"uniqueCode" binding:"required"`
		ClientRequestID string `json:"clientRequestId"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, errcode.ValidationError, err.Error())
		return
	}
	if req.ClientRequestID == "" {
		req.ClientRequestID = c.GetHeader("X-Idempotency-Key")
	}
	line, err := h.biz.ScanInventory(c.Request.Context(), id, req.UniqueCode, mustUserID(c), req.ClientRequestID)
	if err != nil {
		mapBizErr(c, err)
		return
	}
	response.OK(c, line)
}

func (h *BizHandler) SubmitInventory(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	if err := h.biz.SubmitInventory(c.Request.Context(), id); err != nil {
		mapBizErr(c, err)
		return
	}
	response.OK(c, gin.H{"ok": true})
}

func (h *BizHandler) PostInventory(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	if err := h.biz.PostInventory(c.Request.Context(), id); err != nil {
		mapBizErr(c, err)
		return
	}
	response.OK(c, gin.H{"ok": true})
}

func (h *BizHandler) ListInventoryLines(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	list, err := h.biz.ListInventoryLines(c.Request.Context(), id)
	if err != nil {
		mapBizErr(c, err)
		return
	}
	response.OK(c, list)
}

// --- approval / stock adjust ---

type ApprovalHandler struct {
	biz *bizsvc.Service
	e   *casbin.Enforcer
}

func NewApprovalHandler(biz *bizsvc.Service, e *casbin.Enforcer) *ApprovalHandler {
	return &ApprovalHandler{biz: biz, e: e}
}

func (h *ApprovalHandler) UpsertTemplate(c *gin.Context) {
	var req struct {
		BizType string                 `json:"bizType" binding:"required"`
		Name    string                 `json:"name"`
		Levels  []bizsvc.ApprovalLevel `json:"levels" binding:"required"`
		Enabled *bool                  `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, errcode.ValidationError, err.Error())
		return
	}
	en := true
	if req.Enabled != nil {
		en = *req.Enabled
	}
	t, err := h.biz.UpsertApprovalTemplate(c.Request.Context(), req.BizType, req.Name, req.Levels, en)
	if err != nil {
		mapBizErr(c, err)
		return
	}
	response.OK(c, t)
}

func (h *ApprovalHandler) ListTemplates(c *gin.Context) {
	list, err := h.biz.ListApprovalTemplates(c.Request.Context())
	if err != nil {
		mapBizErr(c, err)
		return
	}
	response.OK(c, list)
}

func (h *ApprovalHandler) CreateAdjustment(c *gin.Context) {
	var req struct {
		WarehouseID string  `json:"warehouseId" binding:"required"`
		MaterialID  uint64  `json:"materialId" binding:"required"`
		Delta       float64 `json:"delta" binding:"required"`
		Reason      string  `json:"reason"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, errcode.ValidationError, err.Error())
		return
	}
	adj, order, err := h.biz.CreateStockAdjustment(c.Request.Context(), req.WarehouseID, req.MaterialID, req.Delta, req.Reason, mustUserID(c))
	if err != nil {
		mapBizErr(c, err)
		return
	}
	response.OK(c, gin.H{"adjustment": adj, "approvalOrder": order})
}

func (h *ApprovalHandler) ListAdjustments(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
	list, total, err := h.biz.ListStockAdjustments(c.Request.Context(), page, size)
	if err != nil {
		mapBizErr(c, err)
		return
	}
	response.Page(c, list, page, size, total)
}

func (h *ApprovalHandler) ListTodo(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
	list, total, err := h.biz.ListApprovalTodo(c.Request.Context(), page, size)
	if err != nil {
		mapBizErr(c, err)
		return
	}
	response.Page(c, list, page, size, total)
}

func (h *ApprovalHandler) Act(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	var req struct {
		Action  string `json:"action" binding:"required"`
		Comment string `json:"comment"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, errcode.ValidationError, err.Error())
		return
	}
	uid := mustUserID(c)
	uidStr := fmt.Sprintf("%d", uid)
	hasRole := func(code string) bool {
		ok, _ := h.e.HasGroupingPolicy("user:"+uidStr, "role:"+code)
		return ok
	}
	o, err := h.biz.ActApproval(c.Request.Context(), bizsvc.ActApprovalInput{
		OrderID: id, ActorID: uid, Action: req.Action, Comment: req.Comment, HasRole: hasRole,
	})
	if err != nil {
		mapBizErr(c, err)
		return
	}
	response.OK(c, o)
}

// --- inspection ---

func (h *BizHandler) CreateInspPlan(c *gin.Context) {
	var req struct {
		Name string `json:"name" binding:"required"`
		Cron string `json:"cycleCron"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, errcode.ValidationError, err.Error())
		return
	}
	p, err := h.biz.CreateInspectionPlan(c.Request.Context(), req.Name, req.Cron)
	if err != nil {
		mapBizErr(c, err)
		return
	}
	response.OK(c, p)
}

func (h *BizHandler) ListInspPlans(c *gin.Context) {
	list, err := h.biz.ListInspectionPlans(c.Request.Context())
	if err != nil {
		mapBizErr(c, err)
		return
	}
	response.OK(c, list)
}

func (h *BizHandler) CreateInspRoute(c *gin.Context) {
	var req struct {
		PlanID uint64 `json:"planId" binding:"required"`
		Name   string `json:"name" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, errcode.ValidationError, err.Error())
		return
	}
	r, err := h.biz.CreateInspectionRoute(c.Request.Context(), req.PlanID, req.Name)
	if err != nil {
		mapBizErr(c, err)
		return
	}
	response.OK(c, r)
}

func (h *BizHandler) AddInspPoint(c *gin.Context) {
	var req struct {
		RouteID      uint64  `json:"routeId" binding:"required"`
		Seq          int     `json:"seq"`
		Name         string  `json:"name" binding:"required"`
		Lat          float64 `json:"lat"`
		Lng          float64 `json:"lng"`
		RadiusM      float64 `json:"radiusM"`
		RequirePhoto bool    `json:"requirePhoto"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, errcode.ValidationError, err.Error())
		return
	}
	if req.RadiusM == 0 {
		req.RadiusM = 50
	}
	p, err := h.biz.AddInspectionPoint(c.Request.Context(), req.RouteID, req.Seq, req.Name, req.Lat, req.Lng, req.RadiusM, req.RequirePhoto)
	if err != nil {
		mapBizErr(c, err)
		return
	}
	response.OK(c, p)
}

func (h *BizHandler) ListInspPoints(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	list, err := h.biz.ListInspectionPoints(c.Request.Context(), id)
	if err != nil {
		mapBizErr(c, err)
		return
	}
	response.OK(c, list)
}

func (h *BizHandler) CreateInspTask(c *gin.Context) {
	var req struct {
		PlanID     uint64 `json:"planId" binding:"required"`
		RouteID    uint64 `json:"routeId" binding:"required"`
		AssigneeID uint64 `json:"assigneeId" binding:"required"`
		DueAt      string `json:"dueAt"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, errcode.ValidationError, err.Error())
		return
	}
	var due *time.Time
	if req.DueAt != "" {
		t, err := time.Parse(time.RFC3339, req.DueAt)
		if err == nil {
			due = &t
		}
	}
	t, err := h.biz.CreateInspectionTask(c.Request.Context(), req.PlanID, req.RouteID, req.AssigneeID, due)
	if err != nil {
		mapBizErr(c, err)
		return
	}
	response.OK(c, t)
}

func (h *BizHandler) ListInspTasks(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
	list, total, err := h.biz.ListInspectionTasks(c.Request.Context(), page, size)
	if err != nil {
		mapBizErr(c, err)
		return
	}
	response.Page(c, list, page, size, total)
}

func (h *BizHandler) InspCheckin(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	var req struct {
		PointID         uint64   `json:"pointId" binding:"required"`
		Status          string   `json:"status" binding:"required"`
		Lat             *float64 `json:"lat"`
		Lng             *float64 `json:"lng"`
		PhotoObjectKey  string   `json:"photoObjectKey"`
		Remark          string   `json:"remark"`
		ClientRequestID string   `json:"clientRequestId"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, errcode.ValidationError, err.Error())
		return
	}
	if req.ClientRequestID == "" {
		req.ClientRequestID = c.GetHeader("X-Idempotency-Key")
	}
	ck, alm, err := h.biz.InspectionCheckin(c.Request.Context(), bizsvc.InspectionCheckinInput{
		TaskID: id, PointID: req.PointID, Status: req.Status, Lat: req.Lat, Lng: req.Lng,
		PhotoKey: req.PhotoObjectKey, Remark: req.Remark, ClientRequestID: req.ClientRequestID,
		OperatorID: mustUserID(c),
	})
	if err != nil {
		mapBizErr(c, err)
		return
	}
	response.OK(c, gin.H{"checkin": ck, "alarm": alm})
}

// --- alarm / threshold / geofence ---

func (h *BizHandler) UpsertThreshold(c *gin.Context) {
	wh := c.Param("warehouseId")
	var req struct {
		Metric string   `json:"metric" binding:"required"`
		Upper  *float64 `json:"upper"`
		Lower  *float64 `json:"lower"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, errcode.ValidationError, err.Error())
		return
	}
	r, err := h.biz.UpsertThreshold(c.Request.Context(), wh, req.Metric, req.Upper, req.Lower)
	if err != nil {
		mapBizErr(c, err)
		return
	}
	response.OK(c, r)
}

func (h *BizHandler) ListThresholds(c *gin.Context) {
	list, err := h.biz.ListThresholds(c.Request.Context(), c.Query("warehouseId"))
	if err != nil {
		mapBizErr(c, err)
		return
	}
	response.OK(c, list)
}

func (h *BizHandler) EvaluateTelemetry(c *gin.Context) {
	var req struct {
		WarehouseID string             `json:"warehouseId" binding:"required"`
		DeviceID    string             `json:"deviceId"`
		Metrics     map[string]float64 `json:"metrics" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, errcode.ValidationError, err.Error())
		return
	}
	alarms, err := h.biz.EvaluateTelemetry(c.Request.Context(), req.WarehouseID, req.DeviceID, req.Metrics)
	if err != nil {
		mapBizErr(c, err)
		return
	}
	response.OK(c, alarms)
}

func (h *BizHandler) ListAlarms(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
	list, total, err := h.biz.ListAlarms(c.Request.Context(), page, size, c.Query("status"))
	if err != nil {
		mapBizErr(c, err)
		return
	}
	response.Page(c, list, page, size, total)
}

func (h *BizHandler) AckAlarm(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	if err := h.biz.AckAlarm(c.Request.Context(), id, mustUserID(c)); err != nil {
		mapBizErr(c, err)
		return
	}
	response.OK(c, gin.H{"ok": true})
}

func (h *BizHandler) CloseAlarm(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	if err := h.biz.CloseAlarm(c.Request.Context(), id); err != nil {
		mapBizErr(c, err)
		return
	}
	response.OK(c, gin.H{"ok": true})
}

func (h *BizHandler) UpsertGeofence(c *gin.Context) {
	var req struct {
		ID          uint64  `json:"id"`
		WarehouseID string  `json:"warehouseId" binding:"required"`
		Name        string  `json:"name" binding:"required"`
		ShapeType   string  `json:"shapeType"`
		GeoJSON     string  `json:"geoJson"`
		Circle      *struct {
			Lat     float64 `json:"lat"`
			Lng     float64 `json:"lng"`
			RadiusM float64 `json:"radiusM"`
		} `json:"circle"`
		HysteresisM float64 `json:"hysteresisM"`
		Enabled     *bool   `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, errcode.ValidationError, err.Error())
		return
	}
	geo := req.GeoJSON
	shape := req.ShapeType
	if req.Circle != nil {
		shape = "circle"
		b, _ := json.Marshal(map[string]interface{}{
			"type": "circle", "lat": req.Circle.Lat, "lng": req.Circle.Lng, "radiusM": req.Circle.RadiusM,
		})
		geo = string(b)
	}
	en := true
	if req.Enabled != nil {
		en = *req.Enabled
	}
	g, err := h.biz.UpsertGeofence(c.Request.Context(), req.ID, req.WarehouseID, req.Name, shape, geo, req.HysteresisM, en)
	if err != nil {
		mapBizErr(c, err)
		return
	}
	response.OK(c, g)
}

func (h *BizHandler) ListGeofences(c *gin.Context) {
	list, err := h.biz.ListGeofences(c.Request.Context(), c.Query("warehouseId"))
	if err != nil {
		mapBizErr(c, err)
		return
	}
	response.OK(c, list)
}

func (h *BizHandler) EvaluateLocation(c *gin.Context) {
	var req struct {
		WarehouseID string  `json:"warehouseId" binding:"required"`
		DeviceID    string  `json:"deviceId"`
		Lat         float64 `json:"lat" binding:"required"`
		Lng         float64 `json:"lng" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, errcode.ValidationError, err.Error())
		return
	}
	alarms, err := h.biz.EvaluateLocation(c.Request.Context(), req.WarehouseID, req.DeviceID, req.Lat, req.Lng)
	if err != nil {
		mapBizErr(c, err)
		return
	}
	response.OK(c, alarms)
}

// silence unused middleware import if any
var _ = middleware.CtxUserID
