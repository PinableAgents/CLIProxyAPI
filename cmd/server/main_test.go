package main

import (
	"testing"

	"github.com/router-for-me/CLIProxyAPI/v7/internal/config"
)

func TestVersionRequested(t *testing.T) {
	for _, args := range [][]string{{"--version"}, {"-version"}, {"--version", "--config", "ignored.yaml"}} {
		if !versionRequested(args) {
			t.Fatalf("versionRequested(%q) = false", args)
		}
	}
	for _, args := range [][]string{nil, {"--config", "config.yaml"}, {"--versioned"}} {
		if versionRequested(args) {
			t.Fatalf("versionRequested(%q) = true", args)
		}
	}
}

func TestShouldEnableExampleAPIKeySafeMode(t *testing.T) {
	cfgWithExampleKey := &config.Config{
		SDKConfig: config.SDKConfig{
			APIKeys: []string{"real-key", " your-api-key-1 "},
		},
	}
	cfgWithRealKey := &config.Config{
		SDKConfig: config.SDKConfig{
			APIKeys: []string{"real-key"},
		},
	}

	tests := []struct {
		name                string
		cfg                 *config.Config
		commandMode         bool
		tuiMode             bool
		standalone          bool
		cloudConfigMissing  bool
		homeMode            bool
		ephemeralConfigured bool
		want                bool
	}{
		{
			name: "normal server with example key",
			cfg:  cfgWithExampleKey,
			want: true,
		},
		{
			name:       "standalone tui with example key",
			cfg:        cfgWithExampleKey,
			tuiMode:    true,
			standalone: true,
			want:       true,
		},
		{
			name:        "pure tui client is not blocked",
			cfg:         cfgWithExampleKey,
			tuiMode:     true,
			standalone:  false,
			commandMode: false,
			want:        false,
		},
		{
			name:        "one-shot command is not blocked",
			cfg:         cfgWithExampleKey,
			commandMode: true,
			want:        false,
		},
		{
			name:     "home mode is not blocked",
			cfg:      cfgWithExampleKey,
			homeMode: true,
			want:     false,
		},
		{
			name:               "cloud standby without config is not blocked",
			cfg:                cfgWithExampleKey,
			cloudConfigMissing: true,
			want:               false,
		},
		{
			name:                "ephemeral key keeps server available with example config key",
			cfg:                 cfgWithExampleKey,
			ephemeralConfigured: true,
			want:                false,
		},
		{
			name: "normal server with real key",
			cfg:  cfgWithRealKey,
			want: false,
		},
		{
			name: "nil config",
			cfg:  nil,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := shouldEnableExampleAPIKeySafeMode(tt.cfg, tt.commandMode, tt.tuiMode, tt.standalone, tt.cloudConfigMissing, tt.homeMode, tt.ephemeralConfigured)
			if got != tt.want {
				t.Fatalf("shouldEnableExampleAPIKeySafeMode() = %t, want %t", got, tt.want)
			}
		})
	}
}

func TestModelCatalogUpdaterPlan(t *testing.T) {
	tests := []struct {
		name            string
		localModel      bool
		homeEnabled     bool
		wantModels      bool
		wantCodexClient bool
	}{
		{
			name:            "normal CPA refreshes both catalogs",
			localModel:      false,
			homeEnabled:     false,
			wantModels:      true,
			wantCodexClient: true,
		},
		{
			name:            "home mode keeps models.json local and refreshes codex templates",
			localModel:      false,
			homeEnabled:     true,
			wantModels:      false,
			wantCodexClient: true,
		},
		{
			name:            "local-model disables both remote catalogs",
			localModel:      true,
			homeEnabled:     false,
			wantModels:      false,
			wantCodexClient: false,
		},
		{
			name:            "local-model disables both remote catalogs even under home",
			localModel:      true,
			homeEnabled:     true,
			wantModels:      false,
			wantCodexClient: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotModels, gotCodex := modelCatalogUpdaterPlan(tt.localModel, tt.homeEnabled)
			if gotModels != tt.wantModels || gotCodex != tt.wantCodexClient {
				t.Fatalf("modelCatalogUpdaterPlan(%v, %v) = (%v, %v), want (%v, %v)",
					tt.localModel, tt.homeEnabled, gotModels, gotCodex, tt.wantModels, tt.wantCodexClient)
			}
		})
	}
}
