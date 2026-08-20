# CGV provider contract

This document is the source of truth for CGV identity, schedule discovery, and
the handoff from a discovered showtime to seat selection. Implementations must
fail closed when the provider evidence described here is unavailable. Display
text is never a provider identity.

## Evidence boundary

The examples below were observed on the CGV web application on 2026-08-20.
They contain no cookie, credential, signature, account identifier, or complete
request URL. A later provider change must be captured and reviewed before this
contract is changed.

The theater catalog response from
`/api/v1/content/site/searchAllRegionAndSite` contains stable theater fields:

```json
{
  "siteNo": "0056",
  "siteNm": "용산아이파크몰",
  "regnGrpNm": "서울"
}
```

The schedule response from `/api/v1/booking/searchMovScnInfo` contains the
provider tuple and display metadata for one showtime:

```json
{
  "siteNo": "0056",
  "siteNm": "용산아이파크몰",
  "movNo": "00001234",
  "movNm": "표시 영화명",
  "movfNo": "format-file-01",
  "scnsNo": "0007",
  "scnsNm": "IMAX관",
  "scnYmd": "20260820",
  "scnSseq": "0003",
  "prodNo": "product-01",
  "scnsrtTm": "2530",
  "scnendTm": "2832",
  "frSeatCnt": "2",
  "stcnt": "624"
}
```

`prodNo` and `movfNo` can differ between showtimes for the same `movNo`.
They are product and format metadata, not movie identity.

The schedule page renders a showtime as an ordinary button. The observed
button had only `type`, `class`, and accessibility or disabled attributes. It
did not contain `siteNo`, `movNo`, `scnsNo`, `scnSseq`, `data-source-key`,
`data-showtime-source-key`, or another canonical identity attribute.

```html
<button type="button" class="screenInfo_timeLink__...">
  <span><span>19:30</span><span>- 21:31</span></span>
  <span><span>1</span><span>/184석</span></span>
  <span>2관 (Laser)</span>
</button>
```

Selecting that button moved the application from `/cnm/movieBook/cinema` to
`/cnm/selectVisitorCnt`. The seat page displayed the movie, date, theater,
auditorium, start and end time, and seat count. Showtime identity attributes
were still absent. Selectable seats used `data-seatlocno`, for example:

```html
<button data-seatlocno="00100100050001">...</button>
```

This proves that invented showtime `data-*` attributes are not a valid
integration boundary. `data-seatlocno` is a seat selector only.

## Canonical identity

Provider keys are normalized by trimming surrounding whitespace and by
normalizing `scnYmd` to `YYYY-MM-DD`. Missing components are invalid.

| Entity | Canonical provider source key | Display-only fields |
| --- | --- | --- |
| Theater | `siteNo` | `siteNm`, region name |
| Movie | `movNo` | `movNm`, poster, rating |
| Auditorium | `siteNo/scnsNo` | `scnsNm`, format labels |
| Showtime | `siteNo/YYYY-MM-DD/scnsNo/scnSseq` | title, auditorium name, start/end time |

The Cineko catalog ID is `CatalogID("cgv", kind, sourceKey)`. A display-name
change must not change the ID. Two rows with different `scnSseq` are different
showtimes even when their displayed movie, auditorium, and time are equal.

There is no approved title-derived movie fallback, name-derived theater or
auditorium fallback, or time-derived showtime fallback. A guessed provider key
would split history and can direct a booking to the wrong showtime.

## Provider time semantics

`scnYmd` is the provider's schedule date and remains part of showtime identity.
CGV can express after-midnight starts with extended clocks such as `2530`.
For scheduling and user filters, `20260820` plus `2530` is 2026-08-21 01:30 in
the theater time zone. The showtime source key still uses schedule date
`2026-08-20`; it is not rewritten to the civil date.

A user time window is start-inclusive and end-exclusive. An overnight window
such as 21:00-06:00 accepts both 23:00 and the following civil day's 01:00.
Weekday filtering uses the actual civil start instant: Friday schedule date at
25:00 is Saturday 01:00 and therefore matches Saturday.

## Schedule discovery

Probe owns public schedule discovery. For each requested theater and date it
captures the signed provider schedule response in the browser session and
parses every candidate row. The result is complete only when all candidate rows
are valid. One malformed or missing canonical tuple makes the requested date
fail closed; silently dropping a row can hide the newly opened showtime.

Catalog snapshots are additive observations. Omission is not deletion. A full
catalog refresh may clear its refresh request only after a non-empty, complete
provider capture succeeds.

Central owns scan planning:

1. Active user booking demand has absolute priority for its theater.
2. A hot scan contains the union of active monitors' explicit target dates and
   weekday projections within their horizon.
3. A baseline scan advances one date through the configured horizon. After one
   successful hot scan, one bounded baseline date may run before the next
   recurring, unchanged hot scan so ordinary coverage cannot starve.
4. At most one scan for the same theater is active at once. New or changed hot
   demand preempts a queued baseline; after that baseline completes, hot work
   regains priority.
5. Time windows filter showtimes and downstream deep work. If the provider API
   returns an entire date, a time window does not falsely claim to reduce that
   date request itself.

Only a complete assignment advances the hot-success timestamp or baseline date
cursor. Partial, failed, and missed work remains due and cannot unlock the next
lane.

## Booking handoff

Client owns authenticated booking and seat selection. Launcher owns only
version resolution, verified runtime activation, and process lifecycle. Central
does not drive a user's browser, and Launcher does not choose a showtime.

Client must use this sequence:

1. Receive a Central execution command containing the exact provider tuple.
2. In the authenticated CGV browser, capture the schedule response for the
   commanded theater and date.
3. Find exactly one response row whose canonical tuple equals the command.
4. Associate that row with exactly one rendered schedule button using its
   response display projection: movie title, auditorium name, schedule date,
   start/end time, and observed seat totals. These values select the current UI
   row; they never replace canonical identity.
5. If zero or multiple UI rows match, stop before navigation.
6. Select the row and verify the visitor/seat page displays the same movie,
   date, theater, auditorium, and start/end time as the captured response row.
7. Select seats using the provider-rendered seat label and `data-seatlocno`.
8. If any boundary changes or cannot be proven, report a typed provider-contract
   failure. Do not choose a nearby row, a title match, or the first button.

## Warm browser ownership

Client may keep two authenticated booking processes warm and may grow to a
bounded maximum of three under demand. Every lease is exclusive. A browser
handed to payment remains leased until payment handoff ends or expires.

Each process has one owner responsible for closing the browser, stopping its
driver, terminating descendants, and reaping the root process. Shutdown and
crash replacement are bounded and idempotent. A slot is reusable only after the
previous process tree is proven gone; otherwise the pool must fail closed
rather than accumulate zombie processes.

Central and Launcher must not create this pool. Central only prioritizes and
wakes work; Launcher supervises one Client runtime and its startup readiness.

## Required regression cases

- Theater, movie, and auditorium display names change while canonical IDs stay
  stable.
- Same movie, auditorium, and displayed time with different `scnSseq` stays
  distinct.
- Missing `siteNo`, `movNo`, `scnsNo`, `scnYmd`, or `scnSseq` fails the date.
- Compact and hyphenated dates normalize to the same source key.
- Extended clocks map to the correct civil instant and weekday.
- `오늘 12`, `오늘12`, `12 오늘`, and `12오늘` remain accepted UI labels.
- A schedule button without canonical `data-*` attributes follows the verified
  response-to-display handoff; invented attributes are rejected by tests.
- Ambiguous display rows stop before navigation.
- Seat-page display mismatch stops before seat selection.
- Hot target dates exclude unrelated baseline dates; new or changed hot demand
  preempts a queued baseline, while unchanged recurring hot work alternates
  with at most one bounded baseline date.
- Two warm slots initialize concurrently, leases remain exclusive, crashes are
  replaced, and graceful shutdown leaves no driver or browser process.

## Change rule

A provider contract change requires all three in the same review:

1. sanitized evidence from the current CGV response or DOM,
2. this document update, and
3. focused regression tests in every affected owner (Probe, Central, or Client).

Tests that only invent provider HTML or JSON are insufficient evidence.
