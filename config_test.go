package main

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestLoadConfig_RejectsInvalidConfig(t *testing.T) {
	tests := []struct {
		name        string
		yaml        string
		errContains string
	}{
		{
			name: "bridge missing subnet",
			yaml: `
networks:
  - name: "vm0"
    type: "bridge"
    gateway: "192.168.100.1/24"
`,
			errContains: "subnet is required",
		},
		{
			name: "bridge missing gateway",
			yaml: `
networks:
  - name: "vm0"
    type: "bridge"
    subnet: "192.168.100.0/24"
`,
			errContains: "gateway is required",
		},
		{
			name: "bridge missing both subnet and gateway",
			yaml: `
networks:
  - name: "vm0"
    type: "bridge"
`,
			errContains: "is required",
		},
		{
			name: "dummy missing address",
			yaml: `
networks:
  - name: "dummy0"
    type: "dummy"
`,
			errContains: "address is required",
		},
		{
			name: "unknown type",
			yaml: `
networks:
  - name: "x0"
    type: "vlan"
    address: "10.0.0.1/24"
`,
			errContains: "invalid network type",
		},
		{
			name: "type field omitted",
			yaml: `
networks:
  - name: "x0"
    address: "10.0.0.1/24"
`,
			errContains: "invalid network type",
		},
		{
			name: "empty name",
			yaml: `
networks:
  - name: ""
    type: "bridge"
    subnet: "192.168.100.0/24"
    gateway: "192.168.100.1/24"
`,
			errContains: "network name is required",
		},
		{
			name: "no networks defined (empty list)",
			yaml: `
networks: []
`,
			errContains: "no networks defined",
		},
		{
			name:        "no networks defined (key absent)",
			yaml:        ``,
			errContains: "no networks defined",
		},
		{
			name: "second entry invalid",
			yaml: `
networks:
  - name: "vm0"
    type: "bridge"
    subnet: "192.168.100.0/24"
    gateway: "192.168.100.1/24"
  - name: "dummy0"
    type: "dummy"
`,
			errContains: "address is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := writeTempYAML(t, tt.yaml)
			_, err := LoadConfig(path)
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tt.errContains)
			}
			if !strings.Contains(err.Error(), tt.errContains) {
				t.Fatalf("expected error containing %q, got %q", tt.errContains, err.Error())
			}
		})
	}
}

func TestLoadConfig_AcceptsValidConfig(t *testing.T) {
	tests := []struct {
		name string
		yaml string
		want *Config
	}{
		{
			name: "bridge with masquerade",
			yaml: `
networks:
  - name: "vm0"
    type: "bridge"
    subnet: "192.168.100.0/24"
    gateway: "192.168.100.1/24"
    masquerade: true
`,
			want: &Config{Networks: []Network{{
				Name:       "vm0",
				Type:       "bridge",
				Subnet:     "192.168.100.0/24",
				Gateway:    "192.168.100.1/24",
				Masquerade: true,
			}}},
		},
		{
			name: "bridge without masquerade",
			yaml: `
networks:
  - name: "internal0"
    type: "bridge"
    subnet: "192.168.200.0/24"
    gateway: "192.168.200.1/24"
`,
			want: &Config{Networks: []Network{{
				Name:    "internal0",
				Type:    "bridge",
				Subnet:  "192.168.200.0/24",
				Gateway: "192.168.200.1/24",
			}}},
		},
		{
			name: "dummy",
			yaml: `
networks:
  - name: "dummy0"
    type: "dummy"
    address: "10.0.0.1/24"
`,
			want: &Config{Networks: []Network{{
				Name:    "dummy0",
				Type:    "dummy",
				Address: "10.0.0.1/24",
			}}},
		},
		{
			name: "multiple networks of mixed types",
			yaml: `
networks:
  - name: "vm0"
    type: "bridge"
    subnet: "192.168.100.0/24"
    gateway: "192.168.100.1/24"
    masquerade: true
  - name: "dummy0"
    type: "dummy"
    address: "10.0.0.1/24"
`,
			want: &Config{Networks: []Network{
				{Name: "vm0", Type: "bridge", Subnet: "192.168.100.0/24", Gateway: "192.168.100.1/24", Masquerade: true},
				{Name: "dummy0", Type: "dummy", Address: "10.0.0.1/24"},
			}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := writeTempYAML(t, tt.yaml)
			got, err := LoadConfig(path)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("got %+v, want %+v", got, tt.want)
			}
		})
	}
}

func writeTempYAML(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write temp config: %v", err)
	}
	return path
}
