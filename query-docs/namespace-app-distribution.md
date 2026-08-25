# namespace-app-distribution

Queries in [`dashboards/namespace-app-distribution/namespace-app-distribution.json`](../dashboards/namespace-app-distribution/namespace-app-distribution.json) — "Apica Namespace & App Distribution", one `Overview` tab, data source `apica_ascent_prometheus`.

Per-tenant view: who is sending how much. Panels with `&duration=…&step=…` are time series (`line`); the rest are instant values compared across namespaces (`bar`).

`exported_namespace` is the tenant namespace throughout. The metric's own label is `namespace`, which collides with the one Prometheus adds at scrape time, so Prometheus renames the metric's copy — grouping by plain `namespace` would return the single k8s namespace flash runs in.

---

### 1. Aggregate Messages by Namespace (30d)

```promql
round(sum by (exported_namespace) (increase(logiq_namespace_app_message_count[30d])),1)
```

Total messages ingested per tenant over the last 30 days, one bar per namespace. An instant query — a single total, not a trend.

### 2. Namespace, App distribution (EVENTS) [24h]

```promql
round(logiq:namespace_app_count_daily:increase{exported_namespace!="asmplus"},1)
```

Events ingested per namespace in the last 24 hours, bars grouped by `app`. Reads a pre-computed recording rule rather than the raw counter, so the 24h window is fixed by the rule, not by the query.

`asmplus` is excluded deliberately — see *Why asmplus is filtered out* below.

### 3. Messages by Namespace (30d)

```promql
round(sum by (exported_namespace) (increase(logiq_namespace_app_message_count[24h])),1)&duration=30d&step=1h
```

Rolling 24-hour message totals per tenant, sampled hourly across 30 days. Each point answers "how many messages in the preceding day", so this is the daily-volume trend — panel 1 is its single-number equivalent.

### 4. Namespace, App distribution (GB) [24h]

```promql
round(logiq:namespace_app_bytes_daily:increase{exported_namespace!="asmplus"}/1000000000,0.001)
```

Data volume in GB per namespace over the last 24 hours, bars grouped by `app`. The GB counterpart to panel 2, from the matching recording rule, with `asmplus` excluded for the same reason.

### 5. # Messages/sec by Namespace

```promql
round(sum by (exported_namespace) (rate(logiq_namespace_app_message_count[1h])),1)&duration=30d&step=1h
```

Message ingestion rate in messages/sec per tenant, sampled hourly over 30 days. The `[1h]` window matches the `1h` step, so each point covers the whole hour rather than a slice of it.

### 6. Aggregate GB by Namespace (30d)

```promql
round(sum by (exported_namespace) (increase(logiq_namespace_app_received_bytes[30d]))/1000000000,0.001)
```

Total GB ingested per tenant over the last 30 days — panel 1 in bytes rather than messages. Together they show whether a tenant's volume comes from many small messages or fewer large ones.

---

## Why asmplus is filtered out

`flash/asmplus/convert.go` sets the namespace to a constant `asmplus` and the app name to `checks-<check_guid>` — **one distinct `app` value per synthetic check**. A tenant with a few thousand checks therefore produces a few thousand `app` series, all inside that single namespace.

Panels 2 and 4 group by `app`, so `asmplus` alone would contribute thousands of bar segments and legend entries and bury every real tenant. Both exclude it with `{exported_namespace!="asmplus"}`.

Panels 1, 3, 5 and 6 do **not** filter it, and should not: they aggregate with `sum by (exported_namespace)`, so `asmplus` collapses to a single series like any other tenant. The problem is its app cardinality, not its volume, so it only needs excluding where `app` is a dimension.

## Open items

- **Panels 2 and 4** read the recording rules `logiq:namespace_app_count_daily:increase` and `logiq:namespace_app_bytes_daily:increase`, which are **not defined anywhere in this workspace** — they ship via Helm/Prometheus config. Their `groupBy: "app"` and `x: "exported_namespace"` are inferred from the rule names and from how Prometheus labels recording-rule output; worth confirming against the deployed rules.
- **Panels 1 and 3** are the same measure at different resolutions (30d total vs. daily trend), as are **panels 4 and 6** in GB at different windows (24h vs 30d). Intentional, but worth knowing before adding more.
- No panel caps the number of series, so a cluster with many tenants will crowd both the bar charts and the line charts. `topk(n, …)` is the usual remedy.

All 6 panels reviewed.
