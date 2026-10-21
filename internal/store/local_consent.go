package store

// SetTenantConsent atomically replaces tenant-wide consent for a client/resource.
// Keeping an empty grant is intentional: it must not fall back to auto-consent.
func (s *Store) SetTenantConsent(clientID, resourceID, scope string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`DELETE FROM oauth2_permission_grants WHERE client_id=? AND resource_id=? AND consent_type='AllPrincipals'`, clientID, resourceID); err != nil {
		return err
	}
	if _, err := tx.Exec(`INSERT INTO oauth2_permission_grants (id,client_id,consent_type,resource_id,scope,created_at) VALUES (?,?,'AllPrincipals',?,?,?)`, NewGUID(), clientID, resourceID, scope, s.Now()); err != nil {
		return mapConstraint(err)
	}
	return tx.Commit()
}

// SetApplicationRole is idempotent even when the assignment originated in Graph.
func (s *Store) SetApplicationRole(clientID, resourceID, roleID string, assigned bool) error {
	if !assigned {
		_, err := s.db.Exec(`DELETE FROM app_role_assignments WHERE principal_id=? AND resource_id=? AND app_role_id=? AND principal_type='ServicePrincipal'`, clientID, resourceID, roleID)
		return err
	}
	_, err := s.db.Exec(`INSERT INTO app_role_assignments (id,principal_id,principal_type,resource_id,app_role_id,created_at)
 SELECT ?,?,'ServicePrincipal',?,?,? WHERE NOT EXISTS (SELECT 1 FROM app_role_assignments WHERE principal_id=? AND resource_id=? AND app_role_id=? AND principal_type='ServicePrincipal')`,
		NewGUID(), clientID, resourceID, roleID, s.Now(), clientID, resourceID, roleID)
	return mapConstraint(err)
}
