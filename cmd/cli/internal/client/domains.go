package client

import "fmt"

type Domain struct {
	ID         string `json:"id"`
	ProjectID  string `json:"project_id"`
	Domain     string `json:"domain"`
	Verified   bool   `json:"verified"`
	VerifiedAt string `json:"verified_at,omitempty"`
	CreatedAt  string `json:"created_at"`
}

type DomainListResponse struct {
	Domains []Domain `json:"domains"`
}

type AddDomainRequest struct {
	Domain string `json:"domain"`
}

func (c *Client) ListDomains(projectID string) (*DomainListResponse, error) {
	var resp DomainListResponse
	err := c.Get(fmt.Sprintf("/api/v1/projects/%s/domains", projectID), &resp)
	if err != nil {
		return nil, fmt.Errorf("list domains: %w", err)
	}
	return &resp, nil
}

func (c *Client) AddDomain(projectID string, domain string) (*Domain, error) {
	var resp struct {
		Domain Domain `json:"domain"`
	}
	err := c.Post(fmt.Sprintf("/api/v1/projects/%s/domains", projectID), AddDomainRequest{Domain: domain}, &resp)
	if err != nil {
		return nil, fmt.Errorf("add domain: %w", err)
	}
	return &resp.Domain, nil
}

func (c *Client) DeleteDomain(projectID, domainID string) error {
	err := c.Delete(fmt.Sprintf("/api/v1/projects/%s/domains/%s", projectID, domainID), nil)
	if err != nil {
		return fmt.Errorf("delete domain: %w", err)
	}
	return nil
}

func (c *Client) VerifyDomain(projectID, domainID string) (*Domain, error) {
	var resp struct {
		Domain Domain `json:"domain"`
	}
	err := c.Post(fmt.Sprintf("/api/v1/projects/%s/domains/%s/verify", projectID, domainID), nil, &resp)
	if err != nil {
		return nil, fmt.Errorf("verify domain: %w", err)
	}
	return &resp.Domain, nil
}
