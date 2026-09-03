# Network Partition

A network-level failure: `network unreachable` errors followed by HTTP 5xx
downstream.

```bash
faultlens incident app.log
```

Expected root cause:

```text
Root Cause: Network partition
```

The diagnosis combines the network strong signal with downstream HTTP 5xx and
the temporal relationship — evidence first, not a guess.