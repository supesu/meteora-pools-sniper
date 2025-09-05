package discord

import (
	"fmt"
	"time"

	"github.com/supesu/sniping-bot-v2/pkg/config"
	"github.com/supesu/sniping-bot-v2/pkg/domain"
)

// EmbedBuilder handles Discord embed creation
type EmbedBuilder struct {
	config *config.DiscordConfig
}

// NewEmbedBuilder creates a new embed builder
func NewEmbedBuilder(cfg *config.DiscordConfig) *EmbedBuilder {
	return &EmbedBuilder{
		config: cfg,
	}
}

// CreateTokenCreationEmbed creates an embed for token creation events
func (b *EmbedBuilder) CreateTokenCreationEmbed(event *domain.TokenCreationEvent) DiscordEmbed {
	description := fmt.Sprintf(
		"🚀 **New Token Created**\n\n"+
			"**Token:** %s\n"+
			"**Address:** `%s`\n"+
			"**Creator:** `%s`\n"+
			"**Timestamp:** <t:%d:f>",
		event.GetDisplayName(),
		event.TokenAddress,
		event.CreatorAddress,
		event.Timestamp.Unix(),
	)

	embed := DiscordEmbed{
		Title:       "🪙 Token Creation Alert",
		Description: description,
		Color:       b.getEmbedColor(),
		Timestamp:   event.Timestamp.Format(time.RFC3339),
		Fields: []EmbedField{
			{
				Name:   "💰 Initial Supply",
				Value:  event.InitialSupply,
				Inline: true,
			},
			{
				Name:   "🎯 Decimals",
				Value:  fmt.Sprintf("%d", event.Decimals),
				Inline: true,
			},
		},
	}

	// Add metadata fields if available
	if len(event.Metadata) > 0 {
		for key, value := range event.Metadata {
			if key != "" && value != "" {
				embed.Fields = append(embed.Fields, EmbedField{
					Name:   fmt.Sprintf("📋 %s", key),
					Value:  value,
					Inline: true,
				})
			}
		}
	}

	return embed
}

// CreateTransactionEmbed creates an embed for transaction events
func (b *EmbedBuilder) CreateTransactionEmbed(tx *domain.Transaction) DiscordEmbed {
	description := fmt.Sprintf(
		"⚡ **New Transaction**\n\n"+
			"**Signature:** `%s`\n"+
			"**Program:** `%s`\n"+
			"**Slot:** `%d`\n"+
			"**Timestamp:** <t:%d:f>",
		tx.Signature,
		tx.ProgramID,
		tx.Slot,
		tx.Timestamp.Unix(),
	)

	embed := DiscordEmbed{
		Title:       "💸 Transaction Alert",
		Description: description,
		Color:       b.getEmbedColor(),
		Timestamp:   tx.Timestamp.Format(time.RFC3339),
		Fields: []EmbedField{
			{
				Name:   "📊 Status",
				Value:  tx.Status.String(),
				Inline: true,
			},
		},
	}

	// Add accounts field if available
	if len(tx.Accounts) > 0 {
		accountsValue := ""
		for i, account := range tx.Accounts {
			if i >= 5 { // Limit to 5 accounts to avoid embed limits
				accountsValue += fmt.Sprintf("\n... and %d more", len(tx.Accounts)-5)
				break
			}
			accountsValue += fmt.Sprintf("`%s`\n", account)
		}
		embed.Fields = append(embed.Fields, EmbedField{
			Name:   "👥 Accounts",
			Value:  accountsValue,
			Inline: false,
		})
	}

	// Add metadata fields if available
	if len(tx.Metadata) > 0 {
		for key, value := range tx.Metadata {
			if key != "" && value != "" {
				embed.Fields = append(embed.Fields, EmbedField{
					Name:   fmt.Sprintf("📋 %s", key),
					Value:  value,
					Inline: true,
				})
			}
		}
	}

	return embed
}

// CreateMeteoraPoolEmbed creates an embed for Meteora pool events
func (b *EmbedBuilder) CreateMeteoraPoolEmbed(event *domain.MeteoraPoolEvent) DiscordEmbed {
	description := fmt.Sprintf(
		"🏊 **Meteora Pool Created**\n\n"+
			"**Pool Address:** `%s`\n"+
			"**Token Pair:** %s / %s\n"+
			"**Creator:** `%s`\n"+
			"**Timestamp:** <t:%d:f>",
		event.PoolAddress,
		event.TokenASymbol,
		event.TokenBSymbol,
		event.CreatorWallet,
		event.CreatedAt.Unix(),
	)

	embed := DiscordEmbed{
		Title:       "🌊 Meteora Pool Alert",
		Description: description,
		Color:       MeteoraBrandColor,
		Timestamp:   event.CreatedAt.Format(time.RFC3339),
		Fields: []EmbedField{
			{
				Name:   "🏷️ Pool Type",
				Value:  event.PoolType,
				Inline: true,
			},
			{
				Name:   "💰 Fee Rate",
				Value:  fmt.Sprintf("%.2f%%", float64(event.FeeRate)/10000),
				Inline: true,
			},
			{
				Name:   "💧 Initial Liquidity A",
				Value:  fmt.Sprintf("%.6f %s", float64(event.InitialLiquidityA)/1000000, event.TokenASymbol),
				Inline: true,
			},
			{
				Name:   "💧 Initial Liquidity B",
				Value:  fmt.Sprintf("%.6f %s", float64(event.InitialLiquidityB)/1000000, event.TokenBSymbol),
				Inline: true,
			},
		},
	}

	// Add metadata fields if available
	if len(event.Metadata) > 0 {
		for key, value := range event.Metadata {
			if key != "" && value != "" {
				embed.Fields = append(embed.Fields, EmbedField{
					Name:   fmt.Sprintf("📋 %s", key),
					Value:  value,
					Inline: true,
				})
			}
		}
	}

	return embed
}

// getEmbedColor returns the configured embed color or default
func (b *EmbedBuilder) getEmbedColor() int {
	if b.config.EmbedColor != 0 {
		return b.config.EmbedColor
	}
	return EmbedColorDefault
}
