package response

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"watcher/backend/pkg/errcode"
)

type Body struct {
	Code      string      `json:"code"`
	Message   string      `json:"message,omitempty"`
	Data      interface{} `json:"data,omitempty"`
	RequestID string      `json:"requestId,omitempty"`
}

type PageData struct {
	List     interface{} `json:"list"`
	Page     int         `json:"page"`
	PageSize int         `json:"pageSize"`
	Total    int64       `json:"total"`
}

func OK(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Body{
		Code:      errcode.OK,
		Data:      data,
		RequestID: requestID(c),
	})
}

func Fail(c *gin.Context, httpStatus int, code, message string) {
	c.JSON(httpStatus, Body{
		Code:      code,
		Message:   message,
		RequestID: requestID(c),
	})
}

func Page(c *gin.Context, list interface{}, page, pageSize int, total int64) {
	OK(c, PageData{List: list, Page: page, PageSize: pageSize, Total: total})
}

func requestID(c *gin.Context) string {
	if v, ok := c.Get("request_id"); ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}
