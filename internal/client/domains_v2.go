package client

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

func (c *Client) GetDomainV2(ctx context.Context, customerID, domain string, includes []string) (*Domain, bool, error) {
	query := url.Values{}
	if len(includes) > 0 {
		query.Set("includes", strings.Join(includes, ","))
	}

	var out Domain
	statusCode, err := c.do(ctx, http.MethodGet, buildURL(fmt.Sprintf("/v2/customers/%s/domains/%s", customerID, domain), query), nil, &out, requestOptions{
		PathTemplate: "/v2/customers/{customerId}/domains/{domain}",
	})
	if err != nil {
		return nil, false, err
	}

	return &out, statusCode == http.StatusNonAuthoritativeInfo, nil
}

func (c *Client) GetDomainForwarding(ctx context.Context, customerID, fqdn string, includeSubs bool) (*DomainForwarding, error) {
	query := url.Values{}
	if includeSubs {
		query.Set("includeSubs", "true")
	}

	var out DomainForwarding
	statusCode, err := c.do(ctx, http.MethodGet, buildURL(fmt.Sprintf("/v2/customers/%s/domains/forwards/%s", customerID, fqdn), query), nil, &out, requestOptions{
		PathTemplate:    "/v2/customers/{customerId}/domains/forwards/{fqdn}",
		AllowStatusCode: []int{http.StatusNotFound},
	})
	if err != nil {
		return nil, err
	}
	if statusCode == http.StatusNotFound {
		return nil, &APIError{StatusCode: http.StatusNotFound, Message: "forwarding not found"}
	}
	return &out, nil
}

func (c *Client) CreateDomainForwarding(ctx context.Context, customerID, fqdn string, body DomainForwarding) error {
	_, err := c.do(ctx, http.MethodPost, fmt.Sprintf("/v2/customers/%s/domains/forwards/%s", customerID, fqdn), body, nil, requestOptions{
		PathTemplate: "/v2/customers/{customerId}/domains/forwards/{fqdn}",
		RequestID:    true,
	})
	return err
}

func (c *Client) UpdateDomainForwarding(ctx context.Context, customerID, fqdn string, body DomainForwarding) error {
	_, err := c.do(ctx, http.MethodPut, fmt.Sprintf("/v2/customers/%s/domains/forwards/%s", customerID, fqdn), body, nil, requestOptions{
		PathTemplate: "/v2/customers/{customerId}/domains/forwards/{fqdn}",
		RequestID:    true,
	})
	return err
}

func (c *Client) DeleteDomainForwarding(ctx context.Context, customerID, fqdn string) error {
	statusCode, err := c.do(ctx, http.MethodDelete, fmt.Sprintf("/v2/customers/%s/domains/forwards/%s", customerID, fqdn), nil, nil, requestOptions{
		PathTemplate:    "/v2/customers/{customerId}/domains/forwards/{fqdn}",
		RequestID:       true,
		AllowStatusCode: []int{http.StatusNotFound},
	})
	if err != nil {
		return err
	}
	if statusCode == http.StatusNotFound {
		return nil
	}
	return nil
}

func (c *Client) AddDNSSECRecords(ctx context.Context, customerID, domain string, records []DNSSECRecord) error {
	_, err := c.do(ctx, http.MethodPatch, fmt.Sprintf("/v2/customers/%s/domains/%s/dnssecRecords", customerID, domain), records, nil, requestOptions{
		PathTemplate: "/v2/customers/{customerId}/domains/{domain}/dnssecRecords",
		RequestID:    true,
	})
	return err
}

func (c *Client) PatchDomainContactsV2(ctx context.Context, customerID, domain string, body DomainContactsV2Update) error {
	_, err := c.do(ctx, http.MethodPatch, fmt.Sprintf("/v2/customers/%s/domains/%s/contacts", customerID, domain), body, nil, requestOptions{
		PathTemplate: "/v2/customers/{customerId}/domains/{domain}/contacts",
		RequestID:    true,
	})
	return err
}

func (c *Client) DeleteDNSSECRecords(ctx context.Context, customerID, domain string, records []DNSSECRecord) error {
	_, err := c.do(ctx, http.MethodDelete, fmt.Sprintf("/v2/customers/%s/domains/%s/dnssecRecords", customerID, domain), records, nil, requestOptions{
		PathTemplate: "/v2/customers/{customerId}/domains/{domain}/dnssecRecords",
		RequestID:    true,
	})
	return err
}
