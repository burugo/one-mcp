package proxy

import (
	"net/http"
	"strings"
)

// WrapSessionErrorFixingHandler adjusts invalid session responses to 404
// so clients can re-initialize per MCP streamable HTTP expectations.
func WrapSessionErrorFixingHandler(handler http.Handler) http.Handler {
	if handler == nil {
		return handler
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			wrapper := &sessionErrorFixingResponseWriter{ResponseWriter: w}
			handler.ServeHTTP(wrapper, r)
			wrapper.flushWithFix()
			return
		}
		handler.ServeHTTP(w, r)
	})
}

// sessionErrorFixingResponseWriter wraps http.ResponseWriter to fix mcp-go's incorrect
// status code for invalid/expired sessions. Per MCP spec, invalid session should return
// 404 Not Found (so client re-initializes), but mcp-go returns 400 Bad Request.
// Only buffers small error responses; passes through SSE streams and large responses directly.
type sessionErrorFixingResponseWriter struct {
	http.ResponseWriter
	statusCode    int
	headerWritten bool
	buffer        []byte
	passthrough   bool // true if we've determined this is a streaming response
}

func (w *sessionErrorFixingResponseWriter) WriteHeader(statusCode int) {
	w.statusCode = statusCode
	// Don't write header yet - wait to check if we need to fix it
}

func (w *sessionErrorFixingResponseWriter) Write(data []byte) (int, error) {
	if w.passthrough {
		return w.ResponseWriter.Write(data)
	}
	if !w.headerWritten {
		// Only buffer small responses (likely error messages)
		// For larger responses or SSE streams, switch to passthrough mode
		if len(w.buffer)+len(data) > 512 {
			w.flushWithFix()
			w.passthrough = true
			return w.ResponseWriter.Write(data)
		}
		w.buffer = append(w.buffer, data...)
		return len(data), nil
	}
	return w.ResponseWriter.Write(data)
}

// Flush implements http.Flusher for SSE streaming support
func (w *sessionErrorFixingResponseWriter) Flush() {
	if !w.headerWritten {
		w.flushWithFix()
	}
	w.passthrough = true
	if flusher, ok := w.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

func (w *sessionErrorFixingResponseWriter) flushWithFix() {
	if w.headerWritten {
		return
	}
	w.headerWritten = true

	// Fix: If status is 400 and body contains session-related error, change to 404
	// This follows MCP spec: invalid/expired session should return 404 so client re-initializes
	if w.statusCode == http.StatusBadRequest {
		bodyStr := string(w.buffer)
		if strings.Contains(bodyStr, "Invalid session ID") ||
			strings.Contains(bodyStr, "session not found") {
			w.statusCode = http.StatusNotFound
		}
	}

	if w.statusCode != 0 {
		w.ResponseWriter.WriteHeader(w.statusCode)
	}
	if len(w.buffer) > 0 {
		w.ResponseWriter.Write(w.buffer)
	}
}
