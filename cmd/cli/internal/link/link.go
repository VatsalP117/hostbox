package link

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

const linkFileName = ".hostbox.json"

type LinkConfig struct {
	ProjectID string `json:"project_id"`
	ServerURL string `json:"server_url,omitempty"`
}

// Discover walks up from the current directory looking for .hostbox.json.
func Discover() (*LinkConfig, error) {
	dir, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	for {
		path := filepath.Join(dir, linkFileName)
		data, err := os.ReadFile(path)
		if err == nil {
			var cfg LinkConfig
			if err := json.Unmarshal(data, &cfg); err != nil {
				return nil, err
			}
			return &cfg, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	return nil, errors.New("not linked to a project — run 'hostbox link' first")
}

// Save writes a .hostbox.json file in the current directory.
func Save(projectID, serverURL string) error {
	cfg := LinkConfig{
		ProjectID: projectID,
		ServerURL: serverURL,
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(linkFileName, append(data, '\n'), 0644)
}
