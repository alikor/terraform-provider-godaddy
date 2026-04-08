package client

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"sync"
	"time"
)

type Config struct {
	APIKey         string
	APISecret      string
	BaseURL        string
	ShopperID      string
	CustomerID     string
	AppKey         string
	MarketID       string
	RequestTimeout time.Duration
	PollInterval   time.Duration
	MaxRetries     int
	RateLimitRPM   int
	UserAgent      string
	DebugHTTP      bool
}

type Client struct {
	config           Config
	httpClient       *http.Client
	limiters         *endpointRateLimiter
	customerIDMu     sync.Mutex
	cachedCustomerID string
}

func New(config Config) *Client {
	baseTransport := http.DefaultTransport
	httpClient := &http.Client{
		Timeout: config.RequestTimeout,
		Transport: &authTransport{
			base:      baseTransport,
			apiKey:    config.APIKey,
			apiSecret: config.APISecret,
			userAgent: config.UserAgent,
		},
	}

	return &Client{
		config:           config,
		httpClient:       httpClient,
		limiters:         newEndpointRateLimiter(config.RateLimitRPM),
		cachedCustomerID: config.CustomerID,
	}
}

func (c *Client) ResolveCustomerID(ctx context.Context) (string, error) {
	if c.config.CustomerID != "" {
		return c.config.CustomerID, nil
	}

	c.customerIDMu.Lock()
	if c.cachedCustomerID != "" {
		defer c.customerIDMu.Unlock()
		return c.cachedCustomerID, nil
	}
	c.customerIDMu.Unlock()

	if c.config.ShopperID == "" {
		return "", errors.New("customer_id is required for this operation; set provider.customer_id or provider.shopper_id")
	}

	shopper, err := c.GetShopper(ctx, c.config.ShopperID, true)
	if err != nil {
		return "", err
	}

	if shopper.CustomerID == "" {
		return "", errors.New("unable to resolve customer_id from shopper lookup")
	}

	c.customerIDMu.Lock()
	c.cachedCustomerID = shopper.CustomerID
	c.customerIDMu.Unlock()

	return shopper.CustomerID, nil
}

func ResolveBaseURL(endpoint, override string) string {
	if trimmed := strings.TrimRight(strings.TrimSpace(override), "/"); trimmed != "" {
		return trimmed
	}

	if strings.EqualFold(endpoint, "ote") {
		return "https://api.ote-godaddy.com"
	}

	return "https://api.godaddy.com"
}

func (c *Client) Config() Config {
	return c.config
}
