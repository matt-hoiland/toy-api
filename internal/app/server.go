package app

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	log "github.com/sirupsen/logrus"
)

type Server interface {
	http.Handler
}

type server struct {
	router Router
}

func NewServer(routerDep Router) Server {
	s := &server{
		router: routerDep,
	}
	s.routes()
	return s
}

func (s *server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.router.ServeHTTP(w, r)
}

func (s *server) handleError(code int, err error) http.HandlerFunc {
	type response struct {
		Error string `json:"error"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(code)
		res := response{
			Error: err.Error(),
		}
		json.NewEncoder(w).Encode(res)
	}
}

func (s *server) logRequest(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		headers, err := json.Marshal(r.Header)
		if err != nil {
			s.handleError(http.StatusInternalServerError, err)(w, r)
			return
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			s.handleError(http.StatusInternalServerError, err)(w, r)
		}
		r.Body.Close()
		r.Body = io.NopCloser(bytes.NewBuffer(body))
		requestLogger := log.WithFields(log.Fields{
			"method":   r.Method,
			"url":      r.URL.String(),
			"protocol": r.Proto,
			"headers":  string(headers),
			"body":     string(body),
		})
		requestLogger.Debug("request received")

		h(w, r)
	}
}

type responseWrapper struct {
	http.ResponseWriter
	Status      int
	WroteHeader bool
	Body        []byte
	WriteCount  int
}

func wrapResponseWriter(w http.ResponseWriter) *responseWrapper {
	return &responseWrapper{ResponseWriter: w}
}

func (rw *responseWrapper) WriteHeader(code int) {
	if rw.WroteHeader {
		return
	}
	rw.Status = code
	rw.ResponseWriter.WriteHeader(code)
	rw.WroteHeader = true
}

func (rw *responseWrapper) Write(body []byte) (int, error) {
	rw.WriteCount++
	rw.Body = make([]byte, len(body))
	copy(rw.Body, body)
	return rw.ResponseWriter.Write(body)
}

func (s *server) logResponse(h http.HandlerFunc) http.HandlerFunc {
	logger := log.WithFields(log.Fields{"func": "logResponse"})
	return func(w http.ResponseWriter, r *http.Request) {
		rw := wrapResponseWriter(w)
		h(rw, r)

		headers, err := json.Marshal(rw.Header())
		if err != nil {
			logger.WithError(err).Error("unable to marshal headers")
		}
		requestLogger := logger.WithFields(log.Fields{
			"status":      rw.Status,
			"headers":     string(headers),
			"body":        string(rw.Body),
			"write-count": rw.WriteCount,
		})
		requestLogger.Debug("response sent")
	}
}

func (s *server) handleEcho() http.HandlerFunc {
	type request struct {
		Message *string `json:"req"`
	}
	type response struct {
		Message string `json:"res"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			s.handleError(http.StatusMethodNotAllowed, fmt.Errorf("method not allowed, expected %s", http.MethodPost))(w, r)
			return
		}

		var req request
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			if errors.Is(err, io.EOF) {
				err = fmt.Errorf("missing request body")
			}
			var te *json.UnmarshalTypeError
			if errors.As(err, &te) {
				err = fmt.Errorf("incorrect json type given, expected object, received %s", te.Value)
			}
			var se *json.SyntaxError
			if errors.As(err, &se) {
				err = fmt.Errorf("json syntax error in request body, offset=%d: %v", se.Offset, se)
			}
			s.handleError(http.StatusBadRequest, err)(w, r)
			return
		}

		if req.Message == nil {
			s.handleError(http.StatusBadRequest, fmt.Errorf("missing field in request, expected \"req\""))(w, r)
			return
		}

		res := response{
			Message: *req.Message,
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(res)
	}
}
