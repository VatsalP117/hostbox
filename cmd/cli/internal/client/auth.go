package client

import "fmt"

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

type WhoAmIResponse struct {
	ID          string `json:"id"`
	Email       string `json:"email"`
	DisplayName string `json:"display_name"`
	IsAdmin     bool   `json:"is_admin"`
}

func (c *Client) Login(email, password string) (*LoginResponse, error) {
	var resp LoginResponse
	err := c.Post("/api/v1/auth/login", LoginRequest{Email: email, Password: password}, &resp)
	if err != nil {
		return nil, fmt.Errorf("login failed: %w", err)
	}
	return &resp, nil
}

func (c *Client) WhoAmI() (*WhoAmIResponse, error) {
	var resp WhoAmIResponse
	err := c.Get("/api/v1/auth/me", &resp)
	if err != nil {
		return nil, fmt.Errorf("whoami failed: %w", err)
	}
	return &resp, nil
}
