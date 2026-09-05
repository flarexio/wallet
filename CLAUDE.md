# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this is

FlareX Wallet — a **custodial Solana wallet**. Users never hold a private key: the server derives an ed25519 keypair per user from a Google Cloud KMS key, and every signature is gated behind a passkey (WebAuthn) assertion.

Three deliverables live in one repo:

| Component | Path | Ships as |
|---|---|---|
| Backend API (Go 1.24, Gin + go-kit) | repo root, `cmd/wallet` | `flarexio/wallet` Docker image |
| Wallet PWA (Angular 18 + Material) | `app/` | `flarexio/wallet-app` Docker image (Caddy static server) |
| dApp SDK (TypeScript) | `wallet-adapter/` | npm `@flarex/wallet-adapter` |

## Commands

```bash
# backend
go build ./...
go test ./...
go test -run TestSignTransaction ./            # single test
go test -v ./keys/                             # single package
go run cmd/wallet/main.go --path ./local-conf --port 8080

# frontend (from app/)
npm start          # ng serve on :4200
npm run build
npm test           # karma/jasmine

# sdk (from wallet-adapter/)
npm run build      # tsc -> lib/, which IS committed
```

`TestGoogleKeyService` skips itself unless `GOOGLE_APPLICATION_CREDENTIALS` is set — it hits real KMS. Tests read `config.example.yaml`, so changing that file breaks `conf` and `keys` tests.

CI (`.github/workflows/build.yml`) only builds and tests the Go side. Tagging `v*` triggers `release.yml`, which builds both Docker images and pushes multi-arch manifests to Docker Hub.

## Runtime layout

The server reads everything from one directory — `--path` / `$WALLET_PATH`, default `$HOME/.flarex/wallet`, stored in the package-level `conf.Path`:

- `config.yaml` — see `config.example.yaml`
- `permissions.json` — OPA data document; `policy/data.json` in this repo is the template. The Rego module itself (`rbac.rego`) is embedded in `github.com/flarexio/core/policy`, not here.
- `id.json` — Solana keypair, when the solana persistence driver is used
- badger data dir, named by `persistence.badger.name`

`conf.Path` doubles as the default for badger/solana `path` fields, which is why those configs have custom `UnmarshalYAML` — the zero value gets backfilled at decode time.

## Key derivation (`account/account.go`)

```
subject + random UUID salt → KMS ed25519 Signature() → first 32 bytes as seed
                           → ed25519 private key → Solana address
```

The KMS master key never leaves Google. The account record persists `Salt` and `KeyVersion` alongside the derived key, so an account is reproducible from the KMS key. `keys.Service.Key(ver...)` is 1-indexed against KMS crypto key versions; omitting the version means "latest".

## Signing is always two-phase

`service.go` exposes Initialize/Finalize pairs for both messages and transactions, and the HTTP layer maps them onto POST/PUT of the same URL:

1. **POST** `/accounts/:user/{message,transaction}-signatures` → asks Hanko passkeys for a `CredentialAssertion`, *signs immediately anyway*, and parks the result in the repository under a transaction UUID with a 120s TTL.
2. **PUT** same URL → verifies the assertion, reads the transaction ID out of the returned JWT's `trans` claim, and pops the parked signature (`RemoveTransaction` deletes on read).

So the signature exists server-side before the user approves; the passkey gates *release*, not creation. Keep that invariant in mind when touching `CacheTransaction` / `RemoveTransaction`.

Transaction UUIDs are **client-supplied**, so the parked record carries its owning `Subject` and both sides of the cache are keyed by it (`tx:<subject>:<uuid>`). That is what stops one account from overwriting another's pending transaction, and `RemoveTransaction` re-checks `Subject` on read as a backstop. Anything that caches or pops a transaction must keep the subject in the key — dropping it silently reintroduces a cross-account collision.

`SignMessageEndpoint` / `SignTransactionEndpoint` (the direct, non-passkey variants) exist in `endpoint.go` but are deliberately **not wired up** in `main.go`.

## Cross-window session channel

`CreateSession` signs the payload with the server's ed25519 session key (`keys.session.key` in config), base58-encodes the signature, and uses that as the session ID. Sessions live in an in-memory `map[string][]*Session` bucketed by the **first two characters** of that ID — so lookups in `SessionData`/`AckSession` must index the same way.

- `POST /sessions` is an **SSE stream** (`c.Stream`): emits `session` first, then blocks on the channel and emits `data` (base64) on ack, or `fail` on 120s timeout.
- `GET /sessions/:session` + `POST /sessions/:session/ack` are what the wallet PWA calls.
- These three routes are **unauthenticated**; everything under `/accounts/:user` is not.

## Auth chain

`transport/http/token.go` fetches an Ed25519 JWKS from `jwt.jwksURL` and refreshes every 5 minutes into a mutex-guarded cache. `transport/http/auth.go` then parses the bearer token and hands `{domain, action, who_flags, claims, object}` to the OPA policy. `JWTAuthorizator("wallet::accounts.get", http.Owner)` splits the rule on `.` — a rule string without a dot will panic at wiring time. `who` flags are a bitmask (`Owner|Group|Others|Admin|All`) mirrored in `policy/data.json`'s `who_enum`.

Three distinct actions exist in the `wallet::accounts` domain, so a role can be granted reads without signing rights:

| Routes | Action |
|---|---|
| `GET /accounts/:user` | `get` |
| `POST`/`PUT` `/accounts/:user/message-signatures` | `sign_message` |
| `POST`/`PUT` `/accounts/:user/transaction-signatures` | `sign_transaction` |

Adding or renaming a rule string in `main.go` means updating `policy/data.json` **and** every deployed `permissions.json` — a missing action fails closed with 403. `policy/policy_test.go` evaluates the real data file and will catch drift between the two.

## dApp ↔ wallet transport

`wallet-adapter` is not an injected browser extension. `FlarexWallet` (`src/wallet.ts`) opens a popup at the wallet origin, waits for a `WALLET_READY` postMessage, then exchanges `WalletMessage` / `WalletMessageResponse` envelopes (`src/message.ts`, types `TRUST_SITE` / `SIGN_MESSAGE` / `SIGN_TRANSACTION`) with strict `event.origin` checks. `FlarexWalletAdapter` wraps that in the standard `BaseMessageSignerWalletAdapter` interface and probes `/wallet/v1/health` to decide its `WalletReadyState`.

Payload classes carry hand-written `serialize()`/`deserialize()` because `Uint8Array` doesn't survive structured cloning the way the API's base64 wire format needs. When you change a payload shape, update both sides *and* rebuild `lib/` — it's committed and is what npm consumers get.

## Persistence

`account.Repository` has three drivers behind `persistence.NewAccountRepository`:

- `badger` — local KV, the only fully working one
- `solana` — on-chain; **every method returns "not implemented"**
- `composite` — reads try cache then main; writes go to main synchronously and backfill cache in a goroutine. Transactions (the signing cache) only ever touch the cache repo.

`config.example.yaml` defaults to composite with solana as main, so out of the box the write path fails. Use `driver: badger` for local development.

**Accounts are only as durable as the repository.** The private key is derived from the KMS key plus a random per-account salt, and that salt lives nowhere but the repository — losing the badger store loses the funds, KMS access notwithstanding. There is no export or restore path yet; see the TODO in `service.findOrCreate`.

## Conventions

- Endpoints are go-kit `endpoint.Endpoint` taking a typed request struct; Gin handlers in `transport/http/` do binding, inject `Subject` from the `:user` path param, and translate errors. Non-2xx bodies are plain strings, not JSON.
- Solana transactions cross the wire as raw bytes in JSON, handled by custom `MarshalJSON`/`UnmarshalJSON` on the request/response types rather than struct tags.
- `errors.Is(err, account.ErrAccountNotFound)` is the signal for "create this wallet on first access" (`service.Wallet`); a repository that returns a different error for a missing account will silently break account creation.
