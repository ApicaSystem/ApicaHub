# apica-monitoring

Queries in [`dashboards/apica-monitoring/apica-monitoring.json`](../dashboards/apica-monitoring/apica-monitoring.json) — "Apica Cluster Monitoring", two tabs (`Overview` and `Ingest Endpoint Health`), data source `apica_ascent_prometheus`.

Panels with `&duration=…&step=…` are time series (x = timestamp) — `line` on the Overview tab, `area` on Ingest Endpoint Health. The rest are instant values: a `bar` compared across a label (pod on the Overview tab; reason, endpoint, disposition or pod on Ingest Endpoint Health), a `pie`, or a single-number `counter`, `gauge` or `size` tile.

---

## Overview tab

### 1. Ingest Memory Usage (Go Heap)

```promql
(round(go_memstats_heap_alloc_bytes{pod=~"logiq-flash-\\d{1,3}"}/1000000000,0.1))&duration=1h&step=5m
```

Live Go heap memory in GB for each ingest pod over the last hour, one line per pod. `go_memstats_heap_alloc_bytes` is the Go runtime's own gauge, so it rises and falls with garbage collection.

### 2. Back Pressure Counter

```promql
logiq_back_pressure_counter
```

Current flush backlog per pod — records batched but not yet written out. A high value means flushing is not keeping up with ingest.

### 3. Mover Operation Count

```promql
round(sum by (pod, operation) (increase(logiq_mover_count[5m])),1)
```

Mover activity in the last 5 minutes, per pod and broken down by operation type (`wr` write, `dq` dequeue, `query`, `jok`/`je` JSON ok/error, and so on). Counts the work done in the window, not a running total.

### 4. Data Ingest Rate (GB/h)

```promql
round(sum(increase(logiq_data_received_bytes[5m]))/1000000000*12,0.01)&duration=1h&step=5m
```

Total data received across the whole cluster, as GB per hour, over the last hour. Measured as bytes in a 5-minute window scaled up by 12. Only ingest pods export this counter, so the unqualified `sum()` is a true cluster total with no double counting.

### 5. Per-Pod Ingest Rate (GB/h)

```promql
round(sum by (pod)(increase(logiq_data_received_bytes[5m]))*12/1000000000,0.01)
```

The same GB/hour ingest figure as panel 4, but split per pod and shown as a current value. `sum by (pod)` folds together all ingest protocols, so each pod gets one number. Useful for spotting uneven load across ingest pods.

### 6. API Memory Usage (Go Heap)

```promql
round(go_memstats_heap_alloc_bytes{pod=~"logiq-flash-ml-\\d{1,3}"}/1000000000,0.1)&duration=1h&step=5m
```

Live Go heap memory in GB for the API/ML pods (`logiq-flash-ml-*`) over the last hour — panel 1 for the other half of the cluster.

### 7. Disk Space Available (Ingest)

```promql
round(100 * sum by (pod) (logiq_file_sizes_bytes{pod=~"logiq-flash-\\d+",fileName="free"}) / sum by (pod) (logiq_file_sizes_bytes{pod=~"logiq-flash-\\d+",fileName="all"}),0.01)
```

Free disk on each ingest pod as a percentage of that pod's own volume. `logiq_file_sizes_bytes` carries a single `fileName` label with exactly two values, `all` and `free`, both read from `statfs` on the runtime folder — so dividing one by the other gives the real figure per pod, with no assumed volume size. The `sum by (pod)` on each side exists to strip the differing `fileName` label so the division matches.

### 8. Messages Rate per Namespace

```promql
round(sum by (exported_namespace) (rate(logiq_namespace_app_message_count[5m])),1)
```

Message ingest rate in messages/sec, one line per tenant namespace, over the last hour. Shows which tenants are driving traffic; `app` and `severity` are summed away.

`exported_namespace` is deliberate, not a typo. The metric's own label is `namespace`, which collides with the `namespace` label Prometheus adds at scrape time, so Prometheus renames the metric's copy to `exported_namespace`. Grouping by plain `namespace` would return the k8s namespace flash runs in — a single series — instead of the tenant breakdown.

### 9. Batch Size (5m Avg)

```promql
round(sum by (pod) (increase(logiq_json_batch_size_count_sum[5m])) / sum by (pod) (increase(logiq_json_batch_size_count_count[5m])),1)
```

Average number of **messages** per JSON batch, per pod, over the last 5 minutes — the histogram's summed value divided by its observation count. Despite the name "batch size" this counts messages, not bytes.

The doubled suffix is not a typo: the histogram is itself named `logiq_json_batch_size_count`, so Prometheus exposes `…_count_sum` and `…_count_count`. A pod that received no batches in the window gives `0/0`, which is NaN and renders as a gap rather than a zero.

### 10. Ingest DB File Count

```promql
query=logiq_ingestdb_files_count&duration=1h&step=5m
```

Number of ingest database files per pod over the last hour. A steady climb suggests files are being created faster than they are moved to S3.

The leading `query=` is the canonical form, not a mistake — see *Query string format* below.

### 11. Disk Free % (Per Pod)

```promql
round(100 * sum by (pod) (logiq_file_sizes_bytes{pod=~"logiq-flash-\\d{1,3}",fileName="free"}) / sum by (pod) (logiq_file_sizes_bytes{pod=~"logiq-flash-\\d{1,3}",fileName="all"}),0.01)&duration=1h&step=5m
```

Free disk as a percentage of each ingest pod's own volume, over the last hour. Panel 7 is the same measure as a current-value snapshot; this one is the trend. Both report *free*, so higher is better on each.

## Ingest Endpoint Health tab

Answers one question per ingest endpoint: *is it letting data in, and if not, why?* Every panel reads four counters that flash exports:

| Metric | Labels | Counts |
|---|---|---|
| `apica_ingest_accepted_total` | `endpoint`, `protocol` | admissions — one per HTTP request, one per connection/batch on socket listeners |
| `apica_ingest_rejections_total` | `endpoint`, `protocol`, `reason`, `disposition` | refusals — `reason` is why flash refused, `disposition` is what the sender saw |
| `apica_ingest_dropped_bytes_total` | `protocol`, `reason` | bytes lost where the sender will not retry (oversized payloads, 413) |
| `apica_ingest_channel_full_drops_total` | `brokerId` | batches shed at the broker because every channel was full |

`endpoint` is the URL route for HTTP ingest (`/v1/logs`, `/v1/traces`, `/v1/metrics`, `/v1/json_batch` and its `/otlp*` variants, `/api/v2/logs`) and the protocol name for socket listeners (`syslog`, `relp`, `mqtt_syslog`, `fluentdforward`, `lumberjack`, `grpc_jaeger`). Ingest routes are only mounted on ingest and combined nodes, which are also the only nodes that register these counters — so an unqualified `sum()` is a true cluster total, as with panel 4 on the Overview tab.

Unlike the Overview tab, this tab's widgets carry explicit `position` values on the 12-column grid, so its layout is fixed rather than auto-laid-out. Its time series are `area` charts over the last hour at one-minute steps (`&duration=1.000h&step=1m`).

Two conventions recur below:

- **Pre-accept only, for the headline rejection numbers and the per-endpoint and per-pod rates** (panels 1, 3, 10 and 15). These filter `disposition!="dropped_after_ack"`. An after-ack drop happens *after* admission (data accepted, then discarded), and on socket listeners it is counted per message while admissions are counted per connection — so adding it to a ratio compares different units. After-ack drops have their own tile (5), and the mix and breakdown panels (4, 12, 13 and 14) count every event.
- **The endpoint fold** (panels 10, 11 and 13). `label_replace(…, "endpoint", "$1", "exported_endpoint", "(.{1,})")` handles the same collision as `exported_namespace` on the Overview tab's panel 8: if the scraper attaches its own `endpoint` target label, Prometheus renames the metric's label to `exported_endpoint`. The fold copies it back and does nothing when there is no collision, so these panels work either way. The regex is `{1,}` rather than `+`, because `+` would arrive as a space (see *Query string format*), and not `.*`, which would also match a missing label and blank `endpoint`.

Panel numbers below restart at 1, follow the tab's reading order (left to right, top to bottom) and refer to this tab.

### 1. Rejected (%)

```promql
round(100 * sum(rate(apica_ingest_rejections_total{disposition!="dropped_after_ack"}[5m])) / sum(rate({__name__=~"apica_ingest_accepted_total|apica_ingest_rejections_total",disposition!="dropped_after_ack"}[5m])), 0.01) or vector(0)
```

Share of ingest attempts refused before admission over the last 5 minutes, across all endpoints: pre-accept rejections ÷ (accepted + pre-accept rejections). The gauge's `options.zone` sets its alert threshold at 25 and its danger threshold at 50. Both sides exclude after-ack drops: a socket listener counts one admission per connection but one after-ack drop per message, so including them lets a healthy cluster whose routing rules discard unmatched syslog read close to 100%. The denominator matches both counters with a single `{__name__=~…}` selector, and `or vector(0)` makes a cluster that has never rejected anything read 0% rather than blank.

### 2. Accepted Requests / sec

```promql
round(sum(rate(apica_ingest_accepted_total[5m])), 0.01)
```

Admissions per second, cluster-wide: one per HTTP request, one per connection (or Beats batch) on socket listeners — the same unit the rejection counter uses. There is deliberately no `or vector(0)` here: if this tile shows no data, the cluster is not exporting the new counters at all (for example an older flash), and the zeros in panels 1, 3, 5, 6 and 7 should not be trusted.

### 3. Rejected Requests / sec

```promql
round(sum(rate(apica_ingest_rejections_total{disposition!="dropped_after_ack"}[5m])), 0.01) or vector(0)
```

Pre-accept refusals per second, all endpoints and reasons. After-ack drops are excluded — they are not requests, and panel 5 counts them. `or vector(0)` shows 0 rather than no data when nothing has been rejected yet.

### 4. Rejection Ratio

```promql
sum by (reason)(increase(apica_ingest_rejections_total[1h]))
```

The last hour's rejection events, split by `reason`. It counts every event, after-ack drops and client cancels included, so on a cluster whose routing rules discard unmatched syslog, `no_rules_matched` can dominate the pie — that is real activity, not overload.

### 5. Silent Drops After Ack (1h)

```promql
round(sum(increase(apica_ingest_rejections_total{disposition="dropped_after_ack"}[1h]))) or vector(0)
```

Events where flash accepted data and then discarded it, so the sender saw success. Today these are mostly syslog/RELP messages that matched no routing rule (counted per message), plus OTLP-metrics requests with failed slices on the log-pipeline path. Some no-rules-matched drops are intentional filtering, so watch for a sudden jump rather than a non-zero value.

### 6. Overload Sheds - Channels Full (1h)

```promql
round(sum(increase(apica_ingest_channel_full_drops_total[1h]))) or vector(0)
```

Batches shed at the broker in the last hour because every broker channel was full — the event OPL-15750 was raised for. It reads the broker-factory counter, which fires once per shed whichever listener asked, so it is the authoritative total. Panel 13's `channel_full` bars are the same event seen from the endpoints; the factory total can run higher, because one OTLP-metrics request is split into several batches that each reach the factory.

### 7. Oversized Payloads Lost (MB, 1h)

```promql
round(sum(increase(apica_ingest_dropped_bytes_total{reason="payload_too_large"}[1h])) / 1000000, 0.01) or vector(0)
```

Megabytes rejected with HTTP 413 in the last hour — payloads larger than the ingest rate limiter's burst. It is a `size` tile with `datatype` `mb`: the query already returns MB and the tile scales it up (MB → GB …) for display. These are the only rejections counted as lost volume: the sender is told not to retry, so the data is gone. Retryable rejections (429, gRPC `ResourceExhausted`, severed connections) are re-sent, so they appear as events on this tab but never as lost bytes.

### 8. What's Behind the 429s (1h)

```promql
round(sum by (reason) (increase(apica_ingest_rejections_total{disposition="http_429"}[1h])), 1)
```

Every HTTP 429 in the last hour, split by its real cause. To a client these all look identical — `channel_full` (broker saturated), `rate_limit` (ingest rate limiter), `connection_cap` (too many concurrent requests), `backpressure` (forward buffer full) and `client_cancel` — so this is the panel that separates a capacity problem from limiter tuning.

### 9. Channel-Full Sheds by Broker

```promql
round(sum by (brokerId) (rate(apica_ingest_channel_full_drops_total[5m])) * 60, 0.01)&duration=1.000h&step=1m
```

Panel 6 over time, in sheds per minute, one series per broker: `syslog`, `relp`, `fluentdforward`, `lumberjack`, `lumberjacktls`, `http` and `grpc_jaeger`. `http` is the single broker shared by every HTTP ingest route — json_batch, OTLP and Datadog all publish through it.

### 10. Rejection Rate by Endpoint

```promql
round(sum by (endpoint) (label_replace(rate(apica_ingest_rejections_total{disposition!="dropped_after_ack"}[5m]), "endpoint", "$1", "exported_endpoint", "(.{1,})")), 0.001)&duration=1.000h&step=1m
```

Pre-accept rejections per second, one series per endpoint, over the last hour. The `label_replace` is the endpoint fold described above.

### 11. Accepted Rate by Endpoint

```promql
round(sum by (endpoint) (label_replace(rate(apica_ingest_accepted_total[5m]), "endpoint", "$1", "exported_endpoint", "(.{1,})")), 0.001)&duration=1.000h&step=1m
```

Admissions per second per endpoint. Read it next to panel 10: a flat line with no rejections is a quiet endpoint, not a broken one.

### 12. Rejections by Reason

```promql
sum by (reason) (rate(apica_ingest_rejections_total[5m]))&duration=1.000h&step=1m
```

Every rejection event per second by `reason`, over the last hour — after-ack drops and client cancels included. Unlike the other time series this query is not rounded, so hover values show full precision.

### 13. Rejections by Endpoint & Reason (1h)

```promql
round(sum by (endpoint, reason) (label_replace(increase(apica_ingest_rejections_total[1h]), "endpoint", "$1", "exported_endpoint", "(.{1,})")), 1)
```

The "which door, which error" panel: one bar per (reason, endpoint) pair for the last hour, with the legend on the right. It includes every disposition, so after-ack drops on socket listeners show here too.

### 14. Rejections by Status Code(1h)

```promql
round(sum by (disposition, protocol) (increase(apica_ingest_rejections_total[1h])), 1)
```

What senders experienced, by `disposition` and split by protocol: `http_429` and `http_503` (retry later), `http_400` and `http_413` (do not retry), `http_500`, `grpc_resource_exhausted`, `conn_severed` (socket closed with no error frame), `listener_closed` and `dropped_after_ack`. Despite the title, not every value is an HTTP status — the socket and after-ack outcomes are included. Any `listener_closed` bar deserves attention: one shed tore down a whole listener — every connected Lumberjack client, or the UDP syslog listener.

### 15. Rejections by Pod (1h)

```promql
round(sum by (pod, reason) (increase(apica_ingest_rejections_total{disposition!="dropped_after_ack"}[1h])), 1)
```

Pre-accept rejections per ingest pod, stacked by reason. One pod far above the others usually means uneven load or a single unhealthy pod, rather than a cluster-wide limit.

### Open items (this tab)

- **No alerts attached.** Counter widgets accept an inline `alert` block (see `apica-flow.json`), but none were added: alerts on an imported dashboard notify the customer, so thresholds should be agreed first. Natural candidates: panel 6 above 0 for 2 minutes, and any `listener_closed` in panel 14.
- **Panel 1's thresholds** (alert 25%, danger 50%) live in `options.zone`; revisit them once an ingest SLO is agreed.
- **No per-endpoint failure percentage.** Panel 10 shows each endpoint's rejection *rate*; the only failure *percentage* is cluster-wide (panel 1).
- **Not covered.** The Prometheus remote-write proxy (`/v1/receive/prometheus`) is not instrumented, so it never appears on this tab; nor do upstream (Thanos) errors relayed by the OTLP-metrics store path.
- **OTLP metrics sent to both destinations.** When metrics are converted to log events *and* written to the metrics store, the request is counted once, by the store path's outcome.

---

## Open items (Overview tab)

Noticed while reviewing, not yet changed:

- **Panels 7 and 11** — deliberately the same measure, one snapshot and one trend. Both report percent *free*, matching flash's own internal `freePercent` and its emergency-cleanup threshold. Switching both to percent *used* would allow panel 7 to use the `disk` chart type (as `host-monitoring`/`java-monitoring` do), at the cost of the "higher is better" reading.
- **Panel 3** — its operation labels mix two units on one axis. `inc`, `query`, `dq`, `ewr`, `je`, `jok`, `rec` count *events*; `wr`, `mdq`, `mdone` count *records* and are orders of magnitude larger, flattening the event series. `mq` adds a queue *length* repeatedly, so it has no clean rate meaning at all. Fixing it properly means splitting into two panels.
- **Panel 8** — one line per tenant namespace, so on a cluster with many tenants the chart becomes unreadable. `topk(10, …)` would cap it, at the cost of hiding the rest.

## Query string format

The `query` field is not raw PromQL — it is a URL query string. `redash/query_runner/apica_ascent_prometheus.py` prepends `query=` when it is absent ("for backward compatibility") and then calls `parse_qs`, so `query=<expr>&duration=1h&step=5m` is the canonical form and a bare expression is the legacy one. Both work; panel 10 uses the canonical form, the rest use the legacy one.

**`+` cannot be used in these queries.** `parse_qs` URL-decodes the value, so a `+` becomes a space: `pod=~"logiq-flash-\d+"` arrives at Prometheus as `pod=~"logiq-flash-\d "` and matches no pod at all. It fails silently — an empty panel, not an error. Use `\d{1,3}` for a bounded repeat (as panels 1, 6, 7 and 11 now do), or `%2B` if a literal `+` is genuinely needed. Panel 11 previously carried a `\d|\d\d|\d\d\d` alternation, which was an older workaround for the same limitation. The Ingest Endpoint Health tab's endpoint fold uses `(.{1,})` rather than `(.+)` for the same reason.

`&`, `=`, `{`, `}`, `,` and `*` all pass through safely — only `+` and `%` need care.

## Chart types

Panels comparing a value across pods use `bar`, not `scatter`: x is a category name, so horizontal position carries no meaning and bar heights are what the reader compares. `scatter` was inherited from `dashboards/apica-vanilla/apica-cluster-monitoring.json` and appears nowhere else in this repo. All five pod-axis panels (2, 3, 5, 7, 9) have been converted; no `scatter` remains. The Ingest Endpoint Health tab uses `gauge`, `counter` and `size` tiles for its headline numbers, `area` for its time series, a `pie` for the reason mix and `bar` for its breakdowns.

All 11 Overview panels reviewed; the 15 Ingest Endpoint Health panels are documented above.
