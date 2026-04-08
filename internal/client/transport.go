package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
)

type requestOptions struct {
	PathTemplate    string
	ShopperID       string
	AppKey          string
	MarketID        string
	RequestID       bool
	AllowStatusCode []int
}

func (c *Client) do(ctx context.Context, method, path string, body any, out any, opts requestOptions) (int, error) {
	if opts.PathTemplate == "" {
		opts.PathTemplate = path
	}

	if err := c.limiters.Wait(ctx, opts.PathTemplate); err != nil {
		return 0, err
	}

	var payload []byte
	var err error
	if body != nil {
		payload, err = json.Marshal(body)
		if err != nil {
			return 0, err
		}
	}

	endpoint := strings.TrimRight(c.config.BaseURL, "/") + path
	allowed := make(map[int]struct{}, len(opts.AllowStatusCode))
	for _, code := range opts.AllowStatusCode {
		allowed[code] = struct{}{}
	}

	var statusCode int
	for attempt := 0; attempt <= c.config.MaxRetries; attempt++ {
		var reqBody io.Reader
		if payload != nil {
			reqBody = bytes.NewReader(payload)
		}

		req, reqErr := http.NewRequestWithContext(ctx, method, endpoint, reqBody)
		if reqErr != nil {
			return 0, reqErr
		}

		if opts.ShopperID != "" {
			req.Header.Set("X-Shopper-Id", opts.ShopperID)
		}
		if opts.AppKey != "" {
			req.Header.Set("X-App-Key", opts.AppKey)
		}
		if opts.MarketID != "" {
			req.Header.Set("X-Market-Id", opts.MarketID)
		}
		if opts.RequestID {
			req.Header.Set("X-Request-Id", uuid.NewString())
		}

		if c.config.DebugHTTP {
			log.Printf("godaddy request %s %s", method, endpoint)
		}

		resp, doErr := c.httpClient.Do(req)
		retry, _ := shouldRetry(resp, doErr)
		if doErr != nil {
			if retry && attempt < c.config.MaxRetries {
				sleepWithContext(ctx, backoffDuration(attempt))
				continue
			}
			return 0, doErr
		}

		statusCode = resp.StatusCode
		respBody, readErr := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if readErr != nil {
			return statusCode, readErr
		}

		if _, ok := allowed[resp.StatusCode]; ok {
			if out != nil && len(respBody) > 0 {
				_ = json.Unmarshal(respBody, out)
			}
			return statusCode, nil
		}

		if resp.StatusCode >= 200 && resp.StatusCode <= 299 {
			if out != nil && len(respBody) > 0 {
				if err := json.Unmarshal(respBody, out); err != nil {
					return statusCode, err
				}
			}
			return statusCode, nil
		}

		if retry && attempt < c.config.MaxRetries {
			delay := backoffDuration(attempt)
			var limitErr *RateLimitError
			apiErr := parseAPIError(resp.StatusCode, respBody)
			if resp.StatusCode == http.StatusTooManyRequests && apiErr != nil && errors.As(apiErr, &limitErr) && limitErr.RetryAfterSec > 0 {
				delay = time.Duration(limitErr.RetryAfterSec) * time.Second
			}
			sleepWithContext(ctx, delay)
			continue
		}

		return statusCode, parseAPIError(resp.StatusCode, respBody)
	}

	return statusCode, fmt.Errorf("max retries exceeded for %s %s", method, endpoint)
}

func buildURL(path string, query url.Values) string {
	if len(query) == 0 {
		return path
	}
	return path + "?" + query.Encode()
}

func sleepWithContext(ctx context.Context, d time.Duration) {
	timer := time.NewTimer(d)
	defer timer.Stop()

	select {
	case <-ctx.Done():
	case <-timer.C:
	}
}
