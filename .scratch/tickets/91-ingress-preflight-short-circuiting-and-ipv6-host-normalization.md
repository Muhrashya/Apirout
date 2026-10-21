# 91: Ingress Preflight Short-Circuiting and IPv6 Host Normalization

**What to build:**
Enable browser SPAs to successfully perform cross-origin API calls without encountering `405 Method Not Allowed` on preflight `OPTIONS` requests. In `stage_ingress.go`, intercept incoming `OPTIONS` requests matching route paths prior to exact method matching, responding with `204 No Content` and dynamic `Access-Control-Allow-Methods` and `Access-Control-Allow-Headers`. In addition, normalize `r.Host` with standard `net.SplitHostPort` to safely parse IPv6 literals (`[::1]:8000`) without truncating to `[`.

**Blocked by:** None (can start immediately)

**Status:** completed

- [x] In `internal/gateway/executor/stage_ingress.go`, intercept `r.Method == http.MethodOptions` after route lookup but before returning `ErrMethodNotAllowed`.
- [x] For matching routes, return `204 No Content` with `Access-Control-Allow-Methods` matching the endpoint's allowed methods, and set standard preflight headers (`Access-Control-Allow-Headers`, `Access-Control-Max-Age`).
- [x] Replace naive string splitting on `r.Host` with standard `net.SplitHostPort` to safely support bracketed IPv6 literals (e.g. `[::1]:8000`).
- [x] Write unit tests verifying that `OPTIONS` preflight requests to non-OPTIONS endpoints (e.g. `POST /api/v1/orders`) succeed with `204 No Content`, and that `Host: [::1]:8000` is correctly parsed.
