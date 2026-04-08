package provider

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

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

var _ resource.Resource = (*dnsRecordSetResource)(nil)
var _ resource.ResourceWithConfigure = (*dnsRecordSetResource)(nil)
var _ resource.ResourceWithImportState = (*dnsRecordSetResource)(nil)

type dnsRecordSetResource struct {
	client *client.Client
}

func NewDNSRecordSetResource() resource.Resource {
	return &dnsRecordSetResource{}
}

func (r *dnsRecordSetResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "godaddy_dns_record_set"
}

func dnsRecordAttributes(mode attributeMode) map[string]resourceschema.Attribute {
	switch mode {
	case required:
		return map[string]resourceschema.Attribute{
			"data":     resourceschema.StringAttribute{Required: true, MarkdownDescription: "Record target/value."},
			"ttl":      resourceschema.Int64Attribute{Optional: true, MarkdownDescription: "Time to live in seconds."},
			"priority": resourceschema.Int64Attribute{Optional: true, MarkdownDescription: "MX/SRV priority."},
			"weight":   resourceschema.Int64Attribute{Optional: true, MarkdownDescription: "SRV weight."},
			"port":     resourceschema.Int64Attribute{Optional: true, MarkdownDescription: "SRV port."},
			"protocol": resourceschema.StringAttribute{Optional: true, MarkdownDescription: "SRV protocol."},
			"service":  resourceschema.StringAttribute{Optional: true, MarkdownDescription: "SRV service."},
		}
	default:
		return map[string]resourceschema.Attribute{
			"data":     resourceschema.StringAttribute{Computed: true, MarkdownDescription: "Record target/value."},
			"ttl":      resourceschema.Int64Attribute{Computed: true, MarkdownDescription: "Time to live in seconds."},
			"priority": resourceschema.Int64Attribute{Computed: true, MarkdownDescription: "MX/SRV priority."},
			"weight":   resourceschema.Int64Attribute{Computed: true, MarkdownDescription: "SRV weight."},
			"port":     resourceschema.Int64Attribute{Computed: true, MarkdownDescription: "SRV port."},
			"protocol": resourceschema.StringAttribute{Computed: true, MarkdownDescription: "SRV protocol."},
			"service":  resourceschema.StringAttribute{Computed: true, MarkdownDescription: "SRV service."},
		}
	}
}

type attributeMode string

const (
	required attributeMode = "required"
	computed attributeMode = "computed"
)

func (r *dnsRecordSetResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = resourceschema.Schema{
		MarkdownDescription: "Manage one GoDaddy DNS RRset.",
		Attributes: map[string]resourceschema.Attribute{
			"id": resourceschema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Canonical resource identifier in `domain,type,name` format.",
				PlanModifiers:       []planmodifier.String{useStateForUnknownString()},
			},
			"domain": resourceschema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Domain that owns the RRset.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"type": resourceschema.StringAttribute{
				Required:            true,
				MarkdownDescription: "RRset type. Supported: A, AAAA, CNAME, MX, SRV, TXT.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"name": resourceschema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "RRset name. Defaults to `@`.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"fqdn": resourceschema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Fully qualified domain name for the RRset.",
				PlanModifiers:       []planmodifier.String{useStateForUnknownString()},
			},
			"records": resourceschema.ListNestedAttribute{
				Required:            true,
				MarkdownDescription: "Normalized records owned by the RRset.",
				NestedObject: resourceschema.NestedAttributeObject{
					Attributes: dnsRecordAttributes(required),
				},
			},
		},
	}
}

func (r *dnsRecordSetResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *dnsRecordSetResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data dnsRecordSetResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	domain, recordType, recordName, records, ok := r.normalizeAndValidate(ctx, data, &resp.Diagnostics)
	if !ok {
		return
	}

	existing, err := r.client.GetDNSRecordSet(ctx, domain, recordType, recordName)
	if err == nil && len(existing) > 0 {
		resp.Diagnostics.AddError("DNS record set already exists", "Import the existing RRset instead of creating it.")
		return
	}

	var apiErr *client.APIError
	if err != nil && !(errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusNotFound) {
		resp.Diagnostics.AddError("Unable to inspect existing DNS record set", err.Error())
		return
	}

	if err := r.client.PutDNSRecordSet(ctx, domain, recordType, recordName, records); err != nil {
		resp.Diagnostics.AddError("Unable to create DNS record set", err.Error())
		return
	}

	r.readIntoState(ctx, domain, recordType, recordName, &data, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *dnsRecordSetResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data dnsRecordSetResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	domain, err := parseDomain(data.Domain.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid domain in state", err.Error())
		return
	}

	recordType := normalize.RecordType(data.Type.ValueString())
	recordName := normalize.RecordName(data.Name.ValueString())

	statusRemoved := r.readIntoState(ctx, domain, recordType, recordName, &data, &resp.Diagnostics)
	if statusRemoved {
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *dnsRecordSetResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data dnsRecordSetResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	domain, recordType, recordName, records, ok := r.normalizeAndValidate(ctx, data, &resp.Diagnostics)
	if !ok {
		return
	}

	if err := r.client.PutDNSRecordSet(ctx, domain, recordType, recordName, records); err != nil {
		resp.Diagnostics.AddError("Unable to update DNS record set", err.Error())
		return
	}

	r.readIntoState(ctx, domain, recordType, recordName, &data, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *dnsRecordSetResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data dnsRecordSetResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	domain, err := parseDomain(data.Domain.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid domain in state", err.Error())
		return
	}

	recordType := normalize.RecordType(data.Type.ValueString())
	recordName := normalize.RecordName(data.Name.ValueString())

	if err := r.client.DeleteDNSRecordSet(ctx, domain, recordType, recordName); err != nil {
		resp.Diagnostics.AddError("Unable to delete DNS record set", err.Error())
	}
}

func (r *dnsRecordSetResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.Split(req.ID, ",")
	if len(parts) != 3 {
		resp.Diagnostics.AddError("Invalid import ID", "Expected import ID in the format `domain,type,name`.")
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("domain"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("type"), parts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), parts[2])...)
}

func (r *dnsRecordSetResource) normalizeAndValidate(ctx context.Context, data dnsRecordSetResourceModel, diags *diag.Diagnostics) (string, string, string, []client.DNSRecord, bool) {
	domain, err := parseDomain(data.Domain.ValueString())
	if err != nil {
		diags.AddError("Invalid domain", err.Error())
		return "", "", "", nil, false
	}

	recordType := normalize.RecordType(data.Type.ValueString())
	if err := validateManagedRecordType(recordType); err != nil {
		diags.AddError("Invalid record type", err.Error())
		return "", "", "", nil, false
	}

	recordName := normalize.RecordName(data.Name.ValueString())
	records, err := recordsFromList(ctx, data.Records)
	if err != nil {
		diags.AddError("Invalid record list", err.Error())
		return "", "", "", nil, false
	}

	if len(records) == 0 {
		diags.AddError("Missing records", "At least one record is required.")
		return "", "", "", nil, false
	}
	if err := validateRRset(recordType, records); err != nil {
		diags.AddError("Invalid DNS record set", err.Error())
		return "", "", "", nil, false
	}

	return domain, recordType, recordName, records, true
}

func (r *dnsRecordSetResource) readIntoState(ctx context.Context, domain, recordType, recordName string, data *dnsRecordSetResourceModel, diags *diag.Diagnostics) bool {
	records, err := r.client.GetDNSRecordSet(ctx, domain, recordType, recordName)
	if err != nil {
		var apiErr *client.APIError
		if errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusNotFound {
			return true
		}
		diags.AddError("Unable to refresh DNS record set", err.Error())
		return false
	}

	if len(records) == 0 {
		return true
	}

	data.ID = types.StringValue(fmt.Sprintf("%s,%s,%s", domain, recordType, recordName))
	data.Domain = types.StringValue(domain)
	data.Type = types.StringValue(recordType)
	data.Name = types.StringValue(recordName)
	data.FQDN = types.StringValue(recordSetFQDN(domain, recordName))
	data.Records = recordsToList(records)
	return false
}

func validateRRset(recordType string, records []client.DNSRecord) error {
	switch recordType {
	case "CNAME":
		if len(records) != 1 {
			return fmt.Errorf("CNAME record sets must contain exactly one record")
		}
		for _, record := range records {
			if record.Priority != 0 || record.Weight != 0 || record.Port != 0 || record.Protocol != "" || record.Service != "" {
				return fmt.Errorf("CNAME records must not set priority, weight, port, protocol, or service")
			}
		}
	case "MX":
		for _, record := range records {
			if record.Priority == 0 {
				return fmt.Errorf("MX records must set priority")
			}
		}
	case "SRV":
		for _, record := range records {
			if record.Priority == 0 || record.Weight == 0 || record.Port == 0 || record.Protocol == "" || record.Service == "" {
				return fmt.Errorf("SRV records must set priority, weight, port, protocol, and service")
			}
		}
	default:
		for _, record := range records {
			if record.Priority != 0 || record.Weight != 0 || record.Port != 0 || record.Protocol != "" || record.Service != "" {
				return fmt.Errorf("%s records must not set priority, weight, port, protocol, or service", recordType)
			}
		}
	}

	return nil
}
