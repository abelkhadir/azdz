package httpx

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
)

// DecodeJSON reads one strict JSON object with a small endpoint-specific limit.
func DecodeJSON(w http.ResponseWriter, r *http.Request, maxBytes int64, value any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(value); err != nil {
		return err
	}
	if err := decoder.Decode(new(any)); !errors.Is(err, io.EOF) {
		return errors.New("request body must contain one JSON object")
	}
	return nil
}

func QueryID(r *http.Request, name string) (int64, error) {
	id, err := strconv.ParseInt(r.URL.Query().Get(name), 10, 64)
	if err != nil || id <= 0 {
		return 0, strconv.ErrSyntax
	}
	return id, nil
}

func OptionalQueryID(r *http.Request, name string) (int64, error) {
	value := r.URL.Query().Get(name)
	if value == "" {
		return 0, nil
	}
	id, err := strconv.ParseInt(value, 10, 64)
	if err != nil || id < 0 {
		return 0, strconv.ErrSyntax
	}
	return id, nil
}

func QueryLimit(r *http.Request, fallback, maximum int64) (int64, error) {
	limit, err := OptionalQueryID(r, "limit")
	if err != nil {
		return 0, err
	}
	if limit == 0 {
		return fallback, nil
	}
	if limit > maximum {
		return maximum, nil
	}
	return limit, nil
}
