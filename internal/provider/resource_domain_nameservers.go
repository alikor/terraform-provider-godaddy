package provider

import (
	"context"
	"errors"
	"net/http"
	"slices"

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
			"name_servers": resourceschema.ListAttribute{
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
				MarkdownDescription: "Whether the update used the v2 async path. This implementation currently uses the v1 path.",
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

	domain, current, ok := r.readCurrentDomain(ctx, data.Domain.ValueString(), diags)
	if !ok {
		return
	}

	desired, err := stringsFromList(ctx, data.NameServers)
	if err != nil {
		diags.AddError("Invalid name_servers", err.Error())
		return
	}
	desired = normalize.NormalizeNameservers(desired)
	if len(desired) < 2 {
		diags.AddError("Invalid name_servers", "At least two nameservers are required.")
		return
	}

	currentNS := normalize.NormalizeNameservers(current.NameServers)
	if !slices.Equal(currentNS, desired) {
		if err := r.client.PatchDomain(ctx, domain, client.DomainUpdateRequest{NameServers: desired}); err != nil {
			diags.AddError("Unable to update nameservers", err.Error())
			return
		}

		current, err = r.client.GetDomain(ctx, domain)
		if err != nil {
			diags.AddError("Unable to refresh nameservers", err.Error())
			return
		}
	}

	r.setStateFromDomain(&data, domain, current)
	diags.Append(state.Set(ctx, &data)...)
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
	data.ID = types.StringValue(domain)
	data.Domain = types.StringValue(domain)
	data.NameServers = toStringList(normalize.NormalizeNameservers(current.NameServers))
	data.Status = stringOrNull(current.Status)
	data.UpdatedViaV2 = types.BoolValue(false)
}
