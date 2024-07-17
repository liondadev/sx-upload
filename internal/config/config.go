package config

import (
	"encoding/json"
	"errors"
	"io"
	"os"
)

type User struct {
	MaxUploadSize int64 `json:"max_upload_size"` // in bytes
}

type Config struct {
	Users      map[string]User   `json:"users"`
	Keys       map[string]string `json:"keys"` // maps a unique user key to a user ID
	AdminToken string            `json:"admin_token"`
}

func FromFile(path string) (*Config, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	content, err := io.ReadAll(f)
	if err != nil {
		return nil, err
	}

	var config Config
	if err := json.Unmarshal(content, &config); err != nil {
		return nil, err
	}

	return &config, err
}

func (c *Config) UserFromKey(key string) (*User, error) {
	uid, ok := c.Keys[key]
	if !ok {
		return nil, errors.New("invalid key")
	}

	u, ok := c.Users[uid]
	if !ok {
		return nil, errors.New("key misconfiguration, invalid userid")
	}

	return &u, nil
}
