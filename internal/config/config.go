package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/omnitrix-sh/core.sh/pkg/models"
)

var globalConfig *models.Config


func Load(workDir string) (*models.Config, error) {
	if globalConfig != nil {
		return globalConfig, nil
	}

	
	cfg, err := loadFromFile(filepath.Join(workDir, ".omnitrix.json"))
	if err == nil {
		globalConfig = cfg
		globalConfig.WorkDir = workDir
		return globalConfig, nil
	}

	homeDir, err := os.UserHomeDir()
	if err == nil {
		cfg, err = loadFromFile(filepath.Join(homeDir, ".config", "omnitrix", "config.json"))
		if err == nil {
			globalConfig = cfg
			globalConfig.WorkDir = workDir
			return globalConfig, nil
		}
	}

	
	globalConfig = defaultConfig()
	globalConfig.WorkDir = workDir
	return globalConfig, nil
}

func loadFromFile(path string) (*models.Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg models.Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	if cfg.DataDir != "" {
		cfg.DataDir = expandHome(cfg.DataDir)
	}

	return &cfg, nil
}

func defaultConfig() *models.Config {
	homeDir, _ := os.UserHomeDir()
	dataDir := filepath.Join(homeDir, ".local", "share", "omnitrix")

	return &models.Config{
		DataDir: dataDir,
		Providers: map[models.ProviderType]models.ProviderConfig{
			models.ProviderOllama: {
				Enabled: true,
				BaseURL: "http://localhost:11434",
				Models:  []string{"deepseek-coder:6.7b"},
			},
		},
		DefaultModel: "deepseek-coder:6.7b",
		ContextPaths: []string{
			".cursorrules",
			".github/copilot-instructions.md",
			"omnitrix.md",
		},
		LSP:   make(map[string]models.LSPConfig),
		Debug: false,
	}
}

func expandHome(path string) string {
	if len(path) > 0 && path[0] == '~' {
		homeDir, err := os.UserHomeDir()
		if err == nil {
			return filepath.Join(homeDir, path[1:])
		}
	}
	return path
}


func Get() *models.Config {
	return globalConfig
}
