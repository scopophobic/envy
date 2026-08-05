package handlers

import (
	"log"
	"net/http"

	"github.com/envo/backend/internal/middleware"
	"github.com/gin-gonic/gin"
)

func respondInternalError(c *gin.Context, message string, err error) {
	requestID := middleware.GetRequestID(c)
	log.Printf("[request %s] %s %s: %v", requestID, c.Request.Method, c.Request.URL.Path, err)
	response := gin.H{"error": message}
	if requestID != "" {
		response["request_id"] = requestID
	}
	c.JSON(http.StatusInternalServerError, response)
}
