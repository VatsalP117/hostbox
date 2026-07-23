package client

import "fmt"

type Deployment struct {
	ID                         string  `json:"id"`
	ProjectID                  string  `json:"project_id"`
	CommitSHA                  string  `json:"commit_sha"`
	CommitMessage              *string `json:"commit_message"`
	Branch                     string  `json:"branch"`
	Status                     string  `json:"status"`
	IsProduction               bool    `json:"is_production"`
	DeploymentURL              *string `json:"deployment_url"`
	ErrorMessage               *string `json:"error_message"`
	CreatedAt                  string  `json:"created_at"`
	BuildFramework             *string `json:"build_framework"`
	BuildServingMode           *string `json:"build_serving_mode"`
	BuildPackageManager        *string `json:"build_package_manager"`
	BuildPackageManagerVersion *string `json:"build_package_manager_version"`
	BuildNodeVersion           string  `json:"build_node_version"`
	BuildRootDirectory         string  `json:"build_root_directory"`
	BuildOutputDirectory       *string `json:"build_output_directory"`
	BuildInstallCommand        *string `json:"build_install_command"`
	BuildCommand               *string `json:"build_command"`
	BuildLockFileHash          string  `json:"build_lock_file_hash"`
	BuildManifestResolved      bool    `json:"build_manifest_resolved"`
	SourceRepository           *string `json:"source_repository"`
	SourceInstallationID       *int64  `json:"source_installation_id"`
}

func (c *Client) RebuildDeployment(deployID string) (*Deployment, error) {
	var resp TriggerDeployResponse
	err := c.Post(fmt.Sprintf("/api/v1/deployments/%s/rebuild", deployID), nil, &resp)
	if err != nil {
		return nil, fmt.Errorf("rebuild deployment: %w", err)
	}
	return &resp.Deployment, nil
}

func (c *Client) DeployLatest(projectID, branch string) (*Deployment, error) {
	var resp TriggerDeployResponse
	req := struct {
		Branch string `json:"branch,omitempty"`
	}{Branch: branch}
	err := c.Post(fmt.Sprintf("/api/v1/projects/%s/deploy-latest", projectID), req, &resp)
	if err != nil {
		return nil, fmt.Errorf("deploy latest branch: %w", err)
	}
	return &resp.Deployment, nil
}

type DeploymentListResponse struct {
	Deployments []Deployment `json:"deployments"`
}

type DeploymentLogs struct {
	Lines      []string `json:"lines"`
	TotalLines int      `json:"total_lines"`
	HasMore    bool     `json:"has_more"`
}

type TriggerDeployRequest struct {
	Branch    string `json:"branch,omitempty"`
	CommitSHA string `json:"commit_sha,omitempty"`
}

type TriggerDeployResponse struct {
	Deployment Deployment `json:"deployment"`
}

type RollbackRequest struct {
	DeploymentID string `json:"deployment_id"`
}

func (c *Client) ListDeployments(projectID string) (*DeploymentListResponse, error) {
	var resp DeploymentListResponse
	err := c.Get(fmt.Sprintf("/api/v1/projects/%s/deployments", projectID), &resp)
	if err != nil {
		return nil, fmt.Errorf("list deployments: %w", err)
	}
	return &resp, nil
}

func (c *Client) GetDeployment(deployID string) (*Deployment, error) {
	var resp struct {
		Deployment Deployment `json:"deployment"`
	}
	err := c.Get(fmt.Sprintf("/api/v1/deployments/%s", deployID), &resp)
	if err != nil {
		return nil, fmt.Errorf("get deployment: %w", err)
	}
	return &resp.Deployment, nil
}

func (c *Client) TriggerDeploy(projectID string, req TriggerDeployRequest) (*Deployment, error) {
	var resp TriggerDeployResponse
	err := c.Post(fmt.Sprintf("/api/v1/projects/%s/deployments/trigger", projectID), req, &resp)
	if err != nil {
		return nil, fmt.Errorf("trigger deploy: %w", err)
	}
	return &resp.Deployment, nil
}

func (c *Client) GetDeploymentLogs(deployID string) (*DeploymentLogs, error) {
	var resp DeploymentLogs
	err := c.Get(fmt.Sprintf("/api/v1/deployments/%s/logs", deployID), &resp)
	if err != nil {
		return nil, fmt.Errorf("get deployment logs: %w", err)
	}
	return &resp, nil
}

func (c *Client) Rollback(projectID string, req RollbackRequest) (*Deployment, error) {
	var resp TriggerDeployResponse
	err := c.Post(fmt.Sprintf("/api/v1/projects/%s/rollback", projectID), req, &resp)
	if err != nil {
		return nil, fmt.Errorf("rollback: %w", err)
	}
	return &resp.Deployment, nil
}
