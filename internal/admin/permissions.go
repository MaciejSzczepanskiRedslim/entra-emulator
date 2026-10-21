package admin

import (
	"net/http"
	"regexp"
	"strings"

	"github.com/calvinchengx/entra-emulator/internal/httpx"
	"github.com/calvinchengx/entra-emulator/internal/store"
)

var fixedIDPattern = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

func validFixedID(w http.ResponseWriter, id string) bool {
	if !fixedIDPattern.MatchString(id) || id == store.ZeroGUID {
		httpx.WriteAdminError(w, 400, "validation_error", "id must be a nonzero lowercase UUID.")
		return false
	}
	return true
}

func (a *Admin) registerPermissions(mux *http.ServeMux) {
	mux.HandleFunc("GET /admin/api/apps/{id}/preauthorizedScopes/{resourceId}", a.preauthorizedScopes)
	mux.HandleFunc("PUT /admin/api/apps/{id}/preauthorizedScopes/{resourceId}", a.preauthorizedScopes)
	mux.HandleFunc("GET /admin/api/apps/{id}/roleAssignments", a.listRoleAssignments)
	mux.HandleFunc("PUT /admin/api/apps/{id}/roleAssignments/{resourceId}/{roleId}", a.setRoleAssignment)
	mux.HandleFunc("DELETE /admin/api/apps/{id}/roleAssignments/{resourceId}/{roleId}", a.setRoleAssignment)
}

func (a *Admin) permissionApps(w http.ResponseWriter, r *http.Request) bool {
	client, err := a.Store.GetApp(r.PathValue("id"))
	if err != nil {
		writeStoreErr(w, err)
		return false
	}
	resource, err := a.Store.GetApp(r.PathValue("resourceId"))
	if err != nil {
		writeStoreErr(w, err)
		return false
	}
	if client.TenantID != resource.TenantID {
		httpx.WriteAdminError(w, 400, "invalid_reference", "Client and resource must belong to the same tenant.")
		return false
	}
	return true
}

func (a *Admin) preauthorizedScopes(w http.ResponseWriter, r *http.Request) {
	if !a.permissionApps(w, r) {
		return
	}
	client, resource := r.PathValue("id"), r.PathValue("resourceId")
	if r.Method == http.MethodPut {
		var body struct {
			Scopes []string `json:"scopes"`
		}
		if !decodeBody(w, r, &body) {
			return
		}
		exposed, err := a.Store.ListScopes(resource)
		if err != nil {
			writeStoreErr(w, err)
			return
		}
		known := map[string]bool{}
		for _, scope := range exposed {
			known[scope.Value] = true
		}
		for _, scope := range body.Scopes {
			if !known[scope] {
				httpx.WriteAdminError(w, 400, "invalid_reference", "Scope is not exposed by the resource: "+scope)
				return
			}
		}
		if err := a.Store.SetTenantConsent(client, resource, strings.Join(body.Scopes, " ")); err != nil {
			writeStoreErr(w, err)
			return
		}
	}
	scopes, _, err := a.Store.ConsentedScopes(client, resource, "")
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	// Resource declaration order gives a stable response without a second sort.
	exposed, err := a.Store.ListScopes(resource)
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	values := []string{}
	for _, scope := range exposed {
		if scopes[scope.Value] {
			values = append(values, scope.Value)
		}
	}
	httpx.WriteJSON(w, 200, map[string]any{"scopes": values})
}

func (a *Admin) setRoleAssignment(w http.ResponseWriter, r *http.Request) {
	if !a.permissionApps(w, r) {
		return
	}
	client, resource, roleID := r.PathValue("id"), r.PathValue("resourceId"), r.PathValue("roleId")
	if r.Method == http.MethodDelete {
		if err := a.Store.SetApplicationRole(client, resource, roleID, false); err != nil {
			writeStoreErr(w, err)
			return
		}
		w.WriteHeader(204)
		return
	}
	roles, err := a.Store.ListRoles(resource)
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	valid := false
	for _, role := range roles {
		if role.ID == roleID && role.IsEnabled && strings.Contains(","+role.AllowedMemberTypes+",", ",Application,") {
			valid = true
		}
	}
	if !valid {
		httpx.WriteAdminError(w, 400, "invalid_reference", "Role must be an enabled Application role on the resource.")
		return
	}
	if err := a.Store.SetApplicationRole(client, resource, roleID, true); err != nil {
		writeStoreErr(w, err)
		return
	}
	w.WriteHeader(204)
}

func (a *Admin) listRoleAssignments(w http.ResponseWriter, r *http.Request) {
	if _, err := a.Store.GetApp(r.PathValue("id")); err != nil {
		writeStoreErr(w, err)
		return
	}
	assignments, err := a.Store.ListAppRoleAssignmentsForPrincipal(r.PathValue("id"))
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	values := []map[string]any{}
	for _, assignment := range assignments {
		values = append(values, map[string]any{"id": assignment.ID, "resourceId": assignment.ResourceID, "appRoleId": assignment.AppRoleID})
	}
	httpx.WriteJSON(w, 200, map[string]any{"value": values})
}
