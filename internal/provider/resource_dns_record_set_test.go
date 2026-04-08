package provider

import (
	"context"
	"testing"

	"github.com/alikor/terraform-provider-godaddy/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestValidateRRset(t *testing.T) {
	t.Parallel()

	if err := validateRRset("CNAME", []client.DNSRecord{{Data: "example.net"}, {Data: "example.org"}}); err == nil {
		t.Fatal("expected CNAME validation error")
	}

	if err := validateRRset("MX", []client.DNSRecord{{Data: "mail.example.net"}}); err == nil {
		t.Fatal("expected MX validation error")
	}

	if err := validateRRset("SRV", []client.DNSRecord{{Data: "srv.example.net", Priority: 1, Weight: 1, Port: 443, Protocol: "tcp", Service: "_https"}}); err != nil {
		t.Fatalf("unexpected SRV validation error: %v", err)
	}

	if err := validateRRset("A", []client.DNSRecord{{Data: "203.0.113.1", Priority: 10}}); err == nil {
		t.Fatal("expected A validation error")
	}
}

func TestRecordsToListUsesNullForUnsetOptionalFields(t *testing.T) {
	t.Parallel()

	list := recordsToList([]client.DNSRecord{{
		Data: "codex-test",
		TTL:  600,
	}})

	type dnsRecordModel struct {
		Data     types.String `tfsdk:"data"`
		TTL      types.Int64  `tfsdk:"ttl"`
		Priority types.Int64  `tfsdk:"priority"`
		Weight   types.Int64  `tfsdk:"weight"`
		Port     types.Int64  `tfsdk:"port"`
		Protocol types.String `tfsdk:"protocol"`
		Service  types.String `tfsdk:"service"`
	}

	var models []dnsRecordModel
	diags := list.ElementsAs(context.Background(), &models, false)
	if diags.HasError() {
		t.Fatalf("unable to decode records: %#v", diags)
	}
	if len(models) != 1 {
		t.Fatalf("record count = %d, want 1", len(models))
	}
	if models[0].TTL.IsNull() || models[0].TTL.ValueInt64() != 600 {
		t.Fatalf("ttl = %#v, want 600", models[0].TTL)
	}
	if !models[0].Priority.IsNull() || !models[0].Weight.IsNull() || !models[0].Port.IsNull() {
		t.Fatalf("expected unset optional numeric fields to be null, got priority=%#v weight=%#v port=%#v", models[0].Priority, models[0].Weight, models[0].Port)
	}
}
