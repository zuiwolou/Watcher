package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"watcher/backend/pkg/errcode"
	"watcher/backend/pkg/response"
	"watcher/backend/pkg/storage"
)

type MediaHandler struct {
	store *storage.MinIO
}

func NewMediaHandler(store *storage.MinIO) *MediaHandler {
	return &MediaHandler{store: store}
}

func (h *MediaHandler) UploadURL(c *gin.Context) {
	if h.store == nil {
		response.Fail(c, http.StatusServiceUnavailable, errcode.UpstreamUnavailable, "minio not configured")
		return
	}
	var req struct {
		ContentType string `json:"contentType" binding:"required"`
		Ext         string `json:"ext"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, errcode.ValidationError, err.Error())
		return
	}
	res, err := h.store.PresignPut(c.Request.Context(), req.ContentType, req.Ext, 15*time.Minute)
	if err != nil {
		response.Fail(c, http.StatusBadGateway, errcode.UpstreamUnavailable, err.Error())
		return
	}
	response.OK(c, res)
}
