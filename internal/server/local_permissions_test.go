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
	if app["id"] != appID {
		t.Fatalf("app identity: %v", app)
	}
	localAdmin(t, hts.URL, "POST", "apps/"+appID+"/secrets", map[string]any{"secretText": "known-local-secret"}, 201)
	_, body := postForm(t, http.DefaultClient, hts.URL+"/"+tenant+"/oauth2/v2.0/token", url.Values{"grant_type": {"client_credentials"}, "client_id": {appID}, "client_secret": {"known-local-secret"}, "scope": {"https://graph.microsoft.com/.default"}})
	if token, _ := body["access_token"].(string); token == "" {
		t.Fatalf("fixed-secret authentication: %v", body)
	}
	localAdmin(t, hts.URL, "DELETE", "apps/"+appID, nil, 204)
	recreated := localAdmin(t, hts.URL, "POST", "apps", map[string]any{"id": appID, "displayName": "Local app"}, 201)
	if recreated["id"] != appID {
		t.Fatalf("app ID changed on recreation: %v", recreated)
	}
}
