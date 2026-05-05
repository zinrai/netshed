package main

import (
	"fmt"
	"os"

	"github.com/goccy/go-yaml"
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

func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %v", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %v", err)
	}

	if err := validateConfig(&config); err != nil {
		return nil, fmt.Errorf("invalid config: %v", err)
	}

	return &config, nil
}

func validateConfig(config *Config) error {
	if len(config.Networks) == 0 {
		return fmt.Errorf("no networks defined")
	}
	for _, n := range config.Networks {
		if err := validateNetwork(n); err != nil {
			return err
		}
	}
	return nil
}

func validateNetwork(n Network) error {
	if n.Name == "" {
		return fmt.Errorf("network name is required")
	}
	switch n.Type {
	case "bridge":
		return validateBridge(n)
	case "dummy":
		return validateDummy(n)
	default:
		return fmt.Errorf("invalid network type %s for network %s", n.Type, n.Name)
	}
}

func validateBridge(n Network) error {
	if n.Subnet == "" {
		return fmt.Errorf("subnet is required for bridge network %s", n.Name)
	}
	if n.Gateway == "" {
		return fmt.Errorf("gateway is required for bridge network %s", n.Name)
	}
	return nil
}

func validateDummy(n Network) error {
	if n.Address == "" {
		return fmt.Errorf("address is required for dummy network %s", n.Name)
	}
	return nil
}
