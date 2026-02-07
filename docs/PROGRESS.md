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
- [ ] EIP-712 typed data signing (currently a raw-hash stub; needs correct domain/struct hashing)
- [x] Multi-chain RPC client with failover
- [x] Native token balance queries
- [x] ERC20 token balance queries
- [x] 5 mainnet chains (Ethereum, Base, Arbitrum, Optimism, Polygon)
- [x] 2 testnet chains (Sepolia, Base Sepolia)

### Phase 3: Safe Signing (MVP)
- [x] Send native tokens (preview + confirm gate)
- [x] Send ERC20 tokens (preview + confirm gate)
- [x] Token approvals (preview + confirm gate)
- [x] Transaction receipt persistence (SQLite) + tools (`get_receipt`, `wait_receipt`)

### CLI + REPL
- [x] Interactive REPL with Bubbletea TUI
- [x] Slash commands: /help, /clear, /quit
- [x] CLI commands: wallet, portfolio, auth
- [x] In-app /model + /provider switching
- [x] First-run setup wizard with provider + wallet onboarding
- [x] Environment variable auto-detection

### Authentication
- [x] API key storage in auth.json (0600 permissions)
- [x] Environment variable priority over stored keys
- [x] OAuth flow infrastructure (callback server on :19876)
- [x] GitHub Copilot OAuth support
- [x] Multi-method auth per provider (API key vs OAuth)
- [x] Default provider management
- [x] `clifi auth test` diagnostics (real provider pings)

### DevEx
- [x] Makefile (build, test, lint, fmt, tidy)
- [x] Pre-commit hooks (go fmt, golangci-lint, tests)
- [x] Comprehensive test suite across all packages
- [x] Race condition testing (-race flag)

---

## In Progress

### Reliability + Performance
- [ ] OpenRouter + Claude tool calling reliability (ensure tool calls work and degrade gracefully when they don't)
- [ ] Streaming for tool-call flows (and Anthropic) without breaking tool deltas
- [ ] RPC timeouts/backoff + better surfacing of failover/latency

---

## Planned

### Safe Signing (Polish)
- [ ] Gas estimation and fee display improvements
- [ ] Nonce management, retries/backoff, better error messages
- [ ] Safer policy defaults (spend limits/allowlists) + explainability

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
