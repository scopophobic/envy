package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const RequestIDKey = "request_id"

func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader("X-Request-ID")
		if _, err := uuid.Parse(requestID); err != nil {
			requestID = uuid.NewString()
		}
		c.Set(RequestIDKey, requestID)
		c.Header("X-Request-ID", requestID)
		c.Next()
	}
}

func GetRequestID(c *gin.Context) string {
	requestID, _ := c.Get(RequestIDKey)
	value, _ := requestID.(string)
	return value
}
