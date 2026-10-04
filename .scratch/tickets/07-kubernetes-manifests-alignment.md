# 07: Kubernetes Manifests Alignment, Tag Pinning & Schema Validation

**Parent:** [#6](https://github.com/Muhrashya/Apirout/issues/6)
**Issue:** [#9](https://github.com/Muhrashya/Apirout/issues/9)
**Blocked by:** [#7](https://github.com/Muhrashya/Apirout/issues/7)
**Status:** ready-for-agent

## What to build
Standardize and update all 6 manifests in `deploy/k8s/` (`00-namespace.yaml` through `05-network-policy.yaml`). Pin container images to formal release tag `apirout:v1.0.0`, align ConfigMaps and Secrets with current runtime requirements, and provide an automated manifest validation check.

## Acceptance criteria
- [ ] All deployment manifests pin container images to `apirout:v1.0.0`.
- [ ] ConfigMaps and Secrets in `01-configmaps-secrets.yaml` match current configuration keys (`APP_ENCRYPTION_KEY`, `JWT_SECRET`, `ADMIN_SECRET`).
- [ ] Automated dry-run or schema validation confirms all 6 manifests are syntactically valid.
