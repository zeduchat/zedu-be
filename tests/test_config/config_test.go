package test_config

import (
	"os"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

// Test environment override without full app initialization
func TestEnvironmentVariableOverride(t *testing.T) {
	// This is a simpler test that verifies viper's env override capability
	// without needing full application setup
	
	testEnvContent := `SERVER_PORT=8000
APP_NAME=FileValue`

	tmpFile, err := os.CreateTemp(".", "simple_test_*.env")
	assert.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.WriteString(testEnvContent)
	assert.NoError(t, err)
	tmpFile.Close()

	configName := tmpFile.Name()[:len(tmpFile.Name())-4]

	t.Run("file value without override", func(t *testing.T) {
		v := viper.New()
		v.SetConfigName(configName)
		v.SetConfigType("env")
		v.AddConfigPath(".")
		
		err := v.ReadInConfig()
		assert.NoError(t, err)
		
		v.AutomaticEnv()
		
		assert.Equal(t, "8000", v.GetString("SERVER_PORT"))
		assert.Equal(t, "FileValue", v.GetString("APP_NAME"))
	})

	t.Run("environment variable overrides file", func(t *testing.T) {
		// Set environment variable
		os.Setenv("SERVER_PORT", "4003")
		defer os.Unsetenv("SERVER_PORT")
		
		v := viper.New()
		v.SetConfigName(configName)
		v.SetConfigType("env")
		v.AddConfigPath(".")
		
		// Read file first
		err := v.ReadInConfig()
		assert.NoError(t, err)
		
		// Enable env override
		v.AutomaticEnv()
		
		// Environment should override file
		assert.Equal(t, "4003", v.GetString("SERVER_PORT"))
		// Non-overridden value should come from file
		assert.Equal(t, "FileValue", v.GetString("APP_NAME"))
	})

	t.Run("multiple overrides", func(t *testing.T) {
		os.Setenv("SERVER_PORT", "9000")
		os.Setenv("APP_NAME", "EnvValue")
		defer func() {
			os.Unsetenv("SERVER_PORT")
			os.Unsetenv("APP_NAME")
		}()
		
		v := viper.New()
		v.SetConfigName(configName)
		v.SetConfigType("env")
		v.AddConfigPath(".")
		
		v.ReadInConfig()
		v.AutomaticEnv()
		
		assert.Equal(t, "9000", v.GetString("SERVER_PORT"))
		assert.Equal(t, "EnvValue", v.GetString("APP_NAME"))
	})
}

// Additional test for edge cases
func TestEnvironmentOnly(t *testing.T) {
	t.Run("reads from env when no file exists", func(t *testing.T) {
		os.Setenv("SERVER_PORT", "5555")
		os.Setenv("APP_NAME", "EnvOnlyApp")
		defer func() {
			os.Unsetenv("SERVER_PORT")
			os.Unsetenv("APP_NAME")
		}()
		
		v := viper.New()
		v.SetConfigName("nonexistent_config")
		v.SetConfigType("env")
		v.AddConfigPath(".")
		
		// File read will fail, but that's expected
		v.ReadInConfig()
		
		// Enable env reading
		v.AutomaticEnv()
		
		// Should get values from environment
		assert.Equal(t, "5555", v.GetString("SERVER_PORT"))
		assert.Equal(t, "EnvOnlyApp", v.GetString("APP_NAME"))
	})
}
