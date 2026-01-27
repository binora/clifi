# TODO

## Next Up

### In-App REPL Commands
- [ ] `/model` - list and switch models for current provider
- [ ] `/provider` - list connected providers, switch active provider
- [ ] `/auth` - connect a new provider from within REPL
- [ ] `/logout` - clear all credentials, exit to wizard on next launch
- [ ] `/status` - show current provider, model, wallet, connected providers
- [ ] Agent hot-swap support (SetProvider, SetModel without restart)
- [ ] Export `CreateProvider` for REPL to use

---

## Phase 3: Safe Signing
- [ ] Send native tokens (ETH, MATIC)
- [ ] Send ERC20 tokens
- [ ] Token approvals (ERC20 approve)
- [ ] Safety confirmation gates (show params, require explicit yes)
- [ ] Gas estimation and fee display
- [ ] Transaction receipt tracking
- [ ] Nonce management

---

## Phase 4: Swap Primitive
- [ ] DEX integration (Uniswap v3, etc.)
- [ ] Quote fetching and comparison
- [ ] Slippage protection
- [ ] Multi-hop routing
- [ ] Token price lookups

---

## Phase 5: Bridge Primitive
- [ ] Cross-chain bridge integration
- [ ] Bridge fee comparison
- [ ] Transaction status tracking across chains

---

## Phase 6: Perps Integration
- [ ] Perpetual futures interface
- [ ] Position management (open, close, modify)
- [ ] Risk display (liquidation price, margin)

---

## Phase 7: Plugin SDK
- [ ] Plugin interface for custom tools
- [ ] Plugin discovery and loading
- [ ] Community plugin registry

---

## Improvements
- [ ] Streaming LLM responses in REPL
- [ ] Import wallet via mnemonic phrase
- [ ] Hardware wallet support (Ledger, Trezor)
- [ ] Encrypt auth.json at rest
- [ ] Spend limit policies
- [ ] Transaction history / receipts log
- [ ] Conversation persistence between sessions
- [ ] Multi-wallet context (auto-detect which wallet to use)

---

## Infrastructure
- [ ] Release workflow (goreleaser, GitHub releases)
- [ ] Homebrew formula
- [ ] Docker image
- [ ] Integration tests with testnet RPCs
- [ ] CLI test coverage (internal/cli)
