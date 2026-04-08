package client

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

func (c *Client) GetShopper(ctx context.Context, shopperID string, includeCustomerID bool) (*Shopper, error) {
	query := url.Values{}
	if includeCustomerID {
		query.Set("includes", "customerId")
	}

	var out Shopper
	_, err := c.do(ctx, http.MethodGet, buildURL(fmt.Sprintf("/v1/shoppers/%s", shopperID), query), nil, &out, requestOptions{
		PathTemplate: fmt.Sprintf("/v1/shoppers/%s", "{shopperId}"),
	})
	if err != nil {
		return nil, err
	}
	return &out, nil
}
