package discord

// DiscordMessage represents a Discord webhook message
type DiscordMessage struct {
	Content   string         `json:"content,omitempty"`
	Username  string         `json:"username,omitempty"`
	AvatarURL string         `json:"avatar_url,omitempty"`
	TTS       bool           `json:"tts,omitempty"`
	Embeds    []DiscordEmbed `json:"embeds,omitempty"`
}

// BotMessage represents a Discord bot API message
type BotMessage struct {
	Content string         `json:"content,omitempty"`
	TTS     bool           `json:"tts,omitempty"`
	Embeds  []DiscordEmbed `json:"embeds,omitempty"`
}

// DiscordEmbed represents a Discord embed
type DiscordEmbed struct {
	Title       string       `json:"title,omitempty"`
	Type        string       `json:"type,omitempty"`
	Description string       `json:"description,omitempty"`
	URL         string       `json:"url,omitempty"`
	Timestamp   string       `json:"timestamp,omitempty"`
	Color       int          `json:"color,omitempty"`
	Footer      EmbedFooter  `json:"footer,omitempty"`
	Image       EmbedImage   `json:"image,omitempty"`
	Thumbnail   EmbedImage   `json:"thumbnail,omitempty"`
	Author      EmbedAuthor  `json:"author,omitempty"`
	Fields      []EmbedField `json:"fields,omitempty"`
}

// EmbedFooter represents a Discord embed footer
type EmbedFooter struct {
	Text    string `json:"text,omitempty"`
	IconURL string `json:"icon_url,omitempty"`
}

// EmbedImage represents a Discord embed image
type EmbedImage struct {
	URL    string `json:"url,omitempty"`
	Width  int    `json:"width,omitempty"`
	Height int    `json:"height,omitempty"`
}

// EmbedAuthor represents a Discord embed author
type EmbedAuthor struct {
	Name    string `json:"name,omitempty"`
	URL     string `json:"url,omitempty"`
	IconURL string `json:"icon_url,omitempty"`
}

// EmbedField represents a Discord embed field
type EmbedField struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Inline bool   `json:"inline,omitempty"`
}
