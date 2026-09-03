# Database Outage

A minimal incident chain: connection refused → MySQL errors → HTTP 500. The
HTTP 5xx is a downstream symptom; FaultLens must diagnose the database as the
root cause.

```bash
faultlens incident app.log
```

Expected root cause:

```text
Root Cause: Database unavailable
```

Note that the diagnosis comes from the log evidence (MySQL + connection
failures + downstream 5xx), not from the directory name.

Compare reports over time with `faultlens diff`:

```bash
faultlens incident app.log --output json -o before.json
# ... change the log ...
faultlens diff before.json after.json
```