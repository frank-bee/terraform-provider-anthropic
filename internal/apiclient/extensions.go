package apiclient

import (
	"bytes"
	"encoding/json"
)

// UnmarshalJSON tolerates the two shapes the Anthropic API returns for an
// agent's model field: either a bare string (older `agent-api-*` betas) or
// an object `{"id": "...", "speed": "..."}` (newer `managed-agents-*` betas).
func (a *AgentModelConfig) UnmarshalJSON(data []byte) error {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) > 0 && trimmed[0] == '"' {
		var s string
		if err := json.Unmarshal(trimmed, &s); err != nil {
			return err
		}
		a.Id = s
		a.Speed = nil
		return nil
	}
	type raw AgentModelConfig
	var r raw
	if err := json.Unmarshal(trimmed, &r); err != nil {
		return err
	}
	*a = AgentModelConfig(r)
	return nil
}
