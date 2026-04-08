package provider

import (
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
)

func useStateForUnknownString() planmodifier.String {
	return stringplanmodifier.UseStateForUnknown()
}

func useStateForUnknownBool() planmodifier.Bool {
	return boolplanmodifier.UseStateForUnknown()
}

func useStateForUnknownInt64() planmodifier.Int64 {
	return int64planmodifier.UseStateForUnknown()
}

func useStateForUnknownList() planmodifier.List {
	return listplanmodifier.UseStateForUnknown()
}
