# Audit trail test vectors

`test-vectors/` is a verbatim copy of the shared test vectors of Yontrack:

- repository: `yontrack/yontrack`
- directory: `ontrack-extension-audit-trail/src/test/resources/audit-trail/test-vectors/`
- commit: `6f0cb372c7d7c20d24a0bb8c094b5854e4f920b5` (`main`, 2026-10-08)

They are the reference of the hash format v1 and of the export format: never edit them here. When
Yontrack changes them, copy them again and update the commit above.

- `01-build-created.json`, `02-trail-opened.json`, `03-canonical-forms.json` — trails as
  `{description, schemaVersion, entries: [{envelope, canonical, hash}]}`: each envelope, its
  canonical form and its hash.
- `endorsements/01-rfc8032-test1.json` — the endorsements of the entries of `01-build-created.json`
  by the published key of RFC 8032, section 7.1, TEST 1.
- `exports/01-build-created.json` — an intact export, endorsed by that key.
- `exports/02-tampered-payload.json` — the same export, whose seq 2 payload was edited.
