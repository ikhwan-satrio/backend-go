package httpx

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type ErrorResponse struct {
	Error string `json:"error"`
}

func DecodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	// Limit body size (e.g. 1 MB) to stay safe from huge payloads
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	if err := dec.Decode(dst); err != nil {
		var syntaxErr *json.SyntaxError
		var unmarshalTypeErr *json.UnmarshalTypeError
		var maxBytesErr *http.MaxBytesError

		switch {
		case errors.As(err, &syntaxErr):
			return fmt.Errorf("malformed JSON at position %d", syntaxErr.Offset)
		case errors.Is(err, io.ErrUnexpectedEOF):
			return errors.New("malformed JSON")
		case errors.As(err, &unmarshalTypeErr):
			return fmt.Errorf("invalid value for field %q", unmarshalTypeErr.Field)
		case errors.Is(err, io.EOF):
			return errors.New("request body must not be empty")
		case errors.As(err, &maxBytesErr):
			return fmt.Errorf("request body must not exceed %d bytes", maxBytesErr.Limit)
		case strings.HasPrefix(err.Error(), "json: unknown field "):
			fieldName := strings.TrimPrefix(err.Error(), "json: unknown field ")
			return fmt.Errorf("unknown field %s", fieldName)
		default:
			return err
		}
	}

	// Ensure the body contains only a single JSON object
	if err := dec.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return errors.New("request body must only contain a single JSON object")
	}

	return nil
}
func JSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)

	if data == nil {
		return
	}

	if err := json.NewEncoder(w).Encode(data); err != nil {
		http.Error(w, `{"error":"failed to encode response"}`, http.StatusInternalServerError)
	}
}

func Error(w http.ResponseWriter, status int, message string) {
	JSON(w, status, ErrorResponse{Error: message})
}
