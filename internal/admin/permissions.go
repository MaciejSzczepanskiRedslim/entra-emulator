package admin

import (
	"net/http"
	"regexp"

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
