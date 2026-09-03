# Basic Usage

The simplest way to try FaultLens. Analyze the log and get a full report:

```bash
faultlens app.log
```

Output includes a summary, error groups, and a diagnosis. The log here has a
few database-related errors, so the diagnosis should be database-related.

Try the other report views:

```bash
faultlens errors app.log
faultlens timeline app.log
faultlens incident app.log
```

Or generate an HTML report:

```bash
faultlens app.log --output html -o report.html
```