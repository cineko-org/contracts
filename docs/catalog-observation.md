# Catalog and observation

Catalog identity and timestamped observations are separate data.

## Task inventory

| Capability | Input | Successful output | Completeness rule |
| --- | --- | --- | --- |
| `cgv.catalog.capture` | Provider, locale, time zone, egress policy | One validated catalog payload for the reported scope | Empty or partially parsed provider data fails the task |
| `cgv.schedule.capture` | One theater and an explicit date set | One `Capture` for every requested date | Every candidate showtime must parse; a failed date is `complete=false` and cannot prove absence |
| `cgv.seat-map.capture` | Exact theater and auditorium, a bounded date set, and an optional exact-showtime hint | One atomic `Completed.live_seat` observation; Central persists its layout component as the immutable auditorium snapshot | The Probe must visit a currently bookable showtime for that auditorium; stored metadata alone never proves the layout |
| `cgv.seat-availability.capture` | One exact showtime with its theater and auditorium | One atomic `Completed.live_seat` observation containing the current layout and complete available-seat set | Missing, partial, challenged, or identity-mismatched seat data fails the task; it never proves no seats are available |

The verified full-catalog source currently enumerates theaters only. Movies, auditoriums, and showtimes are added
from structured schedule responses, where their provider identifiers are present. Reporters must not guess a movie
catalog endpoint or fall back to a displayed title merely to make the initial catalog appear complete.

`CatalogSnapshot` upserts the Provider, Theater, Movie, Auditorium, and Showtime metadata it contains. A schedule
result upserts that shared catalog and appends availability observations in the same Central transaction.
Availability, sold-out state, and observed time are observations; they are not authoritative catalog attributes.
Schedule availability is an aggregate hint. Exact preferred-seat rearming uses the separate live-seat observation and
never treats a positive aggregate count as proof that a matching seat group exists.

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
| `completed` | The requested capture is valid. A live-seat capture includes layout and availability from the same provider response. | Commit the observation and advance the relevant coverage cursor. |
| `deferred` | The Probe reached a valid stopping point but cannot observe the requested fact yet. `showtime_not_discovered` means the catalog has no matching future showtime yet; `no_bookable_showtime` means the catalog had candidates but none was currently bookable. | Preserve the typed reason and let Central choose the next attempt. |
| `failed` | The provider or execution boundary could not produce a trustworthy result. | Apply Central's reason-specific retry or blocked policy. |

Both seat-page objectives remain distinct because one discovers a static layout
and the other refreshes an exact show's availability. They share the same
atomic wire result: a successful seat-page response always carries
`Completed.live_seat`, with layout and availability tied to one auditorium and
layout hash. There is no standalone `Completed.seat_map` payload.

Probe reports facts through the `DeferredReason` oneof. Central derives the
durable `WaitingReason` for a resolution: `showtime_not_discovered` means the
catalog has not exposed a matching future showtime yet, while
`no_bookable_showtime` and `target_date_unavailable` may be reported by a Probe
when it has inspected the requested scope. A Probe does not set a `retryable`
boolean or choose a backoff.
An incomplete capture may not remove showtimes or assert that none exist. Central deduplicates exact result replays and
retains observations independently from catalog revisions.

## Seat-map validation

`layoutHash` is the only evidence that two observations describe the same static
layout. Central returns its stored current layout immediately to Clients and
never blocks that read on provider access. Freshness and change detection are a
separate background concern and require revisiting a provider seat page; Central
must not invent a time-to-live.

The response contains an optional cached `Snapshot` and one required
`cineko.collection.State`. `idle` is valid only when the snapshot is present; an empty auditorium
must be represented as `queued`, never as an empty idle response. A cached
snapshot therefore remains usable while validation is queued,
running, waiting for a showtime, scheduled for retry, or blocked. Client does
not receive an assignment ID and does not poll at a fixed cadence. It calls
`ResolveSeatMap` for the current state and subscribes to `WatchSeatMap` for
durable changes; reconnecting always starts with the current Central state.

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
persist free-form trigger strings. A
`waiting_for_showtime` state is not an error and is woken by a catalog/showtime
change. A `retry_scheduled` state is woken by its durable `next_attempt_at`. A
`blocked` state requires a new objective trigger or operator action; the
reconciler must not recreate it every maintenance tick.

Central requests one validation when any of these objective events occurs:

- a Client requests an auditorium that has no stored layout;
- an auditorium has no stored layout and gains a future bookable showtime;
- an active booking monitor gains a new showtime for its auditorium;
- reported auditorium or showtime capacity differs from the active layout;
- provider auditorium metadata changes in a way that may describe another layout;
- an operator explicitly requests validation.

The booking Client consumes only Central's resolved layout and does not choose
or run the collection mechanism. A matching Probe observation only advances
`lastSeenAt`; a different hash creates and activates a new immutable version.
For a missing layout, Central provides a bounded date set and the Probe chooses
the earliest currently bookable showtime in the requested auditorium. An exact
showtime is only a hint and must not be required for first collection. When no
future matching showtime is known, the resolution is `waiting_for_showtime`
with Central's typed `showtime_not_discovered` reason. When matching candidates
exist but none is currently bookable, the Probe assignment is `deferred` with
`no_bookable_showtime`, and Central maps that fact to the same waiting state. In both cases the resolution is not
`fresh` or a generic failure.

When a live-seat response contains a different layout hash, Central stores the
new immutable layout version, stores the availability observation, evaluates
seat presets against that exact layout, and emits any execution signal in one
transaction. A separate follow-up layout request is forbidden for the same
provider response.
