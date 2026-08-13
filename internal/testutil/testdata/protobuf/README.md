# Public protobuf transport fixtures

These transport-level fixtures replace the removed handwritten JSON envelope
fixtures. Values are deliberately stored as reviewable hexadecimal or Base64
text; tests decode them before passing bytes to the generated Connect handler.

- `sound-manifest-valid.hex`: a two-byte `SoundManifestRequest` selecting the
  allowlisted ambient category.
- `malformed-truncated.hex`: a truncated length-delimited protobuf field.
- `unknown-field.hex`: a syntactically valid unknown varint field, used as the
  seed for unknown-field growth up to and beyond the 4 KiB decoded limit.
- `sound-manifest-valid.gzip.base64`: deterministic gzip (`gzip -n`) of the
  valid request, used to exercise compressed Connect input.
- `limits.json`: the accepted/rejected encoded and decoded boundary sizes.

The compatibility edit catalog and committed descriptor baseline live beside
these files as `breaking-fixtures.json` and `proto/compatibility-baseline.binpb`.
