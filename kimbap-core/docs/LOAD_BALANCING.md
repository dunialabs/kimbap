# Load Balancing Guidance

This project no longer exposes a websocket event channel endpoint.

## Active ingress paths

- `/api/v1/*`
- `/admin`
- `/user`
- `/health`, `/ready`
- OAuth endpoints (`/.well-known/*`, `/register`, `/authorize`, `/token`, `/introspect`, `/revoke`, `/oauth/*`)

## Operational notes

- No WebSocket upgrade rule is required for Core.
- Standard HTTP load balancing is sufficient for admin/user/API routes.
- Keep idle timeout tuned for long-lived SSE/streaming paths under MCP traffic.
- Use webhook delivery metrics and API polling latency as primary observability signals for approval UX.
