# Proto usage inventory

This inventory is the source-of-truth review for the latest-only contract cutover.
It records why a message or RPC remains at the wire boundary; it does not treat a
generated self-reference as a consumer.

## Review method

The consumer scan covered only source files under:

- `/Volumes/dev/cineko-org/central`
- `/Volumes/dev/cineko-org/probe`
- `/Volumes/dev/cineko-org/client`
- `/Volumes/dev/cineko-org/launcher`

The scan excluded `gen`, `vendor`, `node_modules`, and `.git` directories. For
each package, message names, RPC names, generated request/response builders,
JSON/Connect handlers, and serialized WebUI schemas were searched with `rg`.
Proto-to-proto references and service declarations were then checked separately.

## Definition inventory

The latest source contains 353 messages, 0 enums, 7 services, and 56 RPCs
across 12 proto files. Every definition is accounted for by one of the
following boundary packages; no definition is retained solely because its
generated file refers to itself:

| Source file | Messages | Services/RPCs | Disposition |
| --- | ---: | ---: | --- |
| `proto/cineko/admin/admin.proto` | 70 | 1 / admin surface | Keep: Central admin API and serialized admin state |
| `proto/cineko/catalog/catalog.proto` | 15 | 0 | Keep: catalog entities and typed provider identities |
| `proto/cineko/client/client.proto` | 68 | 0 | Keep used/public WebUI payloads; remove two unused local request helpers |
| `proto/cineko/client/webui.proto` | 23 | 0 | Keep: local Client API state/action/resource payloads |
| `proto/cineko/collection/collection.proto` | 31 | 0 | Keep: collection reasons, triggers, and lifecycle state |
| `proto/cineko/common/common.proto` | 15 | 0 | Keep: shared boundary primitives and egress/error policy |
| `proto/cineko/execution/execution.proto` | 11 | 0 | Keep: command and heartbeat boundary |
| `proto/cineko/observation/observation.proto` | 19 | 0 | Keep: assignment/task/result boundary |
| `proto/cineko/probe/probe.proto` | 18 | 1 / probe surface | Keep: Probe registration and lease boundary |
| `proto/cineko/release/release.proto` | 13 | 0 | Keep: release resolution and artifact boundary |
| `proto/cineko/seatmap/seatmap.proto` | 9 | 0 | Keep: cached/live seat data and resolution wrapper |
| `proto/cineko/service/services.proto` | 61 | 6 / 37 | Keep: Central service request/response and stream boundary |

The counts above are produced from the latest source definitions, not from
generated code. All message and RPC definitions in the table are therefore
retained as public or nested boundary types unless an exact removal is listed
below.

## Surface disposition

| Package | Runtime/source evidence | Boundary-only evidence | Disposition |
| --- | --- | --- | --- |
| `cineko.admin` | Central admin handlers and frontend operations consume the admin service surface. | Every request/response is attached to `AdminService`; nested status/state messages are response payloads. | Keep |
| `cineko.catalog` | Central, Probe, and Client use catalog values through generated builders and JSON/Connect serialization. Typed CGV identity messages are the pending hard-cutover source of truth. | Catalog messages are embedded in observation, seat-map, and service responses. | Keep |
| `cineko.client` | Client WebUI uses `SeatMapRequest` and `AuditoriumResponse`; resource, monitor, reservation, authentication, and event messages are serialized by the local API and frontend schemas. | Bootstrap, resource, and event payloads are embedded by service RPCs. | Keep used/public messages |
| `cineko.client` local helpers | No consumer, proto service, or serialized endpoint referenced `CatalogRequest` or `AuditoriumRequest`. | Neither message was a service request type. | Removed |
| `cineko.client.webui` | Client WebUI server, event handlers, and frontend schemas use the state/action/resource messages. | State oneofs are serialized API payloads even where a concrete variant is selected dynamically. | Keep |
| `cineko.collection` | Typed deferred/failed reasons are consumed by observation; Central-only waiting reasons, queue triggers, and lifecycle states are consumed by seat-map resolution. | Reasons and state are required oneof payloads, not standalone API requests. | Keep |
| `cineko.common` | Runtime, identity, egress, mutation, pagination, and API error types are used by consumers or imported by public messages. | `PageRequest/PageResponse` and egress variants are service/task boundary fields. | Keep |
| `cineko.execution` | Central and Client use command, heartbeat, and result payloads. | Execution messages are service RPC payloads. | Keep |
| `cineko.observation` | Central and Probe use assignment/task/result payloads; typed completion/deferred/failed payloads are the latest cutover. | Assignment messages are Probe service payloads. | Keep |
| `cineko.probe` | Probe runtime uses registration, health, lease, heartbeat, and result payloads. | Lease messages are `ProbeService` request/response payloads. | Keep |
| `cineko.release` | Launcher/Client release resolution and publish serialization use release sets and artifacts. | Release messages are `ReleaseService` payloads. | Keep |
| `cineko.seatmap` | Client resolves cached layouts and submits the atomic observation from an authenticated pre-booking recheck; Central/Probe persist and validate snapshots and availability. `Resolution` embeds `cineko.collection.State`. | Resolution, submission, and stream messages are `CatalogService` payloads. | Keep |
| `cineko.service` | Client Central HTTP adapters and Central handlers use resource/catalog/execution/release wrappers. | Every declared RPC owns its request/response type; `WatchSeatMap` is the new stream boundary. | Keep |

`AuditoriumResponse` was not removed with `AuditoriumRequest`: the Client local
`/api/auditoriums` endpoint builds it and the frontend validates it. `SeatMapRequest`
is likewise required by the local `/api/seat-map` endpoint.

`Completed.seat_map` was removed in the latest hard cutover. The source scan did
find old accessor calls in the current, not-yet-cutover consumers, including
`/Volumes/dev/cineko-org/probe/probe/runtime.go:600,804`,
`/Volumes/dev/cineko-org/central/internal/central/service.go:350,443`, and
`/Volumes/dev/cineko-org/central/internal/central/postgres/store.go:527,555,575`.
Those are migration evidence, not a reason to retain a duplicate wire payload:
the approved seat-page boundary is `Completed.live_seat`, which carries the
layout and availability from one provider response. Consumer edits are outside
this Contracts-only change and must be completed as the coordinated cutover.

The provider identity fields were checked against the current CGV parser before
keeping the typed showtime tuple. Probe parses `scnSseq`, `scnYmd`, `scnsNo`,
`siteNo`, and `movNo` in
`/Volumes/dev/cineko-org/probe/internal/provider/cgv/schedule_response.go:175-227`;
it uses `row.Sequence` and the tuple
`siteNo/date/screenNo/sequence` in
`/Volumes/dev/cineko-org/probe/internal/provider/cgv/schedule.go:204,230` and
`schedule_capture.go:71`. Client's parser mirrors these fields in
`/Volumes/dev/cineko-org/client/internal/adapters/cgv/schedule_response.go:175-227`.
Therefore `sequence` is the provider's `scnSseq`, not an invented ordinal.

`SeatMapTask` and `SeatAvailabilityTask` remain separate objectives because
their scheduling scope differs, but both successful seat-page results use
`Completed.live_seat`. The collection lifecycle is owned by
`cineko.collection.State`, and queue causes use the typed `Trigger` oneof; no
consumer may introduce a free-form trigger string.

Assignment integrity was checked against the current task creators before
tightening validation. Central creates schedule tasks with a theater, target
dates, locale, time zone, and managed egress, while global catalog tasks carry
only the provider ID, locale, time zone, and egress in
`/Volumes/dev/cineko-org/central/internal/central/reconcile/engine.go:647-683`;
the seat-map backfill creator supplies the same fields plus the auditorium and
bounded target dates in
`/Volumes/dev/cineko-org/central/internal/central/postgres/reconcile_store.go:711-728`;
the exact-showtime creator supplies theater, auditorium, showtime, locale,
time zone, and managed egress in `:838-850`. Probe's schedule validator already
rejects a missing theater, time zone, or target date in
`/Volumes/dev/cineko-org/probe/probe/cgv_executor.go:211-230`.
The wire therefore requires those facts instead of accepting an incomplete
assignment and deferring validation to one provider implementation. A
`Collecting` state also requires the real assignment ID and start time, while
`Queued` deliberately has no assignment ID because no capacity has been
allocated yet. `ResultReceipt` requires assignment ID, run ID, and the
lowercase 64-hex SHA-256 payload hash used by Central at
`/Volumes/dev/cineko-org/central/internal/central/service.go:368-372`.

## Hard-cutover notes

Consumer source still contains old generated accessor names such as
`GetSourceKey`, `GetReady`, and `GetCaptureQueued` while the coordinated
latest-only migration is in progress. Those references are migration work, not
evidence for retaining legacy fields or compatibility messages. This contract
repository intentionally contains no `reserved` declarations, schema/protocol
version fields, versioned DTO names, or duplicate DTO compatibility surface.

The typed identity and collection-state messages may have no direct consumer
name match until Central, Probe, and Client switch together. They remain because
they are required by the public service graph and are referenced by current
wire messages; deleting them to satisfy a source-name count would remove the
approved source of truth.
