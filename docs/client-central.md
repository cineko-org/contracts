# Client and Central

All requests use HTTPS, bearer authentication, and `X-Cineko-Protocol: 3` unless exchanging a launch ticket.

| Method | Path | Purpose |
| --- | --- | --- |
| `POST` | `/v1/client-sessions` | Exchange a one-time launch ticket |
| `GET` | `/v1/client/bootstrap` | Load user, device, feature, and revision state |
| `GET` | `/v1/events/stream` | Continue the ordered event stream |
| `GET` | `/v1/catalog` | Read the shared theater, movie, auditorium, and showtime catalog |
| `POST` | `/v1/catalog/snapshots` | Publish discovered catalog metadata |
| `POST` | `/v1/probe-bootstrap-tickets` | Issue an embedded Probe registration ticket |
| `POST` | `/v1/executions:claim` | Claim one user-scoped booking attempt |
| `PUT` | `/v1/executions/{id}/heartbeat` | Extend an execution lease |
| `PUT` | `/v1/executions/{id}/result` | Commit an execution outcome |

Resources use revision-based optimistic concurrency: create-only writes send `If-None-Match: *`, while updates and
deletes send `If-Match: <revision>`. A failed precondition is a conflict and must be re-read before retrying. Events
are replayed by monotonically increasing sequence.
The team PIN resolves to a Central user ID. Presets, monitors, settings, notification adapters, reservations, and
their history are Central-owned resources scoped by that user ID; Client does not keep an offline domain database.
`GET /v1/events/stream` resumes from `Last-Event-ID` (or `after` for the first connection). Central sends typed
`cineko.control` events containing protocol, release generation, and cursor. `full_resync` means the cursor is older
than the six-month retained outbox or is ahead of Central; Client must reload bootstrap/resources and then reconnect
from the supplied cursor. Transport failures and HTTP 5xx use bounded jittered reconnects. Protocol, content type,
cursor, control payload, and event envelope errors fail closed.

Another device authenticated as the same user receives the same resources from Central. Local persistent state is
limited to device identity, an expiring Central session, verified runtime caches, and private CGV browser sessions
that must never be shared with Central or another user. A CGV account session is optional; a non-member booking
session remains device-local under the same rule.

Catalog entities are shared operational data, not user resources. Client reads them from Central and publishes only
provider metadata it actually observes. Seat availability is not embedded in the catalog; it remains a timestamped
observation.

An execution command contains the exact showtime selected from a Probe observation. After claiming it, Client opens
that showtime with its private CGV browser session and reads the live seat layout and availability in the same flow.
CGV login is not required to enter seat selection; it only changes the later identity, benefit, and booking-history
flow.
Central seat-layout data is not required and must not block preset creation or execution. See
`docs/booking-execution.md`.
