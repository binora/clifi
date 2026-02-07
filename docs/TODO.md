# TODO

## Critical Path (highest impact)
- [ ] Safe signing pipeline polish: keep preview-first flow, but add nonce mgmt, explicit fee + slippage-style warnings, retry/backoff, and better REPL UX for confirm/password prompts.
- [x] Implement send/approve: native sends, ERC20 sends, ERC20 approvals; wired through agent tools with policy gates and previews.
- [ ] EIP-712 correctness: replace current raw-hash stub with proper typed-data hashing/signing (correct domain separator + struct hashing).
- [x] Receipts: persist receipts (SQLite) and expose `get_receipt`/`wait_receipt` tools; optionally surface receipt status more prominently in the REPL.
- [x] REPL operability: `/model`, `/provider`, `/auth`, `/logout`, `/status`; agent hot-swap (SetProvider/SetModel without restart).
- [ ] Streaming + responsiveness: streaming for tool-less chats exists, but enable streaming for Anthropic and (carefully) tool-call flows; propagate timeouts through tools; add RPC timeouts/backoff.
- [x] Auth diagnostics: `clifi auth test` performs real provider pings and guards short keys.
- [ ] OpenRouter + Claude tool calling reliability: ensure tool calls succeed for Anthropic models routed via OpenRouter; keep env-gated canaries/canonical scenarios.

---

## High Impact
- [ ] Observability and resilience: log RPC failover choices, tool errors, and LLM cost; add per-chain timeout/backoff strategy; surface tool-capability warnings in telemetry.
- [ ] Wallet UX/safety: mnemonic import; hardware wallet path (Ledger/Trezor); multi-wallet context and selection; encrypt auth.json at rest; spend-limit policies.
- [ ] Conversation/session UX: conversation persistence between sessions; status panel for active provider/model/wallet/chains.
- [ ] Provider architecture: unify OpenAI-compatible providers (OpenAI/OpenRouter/Copilot/Venice/etc.) behind a single adapter with provider-specific headers/options to avoid rework when adding providers.

---

## Medium Impact (primitives)
- [ ] Swap primitive: DEX integration (Uniswap v3, etc.), quote fetch/compare, slippage protection, multi-hop routing, token price lookups.
- [ ] Bridge primitive: cross-chain bridge integration, fee comparison, transaction status tracking across chains.
- [ ] Perps integration: perpetuals interface, position management, risk display.
- [ ] Plugin SDK: plugin interface, discovery/loading, community registry.

---

## Improvements & Infra
- [ ] Integration tests with mocked testnet RPCs; CLI command coverage (internal/cli).
- [x] OpenRouter live tool-call canary (env-gated) exists; expand with more scenarios (Claude + tools, receipts, send preview) and make it easy to add future scenarios.
- [ ] Release workflow (goreleaser/GitHub releases), Homebrew formula, Docker image.
