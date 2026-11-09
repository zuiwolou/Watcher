package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	bizsvc "watcher/backend/internal/biz/service"
	"watcher/backend/pkg/errcode"
	"watcher/backend/pkg/response"
)

func (h *BizHandler) ListDevices(c *gin.Context) {
	list, err := h.biz.ListDevices(c.Request.Context(), c.Query("warehouseId"))
	if err != nil {
		mapBizErr(c, err)
		return
	}
	response.OK(c, list)
}

func (h *BizHandler) CreateDevice(c *gin.Context) {
	var req struct {
		DeviceCode      string  `json:"deviceCode" binding:"required"`
		Name            string  `json:"name"`
		Type            string  `json:"type"`
		WarehouseID     string  `json:"warehouseId"`
		SchemaProfileID *uint64 `json:"schemaProfileId"`
		AuthType        string  `json:"authType"`
		Status          string  `json:"status"`
		MetaJSON        string  `json:"metaJson"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, errcode.ValidationError, err.Error())
		return
	}
	d, err := h.biz.CreateDevice(c.Request.Context(), bizsvc.DeviceInput{
		DeviceCode: req.DeviceCode, Name: req.Name, Type: req.Type, WarehouseID: req.WarehouseID,
		SchemaProfileID: req.SchemaProfileID, AuthType: req.AuthType, Status: req.Status, MetaJSON: req.MetaJSON,
	})
	if err != nil {
		mapBizErr(c, err)
		return
	}
	writeAudit(c, h.db, "device.create", "device", strconv.FormatUint(d.ID, 10), gin.H{"deviceCode": d.DeviceCode})
	response.OK(c, d)
}

func (h *BizHandler) UpdateDevice(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	var req struct {
		Name            string  `json:"name"`
		Type            string  `json:"type"`
		WarehouseID     string  `json:"warehouseId"`
		SchemaProfileID *uint64 `json:"schemaProfileId"`
		AuthType        string  `json:"authType"`
		Status          string  `json:"status"`
		MetaJSON        string  `json:"metaJson"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, errcode.ValidationError, err.Error())
		return
	}
	d, err := h.biz.UpdateDevice(c.Request.Context(), id, bizsvc.DeviceInput{
		Name: req.Name, Type: req.Type, WarehouseID: req.WarehouseID,
		SchemaProfileID: req.SchemaProfileID, AuthType: req.AuthType, Status: req.Status, MetaJSON: req.MetaJSON,
	})
	if err != nil {
		mapBizErr(c, err)
		return
	}
	writeAudit(c, h.db, "device.update", "device", strconv.FormatUint(d.ID, 10), gin.H{"status": d.Status})
	response.OK(c, d)
}

func (h *BizHandler) BindDeviceMqtt(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	var req struct {
		SchemaProfileID uint64 `json:"schemaProfileId"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, errcode.ValidationError, err.Error())
		return
	}
	d, err := h.biz.BindDeviceMqttSchema(c.Request.Context(), id, req.SchemaProfileID)
	if err != nil {
		mapBizErr(c, err)
		return
	}
	writeAudit(c, h.db, "device.mqtt_binding", "device", strconv.FormatUint(d.ID, 10), gin.H{"schemaProfileId": req.SchemaProfileID})
	response.OK(c, d)
}

func (h *BizHandler) ListOutbox(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
	list, total, err := h.biz.ListOutbox(c.Request.Context(), page, size, c.Query("status"), c.Query("channel"))
	if err != nil {
		mapBizErr(c, err)
		return
	}
	response.Page(c, list, page, size, total)
}

func (h *BizHandler) RetryOutbox(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	row, err := h.biz.RetryOutbox(c.Request.Context(), id)
	if err != nil {
		mapBizErr(c, err)
		return
	}
	writeAudit(c, h.db, "outbox.retry", "outbox", strconv.FormatUint(id, 10), nil)
	response.OK(c, row)
}
