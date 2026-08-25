# apica-logs-overview

Queries in [`dashboards/apica-logs-overview/logs-overview.json`](../dashboards/apica-logs-overview/logs-overview.json) — "Logs Overview", one `Overview` tab, data source `apica_ascent_prometheus`.

Platform-wide ingest summary: how much is coming in, how fast, and from whom. Counters are single-number tiles; `line`/`area` panels are 30-day trends.

---

### 1. GB per hour (30d) · `line`

```promql
query=round(sum(increase(logiq_data_received_bytes[1h]))/1000000000,0.01)&duration=30d&step=1h
```

Data ingested per hour across the whole cluster, one point per hour over 30 days. Only ingest pods export this counter, so the unqualified `sum()` is a true cluster total.

### 2. Log events (24h) · `counter`

```promql
round(sum(increase(logiq_message_count[24h]))/1000000,0.001)
```

Total successfully ingested log events in the last 24 hours, in millions (`options.label` supplies the unit). `logiq_message_count` is a counter labelled by `connectionType`; the `sum()` folds all ingest protocols together.

### 3. Ingest capacity utilization · `gauge`

```promql
((sum(rate(logiq_data_received_bytes[5m]))*3600/1000000000*1)
 /(count (count by (pod) (logiq_data_received_bytes))))*100
```

Current ingest rate as a percentage of capacity. Reads as: cluster GB/hour ÷ number of ingest pods × 100.

**Capacity is hardcoded at 1 GB/hour per pod** — that is what the trailing `*1` represents. The percentage is only meaningful if that matches your real per-pod ceiling; if the true ceiling is 2 GB/h, the gauge reads double. `count(count by (pod) (…))` is the idiom for "number of distinct pods".

### 4. Messages / sec · `counter`

```promql
round(sum(rate(logiq_message_count[5m])),1)
```

Current cluster-wide message ingestion rate, averaged over the last 5 minutes.

### 5. # Messages/sec by Namespace · `line`

```promql
round(sum by (exported_namespace) (rate(logiq_namespace_app_message_count[1h])),1)&duration=30d&step=1h
```

Message rate per tenant namespace over 30 days, one line per tenant. The `[1h]` window matches the `1h` step so each point covers the whole hour. `exported_namespace` rather than `namespace` because the metric's own label collides with the scrape-time one and Prometheus renames it.

### 6. Total GB ingested (30d) · `counter`

```promql
round(sum(increase(logiq_data_received_bytes[30d]))/1000000000,0.01)
```

Single number: total data ingested over the last 30 days, in GB.

### 7. HTTP Connection Rate (6h) · `bar`

```promql
sum by (operation) (rate(logiq_client_connect_count{pod=~"logiq-flash-\\d{1,3}",connectionType="HTTP"}[5m]))&duration=6h&step=5m
```

HTTP client connect and disconnect rate on ingest pods over 6 hours, one series per `operation` (`connect` / `disconnect`). A persistent gap between the two means connections are accumulating.

`sum by (operation)` is load-bearing: `logiq_client_connect_count` carries an **`ip`** label, so one series exists per client IP. Aggregating it away is what keeps this panel from exploding.

### 8. GB per day · `area`

```promql
round(sum(increase(logiq_data_received_bytes[24h]))/1000000000,0.001)&duration=30d&step=24h
```

Data ingested per day over 30 days. The daily counterpart to panel 1, and the coarse trend behind panel 6's single total.

### 9. Connection Rate by Protocol (6h) · `line`

```promql
sum by (connectionType) (rate(logiq_client_connect_count{pod=~"logiq-flash-\\d{1,3}",operation="connect"}[5m]))&duration=6h&step=5m
```

New client connections per second on ingest pods, one line per ingest protocol — `HTTP`, `SYSLOG`, `RELP_SYSLOG`, `LUMBERJACK`, `FLUENTDFORWARD`, `SPAN`. This is the panel that makes a connection storm on a non-HTTP protocol visible; panel 7 is exactly its `HTTP` series, further split by connect vs disconnect.

`operation="connect"` keeps disconnects out, so a line is unambiguously new connections. Values are lowercase (`connect` / `disconnect`) in the source.

### 10. Messages/sec by Protocol (6h) · `line`

```promql
round(sum by (connectionType) (rate(logiq_message_count[5m])),1)&duration=6h&step=5m
```

Message ingestion rate split by protocol. Panel 4 is the same measure with all protocols summed together, so this answers the follow-up question: when the overall rate moves, which pipeline moved.

`rate([5m])` at `step=5m` means each point covers its whole interval — no sampling gap.

---

## Chart types

| panel | data shape | type |
|---|---|---|
| 2, 4, 6 | single scalar | `counter` |
| 3 | single scalar against a ceiling | `gauge` |
| 1, 5 | series over time | `line` |
| 7 | two series over time | `bar` |
| 8 | daily volume over time | `area` |
| 9, 10 | one series per protocol over time | `line` |

`counter` panels take their unit from `options.label`, not `plot.yLabel` — so `yLabel` is inert on panels 2, 4 and 6 and is left as-is rather than given a misleading value.

No gauge anywhere in this repo sets `upperLimit`; all 60 leave it `''` or absent. Panel 3 follows suit, so the gauge has no explicit 0–100 scale.

## Open items

- **Panel 3's 1 GB/hour/pod capacity is an unverified assumption** baked into the query as `*1`. Worth confirming against real per-pod ingest limits — it is the one panel here someone might alert against.
- **Panels 1, 6 and 8 are the same measure at three resolutions** (hourly trend, 30-day total, daily trend). Intentional, but adding a fourth would be redundant.
- **Panel 5 draws one line per tenant**, so it crowds on clusters with many namespaces. `topk(n, …)` would cap it at the cost of hiding the tail.
- **Panels 9 and 10 inherit panel 7's pod filter** (`logiq-flash-\\d{1,3}`). Both collectors are registered only under `if ingestNode`, so a protocol appears only if the pod serving it is an ingest node — `SPAN`, raised from the API path, may therefore be absent depending on how roles are deployed.
- **Panels 9 and 10 use a 6h window** while panels 1, 5 and 8 use 30d. Intentional: protocol mix is an operational question, volume trend is a capacity one.

All 10 panels reviewed.
