# Load Balancing Guidance

This project no longer exposes a websocket event channel endpoint.

## Active ingress paths

- `/v1/*`
- `/v1/health`
- `/console`, `/console/*` (when console embedding is enabled)

## Operational notes

- No WebSocket upgrade rule is required for Core.
- Standard HTTP load balancing is sufficient for connected-mode REST routes.
- Keep idle timeout tuned for long-lived SSE/streaming paths.
- Use webhook delivery metrics and API polling latency as primary observability signals for approval UX.
