# clifi Development Session

**Date**: 2026-01-25
**Project**: Terminal-first Crypto Operator Agent
**Repository**: `github.com/yolodolo42/clifi`

---

## Session Summary

Built the foundation for **clifi** - a CLI crypto agent with **multi-provider LLM support**. Completed Phase 1 (Hello Wallet), Phase 2 (Agent Loop), **Multi-Provider Auth System**, and **Onboarding Wizard**.

---

## Decisions Made

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Language | **Go 1.22+** | Single binary, fast, great CLI ecosystem |
| CLI Framework | Cobra + Viper | Industry standard |
| TUI/REPL | Bubbletea | Rich terminal UI |
| LLM Providers | **Multi-provider** | Anthropic, OpenAI, Venice, Copilot, Gemini |
| Chain Focus | **EVM-first** | Largest DeFi ecosystem |
| Module Path | `github.com/yolodolo42/clifi` | User preference |
| Hardware Wallet | Stub only (Phase 1) | Faster to first milestone |
| Default Testnet | Sepolia | Safe for testing |
| Auth Storage | `~/.clifi/auth.json` | 0600 perms, BYOK only |
| Free Tier | Maybe later | BYOK first |

---

## What Was Built

### Files Created (33 files)

```
clifi/
├── cmd/clifi/main.go              # Entry point
├── internal/
│   ├── agent/
│   │   ├── loop.go                # Core agent loop (multi-provider)
│   │   ├── tools.go               # Tool registry + handlers
│   │   └── conversation.go        # Conversation state
│   ├── auth/                      # Auth subsystem
│   │   ├── auth.go                # Auth manager
│   │   └── store.go               # Credential storage (auth.json)
│   ├── chain/
│   │   ├── client.go              # Multi-chain RPC client
│   │   ├── config.go              # Chain configurations
│   │   └── balance.go             # Balance queries
│   ├── cli/
│   │   ├── root.go                # Cobra root command (with setup check)
│   │   ├── wallet.go              # Wallet commands
│   │   ├── portfolio.go           # Portfolio command
│   │   ├── repl.go                # Bubbletea chat REPL
│   │   ├── auth.go                # Auth commands
│   │   └── setup.go               # NEW: Setup command
│   ├── llm/
│   │   ├── provider.go            # Provider interface + registry
│   │   ├── anthropic.go           # Claude provider
│   │   ├── openai.go              # OpenAI provider
│   │   ├── venice.go              # Venice provider
│   │   ├── copilot.go             # GitHub Copilot provider
│   │   ├── gemini.go              # Google Gemini provider
│   │   └── tools.go               # Tool definitions
│   ├── setup/                     # NEW: Onboarding wizard
│   │   ├── detect.go              # First-run detection
│   │   ├── wizard.go              # Main wizard Bubbletea model
│   │   ├── provider_step.go       # Provider selection + API key
│   │   ├── wallet_step.go         # Wallet creation
│   │   └── styles.go              # Shared lipgloss styles
│   └── wallet/
│       ├── signer.go              # Signer interface
│       ├── keystore.go            # Encrypted keystore
│       └── hardware.go            # Hardware wallet stub
├── config/default.yaml            # Default configuration
├── Makefile                       # Build targets
├── README.md                      # Project documentation
├── .gitignore                     # Git ignore rules
├── go.mod                         # Go module
└── go.sum                         # Dependency checksums
```

### Features Implemented

#### 1. Wallet Subsystem
- `Signer` interface for pluggable signing backends
- `KeystoreSigner` using go-ethereum's encrypted keystore
- Account creation, import (private key), and listing
- Secure password input (no echo)
- Storage: `~/.clifi/keystore/`

#### 2. Multi-Chain Client
- Supports: Ethereum, Base, Arbitrum, Optimism, Polygon
- Testnets: Sepolia, Base Sepolia
- Connection pooling with automatic RPC failover
- Native balance queries
- ERC20 token balance queries

#### 3. Agent Loop (Amp-style)
- Stateless LLM, conversation state maintained locally
- Tool registry with JSON schema definitions
- Tool dispatch and result handling
- Automatic tool call loop until completion

#### 4. Tools Implemented
| Tool | Description |
|------|-------------|
| `get_balances` | Native token balances across chains |
| `get_token_balance` | ERC20 token balance |
| `list_wallets` | List local keystore accounts |
| `get_chain_info` | Chain details (ID, currency, explorer) |
| `list_chains` | All supported chains |

#### 5. CLI Commands
| Command | Description |
|---------|-------------|
| `clifi` | Start interactive REPL (runs setup wizard on first run) |
| `clifi setup` | Run the setup wizard |
| `clifi wallet create` | Create new wallet |
| `clifi wallet import` | Import from private key |
| `clifi wallet list` | List wallets |
| `clifi portfolio` | Show balances |

#### 6. Chat REPL (Bubbletea)
- Rich terminal UI with colors
- Spinner during LLM calls
- Chat history with scrolling viewport
- Commands: `/help`, `/clear`, `/quit`

#### 7. Multi-Provider Auth System
- **Provider Interface**: Common interface for all LLM providers
- **5 Providers Supported**:
  | Provider | Auth | Env Var |
  |----------|------|---------|
  | Anthropic | API Key | `ANTHROPIC_API_KEY` |
  | OpenAI | API Key | `OPENAI_API_KEY` |
  | Venice | API Key | `VENICE_API_KEY` |
  | GitHub Copilot | OAuth | `GITHUB_TOKEN` |
  | Google Gemini | API Key | `GOOGLE_API_KEY` |

- **Auth Priority**: Env vars → config file → stored auth.json
- **Credential Storage**: `~/.clifi/auth.json` (0600 permissions)
- **Auth CLI Commands**:
  | Command | Description |
  |---------|-------------|
  | `clifi auth connect` | Connect to a provider |
  | `clifi auth list` | List connected providers |
  | `clifi auth disconnect` | Remove provider credentials |
  | `clifi auth default` | Get/set default provider |
  | `clifi auth test` | Test provider connection |

#### 8. Onboarding Wizard
- **First-run detection**: Automatically detects if setup is needed
- **Polished TUI**: Bubbletea-based wizard with lipgloss styling
- **Two-step flow**:
  1. Provider selection + API key input (with validation)
  2. Wallet creation (optional: create/skip)
- **Smart step skipping**: Skips already-configured steps
- **Non-TTY handling**: Prints env var instructions in CI/scripts
- **Re-run support**: `clifi setup` command to reconfigure
- **Polish (NEW)**:
  - `bubbles/progress` animated progress bar
  - `bubbles/textinput` for API key and password (with masked input)
  - Environment key detection ("Found ANTHROPIC_API_KEY!")
  - Provider descriptions inline ("Best reasoning & tool use")
  - Wallet benefits explained ("Check balances, Send crypto, DeFi")
  - Actionable error messages (auth, network, rate limit)
  - Quick start tips on completion screen
  - Import wallet disabled with clear "coming soon" message

---

## Dependencies

```go
require (
    github.com/spf13/cobra v1.8.1
    github.com/spf13/viper v1.19.0
    github.com/charmbracelet/bubbletea v1.3.10
    github.com/charmbracelet/bubbles v0.21.0
    github.com/charmbracelet/lipgloss v1.1.0
    github.com/ethereum/go-ethereum v1.14.12
    github.com/liushuangls/go-anthropic/v2 v2.17.0
    github.com/sashabaranov/go-openai v1.41.2      // NEW: OpenAI SDK
    github.com/google/generative-ai-go v0.20.1    // NEW: Gemini SDK
    golang.org/x/term v0.27.0
)
```

---

## How to Run

```bash
# Build
make build

# Set API key
export ANTHROPIC_API_KEY=your-key

# Run interactive mode
./bin/clifi

# Or use commands directly
./bin/clifi wallet create
./bin/clifi portfolio --chains ethereum,base
```

---

## Architecture Notes

### Agent Loop Flow
```
User Input
    ↓
Add to conversation
    ↓
Send to Claude (with tools)
    ↓
┌─────────────────────────────┐
│ While response has tool_use │
│   ↓                         │
│ Execute tool                │
│   ↓                         │
│ Send result to Claude       │
└─────────────────────────────┘
    ↓
Return text response
    ↓
Add to conversation
```

### Safety Model (Planned)
```
Tool Call → Policy Check → Simulation → Human Confirm → Sign → Verify
```

Currently read-only. Safety gates will be added in Phase 3.

---

## Remaining Work

### Phase 3: Safe Signing
- [ ] `TransactionIntent` struct
- [ ] Policy engine (spend limits, allowlists)
- [ ] Simulation gate (`eth_call`)
- [ ] Confirmation gate (explicit yes/no)
- [ ] `send` tool (native transfers)
- [ ] `approve` tool (ERC20 approvals)
- [ ] `verify` tool (check receipts)

### Phase 4: Swap Primitive
- [ ] Swap connector interface
- [ ] 0x or 1inch integration
- [ ] Slippage controls
- [ ] `swap` tool

### Phase 5: Bridge Primitive
- [ ] Bridge connector interface
- [ ] Across or similar integration
- [ ] Cross-chain state tracking

### Phase 6: Perps
- [ ] Hyperliquid or similar integration
- [ ] Risk controls (TP/SL, leverage caps)

### Phase 7: Plugin SDK
- [ ] Public connector interface
- [ ] Documentation
- [ ] Example connectors

---

## Key Files Reference

| File | Purpose |
|------|---------|
| `internal/agent/loop.go` | Core agent orchestration (multi-provider) |
| `internal/agent/tools.go` | Tool registry and handlers |
| `internal/llm/provider.go` | Provider interface + registry |
| `internal/llm/anthropic.go` | Anthropic Claude provider |
| `internal/llm/openai.go` | OpenAI provider |
| `internal/llm/gemini.go` | Google Gemini provider |
| `internal/auth/auth.go` | Auth manager (priority resolution) |
| `internal/auth/store.go` | Credential storage (auth.json) |
| `internal/cli/auth.go` | Auth CLI commands |
| `internal/cli/setup.go` | Setup command |
| `internal/setup/wizard.go` | Onboarding wizard TUI |
| `internal/setup/detect.go` | First-run detection |
| `internal/wallet/keystore.go` | Wallet encryption/signing |
| `internal/chain/client.go` | Multi-chain RPC |
| `internal/cli/repl.go` | Bubbletea chat UI |
| `internal/safety/gates.go` | (TODO) Safety gates |

---

## Testing

```bash
# Run tests
make test

# Build for current platform
make build

# Build for all platforms
make build-all
```

---

## Notes

- The project compiles and runs successfully
- REPL works with any configured LLM provider (not just Anthropic)
- Use `clifi auth connect <provider>` or set env var to configure providers
- Wallet commands work without LLM API key
- Portfolio command queries live RPC endpoints
- All state-changing operations are NOT YET IMPLEMENTED (safety first)

### Supported Providers
```bash
# Anthropic
export ANTHROPIC_API_KEY=sk-ant-...

# OpenAI
export OPENAI_API_KEY=sk-...

# Venice (uncensored, OpenAI-compatible)
export VENICE_API_KEY=...

# GitHub Copilot (requires Copilot subscription)
export GITHUB_TOKEN=$(gh auth token)

# Google Gemini
export GOOGLE_API_KEY=...
```

---

## Session End State

- **Build**: ✅ Passing
- **Phase 1**: ✅ Complete (Wallet + Read Primitives)
- **Phase 2**: ✅ Complete (Agent Loop + REPL)
- **Multi-Provider Auth**: ✅ Complete (5 providers: Anthropic, OpenAI, Venice, Copilot, Gemini)
- **Onboarding Wizard**: ✅ Complete (first-run setup flow)
- **Phase 3**: 🔲 Not started (Safe Signing)

### What's New This Session
- Researched opencode auth patterns for multi-provider support
- Implemented Provider interface with common ChatRequest/ChatResponse
- Added 5 LLM providers with BYOK authentication
- Created auth CLI (`clifi auth connect/list/disconnect/default/test`)
- Auth priority: env vars → config file → stored auth.json
- Credentials stored in `~/.clifi/auth.json` with 0600 permissions
- **NEW: Onboarding wizard** - polished first-run experience:
  - First-run detection with smart step skipping
  - Provider selection with arrow keys
  - API key input with masked display + validation
  - Optional wallet creation
  - `clifi setup` command for reconfiguration

Ready to continue with Phase 3 (Safe Signing) in the next session.
