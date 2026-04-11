package apiclient

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
)

// Skill represents a skill resource from the API.
type Skill struct {
	Type          string  `json:"type"`
	Id            string  `json:"id"`
	CreatedAt     string  `json:"created_at"`
	UpdatedAt     string  `json:"updated_at"`
	DisplayTitle  string  `json:"display_title"`
	Source        string  `json:"source"`
	LatestVersion string  `json:"latest_version"`
}

// SkillVersion represents a specific version of a skill.
type SkillVersion struct {
	Type        string `json:"type"`
	SkillId     string `json:"skill_id"`
	Id          string `json:"id"`
	Version     string `json:"version"`
	Directory   string `json:"directory"`
	Name        string `json:"name"`
	Description string `json:"description"`
	CreatedAt   string `json:"created_at"`
}

// SkillListResponse is the response from listing skills.
type SkillListResponse struct {
	Data     []Skill `json:"data"`
	HasMore  bool    `json:"has_more"`
	NextPage *string `json:"next_page"`
}

// SkillVersionListResponse is the response from listing skill versions.
type SkillVersionListResponse struct {
	Data     []SkillVersion `json:"data"`
	HasMore  bool           `json:"has_more"`
	NextPage *string        `json:"next_page"`
}

// SkillDeletedResponse is the response from deleting a skill.
type SkillDeletedResponse struct {
	Type string `json:"type"`
	Id   string `json:"id"`
}

// SkillsClient provides methods for the Skills API.
type SkillsClient struct {
	baseURL    string
	httpClient *http.Client
	editors    []RequestEditorFn
}

// NewSkillsClient creates a new SkillsClient from a ClientWithResponses.
func NewSkillsClient(client *ClientWithResponses, baseURL string, httpClient *http.Client, editors []RequestEditorFn) *SkillsClient {
	return &SkillsClient{
		baseURL:    baseURL,
		httpClient: httpClient,
		editors:    editors,
	}
}

func (c *SkillsClient) applyEditors(ctx context.Context, req *http.Request) error {
	for _, editor := range c.editors {
		if err := editor(ctx, req); err != nil {
			return err
		}
	}
	// Override the beta header for skills API — must come after editors
	// since the global editor sets agent-api beta header.
	req.Header.Set("anthropic-beta", "skills-2025-10-02")
	return nil
}

// CreateSkill creates a custom skill by uploading a zip file containing SKILL.md.
func (c *SkillsClient) CreateSkill(ctx context.Context, displayTitle, skillName, skillContent string) (*Skill, error) {
	// Build the zip in memory
	var zipBuf bytes.Buffer
	zipWriter := zip.NewWriter(&zipBuf)

	// The folder name inside the zip must match the skill name in SKILL.md
	f, err := zipWriter.Create(skillName + "/SKILL.md")
	if err != nil {
		return nil, fmt.Errorf("creating zip entry: %w", err)
	}
	if _, err := f.Write([]byte(skillContent)); err != nil {
		return nil, fmt.Errorf("writing skill content: %w", err)
	}
	if err := zipWriter.Close(); err != nil {
		return nil, fmt.Errorf("closing zip: %w", err)
	}

	// Build multipart form
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	if err := writer.WriteField("display_title", displayTitle); err != nil {
		return nil, fmt.Errorf("writing display_title field: %w", err)
	}

	part, err := writer.CreateFormFile("files[]", skillName+".zip")
	if err != nil {
		return nil, fmt.Errorf("creating form file: %w", err)
	}
	if _, err := io.Copy(part, &zipBuf); err != nil {
		return nil, fmt.Errorf("copying zip to form: %w", err)
	}
	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("closing multipart writer: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v1/skills", &body)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	if err := c.applyEditors(ctx, req); err != nil {
		return nil, fmt.Errorf("applying request editors: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(respBody))
	}

	var skill Skill
	if err := json.Unmarshal(respBody, &skill); err != nil {
		return nil, fmt.Errorf("unmarshaling response: %w", err)
	}

	return &skill, nil
}

// GetSkill retrieves a skill by ID.
func (c *SkillsClient) GetSkill(ctx context.Context, skillId string) (*Skill, int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/v1/skills/"+skillId, nil)
	if err != nil {
		return nil, 0, fmt.Errorf("creating request: %w", err)
	}
	if err := c.applyEditors(ctx, req); err != nil {
		return nil, 0, fmt.Errorf("applying request editors: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, resp.StatusCode, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(respBody))
	}

	var skill Skill
	if err := json.Unmarshal(respBody, &skill); err != nil {
		return nil, resp.StatusCode, fmt.Errorf("unmarshaling response: %w", err)
	}

	return &skill, resp.StatusCode, nil
}

// ListSkills lists all skills.
func (c *SkillsClient) ListSkills(ctx context.Context) (*SkillListResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/v1/skills", nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	if err := c.applyEditors(ctx, req); err != nil {
		return nil, fmt.Errorf("applying request editors: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(respBody))
	}

	var result SkillListResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("unmarshaling response: %w", err)
	}

	return &result, nil
}

// ListSkillVersions lists all versions of a skill.
func (c *SkillsClient) ListSkillVersions(ctx context.Context, skillId string) (*SkillVersionListResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/v1/skills/"+skillId+"/versions", nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	if err := c.applyEditors(ctx, req); err != nil {
		return nil, fmt.Errorf("applying request editors: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(respBody))
	}

	var result SkillVersionListResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("unmarshaling response: %w", err)
	}

	return &result, nil
}

// DeleteSkillVersion deletes a specific version of a skill.
func (c *SkillsClient) DeleteSkillVersion(ctx context.Context, skillId, version string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, c.baseURL+"/v1/skills/"+skillId+"/versions/"+version, nil)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	if err := c.applyEditors(ctx, req); err != nil {
		return fmt.Errorf("applying request editors: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// DeleteSkill deletes a skill (all versions must be deleted first).
func (c *SkillsClient) DeleteSkill(ctx context.Context, skillId string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, c.baseURL+"/v1/skills/"+skillId, nil)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	if err := c.applyEditors(ctx, req); err != nil {
		return fmt.Errorf("applying request editors: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// DeleteSkillWithVersions deletes all versions of a skill and then the skill itself.
func (c *SkillsClient) DeleteSkillWithVersions(ctx context.Context, skillId string) error {
	versions, err := c.ListSkillVersions(ctx, skillId)
	if err != nil {
		return fmt.Errorf("listing versions: %w", err)
	}

	for _, v := range versions.Data {
		if err := c.DeleteSkillVersion(ctx, skillId, v.Version); err != nil {
			return fmt.Errorf("deleting version %s: %w", v.Version, err)
		}
	}

	return c.DeleteSkill(ctx, skillId)
}
