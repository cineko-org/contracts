# Service boundary inventory

The authoritative RPC definitions are in `proto/cineko/service/services.proto`. This inventory describes ownership;
it does not duplicate field definitions.

| Service | Mutations | Reads and streams | Owner |
| --- | --- | --- | --- |
| ProbeService | register, heartbeat, disconnect, assignment heartbeat, submit result | claim assignment | Central control plane |
| ClientAuthenticationService | PIN exchange, token exchange and refresh, logout, launch-ticket issue and exchange, Probe bootstrap-ticket issue | authentication result | Central identity |
| ClientResourceService | put and delete a typed resource, upsert device | bootstrap, get, list, event stream | Central user state |
| CatalogService | submit a catalog snapshot | catalog, auditoriums, seat-map resolution | Central catalog |
| ExecutionService | heartbeat, complete, and retry execution | claim execution | Central booking coordinator |
| ReleaseService | publish component release sets | resolve runtime or Launcher release | Central release registry |

## State inventory

- Seat-map resolution: ready, capture queued, or objectively unverifiable.
- Probe kind: container or Client-owned.
- Probe health: healthy, degraded, or unhealthy.
- Observation task: schedule, catalog, or seat-map capture.
- Observation result: completed or failed.
- Client resource: settings, preset, monitor, reservation, external operation, or application event.
- Monitor mode: opening or cancellation.
- Monitor state: pending, running, triggered, booked, failed, or stopped.
- Event-stream control: ready, heartbeat, retention gap, or invalid cursor.
- Execution result: completed, failed, or explicit retry requested.

Each group is a required `oneof`; an absent choice is invalid at runtime.
