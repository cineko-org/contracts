# Catalog and observation

Catalog identity and timestamped observations are separate data.

## Task inventory

| Capability | Input | Successful output | Completeness rule |
| --- | --- | --- | --- |
| `cgv.catalog.capture` | One provider ID, locale, time zone, and egress policy; no theater or date scope | One validated catalog payload for the reported provider | Empty or partially parsed provider data fails the task |
| `cgv.schedule.capture` | One theater and an explicit date set | One `Capture` for every requested date | Every candidate showtime must parse; a failed date is `complete=false` and cannot prove absence |
| Client seat-map collection | Central's short-lived `SeatMapCollectionClaim`: claim authority, expiry, and one exact theater/auditorium/showtime task | One claim-authorized `SubmitLiveSeatObservation`; Central persists its layout component as the immutable auditorium snapshot | Current CGV seat pages require member login or non-member phone verification, so anonymous Probe collection is forbidden |
| Client exact-seat task | One exact showtime with its theater and auditorium | One atomic live-seat observation containing the current layout and complete available-seat set | This typed task is local to authenticated Client booking and is never assigned to an anonymous Probe |
| Client live-seat report | Exactly one authority: collection `claimId/claimToken` or execution `commandId/leaseToken`, plus the atomic observation | The active immutable layout snapshot after Central canonicalization | Empty availability means sold out; it does not mean that the required layout is absent |

The verified full-catalog source currently enumerates theaters only. Movies, auditoriums, and showtimes are added
from structured schedule responses, where their provider identifiers are present. Reporters must not guess a movie
catalog endpoint or fall back to a displayed title merely to make the initial catalog appear complete.

Catalog discovery is provider-global and date-independent. `CatalogTask` identifies the provider, not a theater,
and has neither `theater` nor `target_dates`: those scopes belong to schedule work and the generated Client seat-map task. A stale
producer that sends `theater` or `targetDates` is rejected by strict latest-message ProtoJSON decoding instead of
silently narrowing a catalog capture.

`CatalogSnapshot` upserts the Provider, Theater, Movie, Auditorium, and Showtime metadata it contains. A schedule
result upserts that shared catalog and appends availability observations in the same Central transaction.
Availability, sold-out state, and observed time are observations; they are not authoritative catalog attributes.
Schedule availability is an aggregate hint. Central creates one Client execution for every matching future showtime,
including sold-out and zero-seat rows; the authenticated live page alone proves whether a preferred group can be held.

Movie rows are permanent analytical identity. Central never removes a Movie merely because it is no longer offered,
and timestamped availability observations reference its canonical ID. Client catalog responses contain only movies
with a current bookable showtime; historical movies remain available to Central and observability queries instead of
cluttering the booking UI. The `movies` array preserves the provider's first-appearance order as a current
presentation hint. That order never participates in identity or rewrites historical observations.

Catalog capture does not carry an authoritative scope marker, expected entity count, or tombstone set.
Therefore, absence from a catalog payload cannot deactivate or delete an existing entity. Authoritative replacement
requires a future wire revision that makes scope and completeness explicit.

## Identity

- For CGV, `SourceKey` is always a provider identifier. Presentation text is never an identity fallback.
- CGV theater identity is `siteNo` from `searchAllRegionAndSite`.
- CGV movie identity is `movNo` from `searchMovScnInfo` (or the equivalent structured movie response). `prodNo` and
  `movfNo` describe a product/format variant and must remain metadata; they must not split the Movie row.
- CGV auditorium identity is `siteNo/scnsNo`. The screen number is provider data even when the displayed auditorium
  name changes.
- CGV showtime identity is `siteNo/scnYmd/scnsNo/scnSseq`. A displayed title, start-time label, or auditorium name
  change must not create another showtime when this tuple is unchanged.
- A candidate missing any required provider key is rejected and the capture is incomplete. It must not be silently
  converted to a display-text identity.
- `CatalogID(provider, kind, sourceKey)` is the only canonical ID derivation. Reporters do not invent opaque IDs.
- `ObservedAt` records when the reporter saw the value. Central receipt time never replaces it.

## Assignment result states

An assignment result is exactly one of these typed outcomes:

| Outcome | Meaning | Central action |
| --- | --- | --- |
| `completed` | The requested catalog or schedule capture is valid. | Commit the observation and advance the relevant coverage cursor. |
| `deferred` | The Probe reached a valid stopping point but cannot observe the requested schedule date. | Preserve `target_date_unavailable` and let Central choose the next discovery attempt. |
| `failed` | The provider or execution boundary could not produce a trustworthy result. | Apply Central's reason-specific retry or blocked policy. |

Both Client seat-page objectives remain distinct because one fulfills a missing
static layout and the other refreshes an exact show's availability during booking.
They share the same atomic live-seat observation, with layout and availability
tied to one auditorium and layout hash. Anonymous Probe assignment results do not
carry these Client-only objectives.

Probe reports schedule scope through the `DeferredReason` oneof using
`target_date_unavailable`. Central alone derives the durable `WaitingReason` for
a seat-map resolution: `showtime_not_discovered` means the catalog has not
exposed a matching future showtime yet. A Probe does not set a `retryable`
boolean or choose a backoff.
An incomplete capture may not remove showtimes or assert that none exist. Central deduplicates exact result replays and
retains observations independently from catalog revisions.

## Seat-map validation

`layoutHash` is the only evidence that two observations describe the same static
layout. Central returns its stored current layout immediately to Clients and
never blocks that read on provider access. Freshness and change detection are a
separate background concern and require revisiting a provider seat page; Central
must not invent a time-to-live.

The response contains an optional cached `Snapshot`, one required
`cineko.collection.State`, and, on a cache miss claimed for the requesting
authenticated Client, a short-lived `SeatMapCollectionClaim` with authority,
expiry, and an exact `SeatMapTask`. `idle` is valid only when the snapshot is
present. A cached
snapshot therefore remains usable while validation is queued,
running, waiting for a showtime, scheduled for retry, or blocked. Client does
not receive a Probe assignment and does not poll at a fixed cadence. It calls
`ResolveSeatMap` for the current state and subscribes to `WatchSeatMap` for
durable changes; reconnecting always starts with the current Central state.
`WatchSeatMap` carries only the resolution and never exposes claim authority.

The state machine is:

```text
idle -> queued -> collecting -> idle(snapshot)
                         \-> waiting_for_showtime
                         \-> retry_scheduled -> queued
                         \-> blocked
```

`idle` without a snapshot is an invalid Central domain state. The queued state
records a typed trigger (`client_request`, `active_monitor`, `layout_missing`,
`layout_changed`, `catalog_refresh`, or `operator_request`) so Central does not
persist free-form trigger strings. It is internal: `ResolveSeatMap` must
atomically claim it or transition it and can never return a snapshot-less queued
response. `Collecting` stores the claim ID, start time, and expiry. A claim in
the response is valid only when its ID and expiry match that state; a collecting
state without a claim means another Client owns it. A
`waiting_for_showtime` state is not an error and is woken by a catalog/showtime
change. A `retry_scheduled` state is woken by its durable `next_attempt_at`. A
`blocked` state requires a new objective trigger or operator action; the
reconciler must not recreate it every maintenance tick.

Central requests one collection or validation when any of these objective events occurs:

- a Client requests an auditorium that has no stored layout;
- an auditorium has no stored layout and gains any future showtime, including a
  sold-out or zero-availability showtime;
- an active booking monitor gains a new showtime for its auditorium;
- reported auditorium or showtime capacity differs from the active layout;
- provider auditorium metadata changes in a way that may describe another layout;
- an operator explicitly requests validation.

The booking Client consumes Central's resolved layout and never chooses the
collection target. On a cache miss, it runs only Central's exact claimed task
inside its authenticated CGV session and submits the atomic live-seat result
with the matching claim ID and token before expiry.
A matching Client observation only advances `lastSeenAt`; a different hash
creates and activates a new immutable version. The exact showtime is mandatory
for Client collection and is selected from Central's durable catalog. When no
future matching showtime is known, the resolution is `waiting_for_showtime`
with Central's typed `showtime_not_discovered` reason. Sold-out and zero-seat
future showtimes remain valid collection targets because aggregate schedule
availability does not prove that the seat page or static layout is absent.
The task contains no date search range: theater site, auditorium screen, and
showtime site/screen identities must match exactly at contract validation.

When a live-seat response contains a different layout hash, Central stores the
new immutable layout version and the availability observation in one transaction.
The already leased Client evaluates its preset on that exact same-page response;
Central does not emit another execution command. A separate follow-up layout
request is forbidden for the same provider response.

An authenticated Client submits the same `LiveSeatObservation` after its
pre-booking seat-page recheck, authorized by the command ID and lease token.
Central owns normalization and revision state:
the same normalized layout fingerprint refreshes the current revision's
observation time, while a different fingerprint expires the previous current
revision and activates one new immutable revision. The Client does not submit a
revision number, current flag, or storage-shaped DTO.
