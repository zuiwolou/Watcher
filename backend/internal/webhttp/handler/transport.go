package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"watcher/backend/pkg/errcode"
	"watcher/backend/pkg/response"
	bizsvc "watcher/backend/internal/biz/service"
)

func (h *BizHandler) CreateUnit(c *gin.Context) {
	var req struct {
		PlateNo  string `json:"plateNo" binding:"required"`
		Type     string `json:"type"`
		DeviceID string `json:"deviceId"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, errcode.ValidationError, err.Error())
		return
	}
	if req.Type == "" {
		req.Type = "truck"
	}
	u, err := h.biz.CreateTransportUnit(c.Request.Context(), req.PlateNo, req.Type, req.DeviceID)
	if err != nil {
		mapBizErr(c, err)
		return
	}
	response.OK(c, u)
}

func (h *BizHandler) ListUnits(c *gin.Context) {
	list, err := h.biz.ListTransportUnits(c.Request.Context())
	if err != nil {
		mapBizErr(c, err)
		return
	}
	response.OK(c, list)
}

func (h *BizHandler) CreateTransportTask(c *gin.Context) {
	var req struct {
		OutboundOrderID uint64 `json:"outboundOrderId" binding:"required"`
		UnitID          uint64 `json:"unitId" binding:"required"`
		DriverUserID    uint64 `json:"driverUserId" binding:"required"`
		WarehouseID     string `json:"warehouseId"`
		Remark          string `json:"remark"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, errcode.ValidationError, err.Error())
		return
	}
	task, err := h.biz.CreateTransportTask(c.Request.Context(), bizsvc.CreateTransportTaskInput{
		OutboundOrderID: req.OutboundOrderID,
		UnitID:          req.UnitID,
		DriverUserID:    req.DriverUserID,
		WarehouseID:     req.WarehouseID,
		Remark:          req.Remark,
		CreatedBy:       mustUserID(c),
	})
	if err != nil {
		mapBizErr(c, err)
		return
	}
	response.OK(c, task)
}

func (h *BizHandler) ListTransportTasks(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
	list, total, err := h.biz.ListTransportTasks(c.Request.Context(), page, size)
	if err != nil {
		mapBizErr(c, err)
		return
	}
	response.Page(c, list, page, size, total)
}

func (h *BizHandler) SubmitCheckin(c *gin.Context) {
	var req struct {
		TaskID          uint64   `json:"taskId" binding:"required"`
		NodeType        string   `json:"nodeType" binding:"required"`
		UniqueCode      string   `json:"uniqueCode"`
		ClientRequestID string   `json:"clientRequestId"`
		Lat             *float64 `json:"lat"`
		Lng             *float64 `json:"lng"`
		Accuracy        *float64 `json:"accuracy"`
		PhotoObjectKey  string   `json:"photoObjectKey"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, errcode.ValidationError, err.Error())
		return
	}
	if req.ClientRequestID == "" {
		req.ClientRequestID = c.GetHeader("X-Idempotency-Key")
	}
	ck, err := h.biz.SubmitCheckin(c.Request.Context(), bizsvc.SubmitCheckinInput{
		TaskID: req.TaskID, NodeType: req.NodeType, UniqueCode: req.UniqueCode,
		OperatorID: mustUserID(c), ClientRequestID: req.ClientRequestID,
		Lat: req.Lat, Lng: req.Lng, Accuracy: req.Accuracy, PhotoKey: req.PhotoObjectKey,
	})
	if err != nil {
		mapBizErr(c, err)
		return
	}
	response.OK(c, ck)
}

func (h *BizHandler) ListCheckins(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	list, err := h.biz.ListCheckins(c.Request.Context(), id)
	if err != nil {
		mapBizErr(c, err)
		return
	}
	response.OK(c, list)
}
