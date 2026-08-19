# Versioning

Contracts use semantic versions independently of every product.

- Patch: clarification or fixture correction without changing accepted wire data.
- Minor: additive optional fields, endpoints, enum values, or capabilities.
- Major: removed or renamed fields, new required fields, changed meaning, or incompatible authentication.

Consumers pin an exact released contract version. A producer must remain compatible with every supported
consumer version before it deploys. Protocol 3 is the current approved hard cutover: consumers import
`github.com/cineko-org/contracts/v3`; Protocol 1 and 2 are not accepted. A future breaking change requires either a
transition where Central accepts both protocol majors until all active consumers have upgraded or another explicitly
approved hard cutover.

The `X-Cineko-Protocol` header identifies the wire protocol major. Unknown majors fail before request decoding.

Release payload persistence is versioned separately. Publication requests use
`ReleaseEnvelope<ReleaseSet<T>>`, and Central stores each immutable component as `ReleaseEnvelope<T>`. Both shapes
carry `schemaVersion: 2`; a future incompatible persisted payload must introduce a new schema version and retain a
decoder for every schema version still present in the release registry.
