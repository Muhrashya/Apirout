# 61: Non-Root Container Security & Build Ignore Hygiene

**What to build:**  
The production Alpine container image executes under an unprivileged user and group (`apirout:apirout`, UID/GID 10001) rather than root, satisfying CIS Docker Benchmark 4.1. The build ignore manifests (`.dockerignore`, `.gitignore`) strictly filter repository metadata (`.git/`) and private key files (`*.key`, `*.crt`, `*.pem`), preventing sensitive files from leaking into build cache layers.

**Blocked by:** Tickets 58, 59, 60

**Status:** completed

- [x] In `Dockerfile`: create group and user `apirout` with UID/GID 10001, configure permissions for `/app`, and set `USER apirout`.
- [x] In `.dockerignore`: add `.git/`, `*.key`, `*.crt`, `*.pem`, `*.cert`.
- [x] In `.gitignore`: add `*.key`, `*.crt`, `*.pem`, `*.cert`.
- [x] Verify full test suite passes (`go test ./...`), container builds cleanly (`docker compose build`), and runtime responds on health check.
