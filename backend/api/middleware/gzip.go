package middleware

import (
	"compress/gzip"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
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

type lazyGzipWriter struct {
	gin.ResponseWriter
	gzWriter           *gzip.Writer
	wroteHeader        bool
	compressionDecided bool
	enableCompression  bool
	minCompressionSize int // Optional: minimum size to compress
}

func (w *lazyGzipWriter) tryInitCompression() {
	if w.compressionDecided {
		return
	}
	w.compressionDecided = true

	// Check Content-Type. If it's event-stream, disable compression.
	contentType := w.Header().Get("Content-Type")
	if strings.Contains(contentType, "text/event-stream") {
		w.enableCompression = false
		return
	}

	// If we reach here, and client accepts gzip (checked in middleware), enable compression.
	w.enableCompression = true
}

func (w *lazyGzipWriter) WriteHeader(statusCode int) {
	if w.wroteHeader {
		return
	}
	w.tryInitCompression()

	if w.enableCompression {
		w.Header().Set("Content-Encoding", "gzip")
		w.Header().Set("Vary", "Accept-Encoding")
		// Initialize gzip.Writer only if compression is enabled
		if w.gzWriter == nil {
			// It's possible NewWriterLevel could fail, but it's rare with valid levels.
			// For simplicity in this example, error handling is omitted.
			// In production, you might want to handle this error, e.g., by falling back to no compression.
			w.gzWriter, _ = gzip.NewWriterLevel(w.ResponseWriter, gzip.BestCompression)
		}
	}
	w.ResponseWriter.WriteHeader(statusCode)
	w.wroteHeader = true
}

func (w *lazyGzipWriter) Write(data []byte) (int, error) {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}

	if !w.enableCompression {
		return w.ResponseWriter.Write(data)
	}

	// Ensure gzWriter is initialized (it should be if enableCompression is true and WriteHeader was called)
	if w.gzWriter == nil {
		// This case implies WriteHeader wasn't called with a status that would initialize gzWriter
		// OR tryInitCompression decided to enable but gzWriter init failed/skipped.
		// Fallback to uncompressed to be safe, or handle error appropriately.
		// For this refactor, we assume WriteHeader path correctly initializes if enableCompression is true.
		// If gzWriter is still nil here and enableCompression is true, it's an issue.
		// However, the current WriteHeader logic should initialize it.
		// If we hit this, it's a bug or unhandled edge case in initialization logic.
		// Let's proceed as if it's initialized if enableCompression is true.
		// A more robust solution would re-check and initialize here if necessary.
		// For now, relying on WriteHeader's init.
		if w.gzWriter == nil { // Double check, this should ideally not be needed
			return w.ResponseWriter.Write(data) // Fallback if something went wrong
		}
	}
	return w.gzWriter.Write(data)
}

func (w *lazyGzipWriter) WriteString(s string) (int, error) {
	return w.Write([]byte(s))
}

// Close is important to flush the GZIP writer
func (w *lazyGzipWriter) Close() error {
	if w.gzWriter != nil {
		return w.gzWriter.Close()
	}
	return nil
}

// GzipEncodeMiddleware compresses response bodies with gzip
func GzipEncodeMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !strings.Contains(c.Request.Header.Get("Accept-Encoding"), "gzip") {
			c.Next()
			return
		}

		// We remove the early exit based on path or Content-Type here.
		// The decision will be made lazily by the lazyGzipWriter.

		lgw := &lazyGzipWriter{
			ResponseWriter: c.Writer,
			// minCompressionSize: 20, // Example: only compress if > 20 bytes. Set to 0 to always compress if type matches.
		}
		c.Writer = lgw

		defer func() {
			// It's crucial to Close the writer to flush any buffered data for GZIP.
			// Also, ensure headers set by writer are actually sent if Write/WriteHeader wasn't called.
			// Gin typically handles WriteHeader implicitly if not called.
			// The Close here is primarily for the gzip.Writer.
			if lgw.gzWriter != nil { // Only close if it was initialized
				lgw.Close()
			}
		}()

		c.Next()
	}
}
