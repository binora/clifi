# Architecture

## Directory Structure

```
cmd/clifi/main.go              Entry point
internal/
  agent/                       AI agent orchestration
    loop.go                    Conversation loop, provider init, tool call handling
    tools.go                   Tool registry (get_balances, list_wallets, etc.)
    conversation.go            Conversation message types
  auth/                        LLM provider authentication
    auth.go                    Manager facade (env vars > config > auth.json)
    store.go                   Credential persistence (~/.clifi/auth.json)
    oauth.go                   OAuth flow with local callback server
    providers.go               Per-provider auth method configs
  chain/                       Blockchain connectivity
    config.go                  Chain definitions (RPC URLs, chain IDs, explorers)
    client.go                  Multi-chain RPC client with failover
    balance.go                 Native + ERC20 balance queries
  cli/                         CLI commands and REPL
    root.go                    Root command, setup check, REPL launch
    repl.go                    Interactive REPL (Bubbletea TUI)
    auth.go                    clifi auth connect/disconnect/list/default/test
    wallet.go                  clifi wallet create/import/list
    portfolio.go               clifi portfolio
  llm/                         LLM provider implementations
    provider.go                Provider interface, registry, ProviderID constants
    tools.go                   Tool JSON schema definitions
    anthropic.go               Anthropic Claude
    openai.go                  OpenAI GPT (base for OpenAI-compatible providers)
    gemini.go                  Google Gemini
    copilot.go                 GitHub Copilot (wraps OpenAI)
    venice.go                  Venice AI (wraps OpenAI)
    openrouter.go              OpenRouter (wraps OpenAI)
  setup/                       First-run onboarding wizard
    wizard.go                  Bubbletea TUI wizard (steps, views, update handlers)
    provider_step.go           Provider auth validation, OAuth flow
    wallet_step.go             Wallet creation step
    detect.go                  Setup status detection
    styles.go                  UI styles
  wallet/                      Local wallet management
    keystore.go                Encrypted keystore (go-ethereum scrypt)
    signer.go                  Transaction, message, EIP-712 signing
```

## Data Flow

```
User runs `clifi`
    |
    v
root.go: NeedsSetup(~/.clifi)?
    |           |
    | yes       | no
    v           v
wizard.go   RunREPL()
    |           |
    v           v
Save creds  agent.New() -> loads provider from auth.json
    |           |
    +---------->v
            REPL event loop (Bubbletea)
                |
                v
            User input
            /command? --> handleCommand()
            message?  --> agent.Chat()
                            |
                            v
                        LLM provider.Chat()
                            |
                            v
                        Tool calls? --> toolRegistry.ExecuteTool()
                            |               |
                            +<--------------+
                            v
                        Response to user
```

## Provider Interface

All LLM providers implement:

```go
type Provider interface {
    ID() ProviderID
    Name() string
    Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error)
    SupportsTools() bool
    Models() []Model
    DefaultModel() string
}
```

OpenAI-compatible providers (Venice, Copilot, OpenRouter) embed `*OpenAIProvider` and override `ID()`, `Name()`, `Models()`.

## Credential Resolution

```
1. Environment variable (ANTHROPIC_API_KEY, OPENAI_API_KEY, etc.)
2. OAuth token (if available for provider)
3. Stored API key (~/.clifi/auth.json)
```

## Agent Tool Loop

```go
response := provider.Chat(req)
for len(response.ToolCalls) > 0 {
    results := executeTools(response.ToolCalls)
    response = provider.ChatWithToolResults(req, results)
}
return response.Content
```

## Chain Client

- Each chain has 2 fallback RPC URLs
- Connections cached after first dial
- Chain ID verified on connect
- Balance queries use raw `eth_call` for ERC20 (balanceOf, symbol, decimals)

## Wallet Security

- go-ethereum keystore (scrypt encryption)
- Stored in `~/.clifi/keystore/`
- `KeystoreSigner` supports lock/unlock to zero key material
- Signs: transactions (EIP-155), messages (EIP-191), typed data (EIP-712)

## Tech Stack

| Dependency | Purpose |
|---|---|
| go-ethereum | Wallet, signing, RPC |
| go-anthropic | Anthropic Claude API |
| go-openai | OpenAI + compatible APIs |
| generative-ai-go | Google Gemini API |
| bubbletea/bubbles | TUI framework |
| cobra | CLI framework |

## File Layout

```
~/.clifi/
  auth.json       Credentials (0600 permissions)
  keystore/       Encrypted wallet files
  config.yaml     User configuration
```
