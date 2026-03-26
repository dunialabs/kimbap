# DEPRECATED

This directory contains the legacy Kimbap Console — a Next.js operations dashboard built on the legacy MCP proxy stack.

**This codebase is no longer maintained.**

## Why it's deprecated

The legacy console depended on `/admin` and `/user` API endpoints that were part of the MCP proxy server (`cmd/server/`). That entire stack has been removed as Kimbap has evolved into a CLI-first, embedded-runtime product.

The backend endpoints this console relies on no longer exist.

## What replaces it

A lightweight embedded console will be available at `/console` when running `kimbap serve`. It communicates directly with the `/v1` REST API (tokens, policies, approvals, audit).

## Do not invest effort here

Do not attempt to fix, update, or rewire this codebase against the new backend. It is kept only for historical reference.
