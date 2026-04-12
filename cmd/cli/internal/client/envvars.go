package client

import "fmt"

type EnvVar struct {
	ID        string `json:"id"`
	ProjectID string `json:"project_id"`
	Key       string `json:"key"`
	Value     string `json:"value"`
	CreatedAt string `json:"created_at"`
}

type EnvVarListResponse struct {
	EnvVars []EnvVar `json:"env_vars"`
}

type SetEnvVarRequest struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

func (c *Client) ListEnvVars(projectID string) (*EnvVarListResponse, error) {
	var resp EnvVarListResponse
	err := c.Get(fmt.Sprintf("/api/v1/projects/%s/env", projectID), &resp)
	if err != nil {
		return nil, fmt.Errorf("list env vars: %w", err)
	}
	return &resp, nil
}

func (c *Client) SetEnvVar(projectID string, key, value string) error {
	return c.Post(fmt.Sprintf("/api/v1/projects/%s/env", projectID), SetEnvVarRequest{Key: key, Value: value}, nil)
}

func (c *Client) DeleteEnvVar(projectID, envVarID string) error {
	return c.Delete(fmt.Sprintf("/api/v1/projects/%s/env/%s", projectID, envVarID), nil)
}
