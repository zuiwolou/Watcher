package handler

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	bizsvc "watcher/backend/internal/biz/service"
	"watcher/backend/internal/webhttp/middleware"
	"watcher/backend/pkg/errcode"
	"watcher/backend/pkg/response"
)

type BizHandler struct {
	biz *bizsvc.Service
	db  *gorm.DB
}

func NewBizHandler(biz *bizsvc.Service, db *gorm.DB) *BizHandler {
	return &BizHandler{biz: biz, db: db}
}

func (h *BizHandler) ListWarehouses(c *gin.Context) {
	list, err := h.biz.ListWarehouses(c.Request.Context())
	if err != nil {
		mapBizErr(c, err)
		return
	}
	response.OK(c, list)
}

func (h *BizHandler) CreateMaterial(c *gin.Context) {
	var req struct {
		Code      string `json:"code" binding:"required"`
		Name      string `json:"name" binding:"required"`
		WasteCode string `json:"wasteCode"`
		Category  string `json:"category"`
		Unit      string `json:"unit"`
		Spec      string `json:"spec"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, errcode.ValidationError, err.Error())
		return
	}
	m, err := h.biz.CreateMaterial(c.Request.Context(), bizsvc.CreateMaterialInput(req))
	if err != nil {
		mapBizErr(c, err)
		return
	}
	writeAudit(c, h.db, "material.create", "material", strconv.FormatUint(m.ID, 10), gin.H{"code": m.Code})
	response.OK(c, m)
}

func (h *BizHandler) ListMaterials(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
	list, total, err := h.biz.ListMaterials(c.Request.Context(), page, size)
	if err != nil {
		mapBizErr(c, err)
		return
	}
	response.Page(c, list, page, size, total)
}

func (h *BizHandler) UpdateMaterial(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	var req struct {
		Name      string `json:"name"`
		WasteCode string `json:"wasteCode"`
		Category  string `json:"category"`
		Unit      string `json:"unit"`
		Spec      string `json:"spec"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, errcode.ValidationError, err.Error())
		return
	}
	m, err := h.biz.UpdateMaterial(c.Request.Context(), id, bizsvc.CreateMaterialInput{
		Name: req.Name, WasteCode: req.WasteCode, Category: req.Category, Unit: req.Unit, Spec: req.Spec,
	})
	if err != nil {
		mapBizErr(c, err)
		return
	}
	writeAudit(c, h.db, "material.update", "material", strconv.FormatUint(m.ID, 10), gin.H{"name": m.Name})
	response.OK(c, m)
}

func (h *BizHandler) ListUniqueCodes(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
	list, total, err := h.biz.ListUniqueCodes(c.Request.Context(), c.Query("warehouseId"), c.Query("status"), c.Query("q"), page, size)
	if err != nil {
		mapBizErr(c, err)
		return
	}
	response.Page(c, list, page, size, total)
}

func (h *BizHandler) ExportUniqueCodesCSV(c *gin.Context) {
	list, _, err := h.biz.ListUniqueCodes(c.Request.Context(), c.Query("warehouseId"), c.Query("status"), c.Query("q"), 1, 10000)
	if err != nil {
		mapBizErr(c, err)
		return
	}
	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", "attachment; filename=unique-codes.csv")
	var b strings.Builder
	b.WriteString("\ufeffcode,materialId,warehouseId,status\n")
	for _, row := range list {
		b.WriteString(row.Code)
		b.WriteByte(',')
		b.WriteString(strconv.FormatUint(row.MaterialID, 10))
		b.WriteByte(',')
		b.WriteString(row.WarehouseID)
		b.WriteByte(',')
		b.WriteString(row.Status)
		b.WriteByte('\n')
	}
	c.String(http.StatusOK, b.String())
}

func (h *BizHandler) CreateInbound(c *gin.Context) {
	var req struct {
		WarehouseID string  `json:"warehouseId" binding:"required"`
		MaterialID  uint64  `json:"materialId" binding:"required"`
		Qty         float64 `json:"qty" binding:"required"`
		Remark      string  `json:"remark"`
		ManifestNo  string  `json:"manifestNo"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, errcode.ValidationError, err.Error())
		return
	}
	uid := mustUserID(c)
	order, err := h.biz.CreateInboundOrder(c.Request.Context(), bizsvc.CreateInboundInput{
		WarehouseID: req.WarehouseID,
		MaterialID:  req.MaterialID,
		Qty:         req.Qty,
		Remark:      req.Remark,
		ManifestNo:  req.ManifestNo,
		CreatedBy:   uid,
	})
	if err != nil {
		mapBizErr(c, err)
		return
	}
	response.OK(c, order)
}

func (h *BizHandler) ListInbound(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
	list, total, err := h.biz.ListInboundOrders(c.Request.Context(), page, size)
	if err != nil {
		mapBizErr(c, err)
		return
	}
	response.Page(c, list, page, size, total)
}

func (h *BizHandler) IssueCodes(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	var req struct {
		Count int `json:"count" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, errcode.ValidationError, err.Error())
		return
	}
	codes, err := h.biz.IssueCodes(c.Request.Context(), id, req.Count)
	if err != nil {
		mapBizErr(c, err)
		return
	}
	response.OK(c, codes)
}

func (h *BizHandler) ConfirmInbound(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	var req execReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, errcode.ValidationError, err.Error())
		return
	}
	if req.ClientRequestID == "" {
		req.ClientRequestID = c.GetHeader("X-Idempotency-Key")
	}
	res, err := h.biz.ConfirmInbound(c.Request.Context(), bizsvc.ConfirmInboundInput{
		OrderID: id, UniqueCode: req.UniqueCode, OperatorID: mustUserID(c),
		ClientRequestID: req.ClientRequestID, Lat: req.Lat, Lng: req.Lng, PhotoKey: req.PhotoObjectKey,
	})
	if err != nil {
		mapBizErr(c, err)
		return
	}
	writeAudit(c, h.db, "inbound.confirm", "inbound_order", strconv.FormatUint(id, 10), gin.H{"uniqueCode": req.UniqueCode})
	response.OK(c, res)
}

func (h *BizHandler) ListOutbound(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
	list, total, err := h.biz.ListOutboundOrders(c.Request.Context(), page, size)
	if err != nil {
		mapBizErr(c, err)
		return
	}
	response.Page(c, list, page, size, total)
}

func (h *BizHandler) CreateOutbound(c *gin.Context) {
	var req struct {
		WarehouseID string  `json:"warehouseId" binding:"required"`
		MaterialID  uint64  `json:"materialId" binding:"required"`
		Qty         float64 `json:"qty" binding:"required"`
		Remark      string  `json:"remark"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, errcode.ValidationError, err.Error())
		return
	}
	order, err := h.biz.CreateOutboundOrder(c.Request.Context(), req.WarehouseID, mustUserID(c), req.MaterialID, req.Qty, req.Remark)
	if err != nil {
		mapBizErr(c, err)
		return
	}
	response.OK(c, order)
}

func (h *BizHandler) ConfirmOutbound(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	var req struct {
		execReq
		Action string `json:"action"` // lock | receive
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, errcode.ValidationError, err.Error())
		return
	}
	if req.ClientRequestID == "" {
		req.ClientRequestID = c.GetHeader("X-Idempotency-Key")
	}
	action := req.Action
	if action == "" {
		action = c.DefaultQuery("action", "lock")
	}
	var (
		res *bizsvc.ConfirmResult
		err error
	)
	switch action {
	case "receive":
		res, err = h.biz.ReceiveOutbound(c.Request.Context(), bizsvc.ReceiveOutboundInput{
			OrderID: id, UniqueCode: req.UniqueCode, OperatorID: mustUserID(c),
			ClientRequestID: req.ClientRequestID, Lat: req.Lat, Lng: req.Lng, PhotoKey: req.PhotoObjectKey,
		})
	default:
		res, err = h.biz.LockOutbound(c.Request.Context(), bizsvc.LockOutboundInput{
			OrderID: id, UniqueCode: req.UniqueCode, OperatorID: mustUserID(c),
			ClientRequestID: req.ClientRequestID, Lat: req.Lat, Lng: req.Lng, PhotoKey: req.PhotoObjectKey,
		})
	}
	if err != nil {
		mapBizErr(c, err)
		return
	}
	writeAudit(c, h.db, "outbound."+action, "outbound_order", strconv.FormatUint(id, 10), gin.H{"uniqueCode": req.UniqueCode})
	response.OK(c, res)
}

func (h *BizHandler) GetUniqueCode(c *gin.Context) {
	code, err := h.biz.GetUniqueCode(c.Request.Context(), c.Param("code"))
	if err != nil {
		mapBizErr(c, err)
		return
	}
	response.OK(c, code)
}

func (h *BizHandler) VoidUniqueCode(c *gin.Context) {
	var req struct {
		Reason string `json:"reason"`
	}
	_ = c.ShouldBindJSON(&req)
	code, err := h.biz.VoidUniqueCode(c.Request.Context(), c.Param("code"), mustUserID(c), req.Reason)
	if err != nil {
		mapBizErr(c, err)
		return
	}
	writeAudit(c, h.db, "code.void", "unique_code", c.Param("code"), gin.H{"reason": req.Reason})
	response.OK(c, code)
}

func (h *BizHandler) ListStocks(c *gin.Context) {
	list, err := h.biz.ListStocks(c.Request.Context(), c.Query("warehouseId"))
	if err != nil {
		mapBizErr(c, err)
		return
	}
	response.OK(c, list)
}

type execReq struct {
	UniqueCode      string   `json:"uniqueCode" binding:"required"`
	ClientRequestID string   `json:"clientRequestId"`
	Lat             *float64 `json:"lat"`
	Lng             *float64 `json:"lng"`
	PhotoObjectKey  string   `json:"photoObjectKey"`
}

func mustUserID(c *gin.Context) uint64 {
	v, _ := c.Get(middleware.CtxUserID)
	s, _ := v.(string)
	id, _ := strconv.ParseUint(s, 10, 64)
	return id
}

func mapBizErr(c *gin.Context, err error) {
	msg := err.Error()
	switch {
	case strings.Contains(msg, errcode.StateInvalid):
		response.Fail(c, http.StatusConflict, errcode.StateInvalid, msg)
	case strings.Contains(msg, errcode.ValidationError):
		response.Fail(c, http.StatusBadRequest, errcode.ValidationError, msg)
	case strings.Contains(msg, errcode.NotFound):
		response.Fail(c, http.StatusNotFound, errcode.NotFound, msg)
	case strings.Contains(msg, errcode.AuthDenied):
		response.Fail(c, http.StatusForbidden, errcode.AuthDenied, msg)
	case strings.Contains(msg, errcode.IdempotentReplay):
		response.Fail(c, http.StatusOK, errcode.IdempotentReplay, msg)
	default:
		response.Fail(c, http.StatusInternalServerError, errcode.Internal, msg)
	}
}
