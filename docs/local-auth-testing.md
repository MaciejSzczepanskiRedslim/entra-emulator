# Local authorization testing

The emulator supports repeatable local authorization tests with deterministic
identities and explicit permission grants. Strict application-role enforcement
and GUID API audiences are opt-in.
The admin API remains unauthenticated: expose it only to trusted local tools.

## Configuration

- Set `PUBLIC_ORIGIN=https://entra.localhost:<port>` and
  `ISSUER=https://entra.localhost:<port>/<tenant>/v2.0` explicitly. Discovery and
  issued tokens use these values regardless of the incoming HTTP Host header.
  `HOST=::` binds IPv6 (and IPv4 on hosts with dual-stack sockets enabled).
- `REQUIRE_PASSWORD=false` retains the local account picker; password grants
  still check the user's stored password.

Restart the emulator after changing configuration. Directory changes below take
immediate effect for **new** tokens. Previously issued JWTs remain valid until
expiry; refresh tokens, acquire a new token, or sign out and back in.

## Deterministic identities

`POST /admin/api/users` and `POST /admin/api/apps` accept optional `id` fields.
Scope/role creation (`POST /admin/api/apps/{id}/scopes` or `/roles`) also accepts
`id`. Explicit IDs must be lowercase, nonzero UUIDs. Duplicate IDs return 409;
GET then PATCH the existing identity to reconcile partial provisioning. Password
strings go through the existing secret hashing code; import files are unnecessary.

`POST /admin/api/apps/{id}/secrets` accepts optional `secretText` (at least eight
bytes) to replay a declarative local secret. Omit it for the existing random,
show-once behavior. Only its hash and a hint are stored. Use the existing secret
listing/deletion API to avoid accumulating duplicates during reconciliation.
