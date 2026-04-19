package provider

import (
	"context"
	"errors"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/alikor/terraform-provider-godaddy/internal/client"
	"github.com/alikor/terraform-provider-godaddy/internal/normalize"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	resourceschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = (*domainNameserversResource)(nil)
var _ resource.ResourceWithConfigure = (*domainNameserversResource)(nil)
var _ resource.ResourceWithImportState = (*domainNameserversResource)(nil)
var _ resource.ResourceWithModifyPlan = (*domainNameserversResource)(nil)

type domainNameserversResource struct {
	client *client.Client
}

func NewDomainNameserversResource() resource.Resource {
	return &domainNameserversResource{}
}

func (r *domainNameserversResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "godaddy_domain_nameservers"
}

func (r *domainNameserversResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = resourceschema.Schema{
		MarkdownDescription: "Manage the authoritative nameserver set for an existing GoDaddy domain.",
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
			"name_servers": resourceschema.SetAttribute{
				Required:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "Full authoritative nameserver set. Minimum 2 nameservers.",
			},
			"status": resourceschema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Current domain status.",
				PlanModifiers:       []planmodifier.String{useStateForUnknownString()},
			},
			"updated_via_v2": resourceschema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether the most recent managed update used the v2 async nameserver path.",
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
		},
	}
}

func (r *domainNameserversResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *domainNameserversResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() {
		return
	}

	var data nameserversResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	validateNameserverPlan(ctx, data.NameServers, &resp.Diagnostics)
}

func (r *domainNameserversResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	r.apply(ctx, req.Plan, &resp.State, &resp.Diagnostics)
}

func (r *domainNameserversResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data nameserversResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	domain, current, ok := r.readCurrentDomain(ctx, data.Domain.ValueString(), &resp.Diagnostics)
	if !ok {
		resp.State.RemoveResource(ctx)
		return
	}

	// GoDaddy returns old nameservers while the domain is PENDING_DNS (DNS
	// propagation is async). Preserve the known nameservers from state so that
	// a plan immediately after apply does not show spurious drift.
	if current.Status == "PENDING_DNS" {
		if existing, err := stringsFromSet(ctx, data.NameServers); err == nil && len(existing) >= 2 {
			current.NameServers = existing
		}
	}

	r.setStateFromDomain(&data, domain, current)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *domainNameserversResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	r.apply(ctx, req.Plan, &resp.State, &resp.Diagnostics)
}

func (r *domainNameserversResource) Delete(_ context.Context, _ resource.DeleteRequest, resp *resource.DeleteResponse) {
	resp.Diagnostics.AddWarning("State-only delete", "Terraform management for this nameservers resource has been removed, but the remote nameserver set was left unchanged.")
}

func (r *domainNameserversResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("domain"), req.ID)...)
}

func (r *domainNameserversResource) apply(ctx context.Context, getter interface {
	Get(context.Context, any) diag.Diagnostics
}, state interface {
	Set(context.Context, any) diag.Diagnostics
}, diags *diag.Diagnostics) {
	var data nameserversResourceModel

	diags.Append(getter.Get(ctx, &data)...)
	if diags.HasError() {
		return
	}
	updatedViaV2 := existingUpdatedViaV2(data)

	domain, current, ok := r.readCurrentDomain(ctx, data.Domain.ValueString(), diags)
	if !ok {
		return
	}

	desired, err := stringsFromSet(ctx, data.NameServers)
	if err != nil {
		diags.AddError("Invalid name_servers", err.Error())
		return
	}
	desired = normalize.NormalizeNameservers(desired)
	if !validateNormalizedNameservers(desired, diags) {
		return
	}

	currentNS := normalize.NormalizeNameservers(current.NameServers)
	if !slices.Equal(currentNS, desired) {
		usedV2, err := r.updateNameservers(ctx, domain, desired)
		if err != nil {
			summary, detail := describeNameserverUpdateError(err)
			diags.AddError(summary, detail)
			return
		}
		updatedViaV2 = usedV2

		current, err = r.client.GetDomain(ctx, domain)
		if err != nil {
			diags.AddError("Unable to refresh nameservers", err.Error())
			return
		}
		// GoDaddy's API is eventually consistent: the domain may stay ACTIVE
		// (not just PENDING_DNS) while still returning the old nameservers.
		// Since updateNameservers succeeded, always store the desired set so
		// Terraform does not see a spurious inconsistency immediately after apply.
		current.NameServers = desired
	}

	r.setStateFromDomain(&data, domain, current)
	data.UpdatedViaV2 = types.BoolValue(updatedViaV2)
	diags.Append(state.Set(ctx, &data)...)
}

func (r *domainNameserversResource) updateNameservers(ctx context.Context, domain string, desired []string) (bool, error) {
	customerID, useV2, err := r.resolveOptionalCustomerID(ctx)
	if err != nil {
		return false, err
	}
	if useV2 {
		if err := r.client.UpdateDomainNameServersV2(ctx, customerID, domain, desired); err != nil {
			return false, err
		}
		_, err := r.client.PollDomainAction(ctx, customerID, domain, "DOMAIN_UPDATE_NAME_SERVERS", "", 10*time.Minute)
		return true, err
	}

	return false, r.client.PatchDomain(ctx, domain, client.DomainUpdateRequest{NameServers: desired})
}

func (r *domainNameserversResource) resolveOptionalCustomerID(ctx context.Context) (string, bool, error) {
	cfg := r.client.Config()
	if strings.TrimSpace(cfg.CustomerID) == "" && strings.TrimSpace(cfg.ShopperID) == "" {
		return "", false, nil
	}

	customerID, err := r.client.ResolveCustomerID(ctx)
	if err != nil {
		return "", false, err
	}
	return customerID, true, nil
}

func (r *domainNameserversResource) readCurrentDomain(ctx context.Context, rawDomain string, diags *diag.Diagnostics) (string, *client.Domain, bool) {
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

func (r *domainNameserversResource) setStateFromDomain(data *nameserversResourceModel, domain string, current *client.Domain) {
	updatedViaV2 := existingUpdatedViaV2(*data)
	data.ID = types.StringValue(domain)
	data.Domain = types.StringValue(domain)
	data.NameServers = toStringSet(normalize.NormalizeNameservers(current.NameServers))
	data.Status = stringOrNull(current.Status)
	data.UpdatedViaV2 = types.BoolValue(updatedViaV2)
}

func existingUpdatedViaV2(data nameserversResourceModel) bool {
	if !data.UpdatedViaV2.IsNull() && !data.UpdatedViaV2.IsUnknown() {
		return data.UpdatedViaV2.ValueBool()
	}
	return false
}

func describeNameserverUpdateError(err error) (string, string) {
	var apiErr *client.APIError
	if errors.As(err, &apiErr) {
		lower := strings.ToLower(apiErr.Message)
		if strings.Contains(lower, "2fa") || strings.Contains(lower, "two-factor") || strings.Contains(lower, "two factor") {
			return "Nameserver update requires additional verification", "GoDaddy rejected the nameserver update because the domain requires two-factor verification or another interactive protection step that the API cannot complete. Original error: " + err.Error()
		}
		if apiErr.StatusCode == http.StatusForbidden || apiErr.StatusCode == http.StatusUnprocessableEntity || apiErr.StatusCode == http.StatusConflict {
			return "Nameserver update not allowed", "GoDaddy rejected the nameserver update. Protected or high-value domains can require account eligibility checks or interactive verification before nameserver changes are allowed. Original error: " + err.Error()
		}
	}

	return "Unable to update nameservers", err.Error()
}

func validateNameserverPlan(ctx context.Context, planned types.Set, diags *diag.Diagnostics) bool {
	desired, err := stringsFromSet(ctx, planned)
	if err != nil {
		diags.AddError("Invalid name_servers", err.Error())
		return false
	}

	return validateNormalizedNameservers(normalize.NormalizeNameservers(desired), diags)
}

func validateNormalizedNameservers(desired []string, diags *diag.Diagnostics) bool {
	if len(desired) < 2 {
		diags.AddError("Invalid name_servers", "At least two nameservers are required.")
		return false
	}

	return true
}
