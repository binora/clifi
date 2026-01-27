# Progress

## Completed

### Phase 1: Core Agent + Multi-Provider LLM
- [x] Agent conversation loop with tool calling
- [x] Provider interface and registry
- [x] Anthropic Claude provider
- [x] OpenAI GPT provider
- [x] Google Gemini provider
- [x] GitHub Copilot provider
- [x] Venice AI provider
- [x] OpenRouter provider (100+ models via single API)
- [x] Tool registry with 5 crypto tools
- [x] System prompt for crypto agent behavior

### Phase 2: Wallet + Chain Queries
- [x] Encrypted keystore (create, import, list)
- [x] Transaction signing (EIP-155)
- [x] Message signing (EIP-191)
- [x] EIP-712 typed data signing
- [x] Multi-chain RPC client with failover
- [x] Native token balance queries
- [x] ERC20 token balance queries
- [x] 5 mainnet chains (Ethereum, Base, Arbitrum, Optimism, Polygon)
- [x] 2 testnet chains (Sepolia, Base Sepolia)

### CLI + REPL
- [x] Interactive REPL with Bubbletea TUI
- [x] Slash commands: /help, /clear, /quit
- [x] CLI commands: wallet, portfolio, auth
- [x] First-run setup wizard with provider + wallet onboarding
- [x] Environment variable auto-detection

### Authentication
- [x] API key storage in auth.json (0600 permissions)
- [x] Environment variable priority over stored keys
- [x] OAuth flow infrastructure (callback server on :19876)
- [x] GitHub Copilot OAuth support
- [x] Multi-method auth per provider (API key vs OAuth)
- [x] Default provider management

### DevEx
- [x] Makefile (build, test, lint, fmt, tidy)
- [x] Pre-commit hooks (go fmt, golangci-lint, tests)
- [x] Comprehensive test suite across all packages
- [x] Race condition testing (-race flag)

---

## In Progress

### In-App Provider/Model Management
- [ ] /model command - switch models within REPL
- [ ] /provider command - switch providers within REPL
- [ ] /auth command - connect new provider from REPL
- [ ] /logout command - clear credentials, return to wizard
- [ ] /status command - show current config

---

## Planned

### Phase 3: Safe Signing
- [ ] Send native tokens (ETH, MATIC)
- [ ] Send ERC20 tokens
- [ ] Token approvals (ERC20 approve)
- [ ] Safety confirmation gates (show params, require explicit yes)
- [ ] Gas estimation and fee display
- [ ] Transaction receipt tracking

### Phase 4: Swap Primitive
- [ ] DEX integration (Uniswap, etc.)
- [ ] Quote fetching and comparison
- [ ] Slippage protection
- [ ] Multi-hop routing

### Phase 5: Bridge Primitive
- [ ] Cross-chain bridge integration
- [ ] Bridge fee comparison
- [ ] Transaction status tracking across chains

### Phase 6: Perps Integration
- [ ] Perpetual futures interface
- [ ] Position management
- [ ] Risk display

### Phase 7: Plugin SDK
- [ ] Plugin interface for custom tools
- [ ] Plugin discovery and loading
- [ ] Community plugin registry

### Other
- [ ] Hardware wallet support (Ledger, Trezor)
- [ ] Spend limit policies
- [ ] Transaction history
- [ ] Import wallet via mnemonic phrase
- [ ] Token storage encryption (auth.json)
- [ ] Streaming responses in REPL

---

## Providers

| Provider | Status | Auth Methods | Models |
|---|---|---|---|
| Anthropic | Done | API key | Sonnet 4, 3.5 Sonnet, 3.5 Haiku, Opus |
| OpenAI | Done | API key | GPT-4o, GPT-4o Mini, GPT-4 Turbo, GPT-3.5 |
| Google Gemini | Done | API key | 2.0 Flash, 1.5 Pro, 1.5 Flash |
| GitHub Copilot | Done | OAuth, token | Default model |
| Venice AI | Done | API key | Llama 3.3 70B, 3.1 405B, DeepSeek R1 |
| OpenRouter | Done | API key | Claude Sonnet 4, GPT-4o, Gemini 2.5 Pro, DeepSeek R1, Llama 4 |

## Chains

| Chain | Chain ID | Status |
|---|---|---|
| Ethereum | 1 | Done |
| Base | 8453 | Done |
| Arbitrum | 42161 | Done |
| Optimism | 10 | Done |
| Polygon | 137 | Done |
| Sepolia | 11155111 | Done (testnet) |
| Base Sepolia | 84532 | Done (testnet) |
