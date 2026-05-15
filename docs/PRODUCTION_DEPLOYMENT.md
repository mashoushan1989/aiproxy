# Production Deployment Entry Point

This repository uses a private production deployment runbook.

The authoritative production runbook is:

```text
.private/noncommit/DEPLOYMENT.md
```

That private file may contain environment-specific server addresses, paths, rollback steps, and operational details. It is intentionally not committed to the public repository.

Agents handling production deployment, rollback, server connection, or online verification must read `.private/noncommit/DEPLOYMENT.md` before taking action. If the file is missing or unreadable, stop and ask the user for the current production runbook.

Do not infer the production workflow from public scripts alone. In particular:

- Do not manually replace production binaries.
- Do not manually stop or run production Docker containers.
- Do not use public install scripts such as `core/deploy/install.sh` for production updates.
- Use the private `aiproxy-prod` workflow described by the runbook.

Without the private runbook, agents may still perform local code review, tests, builds, and non-production checks, but must not execute production changes.
