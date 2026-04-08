package client

import (
	"encoding/json"
	"fmt"
)

type APIError struct {
	StatusCode int
	Code       string
	Message    string
	Fields     []APIErrorField
	RawBody    []byte
}

func (e *APIError) Error() string {
	if e == nil {
		return ""
	}

	if e.Code != "" && e.Message != "" {
		return fmt.Sprintf("godaddy api error (%d %s): %s", e.StatusCode, e.Code, e.Message)
	}

	if e.Message != "" {
		return fmt.Sprintf("godaddy api error (%d): %s", e.StatusCode, e.Message)
	}

	return fmt.Sprintf("godaddy api error (%d)", e.StatusCode)
}

type APIErrorField struct {
	Path        string `json:"path,omitempty"`
	PathRelated string `json:"pathRelated,omitempty"`
	Code        string `json:"code,omitempty"`
	Message     string `json:"message,omitempty"`
}

type RateLimitError struct {
	APIError
	RetryAfterSec int
}

func (e *RateLimitError) Error() string {
	if e == nil {
		return ""
	}

	if e.RetryAfterSec > 0 {
		return fmt.Sprintf("%s (retry after %ds)", e.APIError.Error(), e.RetryAfterSec)
	}

	return e.APIError.Error()
}

type apiErrorEnvelope struct {
	Code          string          `json:"code"`
	Message       string          `json:"message"`
	Fields        []APIErrorField `json:"fields"`
	RetryAfterSec int             `json:"retryAfterSec"`
}

func parseAPIError(statusCode int, body []byte) error {
	var payload apiErrorEnvelope

	if len(body) > 0 {
		_ = json.Unmarshal(body, &payload)
	}

	base := APIError{
		StatusCode: statusCode,
		Code:       payload.Code,
		Message:    payload.Message,
		Fields:     payload.Fields,
		RawBody:    body,
	}

	if statusCode == 429 {
		return &RateLimitError{
			APIError:      base,
			RetryAfterSec: payload.RetryAfterSec,
		}
	}

	return &base
}
