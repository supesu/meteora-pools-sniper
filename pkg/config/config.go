package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/viper"
)

// Config holds the application configuration
type Config struct {
	Environment string        `mapstructure:"environment"`
	LogLevel    string        `mapstructure:"log_level"`
	GRPC        GRPCConfig    `mapstructure:"grpc"`
	Solana      SolanaConfig  `mapstructure:"solana"`
	Discord     DiscordConfig `mapstructure:"discord"`
}

// GRPCConfig holds gRPC server configuration
type GRPCConfig struct {
	Host        string `mapstructure:"host"`
	Port        int    `mapstructure:"port"`
	MaxRecvSize int    `mapstructure:"max_recv_size"`
	MaxSendSize int    `mapstructure:"max_send_size"`
	Timeout     string `mapstructure:"timeout"`
}

// SolanaConfig holds Solana-related configuration
type SolanaConfig struct {
	RPC          string        `mapstructure:"rpc_url"`
	WSEndpoint   string        `mapstructure:"ws_endpoint"`
	ProgramIDs   []string      `mapstructure:"program_ids"`
	ScanInterval string        `mapstructure:"scan_interval"`
	MaxRetries   int           `mapstructure:"max_retries"`
	RetryDelay   string        `mapstructure:"retry_delay"`
	Meteora      MeteoraConfig `mapstructure:"meteora"`
}

// MeteoraConfig holds Meteora-specific configuration
type MeteoraConfig struct {
	Enabled               bool           `mapstructure:"enabled"`
	MinLiquidityThreshold uint64         `mapstructure:"min_liquidity_threshold"`
	NotificationTypes     []string       `mapstructure:"notification_types"`
	TokenMetadataSources  []string       `mapstructure:"token_metadata_sources"`
	TokenMetadataCacheTTL string         `mapstructure:"token_metadata_cache_ttl"`
	QualityFilters        QualityFilters `mapstructure:"quality_filters"`
}

// QualityFilters holds quality filtering configuration
type QualityFilters struct {
	RequireTokenMetadata bool     `mapstructure:"require_token_metadata"`
	BlockSpamTokens      bool     `mapstructure:"block_spam_tokens"`
	SpamTokenPatterns    []string `mapstructure:"spam_token_patterns"`
	MinLiquidityUSD      float64  `mapstructure:"min_liquidity_usd"`
}

// DiscordConfig holds Discord bot configuration
type DiscordConfig struct {
	BotToken   string `mapstructure:"bot_token"`
	WebhookURL string `mapstructure:"webhook_url"`
	ChannelID  string `mapstructure:"channel_id"`
	Username   string `mapstructure:"username"`
	AvatarURL  string `mapstructure:"avatar_url"`
	Timeout    string `mapstructure:"timeout"`
	MaxRetries int    `mapstructure:"max_retries"`
	RetryDelay string `mapstructure:"retry_delay"`
	EmbedColor int    `mapstructure:"embed_color"`
}

// Load loads configuration from file and environment variables
func Load(configPath string) (*Config, error) {
	v := viper.New()

	// Set defaults
	setDefaults(v)

	// Read environment variables first (highest priority)
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.SetEnvPrefix("SNIPING_BOT")

	// Explicitly bind log level
	v.BindEnv("log_level", "SNIPING_BOT_LOG_LEVEL")
	v.BindEnv("environment", "SNIPING_BOT_ENVIRONMENT")

	// Explicitly bind environment variables to ensure they're read correctly
	v.BindEnv("discord.bot_token", "SNIPING_BOT_DISCORD_BOT_TOKEN")
	v.BindEnv("discord.webhook_url", "SNIPING_BOT_DISCORD_WEBHOOK_URL")
	v.BindEnv("discord.channel_id", "SNIPING_BOT_DISCORD_CHANNEL_ID")
	v.BindEnv("discord.username", "SNIPING_BOT_DISCORD_USERNAME")
	v.BindEnv("discord.avatar_url", "SNIPING_BOT_DISCORD_AVATAR_URL")
	v.BindEnv("discord.timeout", "SNIPING_BOT_DISCORD_TIMEOUT")
	v.BindEnv("discord.max_retries", "SNIPING_BOT_DISCORD_MAX_RETRIES")
	v.BindEnv("discord.retry_delay", "SNIPING_BOT_DISCORD_RETRY_DELAY")
	v.BindEnv("discord.embed_color", "SNIPING_BOT_DISCORD_EMBED_COLOR")

	// Bind Solana configuration
	v.BindEnv("solana.rpc_url", "SNIPING_BOT_SOLANA_RPC_URL")
	v.BindEnv("solana.ws_endpoint", "SNIPING_BOT_SOLANA_WS_ENDPOINT")
	v.BindEnv("solana.program_ids", "SNIPING_BOT_SOLANA_PROGRAM_IDS")

	// Bind Meteora configuration
	v.BindEnv("solana.meteora.enabled", "SNIPING_BOT_METEORA_ENABLED")
	v.BindEnv("solana.meteora.min_liquidity_threshold", "SNIPING_BOT_METEORA_MIN_LIQUIDITY")
	v.BindEnv("solana.meteora.token_metadata_sources", "SNIPING_BOT_TOKEN_METADATA_SOURCES")

	// Try to read from .env file if it exists (only for local development)
	v.SetConfigName(".env")
	v.SetConfigType("env")
	v.AddConfigPath(".")
	if err := v.ReadInConfig(); err == nil {
		// .env file found and read successfully
	}

	// Read from YAML config file (optional)
	if configPath != "" {
		v.SetConfigFile(configPath)
		if err := v.MergeInConfig(); err != nil {
			// Log warning but don't fail - environment variables can provide all config
			fmt.Printf("Warning: Could not read config file %s: %v\n", configPath, err)
		}
	} else {
		// Try to find and merge YAML config files
		v.SetConfigName("config")
		v.SetConfigType("yaml")
		v.AddConfigPath(".")
		v.AddConfigPath("./configs")
		v.AddConfigPath("/etc/sniping-bot")

		if err := v.MergeInConfig(); err != nil {
			// Config file is optional when using environment variables
			if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
				fmt.Printf("Warning: Could not read config file: %v\n", err)
			}
		}
	}

	var config Config
	if err := v.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Handle comma-separated program IDs from environment variable
	if programIDsEnv := os.Getenv("SNIPING_BOT_SOLANA_PROGRAM_IDS"); programIDsEnv != "" {
		config.Solana.ProgramIDs = strings.Split(programIDsEnv, ",")
		// Trim whitespace from each program ID
		for i, id := range config.Solana.ProgramIDs {
			config.Solana.ProgramIDs[i] = strings.TrimSpace(id)
		}
	}

	// Validate required Discord configuration
	if config.Discord.BotToken == "" && config.Discord.WebhookURL == "" {
		return nil, fmt.Errorf("either SNIPING_BOT_DISCORD_BOT_TOKEN or SNIPING_BOT_DISCORD_WEBHOOK_URL must be set")
	}

	if config.Discord.ChannelID == "" && config.Discord.WebhookURL == "" {
		return nil, fmt.Errorf("SNIPING_BOT_DISCORD_CHANNEL_ID must be set when using bot token")
	}

	return &config, nil
}

// setDefaults sets default configuration values
func setDefaults(v *viper.Viper) {
	// General
	v.SetDefault("environment", "development")
	v.SetDefault("log_level", "info")

	// gRPC
	v.SetDefault("grpc.host", "0.0.0.0")
	v.SetDefault("grpc.port", 50051)
	v.SetDefault("grpc.max_recv_size", 4*1024*1024) // 4MB
	v.SetDefault("grpc.max_send_size", 4*1024*1024) // 4MB
	v.SetDefault("grpc.timeout", "30s")

	// Solana
	v.SetDefault("solana.rpc_url", "https://api.mainnet-beta.solana.com")
	v.SetDefault("solana.ws_endpoint", "wss://api.mainnet-beta.solana.com")
	v.SetDefault("solana.scan_interval", "1s")
	v.SetDefault("solana.max_retries", 3)
	v.SetDefault("solana.retry_delay", "5s")

	// Meteora
	v.SetDefault("solana.meteora.enabled", true)
	v.SetDefault("solana.meteora.min_liquidity_threshold", 1000)
	v.SetDefault("solana.meteora.notification_types", []string{"pool_created"})
	v.SetDefault("solana.meteora.token_metadata_sources", []string{"jupiter", "solana-labs"})
	v.SetDefault("solana.meteora.token_metadata_cache_ttl", "1h")
	v.SetDefault("solana.meteora.quality_filters.require_token_metadata", true)
	v.SetDefault("solana.meteora.quality_filters.block_spam_tokens", true)
	v.SetDefault("solana.meteora.quality_filters.spam_token_patterns", []string{"TEST", "FAKE", "SCAM", "SPAM"})
	v.SetDefault("solana.meteora.quality_filters.min_liquidity_usd", 100.0)

	// Discord
	v.SetDefault("discord.username", "Solana Token Sniping Bot")
	v.SetDefault("discord.timeout", "30s")
	v.SetDefault("discord.max_retries", 3)
	v.SetDefault("discord.retry_delay", "5s")
	v.SetDefault("discord.embed_color", 0x00ff00) // Green color
}

// Address returns the gRPC server address
func (c *Config) Address() string {
	return fmt.Sprintf("%s:%d", c.GRPC.Host, c.GRPC.Port)
}
