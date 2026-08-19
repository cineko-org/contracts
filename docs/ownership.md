# Ownership

## Central and Launcher

Authentication exchange, client release discovery, device registration, launch tickets, and runtime artifact
metadata are defined in `docs/launcher-central.md`.

## Central and Client

Client sessions, resources, event streaming, execution signals, settings, and embedded Probe bootstrap are
defined in `docs/client-central.md`. The booking critical path and live-seat ownership are defined in
`docs/booking-execution.md`.

## Central and Probe

Probe registration, heartbeat, assignment leasing, captures, and result commits are defined in
`docs/central-probe.md`.

## Launcher and Client

The local launch handoff and installed runtime manifest are defined in `docs/launcher-client.md`.

Central admin Web APIs are internal to `cineko-central` and are intentionally excluded.
