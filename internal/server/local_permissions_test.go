package server

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/calvinchengx/entra-emulator/internal/store"
)

func localAdmin(t *testing.T, base, method, path string, body any, want int) map[string]any {
	t.Helper()
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	req, err := http.NewRequest(method, base+"/admin/api/"+path, strings.NewReader(string(raw)))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var value map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&value)
	if resp.StatusCode != want {
		t.Fatalf("%s %s: status %d, want %d: %v", method, path, resp.StatusCode, want, value)
	}
	return value
}

func TestLocalExplicitApplicationRoles(t *testing.T) {
	hts, cfg, _ := newTestServer(t)
	cfg.StrictAppRoles, cfg.GUIDAPIAudience = true, true
	resource := "api://" + daemonID
	baseline := appTokenForResource(t, hts.URL, resource)
	if len(rolesOf(baseline)) != 0 || baseline["aud"] != daemonID || baseline["idtyp"] != "app" {
		t.Fatalf("unassigned service: %v", baseline)
	}
	dto := localAdmin(t, hts.URL, "GET", "apps/"+daemonID, nil, 200)
	if baseline["oid"] != dto["servicePrincipalId"] || baseline["oid"] == daemonID {
		t.Fatalf("SP object id: %v / %v", baseline, dto)
	}
	role := dto["appRoles"].([]any)[0].(map[string]any)
	roleID := role["id"].(string)
	path := "apps/" + daemonID + "/roleAssignments/" + daemonID + "/" + roleID
	for range 2 {
		localAdmin(t, hts.URL, "PUT", path, nil, 204)
	}
	assigned := appTokenForResource(t, hts.URL, resource)
	if got := rolesOf(assigned); len(got) != 1 || got[0] != role["value"] {
		t.Fatalf("assigned role: %v", assigned)
	}
	listed := localAdmin(t, hts.URL, "GET", "apps/"+daemonID+"/roleAssignments", nil, 200)
	if len(listed["value"].([]any)) != 1 {
		t.Fatalf("duplicate assignment: %v", listed)
	}
	localAdmin(t, hts.URL, "PATCH", "apps/"+daemonID+"/roles/"+roleID, map[string]any{"isEnabled": false}, 200)
	if got := rolesOf(appTokenForResource(t, hts.URL, resource)); len(got) != 0 {
		t.Fatalf("disabled role in fresh token: %v", got)
	}
	localAdmin(t, hts.URL, "PUT", path, nil, 400)
	for range 2 {
		localAdmin(t, hts.URL, "DELETE", path, nil, 204)
	}
	if got := rolesOf(appTokenForResource(t, hts.URL, resource)); len(got) != 0 {
		t.Fatalf("unassigned role reappeared: %v", got)
	}
}

func TestLocalDefaultDelegatedScopesAcrossGrants(t *testing.T) {
	hts, cfg, _ := newTestServer(t)
	cfg.GUIDAPIAudience = true
	// The client is deliberately different from the API resource.
	consentPath := "apps/" + spaID + "/preauthorizedScopes/" + daemonID
	scopeID := store.NewGUID()
	localAdmin(t, hts.URL, "POST", "apps/"+daemonID+"/scopes", map[string]any{"id": scopeID, "value": "local.read"}, 201)
	localAdmin(t, hts.URL, "POST", "apps/"+daemonID+"/scopes", map[string]any{"value": "unconsented"}, 201)
	localAdmin(t, hts.URL, "POST", "apps/"+daemonID+"/scopes", map[string]any{"value": "disabled", "isEnabled": false}, 201)
	for range 2 {
		localAdmin(t, hts.URL, "PUT", consentPath, map[string]any{"scopes": []string{"local.read", "disabled"}}, 200)
	}
	localAdmin(t, hts.URL, "PUT", consentPath, map[string]any{"scopes": []string{"nonexistent"}}, 400)
	tokenURL := hts.URL + "/" + tenant + "/oauth2/v2.0/token"
	scope := "api://" + daemonID + "/.default"
	password := url.Values{"grant_type": {"password"}, "client_id": {spaID}, "scope": {scope + " openid offline_access"}, "username": {"alice@entraemulator.dev"}, "password": {store.SeedPassword}}
	_, body := postForm(t, http.DefaultClient, tokenURL, password)
	check := func(body map[string]any, want string) {
		t.Helper()
		token, _ := body["access_token"].(string)
		if token == "" {
			t.Fatalf("no token: %v", body)
		}
		claims := decodeJWTPayload(t, token)
		if claims["aud"] != daemonID || claims["scp"] != want {
			t.Fatalf("claims want scp %q/aud %q: %v", want, daemonID, claims)
		}
	}
	check(body, "local.read")
	// Exercise the account picker and code exchange used by MSAL.
	verifier := "local-default-verifier-012345678901234567890"
	query := baseAuthorizeQuery("code")
	query.Set("scope", scope+" openid offline_access")
	query.Set("code_challenge", pkceChallenge(verifier))
	query.Set("code_challenge_method", "S256")
	resp := authorizeAs(t, hts.URL, query)
	resp.Body.Close()
	loc, err := url.Parse(resp.Header.Get("Location"))
	if err != nil {
		t.Fatal(err)
	}
	_, exchange := postForm(t, http.DefaultClient, tokenURL, url.Values{"grant_type": {"authorization_code"}, "client_id": {spaID}, "code": {loc.Query().Get("code")}, "redirect_uri": {redirect}, "code_verifier": {verifier}, "scope": {scope + " openid offline_access"}})
	check(exchange, "local.read")
	// Refresh without an explicit scope must still remove disabled scopes.
	localAdmin(t, hts.URL, "PATCH", "apps/"+daemonID+"/scopes/"+scopeID, map[string]any{"isEnabled": false}, 200)
	_, refreshed := postForm(t, http.DefaultClient, tokenURL, url.Values{"grant_type": {"refresh_token"}, "client_id": {spaID}, "refresh_token": {exchange["refresh_token"].(string)}})
	check(refreshed, "")
	// Re-enabling and then revoking consent changes .default immediately.
	localAdmin(t, hts.URL, "PATCH", "apps/"+daemonID+"/scopes/"+scopeID, map[string]any{"isEnabled": true}, 200)
	localAdmin(t, hts.URL, "PUT", consentPath, map[string]any{"scopes": []string{}}, 200)
	_, revoked := postForm(t, http.DefaultClient, tokenURL, password)
	check(revoked, "")
	// Another client cannot inherit this client's consent.
	localAdmin(t, hts.URL, "PUT", consentPath, map[string]any{"scopes": []string{"local.read"}}, 200)
	password.Set("client_id", daemonID)
	password.Set("client_secret", store.SeedDaemonSecret)
	_, other := postForm(t, http.DefaultClient, tokenURL, password)
	check(other, "")
}

func TestAdminFixedIdentityCreation(t *testing.T) {
	hts, _, _ := newTestServer(t)
	userID, appID := store.NewGUID(), store.NewGUID()
	user := map[string]any{"id": userID, "userPrincipalName": "local@local.test", "displayName": "Local", "password": "password-local"}
	localAdmin(t, hts.URL, "POST", "users", user, 201)
	localAdmin(t, hts.URL, "POST", "users", user, 409)
	localAdmin(t, hts.URL, "GET", "users/"+userID, nil, 200)
	user["id"] = "not-a-uuid"
	user["userPrincipalName"] = "other@local.test"
	localAdmin(t, hts.URL, "POST", "users", user, 400)
	app := localAdmin(t, hts.URL, "POST", "apps", map[string]any{"id": appID, "displayName": "Local app", "isConfidential": true}, 201)
	oid := app["servicePrincipalId"]
	if app["id"] != appID || oid == appID {
		t.Fatalf("app identity: %v", app)
	}
	localAdmin(t, hts.URL, "POST", "apps/"+appID+"/secrets", map[string]any{"secretText": "known-local-secret"}, 201)
	_, body := postForm(t, http.DefaultClient, hts.URL+"/"+tenant+"/oauth2/v2.0/token", url.Values{"grant_type": {"client_credentials"}, "client_id": {appID}, "client_secret": {"known-local-secret"}, "scope": {"https://graph.microsoft.com/.default"}})
	if token, _ := body["access_token"].(string); token == "" || decodeJWTPayload(t, token)["oid"] != oid {
		t.Fatalf("fixed-secret authentication: %v", body)
	}
	localAdmin(t, hts.URL, "DELETE", "apps/"+appID, nil, 204)
	recreated := localAdmin(t, hts.URL, "POST", "apps", map[string]any{"id": appID, "displayName": "Local app"}, 201)
	if recreated["servicePrincipalId"] != oid {
		t.Fatalf("oid changed on reset: %v", recreated)
	}
}
