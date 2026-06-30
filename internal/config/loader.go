package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

func Load(path string) (*Config, error) {

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("cannot read config: %w", err)
	}

	var cfg Config

	err = yaml.Unmarshal(data, &cfg)
	if err != nil {
		return nil, fmt.Errorf("cannot parse yaml: %w", err)
	}

	return &cfg, nil
}
