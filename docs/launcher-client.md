# Launcher and Client

Launcher starts only a verified Client artifact. The handoff contains a one-time launch ticket, Client nonce,
Central URL, installation identity, expected artifact digest, browser revision, and protocol major.

The handoff also includes the release generation used to assemble the runtime. Client compares it with the
`X-Cineko-Release-Generation` received on ordinary Central API responses and the existing event-stream keepalive.
On a mismatch it exits with the update-required result so Launcher resolves and activates the new manifest. No
update-only poll or stream exists.

The Client exchanges the ticket directly with Central. Launcher never forwards a reusable Client session or CGV
credential. Failed update or digest verification prevents launch; the previously installed runtime is not used as
an implicit fallback.

Launcher checks its own version before starting Client. When a newer Launcher is required it stops before Client
activation and presents the portable download link supplied by Central; it never self-installs or self-replaces.

Client, Chromium, and Playwright are installed and cached independently. Launcher downloads only components whose
verified SHA-256 is absent locally, resumes partial downloads with HTTP Range, writes a new active manifest
atomically, and then removes unreferenced component versions, completed download blobs, and failed staging data.

An embedded Probe receives a short-lived ES256 JWS with type `Cineko-Probe-Bootstrap`. The signed claims bind
the user, device, installation, runtime identity, and concurrency of exactly one Client Probe registration. Only
Central owns the private signing key; Launcher and Client distributions contain the public verification keyring.
