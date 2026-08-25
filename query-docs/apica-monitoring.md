# apica-monitoring

Queries in [`dashboards/apica-monitoring/apica-monitoring.json`](../dashboards/apica-monitoring/apica-monitoring.json) — "Apica Cluster Monitoring", one `Overview` tab, data source `apica_ascent_prometheus`.

Panels with `&duration=…&step=…` are time series (`line`, x = timestamp); the rest are instant values compared across pods (`bar`, x = pod).

---

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

---

## Open items

Noticed while reviewing, not yet changed:

- **Panels 7 and 11** — deliberately the same measure, one snapshot and one trend. Both report percent *free*, matching flash's own internal `freePercent` and its emergency-cleanup threshold. Switching both to percent *used* would allow panel 7 to use the `disk` chart type (as `host-monitoring`/`java-monitoring` do), at the cost of the "higher is better" reading.
- **Panel 3** — its operation labels mix two units on one axis. `inc`, `query`, `dq`, `ewr`, `je`, `jok`, `rec` count *events*; `wr`, `mdq`, `mdone` count *records* and are orders of magnitude larger, flattening the event series. `mq` adds a queue *length* repeatedly, so it has no clean rate meaning at all. Fixing it properly means splitting into two panels.
- **Panel 8** — one line per tenant namespace, so on a cluster with many tenants the chart becomes unreadable. `topk(10, …)` would cap it, at the cost of hiding the rest.

## Query string format

The `query` field is not raw PromQL — it is a URL query string. `redash/query_runner/apica_ascent_prometheus.py` prepends `query=` when it is absent ("for backward compatibility") and then calls `parse_qs`, so `query=<expr>&duration=1h&step=5m` is the canonical form and a bare expression is the legacy one. Both work; panel 10 uses the canonical form, the rest use the legacy one.

**`+` cannot be used in these queries.** `parse_qs` URL-decodes the value, so a `+` becomes a space: `pod=~"logiq-flash-\d+"` arrives at Prometheus as `pod=~"logiq-flash-\d "` and matches no pod at all. It fails silently — an empty panel, not an error. Use `\d{1,3}` for a bounded repeat (as panels 1, 6, 7 and 11 now do), or `%2B` if a literal `+` is genuinely needed. Panel 11 previously carried a `\d|\d\d|\d\d\d` alternation, which was an older workaround for the same limitation.

`&`, `=`, `{`, `}`, `,` and `*` all pass through safely — only `+` and `%` need care.

## Chart types

Panels comparing a value across pods use `bar`, not `scatter`: x is a category name, so horizontal position carries no meaning and bar heights are what the reader compares. `scatter` was inherited from `dashboards/apica-vanilla/apica-cluster-monitoring.json` and appears nowhere else in this repo. All five pod-axis panels (2, 3, 5, 7, 9) have been converted; no `scatter` remains.

All 11 panels reviewed.
