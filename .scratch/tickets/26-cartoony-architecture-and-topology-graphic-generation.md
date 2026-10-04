# 26: 2D Cartoony Architecture & Topology Graphic Generation

**What to build:** A high-resolution 2D cartoony tech illustration depicting Apirout's multi-plane system topology (client devices, Data Plane proxy on port 8000, Control Plane on ports 8001/8002, Redis event bus, upstream services, and database backends) in a clean, friendly hand-drawn/isometric style to replace the ASCII text block.

**Blocked by:** None (can start immediately)

**Status:** ready-for-agent

- [ ] Design a 2D cartoony isometric infrastructure graphic showing:
  - Public Client layer (mobile, desktop, external consumer).
  - Data Plane Proxy (`:8000`) with in-memory radix routing, consumer auth, rate limiter, upstream balancer, and instant database proxy.
  - Control Plane (`:8001` Admin API & `:8002` Web Console) with admin operator.
  - Redis Event Bus for real-time config sync & distributed cache invalidation.
  - Upstream Microservices and Database sources (PostgreSQL, MySQL, SQL Server).
- [ ] Render the graphic in a playful, clean cartoon style with crisp outlines, colorful rounded containers, and glowing energy/data lines.
- [ ] Export the illustration as a high-resolution PNG image at `docs/architecture-topology.png`.
- [ ] Verify that the graphic is clearly legible, properly proportioned, and visually balanced on both light and dark GitHub backgrounds.
