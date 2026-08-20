# Central and Probe

All requests use HTTPS and `X-Cineko-Protocol: 3`.

Probe container releases are published independently from desktop components. After a multi-architecture image is
pushed and its manifest digest is verified, CI registers the version, Chromium revision, image reference, and digest
with Central. This record drives Probe inventory and rollout visibility; it does not change the desktop release
generation or restart Client.

| Method | Path | Purpose |
| --- | --- | --- |
| `POST` | `/v1/probes/register` | Register a standalone or Client-owned Probe |
| `PUT` | `/v1/probes/{id}/heartbeat` | Report capacity and receive drain/version policy |
| `POST` | `/v1/probes/{id}/disconnect` | Gracefully remove active capacity |
| `POST` | `/v1/probes/{id}/assignments:claim` | Claim one due assignment |
| `PUT` | `/v1/assignments/{id}/heartbeat` | Extend the assignment lease |
| `PUT` | `/v1/assignments/{id}/result` | Atomically commit observed data |
| `POST` | `/v1/release-registry/probe` | Register one immutable released Probe image set |

Central owns cadence, assignment, retry, and reconciliation. Probe owns one-browser-process execution and reports
only observed data. Schedule results use canonical Provider, Theater, Movie, Auditorium, and Showtime identities;
Central updates the shared catalog and the availability time series in the same result transaction. Assignment and
result writes are idempotent by assignment, run, and lease identity.

The complete runtime and assignment state machines are defined in `docs/probe-runtime.md`. Catalog mutation,
observation completeness, and canonical identity are defined in `docs/catalog-observation.md`; local proxy selection
and lease behavior are defined in `docs/egress.md`.

`cgv.catalog.capture.v1` is a Central-owned bootstrap task. Its successful result carries one validated `catalog`
payload and no schedule `captures`. Central creates at most one active bootstrap assignment, waits when no eligible
Probe is online, and stores the payload before Clients consume the catalog. Protocol v3 treats this payload as an
upsert, not proof that omitted catalog entities disappeared.

Registration capabilities describe what a Probe implementation supports. Every heartbeat separately reports
`availableCapabilities`; Central may assign work only from that current subset.

Catalog and seat-map writes are accepted only as assignment results from a registered, online Probe that advertised
the matching capability. A Client session by itself is not catalog write authority; an embedded Client Probe follows
the same registration, lease, and result contract as a standalone Probe.

A browser Probe may read anonymous live-seat data for observation and analytics, but it never selects or holds a seat
for a user. A Probe reports the newly observed showtime; Central then leases a user-scoped execution to a Client. That
Client enters the exact showtime with its private browser session and reads live seats again before selecting them as
defined in `docs/booking-execution.md`.

## Discovery providers

Browser Probe with the configured residential egress policy is the operational discovery provider. It may discover
public schedules and remaining-seat counts, but it never receives a user's CGV cookie, credential, or booking session.

A serverless HTTP provider is disabled by default. It may be enabled only for public schedule discovery after its own
egress repeatedly returns an unsigned, unauthenticated success response. It must accept a typed theater/date request;
an arbitrary URL proxy is forbidden. `401`, `403`, `429`, challenge responses, or an expired request signature open the
provider circuit and return the work to Browser Probe. Central remains the sole owner of cadence, jitter, deduplication,
retry, and fallback.
