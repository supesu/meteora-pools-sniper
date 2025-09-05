# Sniping Bot v2

A high-performance, microserviced Go application for monitoring Solana blockchain transactions and sending real-time Discord notifications for token creation events. Built following Clean Architecture principles, the system consists of two main services:

1. **gRPC Server**: Handles network requests, processes transactions, and sends Discord notifications
2. **Solana Scanner**: Monitors the Solana blockchain for token creation transactions and forwards them to the gRPC server

## 🏗️ Architecture

```
┌─────────────────┐    gRPC     ┌─────────────────┐
│                 │◄───────────►│                 │
│  Solana Scanner │             │   gRPC Server   │
│                 │             │                 │
└─────────────────┘             └─────────────────┘
         │                               │
         │                               │
         ▼                               ▼
┌─────────────────┐             ┌─────────────────┐
│  Solana Network │             │  Discord Bot    │
│   (WebSocket +  │             │  (Rich Embeds   │
│   RPC Polling)  │             │  & Notifications)│
└─────────────────┘             └─────────────────┘
```

### Flow
1. **Scanner** detects token creation on Solana blockchain
2. **Scanner** parses token information from transaction data  
3. **Scanner** sends transaction to **gRPC Server** via gRPC
4. **gRPC Server** processes transaction using Clean Architecture use cases
5. **gRPC Server** triggers Discord notification for token creation events
6. **Discord Bot** sends rich embed to configured Discord channel

## 🚀 Features

### gRPC Server
- **Transaction Processing**: Receives and processes transactions from scanner services
- **Discord Notifications**: Sends rich embed notifications to Discord for token creation events
- **Real-time Streaming**: WebSocket-like streaming of transaction events to subscribers
- **In-Memory Storage**: Fast in-memory transaction storage for recent data
- **Health Monitoring**: Built-in health checks and status reporting
- **Middleware Support**: Request logging, recovery, and custom middleware

### Solana Scanner
- **Token Creation Detection**: Specifically monitors for token creation transactions
- **Dual Scanning Methods**: 
  - WebSocket subscriptions for real-time updates
  - RPC polling for reliability and historical data
- **Program Filtering**: Monitor specific Solana programs (Token Program, System Program)
- **Automatic Reconnection**: Robust error handling and reconnection logic
- **Configurable Scanning**: Adjustable scan intervals and confirmation levels

### Discord Integration
- **Rich Embeds**: Beautiful, informative Discord embeds with token information
- **Automatic Retry**: Built-in retry logic for Discord API failures
- **Flexible Configuration**: Support for both Discord bots and webhooks
- **Rate Limiting**: Respects Discord API rate limits

### Shared Infrastructure  
- **Clean Architecture**: Follows Uncle Bob's Clean Architecture principles
- **Configuration Management**: YAML-based config with environment variable override
- **Structured Logging**: JSON logging for production, human-readable for development
- **Docker Support**: Complete containerization with docker-compose
- **Monitoring Ready**: Prometheus metrics and Grafana dashboards included

## 📋 Prerequisites

- Go 1.21+
- Protocol Buffers compiler (`protoc`)
- Docker and Docker Compose (for containerized deployment)
- Discord Bot Token and Channel ID (for notifications)

## 🛠️ Installation

### Local Development

1. **Clone the repository**:
   ```bash
   git clone <repository-url>
   cd sniping-bot-v2
   ```

2. **Install dependencies**:
   ```bash
   make deps
   ```

3. **Generate protobuf code**:
   ```bash
   make proto
   ```

4. **Build the services**:
   ```bash
   make build
   ```

### Docker Deployment

#### Option 1: Quick Setup Script (Recommended)
```bash
./scripts/setup.sh
```
This interactive script will:
- Create your `.env` file
- Prompt for Discord credentials  
- Start the services automatically

#### Option 2: Manual Setup
1. **Create environment file**:
   ```bash
   cp env.example .env
   # Edit .env with your Discord bot credentials
   ```

2. **Build and start all services**:
   ```bash
   make docker-up
   ```

3. **Stop services**:
   ```bash
   make docker-down
   ```

## ⚙️ Configuration

Configuration is managed through environment variables with optional YAML file overrides:

### Configuration Priority (highest to lowest)
1. **Environment variables** (e.g., `SNIPING_BOT_DISCORD_BOT_TOKEN`)
2. **`.env` file** (loaded automatically)
3. **YAML config files** (optional)

### Config Files
- `.env` - Environment variables (create from `env.example`)
- `env.example` - Environment variables template
- `configs/config.yaml` - Development configuration (optional)
- `configs/config.prod.yaml` - Production configuration (optional)

### Key Configuration Sections

#### gRPC Server
```yaml
grpc:
  host: 0.0.0.0
  port: 50051
  max_recv_size: 4194304  # 4MB
  timeout: 30s
  enable_tls: false
```

#### Solana Configuration
```yaml
solana:
  rpc_url: https://api.mainnet-beta.solana.com
  ws_endpoint: wss://api.mainnet-beta.solana.com
  program_ids:
    - 11111111111111111111111111111111  # System Program
    - TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA  # Token Program
  scan_interval: 1s
  confirmation_level: confirmed
```

#### Discord Configuration
```yaml
discord:
  bot_token: "YOUR_DISCORD_BOT_TOKEN_HERE"
  channel_id: "YOUR_DISCORD_CHANNEL_ID_HERE"
  guild_id: "YOUR_DISCORD_GUILD_ID_HERE"
  username: "Solana Token Sniping Bot"
  timeout: 30s
  max_retries: 3
  embed_color: 0x00ff00
```

For detailed Discord setup instructions, see [DISCORD_SETUP.md](./DISCORD_SETUP.md).

### Quick Setup with .env

1. **Copy the environment template**:
   ```bash
   cp env.example .env
   ```

2. **Edit `.env` with your Discord credentials**:
   ```bash
   # Required: Get these from Discord Developer Portal
   SNIPING_BOT_DISCORD_BOT_TOKEN=your_actual_bot_token_here
   SNIPING_BOT_DISCORD_CHANNEL_ID=your_actual_channel_id_here  
   SNIPING_BOT_DISCORD_GUILD_ID=your_actual_guild_id_here
   ```

3. **Run with Docker**:
   ```bash
   docker-compose up -d
   ```

The application will automatically load configuration from the `.env` file!

## 🏃‍♂️ Running the Services

### Development Mode

**Start gRPC Server**:
```bash
make run-grpc
# or
go run ./cmd/grpc-server --config configs/config.yaml
```

**Start Solana Scanner**:
```bash
make run-scanner
# or
go run ./cmd/solana-scanner --config configs/config.yaml
```

### Production Mode

**Using Docker Compose**:
```bash
docker-compose up -d
```

**Using Binaries**:
```bash
./bin/grpc-server --config configs/config.prod.yaml
./bin/solana-scanner --config configs/config.prod.yaml
```

## 📊 API Reference

### gRPC Service Definition

The system provides the following gRPC methods:

#### ProcessTransaction
Processes incoming transaction data from scanner services.

```protobuf
rpc ProcessTransaction(ProcessTransactionRequest) returns (ProcessTransactionResponse);
```

#### GetTransactionHistory
Retrieves historical transaction data with filtering and pagination.

```protobuf
rpc GetTransactionHistory(GetTransactionHistoryRequest) returns (GetTransactionHistoryResponse);
```

#### SubscribeToTransactions
Provides real-time streaming of transaction events.

```protobuf
rpc SubscribeToTransactions(SubscribeRequest) returns (stream TransactionEvent);
```

#### GetHealthStatus
Returns service health information.

```protobuf
rpc GetHealthStatus(HealthRequest) returns (HealthResponse);
```

### Message Types

#### Transaction
```protobuf
message Transaction {
  string signature = 1;
  string program_id = 2;
  repeated string accounts = 3;
  bytes data = 4;
  google.protobuf.Timestamp timestamp = 5;
  uint64 slot = 6;
  TransactionStatus status = 7;
  map<string, string> metadata = 8;
}
```

## 🧪 Testing

Run the test suite:

```bash
make test
```

Run tests with coverage:

```bash
make test-cover
```

## 📈 Monitoring

The system includes built-in monitoring capabilities:

### Health Checks
- HTTP health endpoints
- Docker health checks
- Database connectivity checks

### Metrics (Optional)
- Prometheus metrics collection
- Grafana dashboards
- Custom application metrics

### Logging
- Structured JSON logging in production
- Configurable log levels
- Request/response logging with correlation IDs

## 🔧 Development

### Project Structure

```
├── api/proto/              # Protocol buffer definitions
├── cmd/                    # Application entry points
│   ├── grpc-server/       # gRPC server main
│   └── solana-scanner/    # Scanner service main
├── configs/               # Configuration files
├── docker/                # Docker configurations
├── internal/              # Private application code
│   ├── adapters/          # External adapters (Clean Architecture)
│   │   ├── grpc/         # gRPC server adapter
│   │   └── solana/       # Solana blockchain adapter
│   ├── infrastructure/   # Infrastructure implementations
│   │   ├── events/       # Event publishing
│   │   └── repository/   # Data repositories
│   ├── usecase/          # Business use cases
│   └── container/        # Dependency injection
├── pkg/                   # Public library code
│   ├── config/           # Configuration management
│   ├── domain/           # Domain entities and interfaces
│   └── logger/           # Logging utilities
├── scripts/              # Build and deployment scripts
└── monitoring/           # Monitoring configurations
```

### Adding New Features

1. **Update Protocol Buffers**: Modify `api/proto/sniping.proto`
2. **Regenerate Code**: Run `make proto`
3. **Implement Service Logic**: Update service implementations
4. **Add Tests**: Include unit and integration tests
5. **Update Documentation**: Keep README and comments current

### Code Style

- Follow Go conventions and idioms
- Use `gofmt` for formatting
- Run `golangci-lint` for linting
- Include comprehensive error handling
- Add structured logging for observability

## 🚀 Deployment

### Environment Variables

Copy `env.example` to `.env` and customize:

```bash
cp env.example .env
# Edit .env with your configuration
```

### Production Checklist

- [ ] Configure TLS for gRPC communication
- [ ] Set up proper database credentials
- [ ] Configure monitoring and alerting
- [ ] Set appropriate resource limits
- [ ] Enable log aggregation
- [ ] Set up backup procedures
- [ ] Configure firewall rules

### Scaling Considerations

- **Horizontal Scaling**: Multiple scanner instances can run simultaneously
- **Load Balancing**: Use a load balancer in front of gRPC servers
- **Database**: Consider read replicas for high query loads
- **Caching**: Implement Redis for frequently accessed data

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## 📝 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🆘 Support

For support and questions:

- Create an issue in the GitHub repository
- Check the [troubleshooting guide](docs/troubleshooting.md)
- Review the [FAQ](docs/faq.md)

## 🔄 Changelog

See [CHANGELOG.md](CHANGELOG.md) for version history and changes.
