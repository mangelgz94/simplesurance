package internal

import (
	"context"
	"encoding/json"
	"github.com/sirupsen/logrus"
	"net/http"
	"regexp"
	"strings"
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
	service service
}

func NewHandler(service service) *Handler {
	return &Handler{
		service: service,
	}
}

func (s *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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
