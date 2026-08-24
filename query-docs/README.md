# Query Docs

Brief explanations of the queries in each dashboard under [`dashboards/`](../dashboards).

One file per dashboard, named after its folder: `dashboards/apica-monitoring/` → [`apica-monitoring.md`](apica-monitoring.md). Files are added as dashboards are reviewed, so this folder is expected to be incomplete.

Each entry gives the query and a sentence or two on what it shows — enough to read the panel without opening the JSON.

## Repo-wide gotcha: `+` in queries

The `query` field is a URL query string, not raw PromQL. `redash/query_runner/apica_ascent_prometheus.py` runs it through `parse_qs`, which URL-decodes the value — so **any `+` becomes a space** and the query silently matches nothing. No error, just an empty panel.

18 queries in `dashboards/apica-vanilla/kube-pod-data.json` and `dashboards/apica-vanilla/kubernetes-data.json` are affected today: their `=~'.+'` label matchers arrive at Prometheus as `=~'. '`. Fixes are `..*` (exactly equivalent to `.+`), `.{1,}`, or `%2B` for a literal plus.

Verify any query with:

```bash
python3 -c "from urllib.parse import parse_qs; q=input(); print(parse_qs(q if q.startswith('query=') else 'query='+q)['query'][0])"
```

| Dashboard | Doc |
|---|---|
| `apica-monitoring` | [apica-monitoring.md](apica-monitoring.md) |

Not yet documented: `apica-flow`, `apica-logs-overview`, `apica-query-statistics`, `apica-real-usage-monitoring`, `apica-sw`, `apica-vanilla`, `aws-cloudtrail`, `consul-dashboard`, `ec2-monitoring`, `fluent-bit`, `host-monitoring`, `java-monitoring`, `jmxexporter`, `kafka`, `kube-cost`, `kubernetes`, `memcached`, `mongodb`, `mysql`, `namespace-app-distribution`, `postgres`, `prometheus`, `rabbitmq`, `redis`, `slo`, `windows-monitoring`.
