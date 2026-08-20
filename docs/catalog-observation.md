# Catalog and observation

Catalog identity and timestamped observations are separate data.

## Task inventory

| Capability | Input | Successful output | Completeness rule |
| --- | --- | --- | --- |
| `cgv.catalog.capture.v1` | Provider, locale, time zone, egress policy | One validated catalog payload for the reported scope | Empty or partially parsed provider data fails the task |
| `cgv.schedule.capture.v2` | One theater and an explicit date set | One `Capture` for every requested date | Every candidate showtime must parse; a failed date is `complete=false` and cannot prove absence |
| `cgv.seat-map.capture.v1` | Exact auditorium/showtime context | One versioned static layout | Static layout is optional analysis data and never gates booking execution |

The verified full-catalog source currently enumerates theaters only. Movies, auditoriums, and showtimes are added
from structured schedule responses, where their provider identifiers are present. Reporters must not guess a movie
catalog endpoint or fall back to a displayed title merely to make the initial catalog appear complete.

`CatalogSnapshot` upserts the Provider, Theater, Movie, Auditorium, and Showtime metadata it contains. A schedule
result upserts that shared catalog and appends availability observations in the same Central transaction.
Availability, sold-out state, and observed time are observations; they are not authoritative catalog attributes.

Movie rows are permanent analytical identity. Central never removes a Movie merely because it is no longer offered,
and timestamped availability observations reference its canonical ID. Client catalog responses contain only movies
with a current bookable showtime; historical movies remain available to Central and observability queries instead of
cluttering the booking UI. The `movies` array preserves the provider's first-appearance order as a current
presentation hint. That order never participates in identity or rewrites historical observations.

Protocol v3 does not carry an authoritative scope marker, expected entity count, or tombstone set for catalog capture.
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

## Result states

- `completed`: every requested date is complete, or the single catalog/seat-map payload is valid.
- `partial`: at least one requested date is complete and at least one is explicitly incomplete.
- `failed`: no requested date is complete, parsing is ambiguous, or task execution failed.

An incomplete capture may carry diagnostic `errorCode`, but it must not remove showtimes or assert that none exist.
Central deduplicates exact result replays and retains observations independently from catalog revisions.
