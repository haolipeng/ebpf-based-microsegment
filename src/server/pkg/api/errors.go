package api

import (
	"time"

	"github.com/gin-gonic/gin"
)

// ErrorResponse 标准错误响应格式
type ErrorResponse struct {
	Success   bool        `json:"success"`
	Error     *ErrorDetail `json:"error"`
	Timestamp string      `json:"timestamp"`
}

// ErrorDetail 错误详情
type ErrorDetail struct {
	Code    string      `json:"code"`
	Message string      `json:"message"`
	Details interface{} `json:"details,omitempty"`
}

// SuccessResponse 标准成功响应格式
type SuccessResponse struct {
	Success   bool        `json:"success"`
	Data      interface{} `json:"data"`
	Timestamp string      `json:"timestamp"`
}

// 错误代码常量
const (
	ErrCodeBadRequest          = "BAD_REQUEST"
	ErrCodeUnauthorized        = "UNAUTHORIZED"
	ErrCodeForbidden           = "FORBIDDEN"
	ErrCodeNotFound            = "NOT_FOUND"
	ErrCodeConflict            = "CONFLICT"
	ErrCodeInternalServerError = "INTERNAL_SERVER_ERROR"
	ErrCodeInvalidParameter    = "INVALID_PARAMETER"
	ErrCodeDatabaseError       = "DATABASE_ERROR"
)

// RespondWithError 返回标准错误响应
func RespondWithError(c *gin.Context, statusCode int, code string, message string, details interface{}) {
	c.JSON(statusCode, ErrorResponse{
		Success: false,
		Error: &ErrorDetail{
			Code:    code,
			Message: message,
			Details: details,
		},
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	})
}

// RespondWithSuccess 返回标准成功响应
func RespondWithSuccess(c *gin.Context, statusCode int, data interface{}) {
	c.JSON(statusCode, SuccessResponse{
		Success:   true,
		Data:      data,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	})
}

// RespondBadRequest 返回 400 错误
func RespondBadRequest(c *gin.Context, message string, details interface{}) {
	RespondWithError(c, 400, ErrCodeBadRequest, message, details)
}

// RespondNotFound 返回 404 错误
func RespondNotFound(c *gin.Context, message string) {
	RespondWithError(c, 404, ErrCodeNotFound, message, nil)
}

// RespondInternalError 返回 500 错误
func RespondInternalError(c *gin.Context, message string) {
	RespondWithError(c, 500, ErrCodeInternalServerError, message, nil)
}

// RespondUnauthorized 返回 401 错误
func RespondUnauthorized(c *gin.Context, message string) {
	RespondWithError(c, 401, ErrCodeUnauthorized, message, nil)
}

// RespondForbidden 返回 403 错误
func RespondForbidden(c *gin.Context, message string) {
	RespondWithError(c, 403, ErrCodeForbidden, message, nil)
}
