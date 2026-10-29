# Security

This is an independent preview SDK. Compatibility and validation limits are documented in docs/VALIDATION.md, docs/ACCEPTANCE.md and docs/ADDITIONAL_ABI_EVIDENCE.md.

For a security report, use this repository's private vulnerability reporting when available, or contact the maintainers through https://mintcandy.ru/ . Include the affected version and a minimal reproduction using synthetic inputs. Do not send private keys, mnemonic phrases, credentials or a production wallet. Public issues are suitable for reports without sensitive details.

Offline builders do not send transactions. Send/Broadcast are explicit operations; an ambiguous submission must be reconciled by its local hash before retrying. Runtime dependency and toolchain advisories should be reviewed when updating the SDK.
