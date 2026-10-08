package middleware

import (
	"errors"
	"io"
	"net/http"

	"github.com/apirout/apirout/internal/gateway/responder"
)

// PayloadShield creates an ingress HTTP middleware that bounds request body streams to maxBytes.
// If the payload exceeds the limit, it cleanly halts stream processing and emits a standardized
// HTTP 413 error envelope with canonical code PAYLOAD_TOO_LARGE and correlation tracking.
func PayloadShield(maxBytes int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if maxBytes <= 0 {
				next.ServeHTTP(w, r)
				return
			}

			// 1. Fast-path check: Content-Length header explicitly exceeds limit
			if r.ContentLength > maxBytes {
				responder.Error(w, r, http.StatusRequestEntityTooLarge, responder.ErrCodePayloadTooLarge, "Request entity exceeds maximum permitted size")
				return
			}

			if r.Body == nil {
				next.ServeHTTP(w, r)
				return
			}

			// 2. Wrap body stream with http.MaxBytesReader
			var exceeded bool
			wrappedBody := &shieldReadCloser{
				ReadCloser: http.MaxBytesReader(w, r.Body, maxBytes),
				exceeded:   &exceeded,
			}
			r.Body = wrappedBody

			// 3. Wrap ResponseWriter to intercept downstream error handling if MaxBytesError occurs
			srw := &shieldResponseWriter{
				ResponseWriter: w,
				r:              r,
				exceeded:       &exceeded,
			}

			next.ServeHTTP(srw, r)

			// 4. If reading tripped MaxBytesError and downstream didn't write, emit canonical 413
			if exceeded && !srw.written {
				srw.written = true
				responder.Error(w, r, http.StatusRequestEntityTooLarge, responder.ErrCodePayloadTooLarge, "Request entity exceeds maximum permitted size")
			}
		})
	}
}

type shieldReadCloser struct {
	io.ReadCloser
	exceeded *bool
}

func (s *shieldReadCloser) Read(p []byte) (int, error) {
	n, err := s.ReadCloser.Read(p)
	if err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			*s.exceeded = true
		}
	}
	return n, err
}

func (s *shieldReadCloser) Close() error {
	return s.ReadCloser.Close()
}

type shieldResponseWriter struct {
	http.ResponseWriter
	r        *http.Request
	exceeded *bool
	written  bool
}

func (s *shieldResponseWriter) WriteHeader(code int) {
	if *s.exceeded {
		if !s.written {
			s.written = true
			responder.Error(s.ResponseWriter, s.r, http.StatusRequestEntityTooLarge, responder.ErrCodePayloadTooLarge, "Request entity exceeds maximum permitted size")
		}
		return
	}
	s.written = true
	s.ResponseWriter.WriteHeader(code)
}

func (s *shieldResponseWriter) Write(b []byte) (int, error) {
	if *s.exceeded {
		if !s.written {
			s.WriteHeader(http.StatusRequestEntityTooLarge)
		}
		return len(b), nil
	}
	s.written = true
	return s.ResponseWriter.Write(b)
}

func (s *shieldResponseWriter) Flush() {
	if f, ok := s.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}
