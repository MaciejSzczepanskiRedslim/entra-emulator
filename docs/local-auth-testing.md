# Local authorization testing

The emulator supports repeatable local authorization tests with deterministic
identities and explicit permission grants. Strict application-role enforcement
and GUID API audiences are opt-in.
The admin API remains unauthenticated: expose it only to trusted local tools.

## Configuration

- `STRICT_APP_ROLES=true` (`strictAppRoles` in JSON) requires explicit application
  role assignments. A service without assignments receives no roles.
- `GUID_API_AUDIENCE=true` (`guidApiAudience`) uses the registered resource's app
  GUID as `aud` for delegated and app-only v2 tokens, including requests using
  its `api://...` URI. First-party resource carve-outs retain their audiences.
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

Every app DTO includes `servicePrincipalId`. App-only tokens contain:

- `oid`: that service-principal object ID;
- `idtyp`: `app`;
- `azp`, `appid`, and `sub`: the app/client ID.

The object ID is tenant-specific and distinct from the client ID. Its stable
algorithm is SHA-256 of UTF-8
`entra-emulator:service-principal:<tenantId>:<appId>`, first 16 bytes, setting UUID
version bits to 8 and RFC 4122 variant bits, formatted as a lowercase UUID. Prefer
reading the DTO over duplicating this algorithm. Directory resets and recreation
with the same tenant/app IDs preserve it. The Graph directory continues using app
IDs for service-principal navigation; admin URLs below also take **app IDs**.

## Delegated preauthorization

```http
PUT /admin/api/apps/{clientAppId}/preauthorizedScopes/{resourceAppId}
Content-Type: application/json

{"scopes":["access_as_user","user_impersonation"]}
```

This atomically replaces tenant-wide consent for that pair. An empty list revokes
all tenant-wide consent while keeping an explicit empty grant. Scope values must
be declared by the resource; disabled scopes may remain preauthorized but never
appear in a newly minted access token. Client/resource must share a tenant.
`GET` at the same URL returns `{"scopes":[...]}` including disabled declarations.
Per-user grants created through Graph remain independent.

Delegated `<resource>/.default` expands only the requesting client's stored
consent, intersected with enabled exposed scopes. User-specific consent is
intersected again for the actual user before minting. No consent gives an empty
`scp`; it does not implicitly grant every API scope. Authorization code, password,
device code, refresh and OBO resolution pass the requesting client identity. A
refresh chain or code issued before a scope was disabled cannot restore it.
Explicit named delegated scopes retain the existing emulator behavior.

## Application roles

```http
PUT /admin/api/apps/{clientAppId}/roleAssignments/{resourceAppId}/{roleId}
DELETE /admin/api/apps/{clientAppId}/roleAssignments/{resourceAppId}/{roleId}
GET /admin/api/apps/{clientAppId}/roleAssignments
```

PUT and DELETE are idempotent and return 204. PUT requires a registered, enabled
role with `Application` in its allowed member types. DELETE also works after the
role was disabled or deleted. GET returns
`{"value":[{"id":"...","resourceId":"...","appRoleId":"..."}]}`.
Assignment lookup uses the same store as Graph appRoleAssignments. Disabled roles,
User-only roles and roles from another resource never appear in app-only tokens.

Use existing `PATCH /admin/api/apps/{resourceAppId}/roles/{roleId}` or
`/scopes/{scopeId}` with `{"isEnabled":false}` to disable a declaration. Changes
apply to newly issued tokens; refresh tokens or sign in again to obtain updated
claims.
