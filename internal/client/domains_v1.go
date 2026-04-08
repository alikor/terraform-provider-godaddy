package client

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

type domainUpdateRequest struct {
	Locked                       *bool    `json:"locked,omitempty"`
	RenewAuto                    *bool    `json:"renewAuto,omitempty"`
	ExposeRegistrantOrganization *bool    `json:"exposeRegistrantOrganization,omitempty"`
	ExposeWhois                  *bool    `json:"exposeWhois,omitempty"`
	Consent                      *Consent `json:"consent,omitempty"`
}

func (c *Client) GetDomain(ctx context.Context, domain string) (*Domain, error) {
	var out Domain
	_, err := c.do(ctx, http.MethodGet, fmt.Sprintf("/v1/domains/%s", domain), nil, &out, requestOptions{
		PathTemplate: "/v1/domains/{domain}",
		ShopperID:    c.config.ShopperID,
	})
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) ListDomains(ctx context.Context, query url.Values) ([]DomainSummary, error) {
	var out []DomainSummary
	_, err := c.do(ctx, http.MethodGet, buildURL("/v1/domains", query), nil, &out, requestOptions{
		PathTemplate: "/v1/domains",
		ShopperID:    c.config.ShopperID,
	})
	return out, err
}

func (c *Client) GetAgreements(ctx context.Context, query url.Values) ([]Agreement, error) {
	var out []Agreement
	_, err := c.do(ctx, http.MethodGet, buildURL("/v1/domains/agreements", query), nil, &out, requestOptions{
		PathTemplate: "/v1/domains/agreements",
		ShopperID:    c.config.ShopperID,
		MarketID:     c.config.MarketID,
	})
	return out, err
}

func (c *Client) PatchDomain(ctx context.Context, domain string, body domainUpdateRequest) error {
	_, err := c.do(ctx, http.MethodPatch, fmt.Sprintf("/v1/domains/%s", domain), body, nil, requestOptions{
		PathTemplate: "/v1/domains/{domain}",
		ShopperID:    c.config.ShopperID,
	})
	return err
}
