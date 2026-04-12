package client

import "fmt"

type Project struct {
	ID              string  `json:"id"`
	Name            string  `json:"name"`
	Slug            string  `json:"slug"`
	Framework       *string `json:"framework"`
	GitRepo         *string `json:"git_repo"`
	ProductionBranch string `json:"production_branch"`
	CreatedAt       string  `json:"created_at"`
}

type ProjectListResponse struct {
	Projects []Project `json:"projects"`
}

type CreateProjectRequest struct {
	Name             string `json:"name"`
	Slug             string `json:"slug,omitempty"`
	GitRepo          string `json:"git_repo,omitempty"`
	ProductionBranch string `json:"production_branch,omitempty"`
	Framework        string `json:"framework,omitempty"`
	BuildCommand     string `json:"build_command,omitempty"`
	OutputDir        string `json:"output_dir,omitempty"`
	InstallCommand   string `json:"install_command,omitempty"`
}

type CreateProjectResponse struct {
	Project Project `json:"project"`
}

func (c *Client) ListProjects() (*ProjectListResponse, error) {
	var resp ProjectListResponse
	err := c.Get("/api/v1/projects", &resp)
	if err != nil {
		return nil, fmt.Errorf("list projects: %w", err)
	}
	return &resp, nil
}

func (c *Client) GetProject(id string) (*Project, error) {
	var resp struct {
		Project Project `json:"project"`
	}
	err := c.Get(fmt.Sprintf("/api/v1/projects/%s", id), &resp)
	if err != nil {
		return nil, fmt.Errorf("get project: %w", err)
	}
	return &resp.Project, nil
}

func (c *Client) CreateProject(req CreateProjectRequest) (*Project, error) {
	var resp CreateProjectResponse
	err := c.Post("/api/v1/projects", req, &resp)
	if err != nil {
		return nil, fmt.Errorf("create project: %w", err)
	}
	return &resp.Project, nil
}
