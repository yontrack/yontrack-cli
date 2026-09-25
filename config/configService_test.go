package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// inTempWorkspace runs the test from an empty working directory, with an empty
// HOME, no YONTRACK_CONFIG and no --config, and returns both directories.
func inTempWorkspace(t *testing.T) (workDir string, homeDir string) {
	t.Helper()
	workDir = t.TempDir()
	homeDir = t.TempDir()

	previousDir, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(workDir))
	t.Cleanup(func() { _ = os.Chdir(previousDir) })

	t.Setenv("HOME", homeDir)
	t.Setenv("YONTRACK_CONFIG", "")

	previousPath := ConfigFilePath
	ConfigFilePath = ""
	t.Cleanup(func() { ConfigFilePath = previousPath })

	return workDir, homeDir
}

func writeConfigFile(t *testing.T, path string) {
	t.Helper()
	require.NoError(t, os.WriteFile(path, []byte("selected: test\nconfigurations:\n  - name: test\n    url: http://localhost\n"), 0o600))
}

func TestConfigFilePath_ExplicitFlagWinsEvenWhenMissing(t *testing.T) {
	workDir, homeDir := inTempWorkspace(t)
	writeConfigFile(t, filepath.Join(workDir, ".yontrack-config.yaml"))
	writeConfigFile(t, filepath.Join(homeDir, ".yontrack-config.yaml"))
	envFile := filepath.Join(t.TempDir(), "env.yaml")
	writeConfigFile(t, envFile)
	t.Setenv("YONTRACK_CONFIG", envFile)
	ConfigFilePath = filepath.Join(t.TempDir(), "missing.yaml")

	assert.Equal(t, ConfigFilePath, getConfigFilePath())
}

func TestConfigFilePath_EnvVarWinsOverLocalAndHome(t *testing.T) {
	workDir, homeDir := inTempWorkspace(t)
	writeConfigFile(t, filepath.Join(workDir, ".yontrack-config.yaml"))
	writeConfigFile(t, filepath.Join(homeDir, ".yontrack-config.yaml"))
	envFile := filepath.Join(t.TempDir(), "missing.yaml")
	t.Setenv("YONTRACK_CONFIG", envFile)

	assert.Equal(t, envFile, getConfigFilePath())
}

func TestConfigFilePath_LocalWinsOverHome(t *testing.T) {
	workDir, homeDir := inTempWorkspace(t)
	writeConfigFile(t, filepath.Join(workDir, ".yontrack-config.yaml"))
	writeConfigFile(t, filepath.Join(homeDir, ".yontrack-config.yaml"))

	assert.Equal(t, "./.yontrack-config.yaml", getConfigFilePath())
}

func TestConfigFilePath_HomeWhenNoLocal(t *testing.T) {
	_, homeDir := inTempWorkspace(t)
	writeConfigFile(t, filepath.Join(homeDir, ".yontrack-config.yaml"))

	assert.Equal(t, filepath.Join(homeDir, ".yontrack-config.yaml"), getConfigFilePath())

	// ... and it is what gets read
	cfg, err := GetSelectedConfiguration()
	require.NoError(t, err)
	assert.Equal(t, "test", cfg.Name)
}

func TestConfigFilePath_LocalByDefault(t *testing.T) {
	workDir, homeDir := inTempWorkspace(t)

	assert.Equal(t, "./.yontrack-config.yaml", getConfigFilePath())

	// A first 'config create' writes there
	require.NoError(t, AddConfiguration(Config{Name: "prod", URL: "http://localhost"}, false))
	assert.FileExists(t, filepath.Join(workDir, ".yontrack-config.yaml"))
	assert.NoFileExists(t, filepath.Join(homeDir, ".yontrack-config.yaml"))
}

// Selecting a configuration from a repository updates the home file it was
// read from, rather than creating a local one.
func TestConfigFilePath_WritesGoToTheResolvedFile(t *testing.T) {
	workDir, homeDir := inTempWorkspace(t)
	homeFile := filepath.Join(homeDir, ".yontrack-config.yaml")
	writeConfigFile(t, homeFile)
	require.NoError(t, AddConfiguration(Config{Name: "other", URL: "http://other"}, false))

	require.NoError(t, SetSelectedConfiguration("test"))

	assert.NoFileExists(t, filepath.Join(workDir, ".yontrack-config.yaml"))
	root, err := ReadRootConfiguration()
	require.NoError(t, err)
	assert.Equal(t, "test", root.Selected)
	assert.Len(t, root.Configurations, 2)
}

// No selected configuration names the file which was looked at, so that a wrong
// location is visible at once.
func TestNoCurrentConfigurationNamesTheFile(t *testing.T) {
	inTempWorkspace(t)

	_, err := GetSelectedConfiguration()

	assert.EqualError(t, err, "No current configuration (in ./.yontrack-config.yaml)")
}
