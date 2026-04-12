package client

import "fmt"

type Deployment struct {
	ID            string  `json:"id"`
	ProjectID     string  `json:"project_id"`
	CommitSHA     string  `json:"commit_sha"`
	CommitMessage *string `json:"commit_message"`
	Branch        string  `json:"branch"`
	Status        string  `json:"status"`
	IsProduction  bool    `json:"is_production"`
	DeploymentURL *string `json:"deployment_url"`
	ErrorMessage  *string `json:"error_message"`
	CreatedAt     string  `json:"created_at"`
}

type DeploymentListResponse struct {
	Deployments []Deployment `json:"deployments"`
}

type TriggerDeployRequest struct {
	Branch   string `json:"branch,omitempty"`
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

func (c *Client) GetDeployment(projectID, deployID string) (*Deployment, error) {
	var resp struct {
		Deployment Deployment `json:"deployment"`
	}
	err := c.Get(fmt.Sprintf("/api/v1/projects/%s/deployments/%s", projectID, deployID), &resp)
	if err != nil {
		return nil, fmt.Errorf("get deployment: %w", err)
	}
	return &resp.Deployment, nil
}

func (c *Client) TriggerDeploy(projectID string, req TriggerDeployRequest) (*Deployment, error) {
	var resp TriggerDeployResponse
	err := c.Post(fmt.Sprintf("/api/v1/projects/%s/deployments", projectID), req, &resp)
	if err != nil {
		return nil, fmt.Errorf("trigger deploy: %w", err)
	}
	return &resp.Deployment, nil
}

func (c *Client) Rollback(projectID string, req RollbackRequest) (*Deployment, error) {
	var resp TriggerDeployResponse
	err := c.Post(fmt.Sprintf("/api/v1/projects/%s/rollback", projectID), req, &resp)
	if err != nil {
		return nil, fmt.Errorf("rollback: %w", err)
	}
	return &resp.Deployment, nil
}
