package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// RequestBodyLimit rejects known oversized requests immediately and also wraps
// chunked bodies so handlers cannot read beyond the configured limit.
func RequestBodyLimit(maxBytes int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		if maxBytes <= 0 || c.Request.Body == nil {
			c.Next()
			return
		}
		if c.Request.ContentLength > maxBytes {
			c.AbortWithStatusJSON(http.StatusRequestEntityTooLarge, gin.H{"error": "Request body too large"})
			return
		}
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes)
		c.Next()
	}
}

// NoStoreAPI prevents browsers and reverse proxies from persisting API
// responses. Envo's safe performance cache is explicit and in-memory on the
// frontend; plaintext secret responses must never enter an HTTP cache.
func NoStoreAPI() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Cache-Control", "no-store, max-age=0")
		c.Header("Pragma", "no-cache")
		c.Header("Expires", "0")
		c.Next()
	}
}
