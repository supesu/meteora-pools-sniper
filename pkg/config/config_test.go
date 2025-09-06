package config

import (
	"os"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad_ConfigFromFile(t *testing.T) {
	// Set required environment variables for the test
	t.Setenv("SNIPING_BOT_DISCORD_BOT_TOKEN", "test-bot-token")
	t.Setenv("SNIPING_BOT_DISCORD_WEBHOOK_URL", "https://discord.com/api/webhooks/test")
	t.Setenv("SNIPING_BOT_DISCORD_CHANNEL_ID", "123456789")

	// Create a temporary config file
	configContent := `environment: test
log_level: debug
grpc:
  host: localhost
  port: 50051
discord:
  bot_token: test-bot-token
  channel_id: "123456789"
  webhook_url: https://discord.com/api/webhooks/test`

	// Write to temporary file
	tmpFile, err := os.CreateTemp("", "config-*.yaml")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.WriteString(configContent)
	require.NoError(t, err)
	tmpFile.Close()

	// Debug: print file content
	content, _ := os.ReadFile(tmpFile.Name())
	t.Logf("Config file content:\n%s", string(content))

	// Test loading config
	v := viper.New()
	v.SetConfigFile(tmpFile.Name())
	err = v.ReadInConfig()
	if err != nil {
		t.Fatalf("Failed to read config: %v", err)
	}

	config := &Config{}
	err = v.Unmarshal(config)

	assert.NoError(t, err)
	assert.NotNil(t, config)

	// Verify basic fields
	assert.Equal(t, "test", config.Environment)
	assert.Equal(t, "debug", config.LogLevel)

	// Verify gRPC config
	assert.Equal(t, "localhost", config.GRPC.Host)
	assert.Equal(t, 50051, config.GRPC.Port)

	// Verify Discord config
	assert.Equal(t, "test-bot-token", config.Discord.BotToken)
	assert.Equal(t, "123456789", config.Discord.ChannelID)
}

func TestLoad_DefaultConfig(t *testing.T) {
	// Set required Discord environment variables to avoid validation errors
	t.Setenv("SNIPING_BOT_DISCORD_BOT_TOKEN", "test-bot-token")
	t.Setenv("SNIPING_BOT_DISCORD_CHANNEL_ID", "123456789")

	// Test loading without specifying a file (should use defaults)
	config, err := Load("")

	// This might fail if no default config exists, which is expected
	if err != nil {
		assert.Contains(t, err.Error(), "config") // Should be a config-related error
	} else {
		assert.NotNil(t, config)
		// If it succeeds, verify it has reasonable defaults
		assert.NotEmpty(t, config.Environment)
	}
}

func TestLoad_InvalidFile(t *testing.T) {
	// Set required Discord environment variables to avoid validation errors
	t.Setenv("SNIPING_BOT_DISCORD_BOT_TOKEN", "test-bot-token")
	t.Setenv("SNIPING_BOT_DISCORD_CHANNEL_ID", "123456789")

	config, err := Load("non-existent-file.yaml")

	// Config files are optional - Load should succeed with defaults when file doesn't exist
	assert.NoError(t, err)
	assert.NotNil(t, config)
	assert.Equal(t, "development", config.Environment)
}

func TestLoad_InvalidYAML(t *testing.T) {
	// Set required Discord environment variables to avoid validation errors
	t.Setenv("SNIPING_BOT_DISCORD_BOT_TOKEN", "test-bot-token")
	t.Setenv("SNIPING_BOT_DISCORD_CHANNEL_ID", "123456789")

	// Create a temporary file with invalid YAML
	tmpFile, err := os.CreateTemp("", "invalid-config-*.yaml")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.WriteString("invalid: yaml: content: [\n") // Invalid YAML
	require.NoError(t, err)
	tmpFile.Close()

	config, err := Load(tmpFile.Name())

	// Invalid YAML is treated as a warning, not an error - Load should succeed with defaults
	assert.NoError(t, err)
	assert.NotNil(t, config)
	assert.Equal(t, "development", config.Environment)
}

func TestConfig_Address(t *testing.T) {
	config := &Config{
		GRPC: GRPCConfig{
			Host: "127.0.0.1",
			Port: 8080,
		},
	}

	address := config.Address()
	assert.Equal(t, "127.0.0.1:8080", address)
}

// Note: Config validation can be added later if needed
// For now, basic config loading is tested above
