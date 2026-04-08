package provider

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/alikor/terraform-provider-godaddy/internal/client"
	"github.com/alikor/terraform-provider-godaddy/internal/normalize"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

var dnsRecordAttrTypes = map[string]attr.Type{
	"data":     types.StringType,
	"ttl":      types.Int64Type,
	"priority": types.Int64Type,
	"weight":   types.Int64Type,
	"port":     types.Int64Type,
	"protocol": types.StringType,
	"service":  types.StringType,
}

var domainSummaryAttrTypes = map[string]attr.Type{
	"domain":       types.StringType,
	"domain_id":    types.Int64Type,
	"status":       types.StringType,
	"created_at":   types.StringType,
	"expires_at":   types.StringType,
	"name_servers": types.ListType{ElemType: types.StringType},
}

var agreementAttrTypes = map[string]attr.Type{
	"agreement_key": types.StringType,
	"title":         types.StringType,
	"content":       types.StringType,
	"url":           types.StringType,
}

var consentAttrTypes = map[string]attr.Type{
	"agreed_by":      types.StringType,
	"agreed_at":      types.StringType,
	"agreement_keys": types.ListType{ElemType: types.StringType},
}

var mailingAddressAttrTypes = map[string]attr.Type{
	"address1":    types.StringType,
	"address2":    types.StringType,
	"city":        types.StringType,
	"state":       types.StringType,
	"postal_code": types.StringType,
	"country":     types.StringType,
}

var contactAttrTypes = map[string]attr.Type{
	"name_first":      types.StringType,
	"name_middle":     types.StringType,
	"name_last":       types.StringType,
	"organization":    types.StringType,
	"job_title":       types.StringType,
	"email":           types.StringType,
	"phone":           types.StringType,
	"fax":             types.StringType,
	"address_mailing": types.ObjectType{AttrTypes: mailingAddressAttrTypes},
}

func parseDomain(domain string) (string, error) {
	return normalize.Domain(domain)
}

func parseFQDN(name string) (string, error) {
	return normalize.FQDN(name)
}

func toStringList(values []string) types.List {
	if len(values) == 0 {
		return types.ListNull(types.StringType)
	}

	elements := make([]attr.Value, 0, len(values))
	for _, value := range values {
		elements = append(elements, types.StringValue(value))
	}
	return types.ListValueMust(types.StringType, elements)
}

func stringsFromList(ctx context.Context, list types.List) ([]string, error) {
	if list.IsNull() || list.IsUnknown() {
		return nil, nil
	}

	var values []string
	diags := list.ElementsAs(ctx, &values, false)
	if diags.HasError() {
		return nil, fmt.Errorf("unable to decode list: %s", diags.Errors()[0].Summary())
	}
	return values, nil
}

func recordsFromList(ctx context.Context, list types.List) ([]client.DNSRecord, error) {
	if list.IsNull() || list.IsUnknown() {
		return nil, nil
	}

	type dnsRecordModel struct {
		Data     types.String `tfsdk:"data"`
		TTL      types.Int64  `tfsdk:"ttl"`
		Priority types.Int64  `tfsdk:"priority"`
		Weight   types.Int64  `tfsdk:"weight"`
		Port     types.Int64  `tfsdk:"port"`
		Protocol types.String `tfsdk:"protocol"`
		Service  types.String `tfsdk:"service"`
	}

	var models []dnsRecordModel
	diags := list.ElementsAs(ctx, &models, false)
	if diags.HasError() {
		return nil, fmt.Errorf("unable to decode records: %s", diags.Errors()[0].Summary())
	}

	records := make([]client.DNSRecord, 0, len(models))
	for _, model := range models {
		record := client.DNSRecord{
			Data:     model.Data.ValueString(),
			TTL:      model.TTL.ValueInt64(),
			Priority: model.Priority.ValueInt64(),
			Weight:   model.Weight.ValueInt64(),
			Port:     model.Port.ValueInt64(),
			Protocol: model.Protocol.ValueString(),
			Service:  model.Service.ValueString(),
		}
		records = append(records, record)
	}

	return normalize.SortRecords(records), nil
}

func recordsToList(records []client.DNSRecord) types.List {
	if len(records) == 0 {
		return types.ListNull(types.ObjectType{AttrTypes: dnsRecordAttrTypes})
	}

	elements := make([]attr.Value, 0, len(records))
	for _, record := range normalize.SortRecords(records) {
		elements = append(elements, types.ObjectValueMust(
			dnsRecordAttrTypes,
			map[string]attr.Value{
				"data":     types.StringValue(record.Data),
				"ttl":      types.Int64Value(record.TTL),
				"priority": types.Int64Value(record.Priority),
				"weight":   types.Int64Value(record.Weight),
				"port":     types.Int64Value(record.Port),
				"protocol": stringOrNull(record.Protocol),
				"service":  stringOrNull(record.Service),
			},
		))
	}

	return types.ListValueMust(types.ObjectType{AttrTypes: dnsRecordAttrTypes}, elements)
}

func domainSummariesToList(summaries []client.DomainSummary) types.List {
	if len(summaries) == 0 {
		return types.ListNull(types.ObjectType{AttrTypes: domainSummaryAttrTypes})
	}

	elements := make([]attr.Value, 0, len(summaries))
	for _, summary := range summaries {
		elements = append(elements, types.ObjectValueMust(
			domainSummaryAttrTypes,
			map[string]attr.Value{
				"domain":       stringOrNull(summary.Domain),
				"domain_id":    types.Int64Value(summary.DomainID),
				"status":       stringOrNull(summary.Status),
				"created_at":   stringOrNull(summary.CreatedAt),
				"expires_at":   stringOrNull(summary.ExpiresAt),
				"name_servers": toStringList(normalize.NormalizeNameservers(summary.NameServers)),
			},
		))
	}

	return types.ListValueMust(types.ObjectType{AttrTypes: domainSummaryAttrTypes}, elements)
}

func agreementsToList(agreements []client.Agreement) types.List {
	if len(agreements) == 0 {
		return types.ListNull(types.ObjectType{AttrTypes: agreementAttrTypes})
	}

	elements := make([]attr.Value, 0, len(agreements))
	for _, agreement := range agreements {
		elements = append(elements, types.ObjectValueMust(
			agreementAttrTypes,
			map[string]attr.Value{
				"agreement_key": stringOrNull(agreement.AgreementKey),
				"title":         stringOrNull(agreement.Title),
				"content":       stringOrNull(agreement.Content),
				"url":           stringOrNull(agreement.URL),
			},
		))
	}

	return types.ListValueMust(types.ObjectType{AttrTypes: agreementAttrTypes}, elements)
}

func stringOrNull(value string) types.String {
	if strings.TrimSpace(value) == "" {
		return types.StringNull()
	}
	return types.StringValue(value)
}

func stringValueOrNull(value string) attr.Value {
	return stringOrNull(value)
}

func optionalBool(value bool, set bool) types.Bool {
	if !set {
		return types.BoolNull()
	}
	return types.BoolValue(value)
}

func buildDomainQuery(data domainsDataSourceModel, ctx context.Context) (url.Values, error) {
	query := url.Values{}

	statuses, err := stringsFromList(ctx, data.Statuses)
	if err != nil {
		return nil, err
	}
	if len(statuses) > 0 {
		query.Set("statuses", strings.Join(statuses, ","))
	}

	statusGroups, err := stringsFromList(ctx, data.StatusGroups)
	if err != nil {
		return nil, err
	}
	if len(statusGroups) > 0 {
		query.Set("statusGroups", strings.Join(statusGroups, ","))
	}

	includes, err := stringsFromList(ctx, data.Includes)
	if err != nil {
		return nil, err
	}
	if len(includes) > 0 {
		query.Set("includes", strings.Join(includes, ","))
	}

	if !data.Limit.IsNull() && !data.Limit.IsUnknown() && data.Limit.ValueInt64() > 0 {
		query.Set("limit", fmt.Sprintf("%d", data.Limit.ValueInt64()))
	}
	if !data.Marker.IsNull() && !data.Marker.IsUnknown() && data.Marker.ValueString() != "" {
		query.Set("marker", data.Marker.ValueString())
	}
	if !data.ModifiedDate.IsNull() && !data.ModifiedDate.IsUnknown() && data.ModifiedDate.ValueString() != "" {
		query.Set("modifiedDate", data.ModifiedDate.ValueString())
	}

	return query, nil
}

func objectNull(attrTypes map[string]attr.Type) types.Object {
	return types.ObjectNull(attrTypes)
}

func consentObjectFromAPI(value *client.Consent) types.Object {
	if value == nil {
		return objectNull(consentAttrTypes)
	}

	return types.ObjectValueMust(consentAttrTypes, map[string]attr.Value{
		"agreed_by":      stringOrNull(value.AgreedBy),
		"agreed_at":      stringOrNull(value.AgreedAt),
		"agreement_keys": toStringList(value.AgreementKeys),
	})
}

func consentFromObject(ctx context.Context, obj types.Object) (*client.Consent, error) {
	if obj.IsNull() || obj.IsUnknown() {
		return nil, nil
	}

	var model consentModel
	diags := obj.As(ctx, &model, basetypes.ObjectAsOptions{})
	if diags.HasError() {
		return nil, fmt.Errorf("unable to decode consent object: %s", diags.Errors()[0].Summary())
	}

	keys, err := stringsFromList(ctx, model.AgreementKeys)
	if err != nil {
		return nil, err
	}

	return &client.Consent{
		AgreedBy:      model.AgreedBy.ValueString(),
		AgreedAt:      model.AgreedAt.ValueString(),
		AgreementKeys: keys,
	}, nil
}

func mailingAddressObjectFromAPI(value client.MailingAddress) types.Object {
	return types.ObjectValueMust(mailingAddressAttrTypes, map[string]attr.Value{
		"address1":    stringOrNull(value.Address1),
		"address2":    stringOrNull(value.Address2),
		"city":        stringOrNull(value.City),
		"state":       stringOrNull(value.State),
		"postal_code": stringOrNull(value.PostalCode),
		"country":     stringOrNull(value.Country),
	})
}

func mailingAddressFromObject(ctx context.Context, obj types.Object) (client.MailingAddress, error) {
	var model mailingAddressModel
	if obj.IsNull() || obj.IsUnknown() {
		return client.MailingAddress{}, fmt.Errorf("address_mailing is required")
	}

	diags := obj.As(ctx, &model, basetypes.ObjectAsOptions{})
	if diags.HasError() {
		return client.MailingAddress{}, fmt.Errorf("unable to decode address_mailing: %s", diags.Errors()[0].Summary())
	}

	return client.MailingAddress{
		Address1:   model.Address1.ValueString(),
		Address2:   model.Address2.ValueString(),
		City:       model.City.ValueString(),
		State:      model.State.ValueString(),
		PostalCode: model.PostalCode.ValueString(),
		Country:    model.Country.ValueString(),
	}, nil
}

func contactObjectFromAPI(value client.Contact) types.Object {
	return types.ObjectValueMust(contactAttrTypes, map[string]attr.Value{
		"name_first":      stringOrNull(value.NameFirst),
		"name_middle":     stringOrNull(value.NameMiddle),
		"name_last":       stringOrNull(value.NameLast),
		"organization":    stringOrNull(value.Organization),
		"job_title":       stringOrNull(value.JobTitle),
		"email":           stringOrNull(value.Email),
		"phone":           stringOrNull(value.Phone),
		"fax":             stringOrNull(value.Fax),
		"address_mailing": mailingAddressObjectFromAPI(value.AddressMailing),
	})
}

func contactFromObject(ctx context.Context, obj types.Object) (client.Contact, error) {
	var model contactModel
	if obj.IsNull() || obj.IsUnknown() {
		return client.Contact{}, fmt.Errorf("contact is required")
	}

	diags := obj.As(ctx, &model, basetypes.ObjectAsOptions{})
	if diags.HasError() {
		return client.Contact{}, fmt.Errorf("unable to decode contact object: %s", diags.Errors()[0].Summary())
	}

	address, err := mailingAddressFromObject(ctx, model.AddressMailing)
	if err != nil {
		return client.Contact{}, err
	}

	return normalize.Contact(client.Contact{
		NameFirst:      model.NameFirst.ValueString(),
		NameMiddle:     model.NameMiddle.ValueString(),
		NameLast:       model.NameLast.ValueString(),
		Organization:   model.Organization.ValueString(),
		JobTitle:       model.JobTitle.ValueString(),
		Email:          model.Email.ValueString(),
		Phone:          model.Phone.ValueString(),
		Fax:            model.Fax.ValueString(),
		AddressMailing: address,
	}), nil
}
