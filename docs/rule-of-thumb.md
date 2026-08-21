# Contract rule of thumb

`contracts/main` is the only Cineko service-contract authority. A build either uses generated code from that exact
schema or does not ship.

## Hard rules

- Protocol majors, schema-version fields, versioned package paths, compatibility readers, and legacy decoders are
  forbidden.
- `reserved` declarations are forbidden. A breaking field change is a coordinated maintenance cutover, not a
  multi-schema runtime.
- Semantic state, kind, mode, capability, and outcome values use required `oneof` messages. Semantic enums and magic
  strings are forbidden at service boundaries.
- `Any`, `Struct`, `Value`, and untyped byte or JSON payloads are forbidden at service boundaries.
- A service validates every inbound message with Protovalidate before mapping it to its internal domain model.
- Queryable domain facts are persisted in normalized columns. When an aggregate service payload must be retained,
  its only allowed representation is ProtoJSON of the current generated message; handwritten persistence DTOs,
  envelopes, and compatibility shapes are forbidden.
- Generated messages returned to an untrusted renderer redact secret values. The same message carries only an
  explicit presence flag such as `has_password` or `has_secret`; mutation inputs may carry a replacement secret,
  but read responses never do.
- Every durable state and optimistic-concurrency revision needed to resume a mutation after reload survives a Proto
  round trip. The same applies to canonical identities needed to render or reconcile the durable record. Collapsing
  distinct domain states, dropping identity, or silently writing revision zero is forbidden.
- Launcher, Client, Central, and Probe retain independent application SemVer. Those release versions do not identify
  a contract schema.

## Change procedure

1. Change the owning domain proto and its service request or response in one Contracts PR.
2. Run format, lint, rule checks, deterministic Go and TypeScript generation, runtime validation tests, and both
   language compilers.
3. Regenerate every affected consumer from `contracts/main`; hand-written DTO copies are not allowed.
4. Validate all affected normal PRs before merge.
5. Drain old assignments and sessions, deploy the coordinated set, and resume traffic only after every component
   reports healthy.

There is no compatibility window. In-flight work from the previous contract is drained or discarded during the
maintenance cutover.
