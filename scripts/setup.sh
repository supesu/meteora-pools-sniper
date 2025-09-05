#!/bin/bash

# Solana Token Sniping Bot - Quick Setup Script
# This script helps you set up the environment quickly

set -e

echo "🚀 Solana Token Sniping Bot - Quick Setup"
echo "=========================================="
echo

# Check if .env already exists
if [ -f ".env" ]; then
    echo "⚠️  .env file already exists!"
    read -p "Do you want to overwrite it? (y/N): " overwrite
    if [[ $overwrite != [Yy]* ]]; then
        echo "Setup cancelled."
        exit 0
    fi
fi

# Copy env.example to .env
echo "📋 Creating .env file from template..."
cp env.example .env
echo "✅ .env file created successfully!"
echo

# Prompt for Discord configuration
echo "🔧 Discord Configuration Required"
echo "=================================="
echo "You need to get these values from the Discord Developer Portal."
echo "See DISCORD_SETUP.md for detailed instructions."
echo

read -p "Enter your Discord Bot Token: " bot_token
read -p "Enter your Discord Channel ID: " channel_id
read -p "Enter your Discord Guild ID: " guild_id

# Update .env file with user input
if [[ -n "$bot_token" ]]; then
    sed -i.bak "s/YOUR_DISCORD_BOT_TOKEN_HERE/$bot_token/g" .env
fi

if [[ -n "$channel_id" ]]; then
    sed -i.bak "s/YOUR_DISCORD_CHANNEL_ID_HERE/$channel_id/g" .env
fi

if [[ -n "$guild_id" ]]; then
    sed -i.bak "s/YOUR_DISCORD_GUILD_ID_HERE/$guild_id/g" .env
fi

# Clean up backup files
rm -f .env.bak

echo
echo "✅ Configuration updated successfully!"
echo

# Ask if user wants to start the services
echo "🐳 Docker Setup"
echo "==============="
read -p "Do you want to start the services now? (Y/n): " start_services

if [[ $start_services != [Nn]* ]]; then
    echo "🔨 Building and starting services..."

    # Check if Docker is running
    if ! docker info > /dev/null 2>&1; then
        echo "❌ Docker is not running. Please start Docker and run:"
        echo "   docker-compose up -d"
        exit 1
    fi

    # Build and start services
    docker-compose up -d --build

    echo
    echo "🎉 Setup complete!"
    echo "=================="
    echo "✅ Services are starting up..."
    echo "✅ Check logs with: docker-compose logs -f"
    echo "✅ Stop services with: docker-compose down"
    echo
    echo "📱 Your Discord bot should now send notifications to your channel"
    echo "   when new tokens are created on Solana!"
else
    echo
    echo "🎉 Setup complete!"
    echo "=================="
    echo "To start the services later, run:"
    echo "   docker-compose up -d"
    echo
    echo "📖 For more information, see README.md and DISCORD_SETUP.md"
fi

echo
echo "🔗 Useful commands:"
echo "   docker-compose logs -f          # View logs"
echo "   docker-compose down             # Stop services"
echo "   docker-compose up -d --build    # Rebuild and start"
echo
