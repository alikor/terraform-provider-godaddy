package provider

import (
	"context"
	"errors"
	"net/http"
	"reflect"

	"github.com/alikor/terraform-provider-godaddy/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	resourceschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = (*domainContactsResource)(nil)
var _ resource.ResourceWithConfigure = (*domainContactsResource)(nil)
var _ resource.ResourceWithImportState = (*domainContactsResource)(nil)

type domainContactsResource struct {
	client *client.Client
}

func NewDomainContactsResource() resource.Resource {
	return &domainContactsResource{}
}

func (r *domainContactsResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "godaddy_domain_contacts"
}

func (r *domainContactsResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = resourceschema.Schema{
		MarkdownDescription: "Manage the full contact set for an existing GoDaddy domain.",
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
			"identity_document_id": resourceschema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Reserved for later v2 support. The current implementation uses the v1 contacts path.",
			},
		},
		Blocks: map[string]resourceschema.Block{
			"registrant": contactBlock("Registrant contact."),
			"admin":      contactBlock("Admin contact."),
			"tech":       contactBlock("Tech contact."),
			"billing":    contactBlock("Billing contact."),
		},
	}
}

func contactBlock(description string) resourceschema.Block {
	return resourceschema.SingleNestedBlock{
		MarkdownDescription: description,
		Attributes: map[string]resourceschema.Attribute{
			"name_first":   resourceschema.StringAttribute{Required: true, MarkdownDescription: "First name."},
			"name_middle":  resourceschema.StringAttribute{Optional: true, MarkdownDescription: "Middle name."},
			"name_last":    resourceschema.StringAttribute{Required: true, MarkdownDescription: "Last name."},
			"organization": resourceschema.StringAttribute{Optional: true, MarkdownDescription: "Organization name."},
			"job_title":    resourceschema.StringAttribute{Optional: true, MarkdownDescription: "Job title."},
			"email":        resourceschema.StringAttribute{Required: true, MarkdownDescription: "Email address."},
			"phone":        resourceschema.StringAttribute{Required: true, MarkdownDescription: "Phone number."},
			"fax":          resourceschema.StringAttribute{Optional: true, MarkdownDescription: "Fax number."},
		},
		Blocks: map[string]resourceschema.Block{
			"address_mailing": resourceschema.SingleNestedBlock{
				MarkdownDescription: "Mailing address.",
				Attributes: map[string]resourceschema.Attribute{
					"address1":    resourceschema.StringAttribute{Required: true, MarkdownDescription: "Address line 1."},
					"address2":    resourceschema.StringAttribute{Optional: true, MarkdownDescription: "Address line 2."},
					"city":        resourceschema.StringAttribute{Required: true, MarkdownDescription: "City."},
					"state":       resourceschema.StringAttribute{Required: true, MarkdownDescription: "State or province."},
					"postal_code": resourceschema.StringAttribute{Required: true, MarkdownDescription: "Postal code."},
					"country":     resourceschema.StringAttribute{Required: true, MarkdownDescription: "ISO 3166-1 alpha-2 country code."},
				},
			},
		},
	}
}

func (r *domainContactsResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *domainContactsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	r.apply(ctx, req.Plan, &resp.State, &resp.Diagnostics)
}

func (r *domainContactsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data contactsResourceModel
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

func (r *domainContactsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	r.apply(ctx, req.Plan, &resp.State, &resp.Diagnostics)
}

func (r *domainContactsResource) Delete(_ context.Context, _ resource.DeleteRequest, resp *resource.DeleteResponse) {
	resp.Diagnostics.AddWarning("State-only delete", "Terraform management for this contacts resource has been removed, but the remote contact set was left unchanged.")
}

func (r *domainContactsResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("domain"), req.ID)...)
}

func (r *domainContactsResource) apply(ctx context.Context, getter interface {
	Get(context.Context, any) diag.Diagnostics
}, state interface {
	Set(context.Context, any) diag.Diagnostics
}, diags *diag.Diagnostics) {
	var data contactsResourceModel
	diags.Append(getter.Get(ctx, &data)...)
	if diags.HasError() {
		return
	}

	domain, current, ok := r.readCurrentDomain(ctx, data.Domain.ValueString(), diags)
	if !ok {
		return
	}

	contacts, err := contactsPayloadFromModel(ctx, data)
	if err != nil {
		diags.AddError("Invalid contacts", err.Error())
		return
	}

	currentContacts := client.DomainContacts{}
	if current.Contacts != nil {
		currentContacts = normalizedDomainContacts(*current.Contacts)
	}

	if !reflect.DeepEqual(currentContacts, contacts) {
		if err := r.client.PatchDomainContacts(ctx, domain, contacts); err != nil {
			diags.AddError("Unable to update domain contacts", err.Error())
			return
		}

		current, err = r.client.GetDomain(ctx, domain)
		if err != nil {
			diags.AddError("Unable to refresh domain contacts", err.Error())
			return
		}
	}

	r.setStateFromDomain(&data, domain, current)
	diags.Append(state.Set(ctx, &data)...)
}

func (r *domainContactsResource) readCurrentDomain(ctx context.Context, rawDomain string, diags *diag.Diagnostics) (string, *client.Domain, bool) {
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

func (r *domainContactsResource) setStateFromDomain(data *contactsResourceModel, domain string, current *client.Domain) {
	data.ID = types.StringValue(domain)
	data.Domain = types.StringValue(domain)
	if current.Contacts == nil {
		data.Registrant = objectNull(contactAttrTypes)
		data.Admin = objectNull(contactAttrTypes)
		data.Tech = objectNull(contactAttrTypes)
		data.Billing = objectNull(contactAttrTypes)
		return
	}

	contacts := normalizedDomainContacts(*current.Contacts)
	data.Registrant = contactObjectFromAPI(contacts.Registrant)
	data.Admin = contactObjectFromAPI(contacts.Admin)
	data.Tech = contactObjectFromAPI(contacts.Tech)
	data.Billing = contactObjectFromAPI(contacts.Billing)
}

func contactsPayloadFromModel(ctx context.Context, data contactsResourceModel) (client.DomainContacts, error) {
	registrant, err := contactFromObject(ctx, data.Registrant)
	if err != nil {
		return client.DomainContacts{}, err
	}
	admin, err := contactFromObject(ctx, data.Admin)
	if err != nil {
		return client.DomainContacts{}, err
	}
	tech, err := contactFromObject(ctx, data.Tech)
	if err != nil {
		return client.DomainContacts{}, err
	}
	billing, err := contactFromObject(ctx, data.Billing)
	if err != nil {
		return client.DomainContacts{}, err
	}

	return normalizedDomainContacts(client.DomainContacts{
		Registrant: registrant,
		Admin:      admin,
		Tech:       tech,
		Billing:    billing,
	}), nil
}

func normalizedDomainContacts(value client.DomainContacts) client.DomainContacts {
	value.Registrant = normalizeContact(value.Registrant)
	value.Admin = normalizeContact(value.Admin)
	value.Tech = normalizeContact(value.Tech)
	value.Billing = normalizeContact(value.Billing)
	return value
}

func normalizeContact(value client.Contact) client.Contact {
	return client.Contact{
		NameFirst:      value.NameFirst,
		NameMiddle:     value.NameMiddle,
		NameLast:       value.NameLast,
		Organization:   value.Organization,
		JobTitle:       value.JobTitle,
		Email:          value.Email,
		Phone:          value.Phone,
		Fax:            value.Fax,
		AddressMailing: value.AddressMailing,
	}
}
