package provider

import (
	"context"
	"errors"
	"net/http"

	"github.com/alikor/terraform-provider-godaddy/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	resourceschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = (*domainForwardingResource)(nil)
var _ resource.ResourceWithConfigure = (*domainForwardingResource)(nil)
var _ resource.ResourceWithImportState = (*domainForwardingResource)(nil)

type domainForwardingResource struct {
	client *client.Client
}

func NewDomainForwardingResource() resource.Resource {
	return &domainForwardingResource{}
}

func (r *domainForwardingResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "godaddy_domain_forwarding"
}

func (r *domainForwardingResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = resourceschema.Schema{
		MarkdownDescription: "Manage GoDaddy URL forwarding for one FQDN.",
		Attributes: map[string]resourceschema.Attribute{
			"id": resourceschema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Canonical resource identifier equal to the FQDN.",
				PlanModifiers:       []planmodifier.String{useStateForUnknownString()},
			},
			"fqdn": resourceschema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Forwarded FQDN.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"type": resourceschema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Forward type: MASKED, REDIRECT_PERMANENT, or REDIRECT_TEMPORARY.",
			},
			"url": resourceschema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Destination URL.",
			},
			"mask": resourceschema.SingleNestedAttribute{
				Optional:            true,
				MarkdownDescription: "Optional forwarding mask settings.",
				Attributes: map[string]resourceschema.Attribute{
					"title":       resourceschema.StringAttribute{Optional: true, MarkdownDescription: "Mask title."},
					"description": resourceschema.StringAttribute{Optional: true, MarkdownDescription: "Mask description."},
					"keywords":    resourceschema.StringAttribute{Optional: true, MarkdownDescription: "Mask keywords."},
				},
			},
		},
	}
}

func (r *domainForwardingResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *domainForwardingResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data forwardingResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	fqdn, forwarding, customerID, ok := r.expand(ctx, data, &resp.Diagnostics)
	if !ok {
		return
	}

	existing, err := r.client.GetDomainForwarding(ctx, customerID, fqdn, false)
	if err == nil && existing != nil {
		resp.Diagnostics.AddError("Domain forwarding already exists", "Import the existing forwarding instead of creating it.")
		return
	}
	var apiErr *client.APIError
	if err != nil && !(errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusNotFound) {
		resp.Diagnostics.AddError("Unable to inspect existing forwarding", err.Error())
		return
	}

	if err := r.client.CreateDomainForwarding(ctx, customerID, fqdn, forwarding); err != nil {
		resp.Diagnostics.AddError("Unable to create forwarding", err.Error())
		return
	}

	r.refresh(ctx, fqdn, customerID, &data, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *domainForwardingResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data forwardingResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	fqdn, err := parseFQDN(data.FQDN.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid fqdn in state", err.Error())
		return
	}
	customerID, err := r.client.ResolveCustomerID(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Unable to resolve customer_id", err.Error())
		return
	}
	existing, err := r.client.GetDomainForwarding(ctx, customerID, fqdn, false)
	if err != nil {
		var apiErr *client.APIError
		if errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Unable to read forwarding", err.Error())
		return
	}
	r.setState(&data, fqdn, existing)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *domainForwardingResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data forwardingResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	fqdn, forwarding, customerID, ok := r.expand(ctx, data, &resp.Diagnostics)
	if !ok {
		return
	}
	if err := r.client.UpdateDomainForwarding(ctx, customerID, fqdn, forwarding); err != nil {
		resp.Diagnostics.AddError("Unable to update forwarding", err.Error())
		return
	}

	r.refresh(ctx, fqdn, customerID, &data, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *domainForwardingResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data forwardingResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	fqdn, err := parseFQDN(data.FQDN.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid fqdn in state", err.Error())
		return
	}
	customerID, err := r.client.ResolveCustomerID(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Unable to resolve customer_id", err.Error())
		return
	}
	if err := r.client.DeleteDomainForwarding(ctx, customerID, fqdn); err != nil {
		resp.Diagnostics.AddError("Unable to delete forwarding", err.Error())
	}
}

func (r *domainForwardingResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("fqdn"), req.ID)...)
}

func (r *domainForwardingResource) expand(ctx context.Context, data forwardingResourceModel, diags *diag.Diagnostics) (string, client.DomainForwarding, string, bool) {
	fqdn, err := parseFQDN(data.FQDN.ValueString())
	if err != nil {
		diags.AddError("Invalid fqdn", err.Error())
		return "", client.DomainForwarding{}, "", false
	}
	mask, err := forwardMaskFromObject(ctx, data.Mask)
	if err != nil {
		diags.AddError("Invalid mask", err.Error())
		return "", client.DomainForwarding{}, "", false
	}
	customerID, err := r.client.ResolveCustomerID(ctx)
	if err != nil {
		diags.AddError("Unable to resolve customer_id", err.Error())
		return "", client.DomainForwarding{}, "", false
	}
	return fqdn, client.DomainForwarding{
		FQDN: fqdn,
		Type: data.Type.ValueString(),
		URL:  data.URL.ValueString(),
		Mask: mask,
	}, customerID, true
}

func (r *domainForwardingResource) refresh(ctx context.Context, fqdn, customerID string, data *forwardingResourceModel, diags *diag.Diagnostics) {
	existing, err := r.client.GetDomainForwarding(ctx, customerID, fqdn, false)
	if err != nil {
		diags.AddError("Unable to refresh forwarding", err.Error())
		return
	}
	r.setState(data, fqdn, existing)
}

func (r *domainForwardingResource) setState(data *forwardingResourceModel, fqdn string, existing *client.DomainForwarding) {
	data.ID = types.StringValue(fqdn)
	data.FQDN = types.StringValue(fqdn)
	data.Type = stringOrNull(existing.Type)
	data.URL = stringOrNull(existing.URL)
	data.Mask = forwardMaskObjectFromAPI(existing.Mask)
}
