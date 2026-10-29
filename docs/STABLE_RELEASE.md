# Stable release criteria

[Русский](STABLE_RELEASE.ru.md) | [Validation](VALIDATION.md) | [Acceptance resources](ACCEPTANCE.md)

The current preview is not a certificate of production deployment compatibility. A stable release needs all of the following evidence for its advertised scope:

- Reproducible pinned ABI/source artifacts, documented public API and installation/update/rollback checks for both modules.
- Passing Linux/Windows build, unit/integration tests, race checks, bounded fuzz campaigns and no reachable vulnerability findings in the dated dependency scan.
- Genuine local EVM execution for available official fixtures, with provenance, resulting state/events and expected failures. Historical fixtures must remain separate from deployed instances.
- A reviewed contract identity and signing scheme for each supported deployment generation. The bundled historical Checks artifact and sampled official implementation use distinct signing profiles. The sampled code accepts the JS-source helper in local execution; public instance initialization/funding and settlement still require separate acceptance.
- Explicitly authorized bounded testnet acceptance for stateful advertised routes, with existing-role requirements, receipts, fees and state changes. Public transaction submission has not been authorized in this work.
- Exact official ABI/proof sources for bridge/delegation extensions before promoting those deployments to accepted support. Legacy Cosmos descriptor coverage is not proof of standalone native admission; only the pinned node's admitted routes may be advertised that way.

The first three groups have executable preview checks within their documented fixture scope. They do not close the last three groups. Deployment-level Safe/permit acceptance, initialized mainnet Checks, unavailable deployment ABI and public settlement remain unverified. No readiness percentage or stable tag is inferred from passing offline tests.
