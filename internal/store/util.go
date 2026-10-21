package store

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
)

// NewGUID returns a random RFC 4122 version-4 UUID in lowercase form.
func NewGUID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic(err)
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

// NewOpaqueToken returns URL-safe random token material of n random bytes.
func NewOpaqueToken(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return base64.RawURLEncoding.EncodeToString(b)
}

// ServicePrincipalID is the stable tenant-local object identity of an application.
// It deliberately differs from the app/client ID and survives directory resets.
func ServicePrincipalID(tenantID, appID string) string {
	digest := sha256.Sum256([]byte("entra-emulator:service-principal:" + tenantID + ":" + appID))
	digest[6] = (digest[6] & 0x0f) | 0x80 // RFC 9562 custom UUID (version 8)
	digest[8] = (digest[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", digest[:4], digest[4:6], digest[6:8], digest[8:10], digest[10:16])
}
