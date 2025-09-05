# Discord Bot Setup Guide

This guide will help you set up the Discord bot integration for the Solana Token Sniping Bot.

## Prerequisites

- A Discord server where you have administrative permissions
- Discord Developer Portal access

## Step 1: Create a Discord Application

1. Go to the [Discord Developer Portal](https://discord.com/developers/applications)
2. Click "New Application"
3. Enter a name for your application (e.g., "Solana Token Sniping Bot")
4. Click "Create"

## Step 2: Create a Bot

1. In your application, navigate to the "Bot" section
2. Click "Add Bot"
3. Under the "Token" section, click "Copy" to copy your bot token
4. **Important**: Keep this token secret and never share it publicly

## Step 3: Set Bot Permissions

1. In the "Bot" section, scroll down to "Privileged Gateway Intents"
2. Enable the following intents if needed:
   - Server Members Intent (if you plan to mention users)
   - Message Content Intent (if the bot needs to read message content)

## Step 4: Invite Bot to Your Server

1. Go to the "OAuth2" > "URL Generator" section
2. Under "Scopes", select:
   - `bot`
3. Under "Bot Permissions", select:
   - `Send Messages`
   - `Use Slash Commands` (optional)
   - `Embed Links`
   - `Attach Files` (optional)
4. Copy the generated URL and open it in your browser
5. Select your server and authorize the bot

## Step 5: Get Channel and Guild IDs

### Enable Developer Mode
1. In Discord, go to User Settings (gear icon)
2. Go to "Advanced" and enable "Developer Mode"

### Get Guild ID
1. Right-click on your server name
2. Click "Copy Server ID"

### Get Channel ID
1. Right-click on the channel where you want notifications
2. Click "Copy Channel ID"

## Step 6: Configure the Bot

### Option A: Using Environment Variables

Create a `.env` file in your project root:

```bash
SNIPING_BOT_DISCORD_BOT_TOKEN=your_bot_token_here
SNIPING_BOT_DISCORD_CHANNEL_ID=your_channel_id_here
SNIPING_BOT_DISCORD_GUILD_ID=your_guild_id_here
```

### Option B: Using Configuration File

Update your `configs/config.yaml`:

```yaml
discord:
  bot_token: "your_bot_token_here"
  channel_id: "your_channel_id_here"
  guild_id: "your_guild_id_here"
  username: "Solana Token Sniping Bot"
  timeout: 30s
  max_retries: 3
  retry_delay: 5s
  embed_color: 0x00ff00
```

### Option C: Using Webhook (Alternative)

If you prefer using webhooks instead of a bot:

1. In your Discord channel, go to Channel Settings > Integrations
2. Click "Create Webhook"
3. Copy the webhook URL
4. Configure:

```yaml
discord:
  webhook_url: "your_webhook_url_here"
  username: "Solana Token Sniping Bot"
  # ... other settings
```

## Step 7: Test the Integration

1. Start your application
2. The bot should appear online in your Discord server
3. When a token creation is detected, you should see a notification in your configured channel

## Message Format

The bot will send rich embeds with the following information:

- **Token Address**: The new token's address
- **Creator**: The address that created the token
- **Name & Symbol**: If available in the transaction metadata
- **Initial Supply**: If available
- **Decimals**: Token decimal places
- **Transaction Link**: Direct link to view on Solscan

## Troubleshooting

### Bot is offline
- Check that your bot token is correct
- Ensure the bot has been invited to your server

### No messages appearing
- Verify the channel ID is correct
- Check that the bot has permission to send messages in the channel
- Review application logs for Discord API errors

### Messages appear but are malformed
- Check your embed color configuration (should be hex without 0x prefix in YAML)
- Verify all required configuration fields are set

## Security Best Practices

1. **Never commit your bot token to version control**
2. Use environment variables or secure configuration management
3. Regularly rotate your bot token if compromised
4. Only grant necessary permissions to the bot
5. Monitor bot activity and logs for suspicious behavior

## Rate Limiting

Discord has rate limits for bot messages:
- 5 requests per 5 seconds per channel
- The bot includes automatic retry logic with exponential backoff
- Configure `max_retries` and `retry_delay` based on your needs

## Advanced Configuration

### Custom Embed Colors
You can customize the embed color by setting `embed_color` to any hex color value:
- Green: `0x00ff00`
- Blue: `0x0099ff`
- Red: `0xff0000`
- Gold: `0xffd700`

### Multiple Channels
To send notifications to multiple channels, you can:
1. Create multiple webhook URLs
2. Modify the Discord adapter to support multiple channels
3. Use Discord's forum channels for categorized notifications
