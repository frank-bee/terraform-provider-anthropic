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
	"sort"
)

// Skill represents a skill resource from the API.
type Skill struct {
	Type          string `json:"type"`
	Id            string `json:"id"`
	CreatedAt     string `json:"created_at"`
	UpdatedAt     string `json:"updated_at"`
	DisplayTitle  string `json:"display_title"`
	Source        string `json:"source"`
	LatestVersion string `json:"latest_version"`
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

// buildSkillZip creates a zip with the given files placed under the
// `skillName/` folder. files maps relative paths (e.g. "SKILL.md",
// "scripts/helper.sh") to file content. files MUST contain "SKILL.md".
func buildSkillZip(skillName string, files map[string][]byte) ([]byte, error) {
	if _, ok := files["SKILL.md"]; !ok {
		return nil, fmt.Errorf("files map must contain SKILL.md")
	}

	var zipBuf bytes.Buffer
	zipWriter := zip.NewWriter(&zipBuf)

	// Sort keys so zip output is deterministic (helps caching/idempotency).
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		f, err := zipWriter.Create(skillName + "/" + name)
		if err != nil {
			return nil, fmt.Errorf("creating zip entry %s: %w", name, err)
		}
		if _, err := f.Write(files[name]); err != nil {
			return nil, fmt.Errorf("writing zip entry %s: %w", name, err)
		}
	}
	if err := zipWriter.Close(); err != nil {
		return nil, fmt.Errorf("closing zip: %w", err)
	}
	return zipBuf.Bytes(), nil
}

// CreateSkillFromFiles uploads a skill consisting of one or more files.
// files MUST contain "SKILL.md"; additional files (scripts, references, etc.)
// are placed alongside it under the skill's folder.
func (c *SkillsClient) CreateSkillFromFiles(ctx context.Context, displayTitle, skillName string, files map[string][]byte) (*Skill, error) {
	zipBytes, err := buildSkillZip(skillName, files)
	if err != nil {
		return nil, err
	}

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	if err := writer.WriteField("display_title", displayTitle); err != nil {
		return nil, fmt.Errorf("writing display_title field: %w", err)
	}

	part, err := writer.CreateFormFile("files[]", skillName+".zip")
	if err != nil {
		return nil, fmt.Errorf("creating form file: %w", err)
	}
	if _, err := part.Write(zipBytes); err != nil {
		return nil, fmt.Errorf("writing zip to form: %w", err)
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

// CreateSkill creates a single-file skill from a string SKILL.md content.
// Convenience wrapper around CreateSkillFromFiles.
func (c *SkillsClient) CreateSkill(ctx context.Context, displayTitle, skillName, skillContent string) (*Skill, error) {
	return c.CreateSkillFromFiles(ctx, displayTitle, skillName, map[string][]byte{
		"SKILL.md": []byte(skillContent),
	})
}

// CreateSkillVersionFromFiles uploads a new version of an existing skill
// from one or more files. The skill keeps its ID; only the version (and
// contents) advance.
func (c *SkillsClient) CreateSkillVersionFromFiles(ctx context.Context, skillId, skillName string, files map[string][]byte) (*SkillVersion, error) {
	zipBytes, err := buildSkillZip(skillName, files)
	if err != nil {
		return nil, err
	}

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("files[]", skillName+".zip")
	if err != nil {
		return nil, fmt.Errorf("creating form file: %w", err)
	}
	if _, err := part.Write(zipBytes); err != nil {
		return nil, fmt.Errorf("writing zip to form: %w", err)
	}
	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("closing multipart writer: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v1/skills/"+skillId+"/versions", &body)
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

	var version SkillVersion
	if err := json.Unmarshal(respBody, &version); err != nil {
		return nil, fmt.Errorf("unmarshaling response: %w", err)
	}

	return &version, nil
}

// CreateSkillVersion is a convenience wrapper for single-file SKILL.md uploads.
func (c *SkillsClient) CreateSkillVersion(ctx context.Context, skillId, skillName, skillContent string) (*SkillVersion, error) {
	return c.CreateSkillVersionFromFiles(ctx, skillId, skillName, map[string][]byte{
		"SKILL.md": []byte(skillContent),
	})
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
