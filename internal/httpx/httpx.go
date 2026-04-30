package httpx

import (
	"encoding/json"
	"errors"
	"net/http"
)

type ErrorResponse struct {
	Error string `json:"error"`
}

type StatusError struct {
	Code    int
	Message string
}

func (e *StatusError) Error() string {
	return e.Message
}

func BadRequest(msg string) *StatusError {
	return &StatusError{Code: http.StatusBadRequest, Message: msg}
}

func Unauthorized(msg string) *StatusError {
	return &StatusError{Code: http.StatusUnauthorized, Message: msg}
}

func NotFound(msg string) *StatusError {
	return &StatusError{Code: http.StatusNotFound, Message: msg}
}

func Conflict(msg string) *StatusError {
	return &StatusError{Code: http.StatusConflict, Message: msg}
}

func Decode(r *http.Request, dst any) error {
	defer r.Body.Close()
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	return dec.Decode(dst)
}

func JSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func Empty(w http.ResponseWriter, code int) {
	w.WriteHeader(code)
}

func HandleError(w http.ResponseWriter, err error) {
	if err == nil {
		return
	}
	var statusErr *StatusError
	if errors.As(err, &statusErr) {
		JSON(w, statusErr.Code, ErrorResponse{Error: statusErr.Message})
		return
	}
	JSON(w, http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
}

func CORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "authorization, content-type, x-device-id, x-group-id, x-room-device-id, x-room-device-token, x-room-id")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
