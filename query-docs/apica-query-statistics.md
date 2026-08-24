# apica-query-statistics

Queries in [`dashboards/apica-query-statistics/query-statistics.json`](../dashboards/apica-query-statistics/query-statistics.json) — "Apica Query Statistics", one `Overview` tab.

**This dashboard is SQL, not PromQL.** Its `data_source_type` is `pg`, so every panel is a Postgres query against the `queryhistory` table written by `flash/wings/queryhistory.go`. None of the Prometheus-specific notes in the other files apply here.

Useful columns: `createdat` (epoch seconds — hence `to_timestamp(createdat)`), `createdby`, `namespace`, `applicationnames`, `keyword`, `durationsec` (length of the time range the user searched), `timetofirst` (seconds to first returned record), and `requesttype`, an int enum:

| value | meaning |
|---|---|
| 1 | Query |
| 2 | Search |
| 3 | None |
| 4 | AdvanceSearch |
| 5 | Report |
| 6 | Replay |

All windows are hardcoded in the SQL. The header sets `dateTimeRange: true`, but `redash/query_runner/pg.py` performs no time-range substitution, so **the dashboard's date picker does not affect these panels.**

---

### 1. Queries per User (15d) · `Table`

```sql
select count(*) as count, createdby
from queryhistory
where createdby != '' and to_timestamp(createdat) > NOW() - INTERVAL '15 DAY'
group by createdby order by count desc
```

Who is running queries, busiest first. `createdby != ''` drops system/internal queries that carry no user.

### 2. Query history (15d) · `Table`

```sql
select to_timestamp(createdat) as created, createdby, namespace, applicationnames,
       keyword, requesttype, durationsec, timetofirst
from queryhistory
where to_timestamp(createdat) > NOW() - INTERVAL '15 DAY'
order by createdat desc limit 500
```

The 500 most recent queries, newest first — a raw audit trail of who searched what.

### 3. Query count (15d) · `counter`

```sql
select count(*) as value
from queryhistory
where to_timestamp(createdat) > NOW() - INTERVAL '15 DAY'
```

Single number: total queries run in the window. The `as value` alias is required — the counter widget reads the column named in `plot.y`.

### 4. Query search range (15d) · `bar`

```sql
select case when durationsec < 3600   then '<1h'
            when durationsec < 86400  then '1-24h'
            when durationsec < 604800 then '1-7d'
            when durationsec < 2592000 then '7-30d'
            else '>30d' end as searchrange,
       count(*) as count
from queryhistory
where requesttype in (2,4) and durationsec > 0
  and to_timestamp(createdat) > NOW() - INTERVAL '15 DAY'
group by searchrange order by min(durationsec)
```

How far back people actually search — a histogram of the *time range requested*, not of query speed. Restricted to `requesttype in (2,4)` (Search and AdvanceSearch) because only those carry a user-chosen range. `order by min(durationsec)` puts the buckets in size order rather than alphabetical.

### 5. TTFR (Seconds) (15d) · `bar`

```sql
select count(*) as count, timetofirst
from queryhistory
where timetofirst > 0 and to_timestamp(createdat) > NOW() - INTERVAL '15 DAY'
group by timetofirst order by timetofirst
```

Distribution of time-to-first-record: how long users wait before seeing anything. `timetofirst > 0` is meaningful, not cosmetic — `UpdateTimeToFirstRecord` only writes when the first record arrives, so a query that returned nothing stays at `0`. **Failed and empty queries are therefore invisible in this panel.**

### 6. TTFR by Request Type (15d) · `bar`

```sql
select count(*) as count, timetofirst,
       case requesttype when 1 then 'Query' when 2 then 'Search' when 3 then 'None'
                        when 4 then 'AdvanceSearch' when 5 then 'Report'
                        when 6 then 'Replay' else 'Unknown' end as request_type
from queryhistory
where timetofirst > 0 and to_timestamp(createdat) > NOW() - INTERVAL '15 DAY'
group by timetofirst, requesttype order by timetofirst
```

Panel 5 split by request type, so you can see which kinds of request are slow to first record. The `case` maps the raw enum to names — without it the legend reads `1`, `2`, `4`.

---

## Chart types

| panel | data shape | type |
|---|---|---|
| 1 | one row per user | `Table` — exact counts, sortable, scales to any number of users |
| 2 | raw rows | `Table` |
| 3 | single scalar | `counter` |
| 4, 5, 6 | histograms | `bar` |

Panels 4, 5 and 6 are distributions across a category or bucket, not series over time, so `bar` rather than `line`.

`Table` and `counter` ignore `plot.x`, `plot.xLabel` and `plot.yLabel`; only `plot.y` matters, and only to the counter, which uses it to pick the column. Those fields are left as-is on panels 2 and 3 because they are inert there.

## Open items

- **Panel 6 is a superset of panel 5** (same data, broken down by request type). Deliberate, but if the breakdown proves more useful, panel 5 becomes redundant.
- **Panels 5 and 6 group on raw `timetofirst` seconds**, so a long tail of slow queries produces a long tail of single-count bars. If that gets noisy, bucket them the way panel 4 buckets `durationsec`.
- **The date picker is inert** on this dashboard (see above). Making it work would need time-range support in the `pg` runner, not a dashboard change.
- **`timetofirst = 0` conflates "never returned a record" with "returned instantly."** Both panels 5 and 6 drop those rows, so query failures are not visible anywhere on this dashboard.

All 6 panels reviewed.
