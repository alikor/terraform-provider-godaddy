package provider

import (
	"context"
	"errors"
	"net/http"
	"slices"

	"github.com/alikor/terraform-provider-godaddy/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	resourceschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = (*domainSettingsResource)(nil)
var _ resource.ResourceWithConfigure = (*domainSettingsResource)(nil)
var _ resource.ResourceWithImportState = (*domainSettingsResource)(nil)

type domainSettingsResource struct {
	client *client.Client
}

func NewDomainSettingsResource() resource.Resource {
	return &domainSettingsResource{}
}

func (r *domainSettingsResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "godaddy_domain_settings"
}

func (r *domainSettingsResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = resourceschema.Schema{
		MarkdownDescription: "Manage selected mutable settings on an existing GoDaddy domain.",
		Attributes: map[string]resourceschema.Attribute{
			"id": resourceschema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Canonical resource identifier equal to the domain name.",
				PlanModifiers:       []planmodifier.String{useStateForUnknownString()},
			},
			"domain": resourceschema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Existing GoDaddy domain to manage.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"locked": resourceschema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Domain lock flag.",
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"renew_auto": resourceschema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Auto-renew flag.",
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"expose_registrant_organization": resourceschema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "WHOIS organization exposure flag.",
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"expose_whois": resourceschema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "WHOIS exposure flag.",
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"status": resourceschema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Current domain status.",
				PlanModifiers:       []planmodifier.String{useStateForUnknownString()},
			},
			"created_at": resourceschema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Creation timestamp.",
				PlanModifiers:       []planmodifier.String{useStateForUnknownString()},
			},
			"expires_at": resourceschema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Expiration timestamp.",
				PlanModifiers:       []planmodifier.String{useStateForUnknownString()},
			},
			"renew_deadline": resourceschema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Renewal deadline timestamp.",
				PlanModifiers:       []planmodifier.String{useStateForUnknownString()},
			},
			"privacy": resourceschema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Privacy flag.",
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"transfer_protected": resourceschema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Transfer protection flag.",
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"expiration_protected": resourceschema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Expiration protection flag.",
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"hold_registrar": resourceschema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Registrar hold flag.",
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"name_servers": resourceschema.ListAttribute{
				Computed:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "Current nameserver set.",
				PlanModifiers:       []planmodifier.List{listplanmodifier.UseStateForUnknown()},
			},
		},
		Blocks: map[string]resourceschema.Block{
			"consent": resourceschema.SingleNestedBlock{
				MarkdownDescription: "Consent payload used when enabling WHOIS exposure fields.",
				Attributes: map[string]resourceschema.Attribute{
					"agreed_by":      resourceschema.StringAttribute{Required: true, MarkdownDescription: "IP address of the consenting actor."},
					"agreed_at":      resourceschema.StringAttribute{Required: true, MarkdownDescription: "RFC3339 timestamp of consent."},
					"agreement_keys": resourceschema.ListAttribute{Required: true, ElementType: types.StringType, MarkdownDescription: "Agreement keys accepted by the actor."},
				},
			},
		},
	}
}

func (r *domainSettingsResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, diags := configuredClient(req.ProviderData)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	r.client = client
}

func (r *domainSettingsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data settingsResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	domain, current, ok := r.readCurrentDomain(ctx, data.Domain.ValueString(), &resp.Diagnostics)
	if !ok {
		return
	}

	patch, ok := buildSettingsPatch(ctx, current, data, &resp.Diagnostics)
	if !ok {
		return
	}
	if hasSettingsPatchChanges(patch) {
		if err := r.client.PatchDomain(ctx, domain, patch); err != nil {
			resp.Diagnostics.AddError("Unable to update domain settings", err.Error())
			return
		}
	}

	r.refreshState(ctx, domain, data.Consent, &data, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *domainSettingsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data settingsResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	domain, current, ok := r.readCurrentDomain(ctx, data.Domain.ValueString(), &resp.Diagnostics)
	if !ok {
		resp.State.RemoveResource(ctx)
		return
	}

	r.setStateFromDomain(&data, domain, current)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *domainSettingsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data settingsResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	domain, current, ok := r.readCurrentDomain(ctx, data.Domain.ValueString(), &resp.Diagnostics)
	if !ok {
		return
	}

	patch, ok := buildSettingsPatch(ctx, current, data, &resp.Diagnostics)
	if !ok {
		return
	}
	if hasSettingsPatchChanges(patch) {
		if err := r.client.PatchDomain(ctx, domain, patch); err != nil {
			resp.Diagnostics.AddError("Unable to update domain settings", err.Error())
			return
		}
	}

	r.refreshState(ctx, domain, data.Consent, &data, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *domainSettingsResource) Delete(ctx context.Context, _ resource.DeleteRequest, resp *resource.DeleteResponse) {
	resp.Diagnostics.AddWarning("State-only delete", "Terraform management for this domain settings resource has been removed, but the remote domain settings were left unchanged.")
}

func (r *domainSettingsResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("domain"), req.ID)...)
}

func (r *domainSettingsResource) readCurrentDomain(ctx context.Context, rawDomain string, diags *diag.Diagnostics) (string, *client.Domain, bool) {
	domain, err := parseDomain(rawDomain)
	if err != nil {
		diags.AddError("Invalid domain", err.Error())
		return "", nil, false
	}

	current, err := r.client.GetDomain(ctx, domain)
	if err != nil {
		var apiErr *client.APIError
		if errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusNotFound {
			diags.AddError("Domain not found", "The domain must already exist before it can be managed.")
			return "", nil, false
		}
		diags.AddError("Unable to read domain", err.Error())
		return "", nil, false
	}

	return domain, current, true
}

func (r *domainSettingsResource) refreshState(ctx context.Context, domain string, consent types.Object, data *settingsResourceModel, diags *diag.Diagnostics) {
	current, err := r.client.GetDomain(ctx, domain)
	if err != nil {
		diags.AddError("Unable to refresh domain settings", err.Error())
		return
	}

	r.setStateFromDomain(data, domain, current)
	data.Consent = consent
}

func (r *domainSettingsResource) setStateFromDomain(data *settingsResourceModel, domain string, current *client.Domain) {
	data.ID = types.StringValue(domain)
	data.Domain = types.StringValue(domain)
	data.Locked = types.BoolValue(current.Locked)
	data.RenewAuto = types.BoolValue(current.RenewAuto)
	data.ExposeRegistrantOrganization = types.BoolValue(current.ExposeRegistrantOrganization)
	data.ExposeWhois = types.BoolValue(current.ExposeWhois)
	data.Status = stringOrNull(current.Status)
	data.CreatedAt = stringOrNull(current.CreatedAt)
	data.ExpiresAt = stringOrNull(current.ExpiresAt)
	data.RenewDeadline = stringOrNull(current.RenewDeadline)
	data.Privacy = types.BoolValue(current.Privacy)
	data.TransferProtected = types.BoolValue(current.TransferProtected)
	data.ExpirationProtected = types.BoolValue(current.ExpirationProtected)
	data.HoldRegistrar = types.BoolValue(current.HoldRegistrar)
	data.NameServers = toStringList(current.NameServers)
}

func buildSettingsPatch(ctx context.Context, current *client.Domain, data settingsResourceModel, diags *diag.Diagnostics) (client.DomainUpdateRequest, bool) {
	patch := client.DomainUpdateRequest{}

	setBool := func(target *bool, currentValue bool, planned types.Bool) *bool {
		if planned.IsNull() || planned.IsUnknown() {
			return target
		}
		value := planned.ValueBool()
		if value == currentValue {
			return target
		}
		return &value
	}

	patch.Locked = setBool(patch.Locked, current.Locked, data.Locked)
	patch.RenewAuto = setBool(patch.RenewAuto, current.RenewAuto, data.RenewAuto)
	patch.ExposeRegistrantOrganization = setBool(patch.ExposeRegistrantOrganization, current.ExposeRegistrantOrganization, data.ExposeRegistrantOrganization)
	patch.ExposeWhois = setBool(patch.ExposeWhois, current.ExposeWhois, data.ExposeWhois)

	if (patch.ExposeRegistrantOrganization != nil && *patch.ExposeRegistrantOrganization) || (patch.ExposeWhois != nil && *patch.ExposeWhois) {
		consent, err := consentFromObject(ctx, data.Consent)
		if err != nil || consent == nil {
			diags.AddError("Consent required", "Enabling WHOIS exposure fields requires the `consent` block.")
			return client.DomainUpdateRequest{}, false
		}

		if patch.ExposeRegistrantOrganization != nil && *patch.ExposeRegistrantOrganization && !slices.Contains(consent.AgreementKeys, "EXPOSE_REGISTRANT_ORGANIZATION") {
			diags.AddError("Missing consent key", "`EXPOSE_REGISTRANT_ORGANIZATION` must be present in `consent.agreement_keys` when enabling `expose_registrant_organization`.")
			return client.DomainUpdateRequest{}, false
		}
		if patch.ExposeWhois != nil && *patch.ExposeWhois && !slices.Contains(consent.AgreementKeys, "EXPOSE_WHOIS") {
			diags.AddError("Missing consent key", "`EXPOSE_WHOIS` must be present in `consent.agreement_keys` when enabling `expose_whois`.")
			return client.DomainUpdateRequest{}, false
		}

		patch.Consent = consent
	}

	return patch, true
}

func hasSettingsPatchChanges(patch client.DomainUpdateRequest) bool {
	return patch.Locked != nil ||
		patch.RenewAuto != nil ||
		patch.ExposeRegistrantOrganization != nil ||
		patch.ExposeWhois != nil ||
		len(patch.NameServers) > 0
}
