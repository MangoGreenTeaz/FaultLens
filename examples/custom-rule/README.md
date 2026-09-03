# Custom Rule

This example defines a diagnosis rule entirely in YAML — no source code
changes.

## 1. The rule

`.faultlens.yaml` defines a `database_connection_failure` rule with strong
keywords, supporting keywords, downstream impact and recommendations.

## 2. Run

FaultLens auto-loads `.faultlens.yaml` from the current directory:

```bash
faultlens incident app.log
```

Expected root cause:

```text
Root Cause: Database connection failure
```

## 3. Inspect

```bash
faultlens config validate        # configuration is valid
faultlens config show            # print the merged effective config
```

## 4. Without the rule

Without this config file the same log would report
`Insufficient evidence` — proving the custom rule changes the diagnosis.

## Field reference

See the [Custom Rules section](../README.md#custom-rules) of the README for
every field (`id`, `root_cause`, `severity`, `keywords`, `strong_weight`,
`supporting_keywords`, `supporting_weight`, `enable_downstream`,
`recommendations`).