package client

import (
	"context"
	"fmt"
	"net/http"
)

func (c *Client) GetDNSRecordSet(ctx context.Context, domain, recordType, name string) ([]DNSRecord, error) {
	var out []DNSRecord
	statusCode, err := c.do(ctx, http.MethodGet, fmt.Sprintf("/v1/domains/%s/records/%s/%s", domain, recordType, name), nil, &out, requestOptions{
		PathTemplate:    "/v1/domains/{domain}/records/{type}/{name}",
		ShopperID:       c.config.ShopperID,
		AllowStatusCode: []int{http.StatusNotFound},
	})
	if err != nil {
		return nil, err
	}
	if statusCode == http.StatusNotFound {
		return nil, &APIError{StatusCode: http.StatusNotFound, Message: "record set not found"}
	}
	return out, nil
}

func (c *Client) PutDNSRecordSet(ctx context.Context, domain, recordType, name string, records []DNSRecord) error {
	_, err := c.do(ctx, http.MethodPut, fmt.Sprintf("/v1/domains/%s/records/%s/%s", domain, recordType, name), records, nil, requestOptions{
		PathTemplate: "/v1/domains/{domain}/records/{type}/{name}",
		ShopperID:    c.config.ShopperID,
	})
	return err
}

func (c *Client) DeleteDNSRecordSet(ctx context.Context, domain, recordType, name string) error {
	statusCode, err := c.do(ctx, http.MethodDelete, fmt.Sprintf("/v1/domains/%s/records/%s/%s", domain, recordType, name), nil, nil, requestOptions{
		PathTemplate:    "/v1/domains/{domain}/records/{type}/{name}",
		ShopperID:       c.config.ShopperID,
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
