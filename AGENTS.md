# Project Agent Instructions

## Core Principles

- We want to make the product that people love.
- We will never compromise on performance as a first class citizen.
- We will ensure at least 85% test coverage.
- Comments should explain WHY, not WHAT.
- Build a beautiful and performant agent that becomes the default for locked-in degens.
- After completing any plan, run `make fmt`, `make lint`, and `make test`.

## Little Proofs

1. Monotonicity/Immutability - Prefer one-directional state changes; immutable objects cannot be corrupted.
2. Pre/Post-conditions - Define what must be true before and after functions run.
3. Invariants - Identify what must always remain true (like debits = credits in accounting).
4. Isolation - Minimize "blast radius" of changes with structural firewalls.
5. Induction - For recursion, prove base case + inductive step.

Core insight: Judge code quality by how easily you can reason about its correctness.
