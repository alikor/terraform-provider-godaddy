package provider

import (
	"context"
	"testing"

	"github.com/alikor/terraform-provider-godaddy/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestBuildSettingsPatchNoChanges(t *testing.T) {
	t.Parallel()

	current := &client.Domain{
		Locked:                       true,
		RenewAuto:                    true,
		ExposeRegistrantOrganization: false,
		ExposeWhois:                  false,
	}

	patch, ok := buildSettingsPatch(context.Background(), current, settingsResourceModel{
		Locked:                       types.BoolValue(true),
		RenewAuto:                    types.BoolValue(true),
		ExposeRegistrantOrganization: types.BoolValue(false),
		ExposeWhois:                  types.BoolValue(false),
	}, &diag.Diagnostics{})
	if !ok {
		t.Fatal("expected patch build to succeed")
	}
	if hasSettingsPatchChanges(patch) {
		t.Fatalf("expected no-op patch, got %#v", patch)
	}
}

func TestBuildSettingsPatchRequiresConsent(t *testing.T) {
	t.Parallel()

	current := &client.Domain{ExposeWhois: false}
	diags := diag.Diagnostics{}

	_, ok := buildSettingsPatch(context.Background(), current, settingsResourceModel{
		ExposeWhois: types.BoolValue(true),
		Consent:     types.ObjectNull(consentAttrTypes),
	}, &diags)
	if ok {
		t.Fatal("expected patch build to fail without consent")
	}
	if !diags.HasError() {
		t.Fatal("expected diagnostics error")
	}
}

func TestBuildSettingsPatchIncludesConsent(t *testing.T) {
	t.Parallel()

	current := &client.Domain{ExposeWhois: false}
	diags := diag.Diagnostics{}
	consent := types.ObjectValueMust(consentAttrTypes, map[string]attr.Value{
		"agreed_by":      types.StringValue("203.0.113.50"),
		"agreed_at":      types.StringValue("2026-04-08T10:30:00Z"),
		"agreement_keys": types.ListValueMust(types.StringType, []attr.Value{types.StringValue("EXPOSE_WHOIS")}),
	})

	patch, ok := buildSettingsPatch(context.Background(), current, settingsResourceModel{
		ExposeWhois: types.BoolValue(true),
		Consent:     consent,
	}, &diags)
	if !ok {
		t.Fatalf("expected patch build to succeed: %#v", diags)
	}
	if patch.Consent == nil || len(patch.Consent.AgreementKeys) != 1 || patch.Consent.AgreementKeys[0] != "EXPOSE_WHOIS" {
		t.Fatalf("expected consent to be carried into patch, got %#v", patch.Consent)
	}
}
