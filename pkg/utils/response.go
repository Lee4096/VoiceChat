package utils

import "github.com/gin-gonic/gin"

type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

type ErrorDetail struct {
	Code    string      `json:"code"`
	Message string      `json:"message"`
	Details interface{} `json:"details,omitempty"`
}

func NewErrorResponse(c *gin.Context, code string, message string, statusCode int) {
	c.JSON(statusCode, ErrorResponse{
		Error: ErrorDetail{
			Code:    code,
			Message: message,
		},
	})
}

const (
	ErrCodeAuthInvalidToken   = "AUTH_001"
	ErrCodeAuthTokenExpired    = "AUTH_002"
	ErrCodeAuthOAuthFailed     = "AUTH_003"
	ErrCodeRoomNotFound        = "ROOM_001"
	ErrCodeRoomFull            = "ROOM_002"
	ErrCodeRoomUnauthorized    = "ROOM_003"
	ErrCodeVoiceASRUnavailable = "VOICE_001"
	ErrCodeVoiceTTSUnavailable = "VOICE_002"
	ErrCodeLLMUnavailable      = "LLM_001"
	ErrCodeRateLimit           = "RATE_001"
	ErrCodeInternal            = "INTERNAL"
)
