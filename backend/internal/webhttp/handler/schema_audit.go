package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"watcher/backend/internal/webhttp/iam"
	"watcher/backend/pkg/errcode"
	"watcher/backend/pkg/response"
)

type AuditHandler struct{ db *gorm.DB }

func NewAuditHandler(db *gorm.DB) *AuditHandler { return &AuditHandler{db: db} }

func (h *AuditHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 20
	}
	var total int64
	q := h.db.Model(&iam.AuditLog{})
	if action := c.Query("action"); action != "" {
		q = q.Where("action = ?", action)
	}
	if actor := c.Query("actorId"); actor != "" {
		if aid, err := strconv.ParseUint(actor, 10, 64); err == nil {
			q = q.Where("actor_id = ?", aid)
		}
	}
	_ = q.Count(&total)
	var list []iam.AuditLog
	if err := q.Order("id desc").Offset((page - 1) * size).Limit(size).Find(&list).Error; err != nil {
		response.Fail(c, http.StatusInternalServerError, errcode.Internal, err.Error())
		return
	}
	response.Page(c, list, page, size, total)
}

func (h *BizHandler) ListMqttSchemas(c *gin.Context) {
	list, err := h.biz.ListMqttSchemas(c.Request.Context())
	if err != nil {
		mapBizErr(c, err)
		return
	}
	response.OK(c, list)
}

func (h *BizHandler) UpsertMqttSchema(c *gin.Context) {
	var req struct {
		ID           uint64 `json:"id"`
		Code         string `json:"code" binding:"required"`
		Name         string `json:"name"`
		TopicPattern string `json:"topicPattern"`
		MappingJSON  string `json:"mappingJson"`
		Enabled      *bool  `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, errcode.ValidationError, err.Error())
		return
	}
	en := true
	if req.Enabled != nil {
		en = *req.Enabled
	}
	p, err := h.biz.UpsertMqttSchema(c.Request.Context(), req.ID, req.Code, req.Name, req.TopicPattern, req.MappingJSON, en)
	if err != nil {
		mapBizErr(c, err)
		return
	}
	writeAudit(c, h.db, "mqtt.schema.upsert", "mqtt_schema", strconv.FormatUint(p.ID, 10), gin.H{"code": p.Code})
	response.OK(c, p)
}

func (h *BizHandler) DryRunMqttSchema(c *gin.Context) {
	var req struct {
		MappingJSON   string `json:"mappingJson"`
		SamplePayload string `json:"samplePayload"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, errcode.ValidationError, err.Error())
		return
	}
	out, err := h.biz.DryRunMqttSchema(c.Request.Context(), req.MappingJSON, req.SamplePayload)
	if err != nil {
		mapBizErr(c, err)
		return
	}
	response.OK(c, out)
}

func (h *BizHandler) ListIntegrations(c *gin.Context) {
	list, err := h.biz.ListIntegrations(c.Request.Context())
	if err != nil {
		mapBizErr(c, err)
		return
	}
	response.OK(c, list)
}

func (h *BizHandler) UpsertIntegration(c *gin.Context) {
	var req struct {
		ID         uint64 `json:"id"`
		Code       string `json:"code" binding:"required"`
		Name       string `json:"name"`
		ConfigJSON string `json:"configJson"`
		Enabled    *bool  `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, errcode.ValidationError, err.Error())
		return
	}
	en := false
	if req.Enabled != nil {
		en = *req.Enabled
	}
	p, err := h.biz.UpsertIntegration(c.Request.Context(), req.ID, req.Code, req.Name, req.ConfigJSON, en)
	if err != nil {
		mapBizErr(c, err)
		return
	}
	writeAudit(c, h.db, "integration.upsert", "integration", strconv.FormatUint(p.ID, 10), gin.H{"code": p.Code})
	response.OK(c, p)
}

// InternalEvaluateTelemetry is for svc-mqtt (X-Internal-Token), no user JWT.
func (h *BizHandler) InternalEvaluateTelemetry(c *gin.Context) {
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

// InternalEvaluateGeofence is for svc-mqtt location frames (X-Internal-Token).
func (h *BizHandler) InternalEvaluateGeofence(c *gin.Context) {
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
