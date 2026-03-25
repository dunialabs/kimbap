# Load Balancing Review

This review has been refreshed after removing the websocket event channel from Kimbap Core.

## Summary

- Per-user websocket-affinity requirements were removed with realtime channel removal.
- Approval and user notifications now rely on webhook delivery plus console/API polling.
- Routing requirements are now HTTP-first and significantly simpler.

## Current risks to monitor

1. Webhook delivery retries and deduplication correctness.
2. Console polling interval/backoff strategy under load.
3. Approval latency SLOs (P50/P95/P99) and queue depth.

## Recommended controls

- Fast `2xx` webhook ingress with async processing.
- Idempotency key + replay-safe handlers for webhook events.
- Alerting on approval latency and webhook failure rate.
