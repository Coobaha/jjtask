package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/pelletier/go-toml/v2"
	"gopkg.in/yaml.v3"
)

// Config represents .jjtask.toml
type Config struct {
	Workspaces WorkspacesConfig `toml:"workspaces"`
	Prime      PrimeConfig      `toml:"prime"`
}

// WorkspacesConfig holds multi-repo workspace configuration
type WorkspacesConfig struct {
	Repos []Repo `toml:"repos"`
}

// Repo represents a single repo in the config
type Repo struct {
	Path string `toml:"path" yaml:"path"`
	Name string `toml:"name" yaml:"name"`
}

// PrimeConfig holds prime output customization
type PrimeConfig struct {
	Content     string `toml:"content"`
	ContentFile string `toml:"content_file"`
}

var configRoot string
var loadedConfig *Config

// FindConfig locates .jjtask.toml or .jj-workspaces.yaml by traversing up from cwd
// Returns path to config file and root directory
func FindConfig() (cfgPath, root string, err error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", "", err
	}
	for dir != "/" {
		// Prefer .jjtask.toml
		tomlPath := filepath.Join(dir, ".jjtask.toml")
		if _, err := os.Stat(tomlPath); err == nil {
			return tomlPath, dir, nil
		}
		// Fall back to legacy .jj-workspaces.yaml
		yamlPath := filepath.Join(dir, ".jj-workspaces.yaml")
		if _, err := os.Stat(yamlPath); err == nil {
			return yamlPath, dir, nil
		}
		dir = filepath.Dir(dir)
	}
	return "", "", nil
}

// Load reads the config file, supporting both TOML and YAML formats
func Load() (*Config, string, error) {
	if loadedConfig != nil {
		return loadedConfig, configRoot, nil
	}

	cfgPath, root, err := FindConfig()
	if err != nil {
		return nil, "", err
	}
	if cfgPath == "" {
		return nil, "", nil
	}

	data, err := os.ReadFile(cfgPath)
	if err != nil {
		return nil, "", err
	}

	cfg := &Config{}

	if filepath.Ext(cfgPath) == ".yaml" {
		// Legacy YAML format - auto-migrate to TOML
		var yamlCfg struct {
			Repos []Repo `yaml:"repos"`
		}
		if err := yaml.Unmarshal(data, &yamlCfg); err != nil {
			return nil, "", err
		}
		cfg.Workspaces.Repos = yamlCfg.Repos

		// Migrate to .jjtask.toml
		if err := migrateYAMLToTOML(cfgPath, root, cfg); err != nil {
			fmt.Fprintf(os.Stderr, "warning: failed to migrate config: %v\n", err)
		}
	} else {
		if err := toml.Unmarshal(data, cfg); err != nil {
			return nil, "", err
		}
	}

	configRoot = root
	loadedConfig = cfg
	return cfg, root, nil
}

// GetRepos returns list of repo paths
func GetRepos() ([]Repo, string, error) {
	cfg, root, err := Load()
	if err != nil {
		return nil, "", err
	}
	if cfg == nil || len(cfg.Workspaces.Repos) == 0 {
		return []Repo{{Path: ".", Name: "workspace"}}, "", nil
	}
	return cfg.Workspaces.Repos, root, nil
}

// IsMultiRepo returns true if multi-repo config exists
func IsMultiRepo() bool {
	cfg, _, _ := Load()
	return cfg != nil && len(cfg.Workspaces.Repos) > 1
}

// GetPrimeContent returns custom prime content if configured
// Returns content string and bool indicating if custom content exists
func GetPrimeContent() (content string, hasCustom bool, err error) {
	cfg, root, err := Load()
	if err != nil {
		return "", false, err
	}
	if cfg == nil {
		return "", false, nil
	}

	// Inline content takes precedence
	if cfg.Prime.Content != "" {
		return cfg.Prime.Content, true, nil
	}

	// Content file path (relative to config root)
	if cfg.Prime.ContentFile != "" {
		filePath := cfg.Prime.ContentFile
		if !filepath.IsAbs(filePath) {
			filePath = filepath.Join(root, filePath)
		}
		data, err := os.ReadFile(filePath)
		if err != nil {
			return "", false, err
		}
		return string(data), true, nil
	}

	return "", false, nil
}

// Reset clears cached config (for testing)
func Reset() {
	loadedConfig = nil
	configRoot = ""
}

// migrateYAMLToTOML converts .jj-workspaces.yaml to .jjtask.toml
func migrateYAMLToTOML(yamlPath, root string, cfg *Config) error {
	tomlPath := filepath.Join(root, ".jjtask.toml")

	// Don't overwrite existing TOML
	if _, err := os.Stat(tomlPath); err == nil {
		return nil
	}

	// Generate clean inline array syntax
	var content string
	content = "[workspaces]\nrepos = [\n"
	for _, repo := range cfg.Workspaces.Repos {
		content += fmt.Sprintf("  { path = %q, name = %q },\n", repo.Path, repo.Name)
	}
	content += "]\n"

	if err := os.WriteFile(tomlPath, []byte(content), 0o644); err != nil {
		return err
	}

	// Remove old YAML file
	if err := os.Remove(yamlPath); err != nil {
		fmt.Fprintf(os.Stderr, "warning: could not remove %s: %v\n", yamlPath, err)
	}

	fmt.Fprintf(os.Stderr, "migrated %s → %s\n", filepath.Base(yamlPath), filepath.Base(tomlPath))
	return nil
}
