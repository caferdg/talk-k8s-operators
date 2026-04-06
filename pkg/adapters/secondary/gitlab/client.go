package gitlab

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	neturl "net/url"
	"strings"

	"github.com/caferdg/talk-k8s-operators/pkg/domain/entity"
)

const baseURL = "https://gitlab.com/api/v4"

type Client struct {
	token      string
	httpClient *http.Client
}

func NewClient(token string) *Client {
	return &Client{
		token:      token,
		httpClient: &http.Client{},
	}
}

func (c *Client) GetGroup(ctx context.Context, groupID int) (*entity.Group, error) {
	// Fetch group details (includes projects via shared_projects)
	url := fmt.Sprintf("%s/groups/%d", baseURL, groupID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("PRIVATE-TOKEN", c.token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("gitlab API returned %d", resp.StatusCode)
	}

	var result struct {
		ID          int    `json:"id"`
		FullPath    string `json:"full_path"`
		WebURL      string `json:"web_url"`
		Description string `json:"description"`
		Visibility  string `json:"visibility"`
		Projects    []struct {
			Name string `json:"name"`
		} `json:"projects"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	projects := make([]string, len(result.Projects))
	for i, p := range result.Projects {
		projects[i] = p.Name
	}

	// Fetch subgroups via dedicated endpoint
	subgroups, err := c.getNames(ctx, fmt.Sprintf("%s/groups/%d/subgroups", baseURL, groupID))
	if err != nil {
		return nil, fmt.Errorf("fetching subgroups: %w", err)
	}

	// Fetch members via dedicated endpoint
	members, err := c.getNames(ctx, fmt.Sprintf("%s/groups/%d/members", baseURL, groupID))
	if err != nil {
		return nil, fmt.Errorf("fetching members: %w", err)
	}

	return &entity.Group{
		ID:          result.ID,
		FullPath:    result.FullPath,
		WebURL:      result.WebURL,
		Description: result.Description,
		Visibility:  result.Visibility,
		Projects:    projects,
		Subgroups:   subgroups,
		MemberCount: len(members),
	}, nil
}

func (c *Client) getNames(ctx context.Context, url string) ([]string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("PRIVATE-TOKEN", c.token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("gitlab API returned %d", resp.StatusCode)
	}

	var items []struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&items); err != nil {
		return nil, err
	}

	names := make([]string, len(items))
	for i, item := range items {
		names[i] = item.Name
	}
	return names, nil
}

func (c *Client) GetProject(ctx context.Context, groupID int, projectName string) (*entity.Project, error) {
	url := fmt.Sprintf("%s/groups/%d/projects?search=%s", baseURL, groupID, neturl.QueryEscape(projectName))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("PRIVATE-TOKEN", c.token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("gitlab API returned %d", resp.StatusCode)
	}

	var results []struct {
		ID                int    `json:"id"`
		Name              string `json:"name"`
		PathWithNamespace string `json:"path_with_namespace"`
		Description       string `json:"description"`
		Visibility        string `json:"visibility"`
		WebURL            string `json:"web_url"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		return nil, err
	}

	for _, r := range results {
		if strings.EqualFold(r.Name, projectName) {
			return &entity.Project{
				ID:          r.ID,
				Name:        r.Name,
				FullPath:    r.PathWithNamespace,
				Description: r.Description,
				Visibility:  r.Visibility,
				WebURL:      r.WebURL,
			}, nil
		}
	}

	return nil, nil
}

func (c *Client) CreateProject(
	ctx context.Context, groupID int, name, description, visibility string,
) (*entity.Project, error) {
	url := fmt.Sprintf("%s/projects", baseURL)

	body := fmt.Sprintf(`{"name":%q,"namespace_id":%d,"description":%q,"visibility":%q}`,
		name, groupID, description, visibility)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, strings.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("PRIVATE-TOKEN", c.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("gitlab API returned %d", resp.StatusCode)
	}

	var result struct {
		ID                int    `json:"id"`
		Name              string `json:"name"`
		PathWithNamespace string `json:"path_with_namespace"`
		Description       string `json:"description"`
		Visibility        string `json:"visibility"`
		WebURL            string `json:"web_url"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &entity.Project{
		ID:          result.ID,
		Name:        result.Name,
		FullPath:    result.PathWithNamespace,
		Description: result.Description,
		Visibility:  result.Visibility,
		WebURL:      result.WebURL,
	}, nil
}

func (c *Client) UpdateProject(
	ctx context.Context, projectID int, description, visibility string,
) (*entity.Project, error) {
	url := fmt.Sprintf("%s/projects/%d", baseURL, projectID)

	body := fmt.Sprintf(`{"description":%q,"visibility":%q}`, description, visibility)

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, url, strings.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("PRIVATE-TOKEN", c.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("gitlab API returned %d", resp.StatusCode)
	}

	var result struct {
		ID                int    `json:"id"`
		Name              string `json:"name"`
		PathWithNamespace string `json:"path_with_namespace"`
		Description       string `json:"description"`
		Visibility        string `json:"visibility"`
		WebURL            string `json:"web_url"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &entity.Project{
		ID:          result.ID,
		Name:        result.Name,
		FullPath:    result.PathWithNamespace,
		Description: result.Description,
		Visibility:  result.Visibility,
		WebURL:      result.WebURL,
	}, nil
}

const stampTopic = "managed-by-scm-operator"

// Stamp marks a GitLab project as managed by the operator.
func (c *Client) Stamp(ctx context.Context, projectID int) error {
	return c.setTopics(ctx, projectID, fmt.Sprintf(`{"topics":[%q]}`, stampTopic))
}

// Unstamp removes the managed marker from a GitLab project.
func (c *Client) Unstamp(ctx context.Context, projectID int) error {
	return c.setTopics(ctx, projectID, `{"topics":[]}`)
}

func (c *Client) setTopics(ctx context.Context, projectID int, body string) error {
	url := fmt.Sprintf("%s/projects/%d", baseURL, projectID)

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, url, strings.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("PRIVATE-TOKEN", c.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("gitlab API returned %d", resp.StatusCode)
	}
	return nil
}

func (c *Client) GetBranchCommit(ctx context.Context, projectID int, branch string) (*entity.Commit, error) {
	url := fmt.Sprintf("%s/projects/%d/repository/branches/%s", baseURL, projectID, neturl.PathEscape(branch))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("PRIVATE-TOKEN", c.token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("gitlab API returned %d", resp.StatusCode)
	}

	var result struct {
		Commit struct {
			ID        string `json:"id"`
			Message   string `json:"message"`
			Author    string `json:"author_name"`
			CreatedAt string `json:"created_at"`
		} `json:"commit"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &entity.Commit{
		SHA:       result.Commit.ID,
		Message:   result.Commit.Message,
		Author:    result.Commit.Author,
		CreatedAt: result.Commit.CreatedAt,
	}, nil
}

func (c *Client) GetLastPipeline(ctx context.Context, projectID int, branch string) (*entity.Pipeline, error) {
	url := fmt.Sprintf("%s/projects/%d/pipelines?ref=%s&per_page=1&order_by=id&sort=desc",
		baseURL, projectID, neturl.QueryEscape(branch))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("PRIVATE-TOKEN", c.token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("gitlab API returned %d", resp.StatusCode)
	}

	var results []struct {
		ID        int    `json:"id"`
		Status    string `json:"status"`
		WebURL    string `json:"web_url"`
		CreatedAt string `json:"created_at"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		return nil, err
	}

	if len(results) == 0 {
		return nil, nil
	}

	return &entity.Pipeline{
		ID:        results[0].ID,
		Status:    results[0].Status,
		WebURL:    results[0].WebURL,
		CreatedAt: results[0].CreatedAt,
	}, nil
}
