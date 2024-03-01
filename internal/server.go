package internal

import (
	"context"
	"encoding/json"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

type service interface {
	GetPreviousTotalRequests(ctx context.Context) (int, error)
}

type payload struct {
	RequestAmount int `json:"request_amount"`
}

type route struct {
	method  string
	regex   *regexp.Regexp
	handler http.HandlerFunc
}

type Handler struct {
	service      service
	requestsPool chan bool
	config       *ServerConfig
}

type ServerConfig struct {
	MaxConnections int
	SleepTime      int
}

func NewHandler(service service, config *ServerConfig) *Handler {
	return &Handler{
		service:      service,
		requestsPool: make(chan bool, config.MaxConnections),
		config:       config,
	}
}

func (s *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.requestsPool <- true
	defer func() {
		<-s.requestsPool
	}()
	time.Sleep(time.Duration(s.config.SleepTime) * time.Second)

	var allow []string
	route := route{"GET", regexp.MustCompile("^/$"), s.getPreviousTotalRequests}

	matches := route.regex.FindStringSubmatch(r.URL.Path)
	if len(matches) > 0 {
		if r.Method != route.method {
			allow = append(allow, route.method)
			return
		}

		route.handler(w, r.WithContext(r.Context()))
		return
	}

	if len(allow) > 0 {
		w.Header().Set("Allow", strings.Join(allow, ", "))
		http.Error(w, "405 method not allowed", http.StatusMethodNotAllowed)
		return
	}

	http.NotFound(w, r)
}

func (s *Handler) getPreviousTotalRequests(w http.ResponseWriter, r *http.Request) {
	requestAmount, err := s.service.GetPreviousTotalRequests(r.Context())
	if err != nil {
		logrus.Errorf("an error ocurred trying to get previous total requests, %v", err)
		respondWithError(w, http.StatusInternalServerError, "internal server error")

		return
	}

	respondWithJSON(w, http.StatusOK, &payload{RequestAmount: requestAmount})
}

func respondWithError(writer http.ResponseWriter, code int, message string) {
	respondWithJSON(writer, code, map[string]string{"error": message})
}

func respondWithJSON(writer http.ResponseWriter, code int, payload interface{}) {
	response, _ := json.Marshal(payload)

	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(code)
	writer.Write(response)
}
