# Ownership

The proto files own service-boundary shape only. Generated Go and TypeScript are deterministic projections and must
not be edited. Each service owns its internal domain and persistence representation.

## Central and Launcher

Authentication exchange, client release discovery, device registration, launch tickets, and runtime artifact
metadata are owned by `proto/cineko/service/services.proto` and the release/client messages it references. Product
ownership is summarized in `docs/product-specification.md`.

## Central and Client

Client sessions, resources, event streaming, execution signals, settings, and embedded Probe bootstrap are owned by
`proto/cineko/service/services.proto`. The booking critical path and live-seat ownership are defined in
`docs/product-specification.md` and `docs/booking-execution.md`.

## Central and Probe

Probe registration, heartbeat, assignment leasing, captures, and result commits are owned by
`proto/cineko/service/services.proto`, `proto/cineko/probe/probe.proto`, and
`proto/cineko/observation/observation.proto`. Catalog/observation mutation is defined in
`docs/catalog-observation.md`, and local proxy lifecycle in `docs/egress.md`.

## Launcher and Client

The local launch handoff and installed runtime manifest are owned by the generated release and Client launch
messages. Their product boundary is fixed in `docs/product-specification.md`.

Central admin Web APIs are internal to `cineko-central` and are intentionally excluded.
