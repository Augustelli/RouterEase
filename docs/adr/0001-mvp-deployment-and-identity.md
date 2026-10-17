# ADR 0001 — MVP deployment and authentication boundaries

Status: **Accepted at product level; DNS device-attribution mechanism pending separate ADR**.

Decision date: 2026-10-12

## Context
RouterEase must run on a physical OpenWrt router plus one separately hosted Docker backend. There will be LuCI local UI and separate web management UI. Keycloak is retained and parental rules must vary by physical client device.

## Decision
- Backend services share one Docker host and Compose deployment.
- Only small on-router agent/utility and LuCI integration live on physical router; no Docker on router.
- Keycloak secures human admin authentication via OIDC/JWT.
- Each router has its own independently managed API key; that key is not a device/user identity.
- DNS enforcement must preserve authenticated router AND actual device source identity. The protocol and identity correlation approach need an ADR before implementation.
- Backend observability is Prometheus + Grafana + Jaeger, not the legacy lab InfluxDB/Grafana stack.

## Consequences
- New router onboarding/registration and policy management data model are necessary.
- OpenWrt release and physical target must be selected before native packaging.
- Separation allows independent evolution but backend failures require a defined DNS recovery/fallback policy.
- New admin SPA is required; LuCI alone is insufficient for the requested feature set.
