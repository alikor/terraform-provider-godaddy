package provider

import (
	"testing"

	"github.com/alikor/terraform-provider-godaddy/internal/client"
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
