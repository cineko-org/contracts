# Launcher and Central

All requests use HTTPS and `X-Cineko-Protocol: 3`.

Central attaches `X-Cineko-Release-Generation` to every response. This monotonic deployment instruction is
independent of the Central service's own application version: restarting or upgrading Central does not change the
generation. Publishing and selecting a different desktop Launcher or runtime component does. Probe container
inventory does not. Launcher resolves the current manifest only when the observed generation changes.

| Method | Path | Purpose |
| --- | --- | --- |
| `POST` | `/v1/auth/pin` | Exchange a six-digit team PIN for a Client session |
| `POST` | `/v1/auth/refresh` | Rotate an expiring Client session |
| `POST` | `/v1/auth/logout` | Revoke the current Client session |
| `GET` | `/v1/releases/runtime/current` | Resolve independently released Client, Chromium, and Playwright components |
| `GET` | `/v1/releases/launcher/current` | Resolve the Launcher release for platform and architecture |
| `PUT` | `/v1/devices/{installationId}` | Register or refresh a Launcher installation |
| `POST` | `/v1/launch-tickets` | Authorize one exact Client artifact launch |

Client, Chromium, and Playwright are independent releases with separate versions and immutable artifacts. Central
resolves a compatible runtime set for the requested platform: the Client declares its minimum Launcher, minimum
Chromium revision, and Playwright version; Chromium declares the Playwright versions it was verified with. A launch
request selects the newest published Chromium that meets those constraints and the exact declared Playwright
version, so component publishing does not require rebuilding the Client. A launch ticket binds user, installation,
device, release generation, Client version, Client artifact digest, the selected browser revision,
protocol, and a Launcher nonce.

Launcher, Client, and Playwright artifacts are uploaded before their metadata is registered. Their stable object
keys are `{product}/v{version}/{os}-{arch}/{filename}` in the `cineko-releases` bucket and are exposed as
immutable downloads below an operator-configured release origin. Chromium is not redistributed: its release points
to the version-pinned official Chrome for Testing archive for each platform. The Browser release workflow downloads
each official archive, verifies its executable, and records the observed size and SHA-256 before registering the
complete platform set. Central never stores metadata for an artifact that the publisher did not verify.

Release registration uses `{ "schemaVersion": 2, "payload": { "releases": [...] } }`. One request contains every
supported platform for one immutable component version and activates the set in one transaction, advancing the
desktop generation exactly once. Central persists each item as
`{ "schemaVersion": 2, "payload": { ...release } }`; it selects the decoder by schema version before validating the
typed component. Partial sets and later additions or mutations are rejected; replaying the identical complete set is
idempotent. Probe release registration uses the same envelope without advancing the desktop generation.

Launcher checks its own release before resolving the Client runtime. A newer Launcher produces a mandatory update
screen with Central's platform-specific portable download URL. Launcher never installs or replaces itself. Windows
uses a portable executable, Linux uses a portable application image, and macOS uses a compressed app bundle without
an installer. Linux uses a self-contained portable artifact; the release pipeline must not label a host-dependent raw
binary as portable.
