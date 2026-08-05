package response

import (
	"github.com/gofiber/fiber/v3"
)

type JSONResponse struct {
	Code      Code        `json:"code"`
	RequestID string      `json:"requestId"`
	Msg       string      `json:"msg"`
	Data      interface{} `json:"data"`
}

// PageData soybean 对齐的分页结构
type PageData struct {
	Records any   `json:"records"`
	Current int   `json:"current"`
	Size    int   `json:"size"`
	Total   int64 `json:"total"`
}

// NewPageData 构造分页 data
func NewPageData(records any, current, size int, total int64) PageData {
	if current <= 0 {
		current = 1
	}
	if size <= 0 {
		size = 10
	}
	return PageData{Records: records, Current: current, Size: size, Total: total}
}

// JSON 基础返回
func JSON(ctx fiber.Ctx, status int, errcode Code, message string, data interface{}) error {
	if message == "" {
		message = CodeMap[errcode]
	}
	return ctx.Status(status).JSON(JSONResponse{
		Code:      errcode,
		Msg:       message,
		RequestID: ctx.GetRespHeader("X-Request-Id"),
		Data:      data,
	})
}

// SuccessJSON 成功返回
func SuccessJSON(ctx fiber.Ctx, message string, data interface{}) error {
	if message == "" {
		message = Success.Msg()
	}
	return JSON(ctx, fiber.StatusOK, Success, message, data)
}

// BadRequestException 业务失败（HTTP 200 + 业务码，对齐 soybean 前端拦截器）
func BadRequestException(ctx fiber.Ctx, message string) error {
	if message == "" {
		message = CodeMap[RequestParamErr]
	}
	return JSON(ctx, fiber.StatusOK, RequestParamErr, message, nil)
}

// UnauthorizedException 未认证（HTTP 200 + 业务码，便于前端 logoutCodes 拦截）
func UnauthorizedException(ctx fiber.Ctx, message string) error {
	if message == "" {
		message = CodeMap[UnAuthed]
	}
	return JSON(ctx, fiber.StatusOK, UnAuthed, message, nil)
}

// AuthExpiredException 会话过期（HTTP 200 + code，便于前端 refresh 流程）
func AuthExpiredException(ctx fiber.Ctx, message string) error {
	if message == "" {
		message = CodeMap[AuthExpired]
	}
	return JSON(ctx, fiber.StatusOK, AuthExpired, message, nil)
}

// ForbiddenException 403错误
func ForbiddenException(ctx fiber.Ctx, message string) error {
	if message == "" {
		message = CodeMap[Failed]
	}
	return JSON(ctx, fiber.StatusForbidden, Failed, message, nil)
}

// NotFoundException 404错误
func NotFoundException(ctx fiber.Ctx, message string) error {
	if message == "" {
		message = CodeMap[RequestMethodErr]
	}
	return JSON(ctx, fiber.StatusNotFound, RequestMethodErr, message, nil)
}

// InternalServerException 500错误
func InternalServerException(ctx fiber.Ctx, message string) error {
	if message == "" {
		message = CodeMap[InternalErr]
	}
	return JSON(ctx, fiber.StatusInternalServerError, InternalErr, message, nil)
}
