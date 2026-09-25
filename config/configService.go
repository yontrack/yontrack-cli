package config

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v2"
)

// Root configuration
type RootConfig struct {
	// Default configuration name
	Selected string
	// List of configurations
	Configurations []Config
}

// Configuration content
type Config struct {
	// Name of the configuration
	Name string
	// URL of the remote server
	URL string
	// Username for the remote server (when using basic authentication)
	Username string
	// Password for the remote server (when using basic authentication)
	Password string
	// Token for the remote server (when using token-based authentication)
	Token string
	// Is this configuration disabled?
	Disabled bool
	// Connection retry configuration
	ConnectionRetry `yaml:"connectionRetry"`
}

type ConnectionRetry struct {
	MaxWaitTimeSec int `yaml:"maxWaitTimeSec"`
	MaxCount       int `yaml:"maxcount"`
}

// Gets the current configuration
func GetSelectedConfiguration() (*Config, error) {
	root, err := ReadRootConfiguration()
	if err != nil {
		return nil, err
	}
	selected := root.Selected
	if selected != "" {
		for _, item := range root.Configurations {
			if item.Name == selected {
				return &item, nil
			}
		}
		return nil, fmt.Errorf("No configuration named %s", selected)
	}
	return nil, fmt.Errorf("No current configuration (in %s)", getConfigFilePath())
}

// Reads the configuration
func ReadRootConfiguration() (*RootConfig, error) {
	var root RootConfig
	configFilePath := getConfigFilePath()

	// If the config file does not exist, returns an empty root config
	if _, err := os.Stat(configFilePath); err != nil {
		if os.IsNotExist(err) {
			return &root, nil
		}
	}

	reader, _ := os.Open(configFilePath)
	buf, _ := io.ReadAll(reader)
	err := yaml.Unmarshal(buf, &root)
	return &root, err
}

// Adds a new configuration and set as default
func AddConfiguration(config Config, override bool) error {
	root, err := ReadRootConfiguration()
	if err != nil {
		return err
	}
	configurations := root.Configurations
	existing := false
	// Check if the configuration name already exists
	for index, item := range configurations {
		if item.Name == config.Name {
			if override {
				configurations[index] = config
				existing = true
			} else {
				return fmt.Errorf("Configuration with name %s already exists", config.Name)
			}
		}
	}
	// Default selected configuration is the added one
	// Adds the configuration to the list if not existing already

	if !existing {
		configurations = append(root.Configurations, config)
	}
	newRoot := RootConfig{
		Selected:       config.Name,
		Configurations: configurations,
	}
	// Saves the root configuration back
	configFilePath := getConfigFilePath()
	buf, _ := yaml.Marshal(newRoot)
	_, _ = os.OpenFile(configFilePath, os.O_CREATE|os.O_WRONLY, 0600)
	_ = ioutil.WriteFile(configFilePath, buf, 0600)

	// OK
	return nil
}

// Finds an existing configuration
func findConfigurationByName(root *RootConfig, name string) *Config {
	for _, item := range root.Configurations {
		if item.Name == name {
			return &item
		}
	}
	return nil
}

// Replacing an existing configuration
func replaceConfigurationByName(root *RootConfig, config *Config) {
	for index, item := range root.Configurations {
		if item.Name == config.Name {
			root.Configurations[index] = *config
		}
	}
}

// Sets the new selected configuration
func SetSelectedConfiguration(name string) error {
	root, err := ReadRootConfiguration()
	if err != nil {
		return err
	}
	existing := findConfigurationByName(root, name)
	if existing == nil {
		return fmt.Errorf("Configuration with name %s does not exist", name)
	}
	newRoot := RootConfig{
		Selected:       name,
		Configurations: root.Configurations,
	}
	// Saves the root configuration back
	configFilePath := getConfigFilePath()
	buf, _ := yaml.Marshal(newRoot)
	_, _ = os.OpenFile(configFilePath, os.O_CREATE|os.O_WRONLY, 0600)
	_ = os.WriteFile(configFilePath, buf, 0600)

	// OK
	return nil
}

// Disables or enabled a configuration
func SetConfigurationState(name string, disabled bool) error {
	root, err := ReadRootConfiguration()
	if err != nil {
		return err
	}
	existing := findConfigurationByName(root, name)
	if existing == nil {
		return fmt.Errorf("Configuration with name %s does not exist", name)
	}
	// Adjust the existing configuration
	existing.Disabled = disabled
	replaceConfigurationByName(root, existing)
	// Saves the root configuration back
	configFilePath := getConfigFilePath()
	buf, _ := yaml.Marshal(root)
	_, _ = os.OpenFile(configFilePath, os.O_CREATE|os.O_WRONLY, 0600)
	_ = os.WriteFile(configFilePath, buf, 0600)

	// OK
	return nil
}

// Deletes an existing configuration
func DeleteConfiguration(name string) error {
	root, err := ReadRootConfiguration()
	if err != nil {
		return err
	}
	existing := findConfigurationByName(root, name)
	if existing == nil {
		return fmt.Errorf("Configuration with name %s does not exist", name)
	}
	// Filter out the deleted configuration
	configurations := make([]Config, 0, len(root.Configurations)-1)
	for _, item := range root.Configurations {
		if item.Name != name {
			configurations = append(configurations, item)
		}
	}
	// Clear selected if it was the deleted config
	selected := root.Selected
	if selected == name {
		selected = ""
	}
	newRoot := RootConfig{
		Selected:       selected,
		Configurations: configurations,
	}
	// Saves the root configuration back
	configFilePath := getConfigFilePath()
	buf, _ := yaml.Marshal(newRoot)
	_, _ = os.OpenFile(configFilePath, os.O_CREATE|os.O_WRONLY, 0600)
	_ = os.WriteFile(configFilePath, buf, 0600)

	// OK
	return nil
}

// Name of the configuration file, in the working directory or in the home directory
const configFileName = ".yontrack-config.yaml"

// Configuration file in the working directory
const localConfigFilePath = "./" + configFileName

// Environment variable giving the path to the configuration file
const configFileEnv = "YONTRACK_CONFIG"

// Gets the path to the configuration file. The first match wins:
//
//  1. the --config flag, when set, whether the file exists or not
//  2. the YONTRACK_CONFIG environment variable, when not empty, whether the file exists or not
//  3. ./.yontrack-config.yaml, when it exists - what CI writes and reads
//  4. ~/.yontrack-config.yaml, when it exists
//  5. ./.yontrack-config.yaml otherwise, so that a first 'config create' writes there
//
// Reads and writes both go through it, so a configuration is written back to
// the file it was read from.
func getConfigFilePath() string {
	if ConfigFilePath != "" {
		return ConfigFilePath
	}
	if path := os.Getenv(configFileEnv); path != "" {
		return path
	}
	if fileExists(localConfigFilePath) {
		return localConfigFilePath
	}
	if home, err := os.UserHomeDir(); err == nil {
		homeConfigFilePath := filepath.Join(home, configFileName)
		if fileExists(homeConfigFilePath) {
			return homeConfigFilePath
		}
	}
	return localConfigFilePath
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
