# Custom Rules: Disk Full

This example shows how to extend FaultLens with a YAML custom rule — no source
code changes needed.

## 1. The custom rule

`.faultlens.yaml` in this directory defines a `disk_full` rule:

```yaml
custom_rules:
  - id: disk_full
    root_cause: "Disk full"
    severity: critical
    keywords:
      - no space left on device
    strong_weight: 0.40
    supporting_keywords:
      - write error
    supporting_weight: 0.20
    recommendations:
      - "Check filesystem capacity"
      - "Clean temporary files"
```

## 2. Run

```bash
faultlens incident app.log
```

Expected root cause:

```text
Root Cause: Disk full
```

Without the config file the same log would report `Insufficient evidence`,
because disk-full is not one of the built-in rules. Custom rules plug into the
same evidence → confidence → priority engine as the built-ins.

## 3. Validate and inspect

```bash
faultlens config validate          # configuration is valid
faultlens config show             # print the merged effective config
```