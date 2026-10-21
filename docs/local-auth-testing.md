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
