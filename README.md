# portasplit-monitor

A small Go service that monitors stock of the **Midea PortaSplit** portable AC
across several German retailers and pushes a [Pushover](https://pushover.net)
notification whenever availability is found. Already-notified results are
tracked in a local SQLite database so you never get spammed with duplicate
alerts, while a fresh restock (after an item goes out of stock) triggers a new
notification. Offers above `PRICE_MAX` are ignored.

## How it works

1. Every `CHECK_INTERVAL`, each enabled **source** is polled.
2. Each source returns the set of currently in-stock offerings.
3. For any offering not yet recorded, a Pushover message is sent and the result
   is stored in SQLite (dedup key = source + store + product).
4. Offerings still in stock just refresh their `last_checked_at`.
5. When an offering goes out of stock its record is removed, so the next
   restock produces a fresh alert.

## Sources

| Source | What it polls |
| --- | --- |
| `braucheklima` | `https://braucheklima.de/api/availability` (aggregated, all stores) |
| `obi` | `https://www.obi.de/api/pdp/v1/availability/<id>` (single product) |

> Note: `braucheklima.de` sits behind Cloudflare and blocks datacenter IPs
> (403). Either run it from a residential network (e.g. the homelab cluster),
> or set `FLARESOLVERR_URL` to route the feed through a
> [FlareSolverr](https://github.com/FlareSolverr/FlareSolverr) proxy that solves
> the Cloudflare challenge with a real browser. The Helm chart deploys
> FlareSolverr and wires `FLARESOLVERR_URL` automatically (toggle via
> `flaresolverr.enabled`). `obi` works from anywhere.

Sources implement the `model.Source` interface, so adding another retailer is
localised to one file — see **Adding a source** below.

## Configuration

All settings come from environment variables, optionally supplied via a `.env`
file. Copy `.env.example` to `.env` and fill in your Pushover credentials.

The only required values are `PUSHOVER_TOKEN` and `PUSHOVER_USER`. Everything
else has sensible defaults.

## Run locally

```bash
cp .env.example .env   # then edit PUSHOVER_TOKEN / PUSHOVER_USER
make run
```

## SSH + systemd host (deploy.sh)

`deploy/deploy.sh` + `deploy/setup.sh` install the binary under
`/opt/portasplit-monitor` on a remote Linux host and run it as a
`portasplit-monitor` systemd service. This path is independent of the
Kubernetes deployment (which uses the Helm chart + Argo).

```bash
REMOTE=user@host make deploy
```

## Adding a source

1. Implement `model.Source` (`Name() string` + `Check(ctx) ([]Availability, error)`)
   in a new file under `internal/source/`.
2. In `cmd/portasplit-monitor/main.go`, construct it and append to the
   `sources` slice (gated by its own `*_ENABLED` config flag).
3. Map every distinct in-stock result to a stable `Availability.Key` so the
   dedup store can track it.

`Availability` carries: `Source`, `StoreName`, `ProductName`, `Stock`,
`Price`, `URL`, `Location`, `Key`.
