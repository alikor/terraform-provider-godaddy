package client

import (
	"errors"
	"math"
	"math/rand"
	"net"
	"net/http"
	"time"
)

func shouldRetry(resp *http.Response, err error) (bool, time.Duration) {
	if err != nil {
		var netErr net.Error
		if errors.As(err, &netErr) {
			return true, 0
		}
		return true, 0
	}

	if resp == nil {
		return false, 0
	}

	if resp.StatusCode == http.StatusTooManyRequests {
		return true, 0
	}

	if resp.StatusCode >= 500 && resp.StatusCode <= 599 {
		return true, 0
	}

	return false, 0
}

func backoffDuration(attempt int) time.Duration {
	base := time.Second * time.Duration(math.Pow(2, float64(attempt)))
	if base > 16*time.Second {
		base = 16 * time.Second
	}

	jitterRange := int64(base / 5)
	if jitterRange == 0 {
		return base
	}

	jitter := time.Duration(rand.Int63n(jitterRange*2) - jitterRange)
	return base + jitter
}
