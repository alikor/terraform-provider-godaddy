package provider

import (
	"testing"

	"github.com/alikor/terraform-provider-godaddy/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestRequestedDomainIncludes(t *testing.T) {
	t.Parallel()

	got := requestedDomainIncludes(domainDataSourceModel{
		IncludeAuthCode:            types.BoolValue(true),
		IncludeDNSSECRecords:       types.BoolValue(true),
		IncludeRegistryStatusCodes: types.BoolValue(true),
	})

	want := []string{"authCode", "dnssecRecords", "registryStatusCodes"}
	if len(got) != len(want) {
		t.Fatalf("requestedDomainIncludes() len = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("requestedDomainIncludes()[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestApplyDomainResultOmitsAdvancedFieldsWhenNotRequested(t *testing.T) {
	t.Parallel()

	data := domainDataSourceModel{}
	applyDomainResult(&data, "example.com", &client.Domain{
		DomainID:            42,
		AuthCode:            "secret",
		Actions:             []client.DomainAction{{Type: "NAMESERVERS_CHANGE"}},
		DNSSECRecords:       []client.DNSSECRecord{{KeyTag: 123}},
		RegistryStatusCodes: []string{"clientTransferProhibited"},
		NameServers:         []string{"ns1.example.com", "ns2.example.com"},
		Contacts:            &client.DomainContacts{},
		ExposeWhois:         true,
		TransferProtected:   true,
		ExpirationProtected: true,
	}, false)

	if !data.AuthCode.IsNull() {
		t.Fatalf("AuthCode should be null when not requested")
	}
	if !data.Actions.IsNull() {
		t.Fatalf("Actions should be null when not requested")
	}
	if !data.DNSSECRecords.IsNull() {
		t.Fatalf("DNSSECRecords should be null when not requested")
	}
	if !data.RegistryStatusCodes.IsNull() {
		t.Fatalf("RegistryStatusCodes should be null when not requested")
	}
	if data.Contacts.IsNull() {
		t.Fatalf("Contacts should be populated from the base domain payload")
	}
	if data.Partial.ValueBool() {
		t.Fatalf("Partial should be false")
	}
}

func TestApplyDomainResultIncludesRequestedAdvancedFields(t *testing.T) {
	t.Parallel()

	data := domainDataSourceModel{
		IncludeAuthCode:            types.BoolValue(true),
		IncludeActions:             types.BoolValue(true),
		IncludeDNSSECRecords:       types.BoolValue(true),
		IncludeRegistryStatusCodes: types.BoolValue(true),
	}
	applyDomainResult(&data, "example.com", &client.Domain{
		DomainID:            42,
		AuthCode:            "secret",
		Actions:             []client.DomainAction{{Type: "DNSSEC_UPDATE"}},
		DNSSECRecords:       []client.DNSSECRecord{{KeyTag: 123, Algorithm: "RSASHA256"}},
		RegistryStatusCodes: []string{"clientTransferProhibited"},
	}, true)

	if data.AuthCode.IsNull() || data.AuthCode.ValueString() != "secret" {
		t.Fatalf("AuthCode = %#v, want secret", data.AuthCode)
	}
	if data.Actions.IsNull() {
		t.Fatalf("Actions should be populated when requested")
	}
	if data.DNSSECRecords.IsNull() {
		t.Fatalf("DNSSECRecords should be populated when requested")
	}
	if data.RegistryStatusCodes.IsNull() {
		t.Fatalf("RegistryStatusCodes should be populated when requested")
	}
	if !data.Partial.ValueBool() {
		t.Fatalf("Partial should be true")
	}
}

func TestPartialDomainWarning(t *testing.T) {
	t.Parallel()

	got := partialDomainWarning([]string{"authCode", "actions"})
	want := "GoDaddy returned a partial v2 domain response. Optional sections may be unavailable for these requested includes: authCode, actions."
	if got != want {
		t.Fatalf("partialDomainWarning() = %q, want %q", got, want)
	}
}
