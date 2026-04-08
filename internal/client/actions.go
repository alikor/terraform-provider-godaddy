package client

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"
)

var ErrTimeout = errors.New("timed out waiting for GoDaddy action")

type ErrActionFailed struct {
	Action *DomainAction
}

func (e ErrActionFailed) Error() string {
	return fmt.Sprintf("godaddy action %s failed", e.Action.Type)
}

type ErrActionAwaitingInput struct {
	Action *DomainAction
}

func (e ErrActionAwaitingInput) Error() string {
	return fmt.Sprintf("godaddy action %s is awaiting input", e.Action.Type)
}

func (c *Client) GetDomainAction(ctx context.Context, customerID, domain, actionType string) (*DomainAction, error) {
	var out DomainAction
	statusCode, err := c.do(ctx, http.MethodGet, fmt.Sprintf("/v2/customers/%s/domains/%s/actions/%s", customerID, domain, actionType), nil, &out, requestOptions{
		PathTemplate:    "/v2/customers/{customerId}/domains/{domain}/actions/{type}",
		AllowStatusCode: []int{http.StatusNotFound},
	})
	if err != nil {
		return nil, err
	}
	if statusCode == http.StatusNotFound {
		return nil, &APIError{StatusCode: http.StatusNotFound, Message: "action not found"}
	}
	return &out, nil
}

func (c *Client) ListDomainActions(ctx context.Context, customerID, domain string) ([]DomainAction, error) {
	var out []DomainAction
	_, err := c.do(ctx, http.MethodGet, fmt.Sprintf("/v2/customers/%s/domains/%s/actions", customerID, domain), nil, &out, requestOptions{
		PathTemplate: "/v2/customers/{customerId}/domains/{domain}/actions",
	})
	return out, err
}

func (c *Client) PollDomainAction(ctx context.Context, customerID, domain, actionType, requestID string, timeout time.Duration) (*DomainAction, error) {
	deadline := time.Now().Add(timeout)

	for {
		if time.Now().After(deadline) {
			return nil, ErrTimeout
		}

		action, err := c.GetDomainAction(ctx, customerID, domain, actionType)
		if err != nil {
			var apiErr *APIError
			if errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusNotFound {
				select {
				case <-ctx.Done():
					return nil, ctx.Err()
				case <-time.After(c.config.PollInterval):
					continue
				}
			}
			return nil, err
		}

		if requestID != "" && action.RequestID != "" && action.RequestID != requestID {
			actions, listErr := c.ListDomainActions(ctx, customerID, domain)
			if listErr == nil {
				for i := range actions {
					if actions[i].Type == actionType && actions[i].RequestID == requestID {
						action = &actions[i]
						break
					}
				}
			}
		}

		switch action.Status {
		case "SUCCESS":
			return action, nil
		case "FAILED", "CANCELLED":
			return action, ErrActionFailed{Action: action}
		case "AWAITING":
			return action, ErrActionAwaitingInput{Action: action}
		default:
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(c.config.PollInterval):
			}
		}
	}
}
