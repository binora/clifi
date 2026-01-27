# clifi

**Terminal-first crypto operator agent** - An AI-powered CLI for managing crypto wallets and interacting with EVM blockchains.

## Features

- **Wallet Management**: Create, import, and manage Ethereum accounts securely with encrypted keystore
- **Multi-Chain Support**: Query balances across Ethereum, Base, Arbitrum, Optimism, Polygon, and testnets
- **AI-Powered Chat**: Natural language interface powered by Claude for querying your portfolio
- **Safety-First Design**: Human-in-the-loop confirmation for all state-changing operations

## Installation

```bash
# Clone the repository
git clone https://github.com/yolodolo42/clifi.git
cd clifi

# Build
make build

# Install to $GOPATH/bin
make install
```

## Usage

### Interactive Mode (REPL)

```bash
# Set your Anthropic API key
export ANTHROPIC_API_KEY=your-api-key

# Start the interactive agent
clifi
```

In the REPL, you can ask natural language questions:
- "What's my portfolio?"
- "Show my ETH balance on Base"
- "What chains are supported?"
- "List my wallets"

### Command Mode

```bash
# Wallet management
clifi wallet create           # Create a new wallet
clifi wallet import --key ... # Import from private key
clifi wallet list             # List all wallets

# Portfolio
clifi portfolio               # Show balances across chains
clifi portfolio --chains ethereum,base --testnet
```

## Configuration

Config file location: `~/.clifi/config.yaml`

```yaml
# Default chain
chain: ethereum

# LLM settings
llm:
  provider: anthropic
  model: claude-sonnet-4-20250514

# Safety settings
safety:
  confirm_all: true
  max_slippage: 1.0
```

## Supported Chains

### Mainnets
- Ethereum (Chain ID: 1)
- Base (Chain ID: 8453)
- Arbitrum One (Chain ID: 42161)
- Optimism (Chain ID: 10)
- Polygon (Chain ID: 137)

### Testnets
- Sepolia (Chain ID: 11155111)
- Base Sepolia (Chain ID: 84532)

## Project Structure

```
clifi/
├── cmd/clifi/          # Entry point
├── internal/
│   ├── agent/          # AI agent loop and tools
│   ├── chain/          # Multi-chain RPC client
│   ├── cli/            # Cobra commands and Bubbletea REPL
│   ├── llm/            # Anthropic Claude integration
│   ├── wallet/         # Wallet management (keystore)
│   └── safety/         # Safety gates (TODO)
└── config/             # Default configuration
```

## Roadmap

- [x] Phase 1: Wallet + Read Primitives
- [x] Phase 2: Agent Loop + REPL
- [ ] Phase 3: Safe Signing (send, approve)
- [ ] Phase 4: Swap Primitive
- [ ] Phase 5: Bridge Primitive
- [ ] Phase 6: Perps Integration
- [ ] Phase 7: Plugin SDK

## Security

- Private keys are encrypted using go-ethereum's keystore (scrypt)
- Keys are stored in `~/.clifi/keystore/`
- All state-changing operations require explicit confirmation
- Policy engine for spend limits and contract allowlists (coming soon)

## License

MIT

## Contributing

Contributions welcome! Please read the contributing guidelines first.
