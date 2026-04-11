package provider

import (
	"github.com/frank-bee/terraform-provider-anthropic/internal/apiclient"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type AgentModel struct {
	Id         types.String     `tfsdk:"id"`
	Version    types.String     `tfsdk:"version"`
	Name       types.String     `tfsdk:"name"`
	System     types.String     `tfsdk:"system"`
	Model      types.String     `tfsdk:"model"`
	Tools      []AgentToolModel `tfsdk:"tools"`
	McpServers []McpServerModel `tfsdk:"mcp_servers"`
	Skills     []SkillModel     `tfsdk:"skills"`
}

type AgentToolModel struct {
	Type types.String `tfsdk:"type"`
}

type McpServerModel struct {
	Name types.String `tfsdk:"name"`
	Type types.String `tfsdk:"type"`
	Url  types.String `tfsdk:"url"`
}

type SkillModel struct {
	SkillId types.String `tfsdk:"skill_id"`
	Type    types.String `tfsdk:"type"`
	Version types.String `tfsdk:"version"`
}

func (m *AgentModel) Fill(a apiclient.Agent) error {
	m.Id = types.StringValue(a.Id)
	m.Version = types.StringValue(a.Version)
	m.Name = types.StringValue(a.Name)
	m.System = types.StringPointerValue(a.System)
	m.Model = types.StringPointerValue(a.Model)

	if a.Tools != nil {
		tools := make([]AgentToolModel, len(*a.Tools))
		for i, t := range *a.Tools {
			tools[i] = AgentToolModel{
				Type: types.StringValue(t.Type),
			}
		}
		m.Tools = tools
	} else {
		m.Tools = []AgentToolModel{}
	}

	if a.McpServers != nil {
		servers := make([]McpServerModel, len(*a.McpServers))
		for i, s := range *a.McpServers {
			servers[i] = McpServerModel{
				Name: types.StringValue(s.Name),
				Type: types.StringValue(s.Type),
				Url:  types.StringValue(s.Url),
			}
		}
		m.McpServers = servers
	} else {
		m.McpServers = []McpServerModel{}
	}

	if a.Skills != nil {
		skills := make([]SkillModel, len(*a.Skills))
		for i, s := range *a.Skills {
			skills[i] = SkillModel{
				SkillId: types.StringValue(s.SkillId),
				Type:    types.StringValue(s.Type),
				Version: types.StringValue(s.Version),
			}
		}
		m.Skills = skills
	} else {
		m.Skills = []SkillModel{}
	}

	return nil
}
