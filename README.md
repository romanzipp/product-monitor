# product-monitor

A small Go service that monitors stock of the **Midea PortaSplit** portable AC
across several German retailers and pushes a [Pushover](https://pushover.net)
notification whenever availability is found. Already-notified results are
tracked in a local SQLite database so you never get spammed with duplicate
alerts, while a fresh restock (after an item goes out of stock) triggers a new
notification. Offers above `priceMax` are ignored.

> [!NOTE]
> This project is primarily developed on [Codeberg](https://codeberg.org/romanzipp/product-monitor) and only mirrored to GitHub. Please open issues and pull requests on Codeberg.

## How it works

1. Every `checkInterval`, each configured **source** is polled.
2. Each source returns the set of currently in-stock offerings.
3. For any offering not yet recorded, a Pushover message is sent and the result
   is stored in SQLite (dedup key = source + store + product).
4. Offerings still in stock just refresh their `last_checked_at`.
5. When an offering goes out of stock its record is removed, so the next
   restock produces a fresh alert.

## Configuration

Non-secret settings live in a YAML file (`-config`, default `config.yaml`); there
are no built-in defaults, so copy `config.example.yaml` and edit it. Secrets
(`PUSHOVER_TOKEN`, `PUSHOVER_USER`) come from the environment or a `.env` file.

Config is product-centric: each entry under `products` has a `name` (shown in the
notification) and a `sources` map. A source runs only if it is listed under a
product (there is no `enabled` flag), and the same retailer can appear under
several products. Source keys and their config are in [Sources](#sources).

## Run locally (Go)

```bash
cp .env.example .env                # PUSHOVER_TOKEN / PUSHOVER_USER
cp config.example.yaml config.yaml
make run
```

## Docker Compose

The Docker Compose stack also include a [Flaresolverr](https://github.com/Flaresolverr/Flaresolverr)
instance which will be automatically wired.

```bash
cp .env.example .env
cp config.example.yaml config.yaml  # set dbPath: /data/product-monitor.db
docker compose up -d --build        # --build needed on first run and after code changes
```

`docker compose logs -f` to tail, `docker compose down -v` to stop and drop the DB.

## Deploy to a host

`deploy/deploy.sh` rsyncs the repo to a remote host over SSH and runs
`docker compose up -d --build` there (needs Docker + the compose plugin).
Independent of the Kubernetes/Helm + Argo path.

```bash
REMOTE=user@host make deploy
# or: REMOTE=user@host INSTALL_DIR=/srv/product-monitor ./deploy/deploy.sh
```

`.env` and `config.yaml` are uploaded only on the first deploy; later runs leave
the server-side copies and the database volume untouched.

## Sources

Every source takes a **list**, so one product can be watched at several URLs
(multiple `urls`, `productIDs`, `products`, or `stores`).

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
| `bauhausStore` | Bauhaus `/api/purchasability`, per product×store | `productIDs` + `stores` | in-store |

Most product-page sources read the `schema.org` JSON-LD availability. A few need
custom handling: Amazon uses its buybox add-to-cart button, `expert` calls a
price/stock JSON API, `grz` reads the delivery lead time, and `bueromarkt` reads
its "sofort lieferbar" / "derzeit nicht verfügbar" status. `bauhausStore` checks
a store's pickup stock via the `/api/purchasability` endpoint; its result is
store-targeted, so it bypasses the `localPLZPrefixes` filter.

### Pre-orders

A result is flagged as a **pre-order** when it is orderable but not immediately in
stock (schema.org `PreOrder`/`BackOrder`, a German "Vorbestellung" note, expert's
`PREORDER` state, or a long delivery lead time like grz's `Lieferzeit N Werktage`).
The Pushover message then carries a "Vorbestellung / lange Lieferzeit" line.

### Online vs in-store

Each result is tagged **online** (shipped) or **in-store** (pickup); the message
states which. In-store results are filtered to nearby stores via `localPLZPrefixes`
(e.g. `["36"]` for the Fulda region): kept only if the store's postal code starts
with a listed prefix. An empty list disables the filter; online is never filtered.

## Metrics

Prometheus metrics are exposed at `/metrics` (`metricsAddr`, default `:8080`), one
`product_source_*` series per source (up, stock, available, min price, checks,
notifications). The Helm chart ships a `ServiceMonitor` for auto-scraping; a
Grafana dashboard lives in the `solum` repo at
`kube-prometheus-extras/dashboards/product-monitor.json`.

## Adding a source

1. Implement `model.Source` (`Name() string` + `Check(ctx) ([]Availability, error)`)
   in a new file under `internal/source/`.
2. Add its config to `ProductSources` in `internal/config`, wire it into the
   product loop in `cmd/product-monitor/main.go`, and add it to `config.example.yaml`.
3. Map every distinct in-stock result to a stable `Availability.Key` so the
   dedup store can track it.

`Availability` carries: `Source`, `StoreName`, `ProductName`, `Stock`, `Price`,
`URL`, `Location`, `Key`.
