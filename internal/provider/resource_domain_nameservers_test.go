package provider

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/alikor/terraform-provider-godaddy/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	resourceschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestDomainNameserversUpdateUsesV2WhenCustomerIDAvailable(t *testing.T) {
	t.Parallel()

	var mu sync.Mutex
	seenPut := false
	seenPoll := false

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()

		switch {
		case r.Method == http.MethodPut && r.URL.Path == "/v2/customers/customer-123/domains/example.com/nameServers":
			seenPut = true

			var payload client.DomainNameServerUpdateV2
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Fatalf("unable to decode payload: %v", err)
			}
			if len(payload.NameServers) != 2 || payload.NameServers[0] != "ns1.example.net" || payload.NameServers[1] != "ns2.example.net" {
				t.Fatalf("NameServers = %#v", payload.NameServers)
			}
			w.WriteHeader(http.StatusAccepted)
		case r.Method == http.MethodGet && r.URL.Path == "/v2/customers/customer-123/domains/example.com/actions/DOMAIN_UPDATE_NAME_SERVERS":
			seenPoll = true
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(client.DomainAction{
				Type:   "DOMAIN_UPDATE_NAME_SERVERS",
				Status: "SUCCESS",
			})
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	r := &domainNameserversResource{
		client: client.New(client.Config{
			APIKey:         "key",
			APISecret:      "secret",
			BaseURL:        server.URL,
			CustomerID:     "customer-123",
			RequestTimeout: time.Second,
			PollInterval:   10 * time.Millisecond,
			MaxRetries:     0,
			RateLimitRPM:   60,
		}),
	}

	usedV2, err := r.updateNameservers(context.Background(), "example.com", []string{"ns1.example.net", "ns2.example.net"})
	if err != nil {
		t.Fatalf("updateNameservers() returned error: %v", err)
	}
	if !usedV2 {
		t.Fatalf("usedV2 = false, want true")
	}
	if !seenPut || !seenPoll {
		t.Fatalf("expected PUT and poll requests, got seenPut=%t seenPoll=%t", seenPut, seenPoll)
	}
}

func TestDomainNameserversUpdateFallsBackToV1WithoutCustomerContext(t *testing.T) {
	t.Parallel()

	seenPatch := false

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch || r.URL.Path != "/v1/domains/example.com" {
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		seenPatch = true

		var payload client.DomainUpdateRequest
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("unable to decode payload: %v", err)
		}
		if len(payload.NameServers) != 2 || payload.NameServers[0] != "ns1.example.net" || payload.NameServers[1] != "ns2.example.net" {
			t.Fatalf("NameServers = %#v", payload.NameServers)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	r := &domainNameserversResource{
		client: client.New(client.Config{
			APIKey:         "key",
			APISecret:      "secret",
			BaseURL:        server.URL,
			RequestTimeout: time.Second,
			PollInterval:   10 * time.Millisecond,
			MaxRetries:     0,
			RateLimitRPM:   60,
		}),
	}

	usedV2, err := r.updateNameservers(context.Background(), "example.com", []string{"ns1.example.net", "ns2.example.net"})
	if err != nil {
		t.Fatalf("updateNameservers() returned error: %v", err)
	}
	if usedV2 {
		t.Fatalf("usedV2 = true, want false")
	}
	if !seenPatch {
		t.Fatal("expected v1 PATCH request")
	}
}

func TestDomainNameserversSetStatePreservesUpdatedViaV2(t *testing.T) {
	t.Parallel()

	resource := &domainNameserversResource{}
	data := nameserversResourceModel{
		UpdatedViaV2: types.BoolValue(true),
	}

	resource.setStateFromDomain(&data, "example.com", &client.Domain{
		Status:      "ACTIVE",
		NameServers: []string{"ns1.example.net", "ns2.example.net"},
	})

	if data.ID.ValueString() != "example.com" {
		t.Fatalf("id = %q, want example.com", data.ID.ValueString())
	}
	if !data.UpdatedViaV2.ValueBool() {
		t.Fatalf("UpdatedViaV2 = false, want true")
	}
}

func TestDescribeNameserverUpdateErrorTwoFactor(t *testing.T) {
	t.Parallel()

	summary, detail := describeNameserverUpdateError(&client.APIError{
		StatusCode: http.StatusForbidden,
		Message:    "Domain requires 2FA confirmation before changing nameservers",
	})

	if summary != "Nameserver update requires additional verification" {
		t.Fatalf("summary = %q", summary)
	}
	if detail == "" {
		t.Fatal("expected non-empty detail")
	}
}

func TestDomainNameserversModifyPlanRequiresMinimumTwoNameservers(t *testing.T) {
	t.Parallel()

	resp := runDomainNameserversModifyPlan(t, nameserversResourceModel{
		Domain:      types.StringValue("example.com"),
		NameServers: types.SetValueMust(types.StringType, []attr.Value{types.StringValue("ns1.example.net")}),
	})

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected diagnostics error")
	}
	assertDiagContains(t, resp.Diagnostics, "At least two nameservers are required")
}

func TestDomainNameserversModifyPlanAllowsNormalizedPair(t *testing.T) {
	t.Parallel()

	resp := runDomainNameserversModifyPlan(t, nameserversResourceModel{
		Domain: types.StringValue("example.com"),
		NameServers: types.SetValueMust(types.StringType, []attr.Value{
			types.StringValue("NS2.EXAMPLE.NET."),
			types.StringValue("ns1.example.net"),
			types.StringValue("ns1.example.net"),
		}),
	})

	if resp.Diagnostics.HasError() {
		t.Fatalf("expected no diagnostics, got %#v", resp.Diagnostics)
	}
}

func runDomainNameserversModifyPlan(t *testing.T, planModel nameserversResourceModel) resource.ModifyPlanResponse {
	t.Helper()

	schema := testDomainNameserversSchema(t)
	ctx := context.Background()

	if planModel.NameServers.ElementType(context.Background()) == nil {
		planModel.NameServers = types.SetNull(types.StringType)
	}

	plan := tfsdk.Plan{Schema: schema}
	if diags := plan.Set(ctx, planModel); diags.HasError() {
		t.Fatalf("unable to encode plan: %#v", diags)
	}

	req := resource.ModifyPlanRequest{
		Plan: plan,
		State: tfsdk.State{
			Schema: schema,
			Raw:    tftypes.NewValue(schema.Type().TerraformType(ctx), nil),
		},
	}
	resp := resource.ModifyPlanResponse{Plan: plan}

	r := &domainNameserversResource{}
	r.ModifyPlan(ctx, req, &resp)
	return resp
}

func testDomainNameserversSchema(t *testing.T) resourceschema.Schema {
	t.Helper()

	var resp resource.SchemaResponse
	r := &domainNameserversResource{}
	r.Schema(context.Background(), resource.SchemaRequest{}, &resp)
	return resp.Schema
}
