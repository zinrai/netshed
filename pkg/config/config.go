package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Network struct {
	Name       string `yaml:"name"`
	Type       string `yaml:"type"`
	Subnet     string `yaml:"subnet,omitempty"`
	Gateway    string `yaml:"gateway,omitempty"`
	Address    string `yaml:"address,omitempty"`
	Masquerade bool   `yaml:"masquerade,omitempty"`
}

type Config struct {
	Networks []Network `yaml:"networks"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %v", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %v", err)
	}

	if err := validate(&config); err != nil {
		return nil, fmt.Errorf("invalid config: %v", err)
	}

	return &config, nil
}

func validate(config *Config) error {
	if len(config.Networks) == 0 {
		return fmt.Errorf("no networks defined")
	}

	for _, network := range config.Networks {
		if network.Name == "" {
			return fmt.Errorf("network name is required")
		}

		switch network.Type {
		case "bridge":
			if network.Subnet == "" {
				return fmt.Errorf("subnet is required for bridge network %s", network.Name)
			}
			if network.Gateway == "" {
				return fmt.Errorf("gateway is required for bridge network %s", network.Name)
			}
		case "dummy":
			if network.Address == "" {
				return fmt.Errorf("address is required for dummy network %s", network.Name)
			}
		default:
			return fmt.Errorf("invalid network type %s for network %s", network.Type, network.Name)
		}
	}

	return nil
}
