# 47: Contextual Attack Surface Guidance in Banner and End-to-End Verification

**What to build:** Provides operational security feedback for complex multi-auth configurations. When $\ge 2$ active plugins are attached, the Logical OR banner is rendered; when $\ge 3$ active plugins are attached, an operational security tip is displayed warning operators about credential attack surface expansion. When $< 2$ active plugins exist, the banner is hidden. Verifies the complete Web Console production bundle (`npm run build`) and live Data Plane test suite (`go test -v ./tests/e2e/...`).

**Blocked by:** 46

**Status:** completed

- [x] Logical OR banner renders when $\ge 2$ active plugins are attached or staged
- [x] Operational security guidance tip renders inside banner when $\ge 3$ active plugins are attached or staged
- [x] Banner automatically hidden when $< 2$ active plugins exist
- [x] `web/` builds cleanly without errors (`npm run build`)
- [x] Full E2E suite passes 100% (`go test -v ./tests/e2e/...`)
