# CI Integration

A ready-to-use GitHub Actions workflow that runs FaultLens on your logs and
publishes JSON + HTML artifacts plus a job summary.

## Setup

1. Copy `workflow.yml` into `.github/workflows/` in your repository.
2. Put your logs somewhere the workflow can read (e.g. `./logs/*.log`), or
   adjust the paths in the two `go run` steps.
3. Push. The workflow runs on `workflow_dispatch` or every push to `main`.

## What it produces

- `faultlens.json` — machine-readable analysis (CI consumers)
- `faultlens-report.html` — offline report with timeline chart
- A **job summary** with the root cause and confidence

## Alternative: official composite action

If you use the FaultLens repository directly, the built-in action is:

```yaml
- uses: ./.github/actions/faultlens
  with:
    log-file: testdata/incidents/mysql-outage.log
    output: html
    output-file: faultlens-report.html
```