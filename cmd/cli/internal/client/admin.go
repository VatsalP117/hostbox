package client

type BackupResult struct {
	Filename  string `json:"filename"`
	Path      string `json:"path"`
	SizeBytes int64  `json:"size_bytes"`
}

type BackupListResponse struct {
	Backups []BackupResult `json:"backups"`
}

func (c *Client) CreateBackup(compress bool) (*BackupResult, error) {
	var resp struct {
		Backup BackupResult `json:"backup"`
	}
	err := c.Post("/api/v1/admin/backup", map[string]bool{"compress": compress}, &resp)
	if err != nil {
		return nil, err
	}
	return &resp.Backup, nil
}

func (c *Client) ListBackups() (*BackupListResponse, error) {
	var resp BackupListResponse
	err := c.Get("/api/v1/admin/backups", &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) RestoreBackup(path string) error {
	return c.Post("/api/v1/admin/restore", map[string]string{"path": path}, nil)
}
