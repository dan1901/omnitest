package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoad(t *testing.T) {
	tests := []struct {
		name    string
		file    string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid sample config",
			file:    "sample.yaml",
			wantErr: false,
		},
		{
			name:    "valid minimal config",
			file:    "minimal.yaml",
			wantErr: false,
		},
		{
			name:    "valid full config",
			file:    "full.yaml",
			wantErr: false,
		},
		{
			name:    "invalid yaml format",
			file:    "invalid.yaml",
			wantErr: true,
		},
		{
			name:    "nonexistent file",
			file:    "nonexistent.yaml",
			wantErr: true,
			errMsg:  "failed to read scenario file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join("..", "..", "testdata", tt.file)
			cfg, err := Load(path)

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error but got nil")
				}
				if tt.errMsg != "" && err != nil {
					if !strings.Contains(err.Error(), tt.errMsg) {
						t.Errorf("error = %q, want substring %q", err.Error(), tt.errMsg)
					}
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if cfg == nil {
				t.Fatal("expected config but got nil")
			}
			if len(cfg.Targets) == 0 {
				t.Error("expected at least one target")
			}
			if len(cfg.Scenarios) == 0 {
				t.Error("expected at least one scenario")
			}
		})
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		file    string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "missing targets",
			file:    "no_targets.yaml",
			wantErr: true,
			errMsg:  "at least one target is required",
		},
		{
			name:    "missing scenarios",
			file:    "no_scenarios.yaml",
			wantErr: true,
			errMsg:  "at least one scenario is required",
		},
		{
			name:    "missing base_url",
			file:    "missing_url.yaml",
			wantErr: true,
			errMsg:  "base_url is required",
		},
		{
			name:    "zero vusers",
			file:    "zero_vusers.yaml",
			wantErr: true,
			errMsg:  "vusers must be > 0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join("..", "..", "testdata", tt.file)
			_, err := Load(path)

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error but got nil")
				}
				if tt.errMsg != "" && err != nil {
					if !strings.Contains(err.Error(), tt.errMsg) {
						t.Errorf("error = %q, want substring %q", err.Error(), tt.errMsg)
					}
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestExpandEnvVars(t *testing.T) {
	t.Setenv("TEST_TOKEN", "my-secret-token")

	got := expandEnvVars("Bearer ${TEST_TOKEN}")
	want := "Bearer my-secret-token"
	if got != want {
		t.Errorf("expandEnvVars() = %q, want %q", got, want)
	}
}

func TestExpandEnvVars_Missing(t *testing.T) {
	os.Unsetenv("NONEXISTENT_VAR")

	got := expandEnvVars("${NONEXISTENT_VAR}")
	want := "${NONEXISTENT_VAR}"
	if got != want {
		t.Errorf("expandEnvVars() = %q, want %q (should keep original)", got, want)
	}
}
