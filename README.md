# product-monitor

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

Every source takes a **list** in `config.yaml`, so one source can watch several
products (multiple `urls`, `productIDs`, `products`, or `storeIDs`).

| Source | What it polls | Config | Channels |
| --- | --- | --- | --- |
| `braucheklima` | aggregated feed, ~1200 physical stores | `url` + `products` | in-store only |
| `obi` | `obi.de` availability API, per postal code | `productIDs` + `postalCodes` | online + in-store |
| `mediamarkt` | MediaMarkt product pages | `urls` | online |
| `euronics` | Euronics product pages | `urls` | online |
| `globus` | Globus Baumarkt product pages | `urls` | online |
| `amazon` | Amazon product pages, buybox check | `urls` | online |
| `bauhaus` | Bauhaus product pages | `urls` | online |
| `hagebau` | Hagebau product pages | `urls` | online |
| `hornbach` | Hornbach product pages | `urls` | online |
| `toom` | toom product pages | `urls` | online |
| `solarprofi` | solarprofi-24.de product pages | `urls` | online |
| `galaxus` | galaxus.de product pages (Akamai/CAPTCHA, needs FlareSolverr) | `urls` | online |
| `solario24` | solario24.com product pages | `urls` | online |
| `evolarshop` | evolarshop.de product pages | `urls` | online |
| `bueromarkt` | bueromarkt-ag.de pages (Incapsula, prefers FlareSolverr) | `urls` | online |
| `expert` | expert.de price/stock JSON API | `urls` + `storeId` | online |
| `prosatech` | prosatech.de product pages | `urls` | online |
| `tado` | shop.tado.com product pages | `urls` | online |
| `solarhandel24` | solarhandel24.de product pages | `urls` | online |
| `schwabklima` | schwab-klima.de product pages | `urls` | online |
| `grz` | grz-haustechnik.de pages, delivery lead time | `urls` | online |
| `selfio` | selfio.de product pages | `urls` | online |
| `klimavertrieb` | klima-vertrieb.de product pages | `urls` | online |
| `groupsumi` | groupsumi.de product pages | `urls` | online |
| `weinmannschanz` | weinmann-schanz.de (B2B, no price) | `urls` | online |
| `talentking` | talent-king.de product pages | `urls` | online |
| `heizungbilliger` | heizung-billiger.de (Cloudflare, needs FlareSolverr) | `urls` | online |
| `tecedo` | tecedo.de product pages | `urls` | online |
| `mediadeal` | mediadeal.de product pages | `urls` | online |
| `klimafy` | klimafy.de product pages | `urls` | online |
| `entratek` | entratek-shop.de product pages | `urls` | online |
| `bobselektro` | bobselektro.de, delivery-badge check | `urls` | online |
| `grsolar` | gr-solar.de product pages | `urls` | online |
| `bauhaus-store` | Bauhaus `/api/purchasability`, per product×store | `productIDs` + `stores` | in-store |

Online retailers each have their own source (direct product-page check); the
`braucheklima` feed contributes physical-store stock only, so online stores are
not double-counted. Most product-page sources read the `schema.org` JSON-LD
availability. A few need custom handling: Amazon uses its buybox add-to-cart
button, `expert` calls a price/stock JSON API, `grz` reads the delivery lead time,
and `bueromarkt` reads its "sofort lieferbar" / "derzeit nicht verfügbar" status.

`bauhaus-store` checks a Bauhaus store's pickup stock via the
`/api/purchasability` endpoint. Its result is store-targeted, so it bypasses the
`localPLZPrefixes` filter.

### Pre-orders

A result can be flagged as a **pre-order** when it is orderable but not immediately
in stock (schema.org `PreOrder`/`BackOrder`, a German "Vorbestellung" note, expert's
`PREORDER`/`ORDER_IN_ADVANCE` state, or a long delivery lead time such as
grz-haustechnik's `Lieferzeit N Werktage`). The Pushover message then carries a
"Vorbestellung / lange Lieferzeit" line so a pre-order is not mistaken for stock on hand.

### Online vs in-store

Each result is tagged as **online** (shipped/delivery) or **in-store**
(physically stocked for pickup); the Pushover message states which. In-store
results are filtered to nearby stores via `localPLZPrefixes` (e.g. `["36"]` for
the Fulda region) — a store is kept only if its postal code starts with one of
the listed prefixes. An empty list disables the filter; online is never filtered.

Sources implement the `model.Source` interface, so adding another retailer is
localised to one file — see **Adding a source** below.

## Metrics

The service exposes Prometheus metrics at `/metrics` (listen address
`METRICS_ADDR`, default `:8080`), with one series per source:

| Metric | Type | Description |
| --- | --- | --- |
| `product_source_up` | gauge | last check succeeded (1) or failed (0) |
| `product_source_last_check_timestamp_seconds` | gauge | unix time of last check |
| `product_source_available` | gauge | in-stock offerings from last check |
| `product_source_stock` | gauge | total units from last check |
| `product_source_min_price_euros` | gauge | lowest known price (when available) |
| `product_source_checks_total` | counter | checks by `result` (`success`/`error`) |
| `product_source_notifications_total` | counter | notifications sent |

The Helm chart ships a headless `-metrics` Service and a `ServiceMonitor`
(`serviceMonitor.enabled`, default on) so the Prometheus Operator scrapes it
automatically. A Grafana dashboard lives in the `solum` repo under
`kube-prometheus-extras/dashboards/product-monitor.json`.

## Configuration

All non-secret settings live in a YAML file (`config.yaml`, path via `-config`,
default `config.yaml`). There are **no built-in defaults** — every value comes
from the file, so start from `config.example.yaml` (a complete, working config)
and edit it.

Config is **product-centric**: define one or more `products`, each with a `name`
(shown in the notification title) and a `sources` map. A source is checked only
if it appears under a product's `sources` — omit it to skip it (there is no
`enabled` flag). The same retailer can be listed under several products. The
`Config` column in the table above is the key to use under `sources.<retailer>`.

**Secrets stay in the environment** (or a `.env` file): only `PUSHOVER_TOKEN` and
`PUSHOVER_USER`, both required. Nothing else reads the environment.

## Run locally (Go)

```bash
cp .env.example .env               # PUSHOVER_TOKEN / PUSHOVER_USER
cp config.example.yaml config.yaml # complete working config; edit as needed
make run
```

## Run with Docker Compose

```bash
cp .env.example .env               # PUSHOVER_TOKEN / PUSHOVER_USER
cp config.example.yaml config.yaml
```

Then edit `config.yaml` for the container: set `dbPath: /data/product-monitor.db`.
It is mounted read-only into the monitor container. Finally:

```bash
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
# or: REMOTE=user@host INSTALL_DIR=/srv/product-monitor ./deploy/deploy.sh
```

The repo is synced to `/opt/product-monitor` (override with `INSTALL_DIR`).
Your local `.env` and `config.yaml` are uploaded only on the first deploy; later
runs leave the server-side copies and the database volume untouched.

## Adding a source

1. Implement `model.Source` (`Name() string` + `Check(ctx) ([]Availability, error)`)
   in a new file under `internal/source/`.
2. In `cmd/product-monitor/main.go`, construct it and append to the
   `sources` slice (gated by its own config flag), and add the field to
   `internal/config` + `config.example.yaml`.
3. Map every distinct in-stock result to a stable `Availability.Key` so the
   dedup store can track it.

`Availability` carries: `Source`, `StoreName`, `ProductName`, `Stock`,
`Price`, `URL`, `Location`, `Key`.
