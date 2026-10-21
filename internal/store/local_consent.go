package store

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
