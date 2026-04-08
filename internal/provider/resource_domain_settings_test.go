package provider

import (
	"context"
	"strings"
	"testing"

	"github.com/alikor/terraform-provider-godaddy/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
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

func TestDomainSettingsModifyPlanRequiresConsentOnCreate(t *testing.T) {
	t.Parallel()

	resp := runDomainSettingsModifyPlan(t, settingsResourceModel{
		Domain:      types.StringValue("example.com"),
		ExposeWhois: types.BoolValue(true),
		Consent:     types.ObjectNull(consentAttrTypes),
	}, nil)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected diagnostics error")
	}
	assertDiagContains(t, resp.Diagnostics, "Consent required")
}

func TestDomainSettingsModifyPlanRequiresConsentOnEnableUpdate(t *testing.T) {
	t.Parallel()

	state := settingsResourceModel{
		ID:          types.StringValue("example.com"),
		Domain:      types.StringValue("example.com"),
		ExposeWhois: types.BoolValue(false),
		Consent:     types.ObjectNull(consentAttrTypes),
	}
	plan := settingsResourceModel{
		ID:          types.StringValue("example.com"),
		Domain:      types.StringValue("example.com"),
		ExposeWhois: types.BoolValue(true),
		Consent:     types.ObjectNull(consentAttrTypes),
	}

	resp := runDomainSettingsModifyPlan(t, plan, &state)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected diagnostics error")
	}
	assertDiagContains(t, resp.Diagnostics, "Consent required")
}

func TestDomainSettingsModifyPlanAllowsExistingExposureWithoutConsent(t *testing.T) {
	t.Parallel()

	state := settingsResourceModel{
		ID:          types.StringValue("example.com"),
		Domain:      types.StringValue("example.com"),
		ExposeWhois: types.BoolValue(true),
		Consent:     types.ObjectNull(consentAttrTypes),
	}
	plan := settingsResourceModel{
		ID:          types.StringValue("example.com"),
		Domain:      types.StringValue("example.com"),
		ExposeWhois: types.BoolValue(true),
		Consent:     types.ObjectNull(consentAttrTypes),
	}

	resp := runDomainSettingsModifyPlan(t, plan, &state)

	if resp.Diagnostics.HasError() {
		t.Fatalf("expected no diagnostics, got %#v", resp.Diagnostics)
	}
}

func TestDomainSettingsModifyPlanRequiresConsentKey(t *testing.T) {
	t.Parallel()

	consent := types.ObjectValueMust(consentAttrTypes, map[string]attr.Value{
		"agreed_by":      types.StringValue("203.0.113.50"),
		"agreed_at":      types.StringValue("2026-04-08T10:30:00Z"),
		"agreement_keys": types.ListValueMust(types.StringType, []attr.Value{types.StringValue("EXPOSE_WHOIS")}),
	})

	resp := runDomainSettingsModifyPlan(t, settingsResourceModel{
		Domain:                       types.StringValue("example.com"),
		ExposeRegistrantOrganization: types.BoolValue(true),
		Consent:                      consent,
	}, nil)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected diagnostics error")
	}
	assertDiagContains(t, resp.Diagnostics, "EXPOSE_REGISTRANT_ORGANIZATION")
}

func runDomainSettingsModifyPlan(t *testing.T, planModel settingsResourceModel, stateModel *settingsResourceModel) resource.ModifyPlanResponse {
	t.Helper()

	schema := testDomainSettingsSchema(t)
	ctx := context.Background()
	planModel = normalizeSettingsTestModel(planModel)

	plan := tfsdk.Plan{Schema: schema}
	if diags := plan.Set(ctx, planModel); diags.HasError() {
		t.Fatalf("unable to encode plan: %#v", diags)
	}

	state := tfsdk.State{
		Schema: schema,
		Raw:    tftypes.NewValue(schema.Type().TerraformType(ctx), nil),
	}
	if stateModel != nil {
		normalizedState := normalizeSettingsTestModel(*stateModel)
		if diags := state.Set(ctx, normalizedState); diags.HasError() {
			t.Fatalf("unable to encode state: %#v", diags)
		}
	}

	req := resource.ModifyPlanRequest{
		Plan:  plan,
		State: state,
	}
	resp := resource.ModifyPlanResponse{
		Plan: plan,
	}

	r := &domainSettingsResource{}
	r.ModifyPlan(ctx, req, &resp)
	return resp
}

func normalizeSettingsTestModel(model settingsResourceModel) settingsResourceModel {
	if model.NameServers.ElementType(context.Background()) == nil {
		model.NameServers = types.ListNull(types.StringType)
	}
	if len(model.Consent.AttributeTypes(context.Background())) == 0 {
		model.Consent = types.ObjectNull(consentAttrTypes)
	}
	return model
}

func testDomainSettingsSchema(t *testing.T) schema.Schema {
	t.Helper()

	var resp resource.SchemaResponse
	r := &domainSettingsResource{}
	r.Schema(context.Background(), resource.SchemaRequest{}, &resp)
	return resp.Schema
}

func assertDiagContains(t *testing.T, diags diag.Diagnostics, want string) {
	t.Helper()

	if !diags.HasError() {
		t.Fatalf("expected diagnostics containing %q, got none", want)
	}

	for _, errDiag := range diags.Errors() {
		if strings.Contains(errDiag.Summary(), want) || strings.Contains(errDiag.Detail(), want) {
			return
		}
	}

	t.Fatalf("expected diagnostics containing %q, got %#v", want, diags)
}
