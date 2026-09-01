# Security Policy

FaultLens is a local-first, offline-first tool: all analysis happens on the
user's machine and no data ever leaves it.

## Reporting a Vulnerability

If you find a security issue in FaultLens, please report it privately instead
of opening a public issue:

- **Email:** hulei15082452670@gmail.com
- **Subject:** `[FaultLens Security] <short description>`

Please include:

- A description of the vulnerability and its impact
- Steps to reproduce (including the log input if relevant)
- Affected version or commit
- Any suggested fix, if you have one

We will acknowledge your report within 5 business days and work with you on a
fix before any public disclosure.

## Scope

The following are **not** considered security vulnerabilities:

- False positives or false negatives in log diagnosis rules (these are
  correctness issues, report them as regular issues)
- Performance problems on very large log files
- Features explicitly out of scope for the current version (see `plan-v2.md`)

## Responsible Disclosure

Please give us a reasonable window (30 days by default) to address the issue
before publicly disclosing it.