package responder

// HTTP 状态码相关的错误码
const (
	// 4xxx - 客户端错误
	ErrCodeBadRequest       = 4000 // 请求格式错误
	ErrCodeBindFailed       = 4001 // 参数绑定错误
	ErrCodeValidationFailed = 4002 // 数据验证失败
	ErrCodeNotFound         = 4003 // 资源不存在
	ErrCodeRouteNotFound    = 4004 // 路由不存在
	ErrCodeForbidden        = 4005 // 权限不足
	ErrCodeUnauthorized     = 4006 // 认证失败
	ErrCodeDuplicate        = 4007 // 资源已存在
	ErrCodeConflict         = 4008 // 数据冲突
	ErrCodeTooManyRequests  = 4009 // 请求过于频繁
	ErrCodeRateLimitExceeded = 4009 // 速率限制超出 (别名)

	// 5xxx - 服务端错误
	ErrCodeInternalServer  = 5000 // 内部服务器错误
	ErrCodeDatabase        = 5001 // 数据库错误
	ErrCodeBusinessLogic   = 5002 // 业务逻辑错误
	ErrCodeFileUpload      = 5003 // 文件上传失败
	ErrCodeStorageService  = 5004 // 存储服务错误
	ErrCodeExternalService = 5005 // 外部服务错误
	ErrCodeTimeout         = 5006 // 请求超时
)

// 错误消息映射
var errorMessages = map[int]string{
	ErrCodeBadRequest:       "Bad Request",
	ErrCodeBindFailed:       "Invalid Request Body",
	ErrCodeValidationFailed: "Validation Failed",
	ErrCodeNotFound:         "Resource Not Found",
	ErrCodeRouteNotFound:    "Route Not Found",
	ErrCodeForbidden:        "Forbidden",
	ErrCodeUnauthorized:     "Unauthorized",
	ErrCodeDuplicate:        "Resource Already Exists",
	ErrCodeConflict:         "Data Conflict",
	ErrCodeTooManyRequests:  "Too Many Requests",
	ErrCodeInternalServer:   "Internal Server Error",
	ErrCodeDatabase:         "Database Error",
	ErrCodeBusinessLogic:    "Business Logic Error",
	ErrCodeFileUpload:       "File Upload Failed",
	ErrCodeStorageService:   "Storage Service Error",
	ErrCodeExternalService:  "External Service Error",
	ErrCodeTimeout:          "Request Timeout",
}

// GetErrorMessage returns the default message for an error code
func GetErrorMessage(code int) string {
	if msg, ok := errorMessages[code]; ok {
		return msg
	}
	return "Unknown Error"
}

// NewError creates a new Error with code and message
func NewError(code int, message string) Error {
	if message == "" {
		message = GetErrorMessage(code)
	}
	return Error{
		Code:    code,
		Message: message,
	}
}

// NewErrorWithDetails creates a new Error with code, message and details
func NewErrorWithDetails(code int, message string, details any) Error {
	if message == "" {
		message = GetErrorMessage(code)
	}
	return Error{
		Code:    code,
		Message: message,
		Details: details,
	}
}

// Predefined errors for common scenarios
var (
	ErrBadRequest       = NewError(ErrCodeBadRequest, "")
	ErrBindFailed       = NewError(ErrCodeBindFailed, "")
	ErrValidationFailed = NewError(ErrCodeValidationFailed, "")
	ErrNotFound         = NewError(ErrCodeNotFound, "")
	ErrRouteNotFound    = NewError(ErrCodeRouteNotFound, "")
	ErrForbidden        = NewError(ErrCodeForbidden, "")
	ErrUnauthorized     = NewError(ErrCodeUnauthorized, "")
	ErrDuplicate        = NewError(ErrCodeDuplicate, "")
	ErrConflict         = NewError(ErrCodeConflict, "")
	ErrTooManyRequests  = NewError(ErrCodeTooManyRequests, "")
	ErrInternalServer   = NewError(ErrCodeInternalServer, "")
	ErrDatabase         = NewError(ErrCodeDatabase, "")
	ErrBusinessLogic    = NewError(ErrCodeBusinessLogic, "")
	ErrFileUpload       = NewError(ErrCodeFileUpload, "")
	ErrStorageService   = NewError(ErrCodeStorageService, "")
	ErrExternalService  = NewError(ErrCodeExternalService, "")
	ErrTimeout          = NewError(ErrCodeTimeout, "")
)
