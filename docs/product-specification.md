# Product specification

This document is the product-level source of truth for Cineko. Domain-specific
documents may explain wire or persistence details, but they must not redefine the
behaviour below.

## Product invariants

1. A user creates one booking monitor. The same monitor detects a newly opened
   showtime, waits for a matching cancellation seat when necessary, prepares one
   payment handoff, and remains recoverable until it expires or the user stops it.
   There are no opening-only and cancellation-only monitor products.
2. Central owns shared discovery, work priority, cadence, deduplication, execution
   leases, and durable user state. Client owns the user's private browser, live
   seat selection, CGV session, and payment handoff. Probe owns anonymous provider
   observation and physical egress selection.
3. Client never controls provider polling intervals and an operator never enters
   raw cadence seconds per theater. The Admin policy selects the theater, whether
   observation is enabled, and a rolling horizon of at most 14 provider dates.
4. A provider schedule date and the civil start date are different facts. CGV's
   `scnYmd` remains part of showtime identity, while weekday and time-window
   matching use the civil `startsAt` in the theater time zone.
5. Static seat layout and live seat availability are different data. A layout is
   versioned per auditorium by `layoutHash`; availability is timestamped per exact
   showtime. Neither one is inferred from the other.
6. Client always reads live seats again before selecting them. A Central signal is
   a fast wake-up hint, never authority that a seat is still available.
7. A provider request is shared at the narrowest safe identity. Schedule discovery
   is shared by theater and date set. Live-seat observation is shared by exact
   showtime. User count never multiplies those requests.
8. Unknown results remain unknown. Cineko never reports a booking or cancellation
   as successful without authoritative provider evidence.

## User journey

1. Central bootstraps the theater catalog when a capable Probe exists. With no
   Probe or no catalog, the system reports a durable waiting reason; Client does
   not fall back to a private full-catalog scan.
2. The user creates a seat preset. Central must resolve and return its cached
   auditorium layout for preview before the preset is saved. If the layout is
   not cached, the user sees the Central collection state and the preset remains
   unsaved until a layout can be displayed. This does not make the cached layout
   authoritative for final seat selection.
3. The user chooses a movie, preset, explicit dates and/or weekdays, a rolling
   horizon, and an optional civil showtime window. Central derives every provider
   date that must be scanned.
4. Central observes schedules and creates one fenced execution command when a
   matching showtime first becomes selectable.
5. Client acquires a ready local browser, revalidates the exact provider tuple,
   reads live seats, ranks them using the preset, and stops at the visible payment
   handoff.
6. If no acceptable seat exists, Central keeps the same monitor armed and watches
   that exact showtime. A transition from no matching group to a matching group
   creates one new command. Replaying an unchanged snapshot creates none.
7. Abandoning a payment handoff rearms the monitor in Central. Retrying never
   starts a second Client-side schedule discovery loop.

## Scheduling and capacity

Central chooses a lane before due time. Oldest due work wins inside a lane.

| Lane | Work | Target cadence |
| --- | --- | --- |
| `P0` | Active monitor with no matching showtime | randomized 2-5 seconds |
| `P1` | Exact-showtime live-seat watch for an active monitor | randomized 2-5 seconds |
| `P2` | Recently changed schedule coverage | randomized 15-30 seconds |
| `P3` | Ordinary rolling catalog coverage | randomized 5-15 minutes |

The target cadence is measured from completion, so slow provider work never
creates overlapping requests. One active schedule assignment exists per theater;
one active availability assignment exists per exact showtime. Capacity is reserved
for `P3`, so continuous demand cannot starve catalog coverage. These values are
safe initial product defaults and may be tuned from production telemetry in one
Central configuration change; they are not stored in Client resources or copied
into every observation policy. A general maintenance tick must not add another
fixed interval to `P0` or `P1`; Central wakes for their next due deadline and stays
idle when no deadline exists.

## Live-seat signal

- Probe reports the exact showtime ID, auditorium ID, layout hash, observed time,
  and the normalized set of currently available seat IDs.
- Central hashes the complete availability set. An unchanged hash is acknowledged
  but does not create another durable snapshot or execution command.
- Central joins the availability set with the layout returned in the same live
  observation and evaluates every matching preset. It wakes only monitors for
  which at least one valid seat group exists. If the layout hash changed, Central
  atomically stores the new immutable layout and availability observation before
  emitting the signal. A coarse positive wake without exact layout proof is
  forbidden.
- A failed command with `preferred_seats_unavailable` or
  `showtime_unavailable` is rearmed only by a later, distinct positive signal.
  Time alone and a positive aggregate seat count are insufficient.
- Availability history stores only distinct snapshots. Available seats are
  normalized as child rows of the snapshot so analysis can answer when a concrete
  seat or group appeared without storing unchanged high-frequency duplicates.

## Session and browser readiness

- CGV login is optional for entering the non-member booking path. An active monitor
  therefore requests warm Client capacity even when no saved credentials exist.
- If a member session snapshot exists, each warm slot restores and validates it;
  otherwise the slot is prepared for the non-member path. Session expiry asks the
  user to log in again but does not silently send credentials or bypass CAPTCHA.
- Credentials, cookies, origin storage, non-member details, and payment data remain
  local. Saved credentials may prefill a visible login flow only.
- Client keeps a target of two ready, isolated browser processes and a hard cap of
  three while it has active work. Each process owns one profile and one page and is
  reaped before replacement. The same showtime is never raced by two local slots.
- A claimed seat page may refresh at randomized 1.5-2.5 second intervals for at
  most 45 seconds and 20 requests. Protection signals stop the loop immediately.

## Egress and provider protection

- Client may use direct networking or a user-configured standard proxy. Its
  authenticated identity stays stable for the whole session; it never rotates in
  the middle of login, seat selection, or payment.
- Central selects a semantic egress policy for anonymous work. Probe resolves that
  policy to concrete Soxy/proxy inventory and reports protection failures.
- A blocked egress identity is quarantined. Rotation is rate-limited by the egress
  owner; a quarantined member is removed from selection until its cooldown expires.
  Remaining healthy members continue serving work. Central does not repeatedly
  command rotations or expose proxy credentials.
- HTTP 403/429, CAPTCHA, provider-contract drift, and page-identity mismatch are
  separate reason codes and metrics. None is treated as an ordinary empty result.

## Durable state machines

### Monitor

`pending -> running -> triggered -> booked`

- `pending` and `running` are eligible for shared observation and execution.
- `triggered` means a payment handoff exists; no second command may be claimed.
- `payment_unknown` is terminal until explicit user retry.
- `failed` is an unrecoverable application failure; `stopped` is cancellation or
  expiry. Retry from a terminal state abandons any retained payment browser and
  returns the same monitor to `pending`. That Monitor transition is the only
  user retry boundary; Clients never retry an opaque execution command directly.
- The current manual payment handoff normally stops at `triggered`; `booked` is
  used only when an authoritative receipt is available.

### Execution command

`queued -> leased -> completed | failed`

Lease token and expiry fence every heartbeat and result. Losing the lease cancels
browser work. Invalid payload is terminal. A transient infrastructure failure uses
the bounded command retry budget; a seat-unavailable result waits for a distinct
Central availability signal.

The result `oneof`, rather than a reason-string allowlist, decides retry ownership:

| Client result | Examples | Command effect | Monitor effect |
| --- | --- | --- | --- |
| `completed` | payment handoff prepared | complete | Client has already moved it to `triggered` |
| `retry_requested` | transient booking preparation failure | queue while the three-attempt budget remains | stay `running`; fail when the budget is exhausted |
| `failed` with unavailable reason | showtime or preferred seats unavailable | fail until a distinct false-to-true live-seat edge | stay `running` |
| `failed` with user-action reason | login, CAPTCHA, provider block, or contract change | terminal | `failed` with the same stable reason |
| `failed` with ambiguous reason | lease lost or Client interrupted | terminal | `payment_unknown` until the user verifies CGV history |

Deleting a monitor invalidates every queued or leased execution for that monitor.
A deleted monitor can never be claimed, rearmed, or recreated by a late observation.

### Observation

`queued -> leased -> completed | partial | failed | missed`

Only complete requested dates prove absence. Partial, failed, and missed work never
advances the corresponding coverage cursor. First discovery is left-censored: the
opening interval is bounded by the last complete absence and first complete
presence, not reported as an exact opening timestamp.

## Deliberately deferred capabilities

The following are explicit non-capabilities, not unspecified behaviour:

- automatic payment success detection without an authoritative receipt;
- CAPTCHA solving or background credential submission;
- Central-hosted user login, seat selection, or payment iframe;
- mobile-native applications (the responsive UI and service contracts may be
  reused later);
- exact provider-protection thresholds before production telemetry exists.

## Acceptance evidence

Every change to this specification must update the owning Proto and prove:

- duplicate user demand creates one shared provider assignment;
- extended-clock showtimes match the correct civil weekday and time window;
- unchanged live-seat snapshots create no duplicate command;
- a false-to-true preferred-seat transition creates exactly one fenced command;
- an explicit retry request consumes the bounded budget while a terminal failure
  never retries by inference;
- loss of lease cancels Client browser work;
- deletion prevents all later claims and rearms for that monitor;
- a two-second fast deadline is not delayed by the general maintenance interval,
  and an idle scheduler does not busy-loop;
- missing Probe, catalog, layout, session, and healthy egress each expose a distinct
  waiting or terminal reason;
- provider protection signals stop or quarantine work instead of being parsed as
  successful absence.
