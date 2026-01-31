# TODO

## Critical Path (highest impact)
- [ ] Safe signing pipeline: intent struct, policy checks (spend limits/allowlists), simulation via `eth_call`, explicit confirmation gates, nonce management, gas estimation/fee display.
- [ ] Implement send/approve: native sends, ERC20 sends, ERC20 approvals; wire through tools + CLI with prompts.
- [ ] EIP-712 correctness: build proper typed-data hashing/signing; replace current raw-hash stub.
- [ ] Receipts/logging: persist receipts (SQLite path in config), add `get_receipt`/`wait_receipt` tool + CLI, surface status in REPL.
- [ ] REPL operability: `/model`, `/provider`, `/auth`, `/logout`, `/status`; agent hot-swap (SetProvider/SetModel without restart); export provider factory for REPL.
- [ ] Streaming + responsiveness: enable streaming for Anthropic/OpenAI; propagate contexts/timeouts through tools (no `context.Background()` in handlers); add RPC call timeouts.
- [ ] Auth diagnostics: make `auth test` hit provider minimally and guard short keys from slicing.

---

## High Impact
- [ ] Observability and resilience: log RPC failover choices, tool errors, and LLM cost; add per-chain timeout/backoff strategy.
- [ ] Wallet UX/safety: mnemonic import; hardware wallet path (Ledger/Trezor); multi-wallet context and selection; encrypt auth.json at rest; spend-limit policies.
- [ ] Conversation/session UX: conversation persistence between sessions; status panel for active provider/model/wallet/chains.

---

## Medium Impact (primitives)
- [ ] Swap primitive: DEX integration (Uniswap v3, etc.), quote fetch/compare, slippage protection, multi-hop routing, token price lookups.
- [ ] Bridge primitive: cross-chain bridge integration, fee comparison, transaction status tracking across chains.
- [ ] Perps integration: perpetuals interface, position management, risk display.
- [ ] Plugin SDK: plugin interface, discovery/loading, community registry.

---

## Improvements & Infra
- [ ] Integration tests with mocked testnet RPCs; CLI command coverage (internal/cli).
- [ ] Release workflow (goreleaser/GitHub releases), Homebrew formula, Docker image.
