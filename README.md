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

| Source | What it polls | Channels |
| --- | --- | --- |
| `braucheklima` | `https://braucheklima.de/api/availability` (aggregated feed, ~1200 physical stores) | in-store only |
| `obi` | `https://www.obi.de/api/pdp/v1/availability/<id>` (single product) | online + in-store |
| `mediamarkt` | MediaMarkt product page (`MEDIAMARKT_URL`) | online |
| `euronics` | Euronics product page (`EURONICS_URL`) | online |
| `globus` | Globus Baumarkt product page (`GLOBUS_URL`) | online |
| `amazon` | Amazon product page, buybox check (`AMAZON_URL`) | online |
| `bauhaus` | Bauhaus product page (`BAUHAUS_URL`) | online |
| `hagebau` | Hagebau product page (`HAGEBAU_URL`) | online |
| `hornbach` | Hornbach product page (`HORNBACH_URL`) | online |
| `toom` | toom product page (`TOOM_URL`) | online |
| `bauhaus-store` | Bauhaus `/api/purchasability` for one store (`BAUHAUS_STORE_ID`, default Frankfurt) | in-store |

Online retailers each have their own source (direct product-page check); the
`braucheklima` feed now contributes physical-store stock only, so online stores
are not double-counted. Online product-page sources rely on `FlareSolverr`
(several retailers are anti-bot protected or JS-rendered) and most use the
`schema.org` JSON-LD availability; Amazon has no such marker, so its buybox
add-to-cart button is used and it reports no price.

`bauhaus-store` checks one Bauhaus store's pickup stock via the
`/api/purchasability` endpoint. That endpoint is XHR-only behind Cloudflare, so a
FlareSolverr session (clearance cookie + user agent) is harvested and the API is
called directly. This requires the monitor and FlareSolverr to share an egress IP
(the clearance cookie is IP-bound) — true for the Docker Compose / single-host
setup. Its result is store-targeted, so it bypasses the `LOCAL_PLZ_PREFIXES`
filter.

### Online vs in-store

Each result is tagged as **online** (shipped/delivery) or **in-store**
(physically stocked for pickup); the Pushover message states which. In-store
results are filtered to nearby stores via `LOCAL_PLZ_PREFIXES` (default `36`, the
Fulda region) — a store is kept only if its postal code starts with one of the
listed prefixes. Online results are never filtered.

Sources implement the `model.Source` interface, so adding another retailer is
localised to one file — see **Adding a source** below.

## Metrics

The service exposes Prometheus metrics at `/metrics` (listen address
`METRICS_ADDR`, default `:8080`), with one series per source:

| Metric | Type | Description |
| --- | --- | --- |
| `portasplit_source_up` | gauge | last check succeeded (1) or failed (0) |
| `portasplit_source_last_check_timestamp_seconds` | gauge | unix time of last check |
| `portasplit_source_available` | gauge | in-stock offerings from last check |
| `portasplit_source_stock` | gauge | total units from last check |
| `portasplit_source_min_price_euros` | gauge | lowest known price (when available) |
| `portasplit_source_checks_total` | counter | checks by `result` (`success`/`error`) |
| `portasplit_source_notifications_total` | counter | notifications sent |

The Helm chart ships a headless `-metrics` Service and a `ServiceMonitor`
(`serviceMonitor.enabled`, default on) so the Prometheus Operator scrapes it
automatically. A Grafana dashboard lives in the `solum` repo under
`kube-prometheus-extras/dashboards/portasplit.json`.

## Configuration

All settings come from environment variables, optionally supplied via a `.env`
file. Copy `.env.example` to `.env` and fill in your Pushover credentials.

The only required values are `PUSHOVER_TOKEN` and `PUSHOVER_USER`. Everything
else has sensible defaults.

## Run locally (Go)

```bash
cp .env.example .env   # then edit PUSHOVER_TOKEN / PUSHOVER_USER
make run
```

## Run with Docker Compose

```bash
cp .env.example .env   # then edit PUSHOVER_TOKEN / PUSHOVER_USER
docker compose up -d --build
```

`--build` is required on the first run (and after code changes), since the
`monitor` image is built from the local `Dockerfile` rather than pulled. Useful
follow-ups:

```bash
docker compose logs -f          # tail logs
docker compose up -d --build    # redeploy after changes
docker compose down             # stop (add -v to also drop the DB volume)
```

## Deploy to a host (Docker Compose)

`deploy/deploy.sh` syncs the repo to a remote Linux host over SSH and runs
`docker compose up -d --build` there, so the host needs Docker with the compose
plugin and an SSH user that can run Docker. This path is independent of the
Kubernetes deployment (which uses the Helm chart + Argo).

```bash
REMOTE=user@host make deploy
# or: REMOTE=user@host INSTALL_DIR=/srv/portasplit-monitor ./deploy/deploy.sh
```

The repo is synced to `/opt/portasplit-monitor` (override with `INSTALL_DIR`).
Your local `.env` is uploaded only on the first deploy; later runs leave the
server-side `.env` and the database volume untouched.

## Adding a source

1. Implement `model.Source` (`Name() string` + `Check(ctx) ([]Availability, error)`)
   in a new file under `internal/source/`.
2. In `cmd/portasplit-monitor/main.go`, construct it and append to the
   `sources` slice (gated by its own `*_ENABLED` config flag).
3. Map every distinct in-stock result to a stable `Availability.Key` so the
   dedup store can track it.

`Availability` carries: `Source`, `StoreName`, `ProductName`, `Stock`,
`Price`, `URL`, `Location`, `Key`.
