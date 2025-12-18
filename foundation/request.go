package foundation

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const maxBodySize = 1 << 20 // 1 MB

// Decode reads and decodes the JSON body of an HTTP request into a value of T.
// It limits the request body size, disallows unknown JSON fields, and rejects
// bodies containing more than a single JSON value.
func Decode[T any](w http.ResponseWriter, r *http.Request) (T, error) {
	body := http.MaxBytesReader(w, r.Body, maxBodySize)
	defer body.Close()

	dec := json.NewDecoder(body)
	dec.DisallowUnknownFields()

	var data T
	if err := dec.Decode(&data); err != nil {
		return data, fmt.Errorf("request: decode: %w", err)
	}

	// Ensure there is exactly one JSON value in the request body.
	// json.Decoder permits multiple values by default (e.g. "{}{}"), which we treat
	// as invalid input.
	var trailing struct{}
	if err := dec.Decode(&trailing); err != io.EOF {
		if err == nil {
			return data, fmt.Errorf("request: decode: body must contain a single JSON value")
		}
		return data, fmt.Errorf("request: decode: %w", err)
	}

	return data, nil
}
