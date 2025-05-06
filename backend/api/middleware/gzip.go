package middleware

import (
	"compress/gzip"
	"github.com/gin-gonic/gin"
	"io"
	"net/http"
	"strings"
)

// GzipDecodeMiddleware decompresses gzipped request bodies
func GzipDecodeMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.GetHeader("Content-Encoding") == "gzip" {
			gzipReader, err := gzip.NewReader(c.Request.Body)
			if err != nil {
				c.AbortWithStatus(http.StatusBadRequest)
				return
			}
			defer gzipReader.Close()

			// Replace the request body with the decompressed data
			c.Request.Body = io.NopCloser(gzipReader)
		}

		// Continue processing the request
		c.Next()
	}
}

// gzipWriter implements ResponseWriter and compresses data using gzip
type gzipWriter struct {
	gin.ResponseWriter
	writer *gzip.Writer
}

// Write compresses the data and writes it to the underlying response writer
func (g *gzipWriter) Write(data []byte) (int, error) {
	return g.writer.Write(data)
}

// WriteString compresses the string and writes it to the underlying response writer
func (g *gzipWriter) WriteString(s string) (int, error) {
	return g.writer.Write([]byte(s))
}

// GzipEncodeMiddleware compresses response bodies with gzip
func GzipEncodeMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check if client accepts gzip encoding
		if !strings.Contains(c.Request.Header.Get("Accept-Encoding"), "gzip") {
			c.Next()
			return
		}

		// Skip compression for SSE which need to flush each message
		if strings.Contains(c.GetHeader("Content-Type"), "text/event-stream") {
			c.Next()
			return
		}

		// Initialize gzip writer
		gz, err := gzip.NewWriterLevel(c.Writer, gzip.BestCompression)
		if err != nil {
			c.Next()
			return
		}
		defer gz.Close()

		// Create a gzipWriter to replace the original response writer
		gzipWriter := &gzipWriter{
			ResponseWriter: c.Writer,
			writer:         gz,
		}

		c.Writer = gzipWriter
		c.Header("Content-Encoding", "gzip")
		c.Header("Vary", "Accept-Encoding")

		c.Next()
	}
}
