package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gopal-chhetri/url-shortener/internal/utils"
)

// swagger:model
type Response struct {
	ErrorMessage string `json:"message,omitempty"`
	Data         any    `json:"data,omitempty"`
	Error        bool   `json:"error"`
}

//swagger:model
type PaginatedResponse struct {
	ErrorMessage string                    `json:"message,omitempty"`
	Data         any                       `json:"data,omitempty"`
	Error        bool                      `json:"error"`
	Metadata     *utils.PaginationResponse `json:"metadata,omitempty"`
}

type HttpError struct {
	Message string `json:"message"`
	Code    int    `json:"code"`
}

func (e HttpError) Error() string {
	return e.Message
}

func NewHttpError(ctx *gin.Context, statusCode int, message string) {
	result := HttpError{
		Message: message,
		Code:    statusCode,
	}
	ctx.JSON(statusCode, result)
}

func SuccessResponse(ctx *gin.Context, data interface{}) {
	result := Response{
		Data:  data,
		Error: false,
	}
	ctx.JSON(http.StatusOK, result)
}

func SuccessCreatedResponse(ctx *gin.Context, data interface{}) {
	result := Response{
		Data:  data,
		Error: false,
	}
	ctx.JSON(http.StatusCreated, result)
}

func SuccessPaginatedResponse(ctx *gin.Context, data interface{}, pagination *utils.PaginationResponse) {
	result := PaginatedResponse{
		Data:     data,
		Error:    false,
		Metadata: pagination,
	}
	ctx.JSON(http.StatusOK, result)
}

// SuccessWithPaginationResponse returns a success response with pagination metadata
func SuccessWithPaginationResponse(ctx *gin.Context, data interface{}, pagination *utils.PaginationResponse) {
	result := PaginatedResponse{
		Data:     data,
		Error:    false,
		Metadata: pagination,
	}
	ctx.JSON(http.StatusOK, result)
}

func ServerError(ctx *gin.Context, message string) {
	NewHttpError(ctx, http.StatusInternalServerError, message)
}

func InternalServerError(ctx *gin.Context) {
	ServerError(ctx, "Internal Server Error")
}

func BadRequestResponse(ctx *gin.Context) {
	NewHttpError(ctx, http.StatusBadRequest, "Bad Request")
}

func NotFoundResponse(ctx *gin.Context, err string) {
	NewHttpError(ctx, http.StatusNotFound, err)
}

func UnauthorizedResponse(ctx *gin.Context, message string) {
	NewHttpError(ctx, http.StatusUnauthorized, message)
}

func ServerErrorResponse(ctx *gin.Context, err error) {
	InternalServerError(ctx)
}

func ValidationErrorResponse(ctx *gin.Context, err error) {
	NewHttpError(ctx, http.StatusBadRequest, err.Error())
}

func ErrorResponse(ctx *gin.Context, err error) {
	switch err.(type) {
	case InvalidOperation:
		NewHttpError(ctx, http.StatusBadRequest, err.Error())
	case PermissionDeniedError:
		NewHttpError(ctx, http.StatusForbidden, err.Error())
	case AppError:
		NewHttpError(ctx, http.StatusBadRequest, err.Error())
	case NotFoundError:
		NotFoundResponse(ctx, err.Error())
	case DuplicateData:
		NewHttpError(ctx, http.StatusBadRequest, err.Error())

	default:
		ServerErrorResponse(ctx, err)
	}
}
