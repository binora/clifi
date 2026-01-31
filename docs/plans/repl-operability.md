# Plan: REPL Operability + Hot-Swap

## Goals
- Enable full provider/model control from the REPL (/provider, /model, /auth, /logout, /status).
- Allow hot-swapping providers/models without restart; conversation resets appropriately.
- Improve responsiveness (timeouts/contexts) so the REPL doesn’t hang on RPC/tool calls.

## Workstream 1: Agent/Provider API
- Add SetProvider(CreateProvider) entry point to agent that reuses auth.Manager/env/config/auth.json.
- Ensure SetModel/SetProvider reset conversation and track the actually selected model (not just default).
- Export provider factory helper for REPL to construct providers on demand.

## Workstream 2: Context & Timeouts
- Remove context.Background in tool handlers; plumb REPL ctx with timeouts to RPC calls.
- Add sensible per-request deadlines to prevent hangs when switching/testing.

## Workstream 3: REPL Commands
- Implement /provider (list/switch), /model (list/switch), /auth (connect provider), /logout (clear creds, exit), /status (show provider/model/wallet/chains).
- Surface errors inline; add a lightweight status message on successful switch.

## Workstream 4: Auth Diagnostics
- Update auth test to make a minimal live probe per provider and guard short keys.
- Reuse diagnostics in REPL /status or after /auth to confirm connectivity.

## Workstream 5: Polish & Tests
- Update REPL help text; docs if needed.
- Add tests for command handlers where practical (cli/repl command parsing, agent SetProvider/SetModel behavior).

## Risks / Watchouts
- Provider re-init failure states (bad key, missing env) should not leave the REPL unusable—fallback to previous provider.
- Avoid leaking conversations across provider/model switches; always reset on switch.
- Timeouts must be tuned to avoid flakiness on slower public RPCs.
