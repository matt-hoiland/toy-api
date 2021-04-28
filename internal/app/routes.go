package app

import "net/http"

type Router interface {
	HandleFunc(pattern string, handler func(http.ResponseWriter, *http.Request))
	ServeHTTP(w http.ResponseWriter, r *http.Request)
}

func (s *server) routes() {
	s.router.HandleFunc("/echo", s.logRequest(s.logResponse(s.handleEcho())))
}
