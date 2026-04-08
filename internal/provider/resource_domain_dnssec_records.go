package provider

import (
	"context"
	"errors"
	"net/http"
	"slices"
	"time"

	"github.com/alikor/terraform-provider-godaddy/internal/client"
	"github.com/alikor/terraform-provider-godaddy/internal/normalize"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	resourceschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = (*domainDNSSECRecordsResource)(nil)
var _ resource.ResourceWithConfigure = (*domainDNSSECRecordsResource)(nil)
var _ resource.ResourceWithImportState = (*domainDNSSECRecordsResource)(nil)

type domainDNSSECRecordsResource struct {
	client *client.Client
}

func NewDomainDNSSECRecordsResource() resource.Resource {
	return &domainDNSSECRecordsResource{}
}

func (r *domainDNSSECRecordsResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "godaddy_domain_dnssec_records"
}

func (r *domainDNSSECRecordsResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = resourceschema.Schema{
		MarkdownDescription: "Manage the full DNSSEC record set for an existing GoDaddy domain.",
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
			"records": resourceschema.ListNestedAttribute{
				Required:            true,
				MarkdownDescription: "Full desired DNSSEC record set.",
				NestedObject: resourceschema.NestedAttributeObject{
					Attributes: map[string]resourceschema.Attribute{
						"key_tag":            resourceschema.Int64Attribute{Required: true, MarkdownDescription: "Key tag."},
						"algorithm":          resourceschema.StringAttribute{Required: true, MarkdownDescription: "Algorithm."},
						"digest_type":        resourceschema.StringAttribute{Required: true, MarkdownDescription: "Digest type."},
						"digest":             resourceschema.StringAttribute{Required: true, MarkdownDescription: "Digest."},
						"flags":              resourceschema.StringAttribute{Optional: true, MarkdownDescription: "Optional key role flags."},
						"public_key":         resourceschema.StringAttribute{Optional: true, MarkdownDescription: "Optional public key."},
						"max_signature_life": resourceschema.Int64Attribute{Optional: true, MarkdownDescription: "Optional max signature life."},
					},
				},
			},
		},
	}
}

func (r *domainDNSSECRecordsResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *domainDNSSECRecordsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data dnssecResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	domain, desired, customerID, current, partial, ok := r.expand(ctx, data, &resp.Diagnostics)
	if !ok {
		return
	}
	if partial && len(current) == 0 {
		resp.Diagnostics.AddError("Partial DNSSEC read", "GoDaddy returned a partial response without DNSSEC records; refusing to continue.")
		return
	}
	if len(current) > 0 {
		resp.Diagnostics.AddError("DNSSEC records already exist", "Import the existing DNSSEC set instead of creating it.")
		return
	}
	if err := r.client.AddDNSSECRecords(ctx, customerID, domain, desired); err != nil {
		resp.Diagnostics.AddError("Unable to create DNSSEC records", err.Error())
		return
	}
	if _, err := r.client.PollDomainAction(ctx, customerID, domain, "DNSSEC_CREATE", "", 10*time.Minute); err != nil {
		resp.Diagnostics.AddError("DNSSEC create action failed", err.Error())
		return
	}
	r.refresh(ctx, domain, customerID, &data, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *domainDNSSECRecordsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data dnssecResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	domain, err := parseDomain(data.Domain.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid domain in state", err.Error())
		return
	}
	customerID, err := r.client.ResolveCustomerID(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Unable to resolve customer_id", err.Error())
		return
	}
	domainV2, partial, err := r.client.GetDomainV2(ctx, customerID, domain, []string{"dnssecRecords"})
	if err != nil {
		var apiErr *client.APIError
		if errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Unable to read DNSSEC records", err.Error())
		return
	}
	if partial && len(domainV2.DNSSECRecords) == 0 {
		resp.Diagnostics.AddError("Partial DNSSEC read", "GoDaddy returned a partial response without DNSSEC records.")
		return
	}
	if len(domainV2.DNSSECRecords) == 0 {
		resp.State.RemoveResource(ctx)
		return
	}
	data.ID = types.StringValue(domain)
	data.Domain = types.StringValue(domain)
	data.Records = dnssecRecordsToList(domainV2.DNSSECRecords)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *domainDNSSECRecordsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data dnssecResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	domain, desired, customerID, current, partial, ok := r.expand(ctx, data, &resp.Diagnostics)
	if !ok {
		return
	}
	if partial && len(current) == 0 {
		resp.Diagnostics.AddError("Partial DNSSEC read", "GoDaddy returned a partial response without DNSSEC records; refusing to continue.")
		return
	}

	toAdd, toRemove := diffDNSSECRecords(current, desired)
	if len(toAdd) > 0 {
		if err := r.client.AddDNSSECRecords(ctx, customerID, domain, toAdd); err != nil {
			resp.Diagnostics.AddError("Unable to add DNSSEC records", err.Error())
			return
		}
		if _, err := r.client.PollDomainAction(ctx, customerID, domain, "DNSSEC_CREATE", "", 10*time.Minute); err != nil {
			resp.Diagnostics.AddError("DNSSEC create action failed", err.Error())
			return
		}
	}
	if len(toRemove) > 0 {
		if err := r.client.DeleteDNSSECRecords(ctx, customerID, domain, toRemove); err != nil {
			resp.Diagnostics.AddError("Unable to remove DNSSEC records", err.Error())
			return
		}
		if _, err := r.client.PollDomainAction(ctx, customerID, domain, "DNSSEC_DELETE", "", 10*time.Minute); err != nil {
			resp.Diagnostics.AddError("DNSSEC delete action failed", err.Error())
			return
		}
	}

	r.refresh(ctx, domain, customerID, &data, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *domainDNSSECRecordsResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data dnssecResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	domain, err := parseDomain(data.Domain.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid domain in state", err.Error())
		return
	}
	customerID, err := r.client.ResolveCustomerID(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Unable to resolve customer_id", err.Error())
		return
	}
	domainV2, partial, err := r.client.GetDomainV2(ctx, customerID, domain, []string{"dnssecRecords"})
	if err != nil {
		resp.Diagnostics.AddError("Unable to read DNSSEC records", err.Error())
		return
	}
	if partial && len(domainV2.DNSSECRecords) == 0 {
		resp.Diagnostics.AddError("Partial DNSSEC read", "GoDaddy returned a partial response without DNSSEC records.")
		return
	}
	if len(domainV2.DNSSECRecords) == 0 {
		return
	}
	if err := r.client.DeleteDNSSECRecords(ctx, customerID, domain, normalize.SortDNSSECRecords(domainV2.DNSSECRecords)); err != nil {
		resp.Diagnostics.AddError("Unable to delete DNSSEC records", err.Error())
		return
	}
	if _, err := r.client.PollDomainAction(ctx, customerID, domain, "DNSSEC_DELETE", "", 10*time.Minute); err != nil {
		resp.Diagnostics.AddError("DNSSEC delete action failed", err.Error())
	}
}

func (r *domainDNSSECRecordsResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("domain"), req.ID)...)
}

func (r *domainDNSSECRecordsResource) expand(ctx context.Context, data dnssecResourceModel, diags *diag.Diagnostics) (string, []client.DNSSECRecord, string, []client.DNSSECRecord, bool, bool) {
	domain, err := parseDomain(data.Domain.ValueString())
	if err != nil {
		diags.AddError("Invalid domain", err.Error())
		return "", nil, "", nil, false, false
	}
	desired, err := dnssecRecordsFromList(ctx, data.Records)
	if err != nil {
		diags.AddError("Invalid DNSSEC records", err.Error())
		return "", nil, "", nil, false, false
	}
	customerID, err := r.client.ResolveCustomerID(ctx)
	if err != nil {
		diags.AddError("Unable to resolve customer_id", err.Error())
		return "", nil, "", nil, false, false
	}
	domainV2, partial, err := r.client.GetDomainV2(ctx, customerID, domain, []string{"dnssecRecords"})
	if err != nil {
		diags.AddError("Unable to read current DNSSEC records", err.Error())
		return "", nil, "", nil, false, false
	}
	return domain, desired, customerID, normalize.SortDNSSECRecords(domainV2.DNSSECRecords), partial, true
}

func (r *domainDNSSECRecordsResource) refresh(ctx context.Context, domain, customerID string, data *dnssecResourceModel, diags *diag.Diagnostics) {
	domainV2, partial, err := r.client.GetDomainV2(ctx, customerID, domain, []string{"dnssecRecords"})
	if err != nil {
		diags.AddError("Unable to refresh DNSSEC records", err.Error())
		return
	}
	if partial && len(domainV2.DNSSECRecords) == 0 {
		diags.AddError("Partial DNSSEC read", "GoDaddy returned a partial response without DNSSEC records.")
		return
	}
	data.ID = types.StringValue(domain)
	data.Domain = types.StringValue(domain)
	data.Records = dnssecRecordsToList(domainV2.DNSSECRecords)
}

func diffDNSSECRecords(current, desired []client.DNSSECRecord) ([]client.DNSSECRecord, []client.DNSSECRecord) {
	current = normalize.SortDNSSECRecords(current)
	desired = normalize.SortDNSSECRecords(desired)

	contains := func(list []client.DNSSECRecord, target client.DNSSECRecord) bool {
		return slices.ContainsFunc(list, func(candidate client.DNSSECRecord) bool {
			return candidate == target
		})
	}

	var toAdd []client.DNSSECRecord
	for _, record := range desired {
		if !contains(current, record) {
			toAdd = append(toAdd, record)
		}
	}

	var toRemove []client.DNSSECRecord
	for _, record := range current {
		if !contains(desired, record) {
			toRemove = append(toRemove, record)
		}
	}

	return toAdd, toRemove
}
