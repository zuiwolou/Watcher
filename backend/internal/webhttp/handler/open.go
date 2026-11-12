package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"watcher/backend/pkg/response"
)

func (h *BizHandler) OpenListMaterials(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
	list, total, err := h.biz.ListMaterials(c.Request.Context(), page, size)
	if err != nil {
		mapBizErr(c, err)
		return
	}
	response.Page(c, list, page, size, total)
}

func (h *BizHandler) OpenListStocks(c *gin.Context) {
	list, err := h.biz.ListStocks(c.Request.Context(), c.Query("warehouseId"))
	if err != nil {
		mapBizErr(c, err)
		return
	}
	response.OK(c, list)
}

func (h *BizHandler) OpenListAlarms(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
	list, total, err := h.biz.ListAlarms(c.Request.Context(), page, size, c.Query("status"), c.Query("level"))
	if err != nil {
		mapBizErr(c, err)
		return
	}
	response.Page(c, list, page, size, total)
}

func (h *BizHandler) OpenListUniqueCodes(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
	list, total, err := h.biz.ListUniqueCodes(c.Request.Context(), c.Query("warehouseId"), c.Query("status"), c.Query("q"), page, size)
	if err != nil {
		mapBizErr(c, err)
		return
	}
	response.Page(c, list, page, size, total)
}
