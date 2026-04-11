package provider

import (
	"context"

	"github.com/frank-bee/terraform-provider-anthropic/internal/apiclient"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type EnvironmentModel struct {
	Id             types.String `tfsdk:"id"`
	Name           types.String `tfsdk:"name"`
	ConfigType     types.String `tfsdk:"config_type"`
	NetworkingType types.String `tfsdk:"networking_type"`
	Packages       types.Map    `tfsdk:"packages"`
	CreatedAt      types.String `tfsdk:"created_at"`
	UpdatedAt      types.String `tfsdk:"updated_at"`
}

func (m *EnvironmentModel) Fill(ctx context.Context, e apiclient.Environment) error {
	m.Id = types.StringValue(e.Id)
	m.Name = types.StringValue(e.Name)
	m.ConfigType = types.StringValue(e.Config.Type)
	m.NetworkingType = types.StringValue(e.Config.Networking.Type)
	m.CreatedAt = types.StringPointerValue(e.CreatedAt)
	m.UpdatedAt = types.StringPointerValue(e.UpdatedAt)

	if e.Config.Packages != nil && len(*e.Config.Packages) > 0 {
		elems := make(map[string]attr.Value, len(*e.Config.Packages))
		for k, v := range *e.Config.Packages {
			elems[k] = types.StringValue(v)
		}
		m.Packages = types.MapValueMust(types.StringType, elems)
	} else {
		m.Packages = types.MapNull(types.StringType)
	}

	return nil
}
