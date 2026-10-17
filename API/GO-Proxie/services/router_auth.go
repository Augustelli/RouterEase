package services

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

var ErrRouterUnauthorized = errors.New("router API key is invalid, expired, revoked, or lacks scope")

func IssueRouterKey(db *gorm.DB, routerID string, scopes []string, expiresAt *time.Time) (RouterAPIKey, string, error) {
	if len(scopes) == 0 {
		scopes = []string{"dns:query", "inventory:write"}
	}
	// 192 bits keeps the complete key at bcrypt's 72-byte input limit while
	// leaving ample entropy for a router credential.
	secret := make([]byte, 24)
	if _, err := rand.Read(secret); err != nil {
		return RouterAPIKey{}, "", err
	}
	keyID := newID()
	raw := "re_" + keyID + "_" + base64.RawURLEncoding.EncodeToString(secret)
	hash, err := bcrypt.GenerateFromPassword([]byte(raw), bcrypt.DefaultCost)
	if err != nil {
		return RouterAPIKey{}, "", err
	}
	key := RouterAPIKey{ID: keyID, RouterID: routerID, KeyHash: string(hash), Scopes: pq.StringArray(scopes), ExpiresAt: expiresAt}
	if err := db.Create(&key).Error; err != nil {
		return RouterAPIKey{}, "", err
	}
	return key, raw, nil
}

func AuthenticateRouterKey(db *gorm.DB, raw, scope string) (RouterAPIKey, error) {
	parts := strings.SplitN(raw, "_", 3)
	if len(parts) != 3 || parts[0] != "re" || len(raw) > 512 {
		return RouterAPIKey{}, ErrRouterUnauthorized
	}
	var key RouterAPIKey
	if err := db.Where("id = ? AND revoked_at IS NULL AND (expires_at IS NULL OR expires_at > now())", parts[1]).First(&key).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return RouterAPIKey{}, ErrRouterUnauthorized
		}
		return RouterAPIKey{}, err
	}
	if bcrypt.CompareHashAndPassword([]byte(key.KeyHash), []byte(raw)) != nil || !hasScope(key.Scopes, scope) {
		return RouterAPIKey{}, ErrRouterUnauthorized
	}
	db.Model(&RouterAPIKey{}).Where("id = ?", key.ID).Update("last_used_at", time.Now().UTC())
	return key, nil
}

func hasScope(scopes pq.StringArray, wanted string) bool {
	for _, scope := range scopes {
		if scope == wanted {
			return true
		}
	}
	return false
}

func RevokeRouterKey(db *gorm.DB, id string) error {
	result := db.Model(&RouterAPIKey{}).Where("id = ? AND revoked_at IS NULL", id).Update("revoked_at", time.Now().UTC())
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("router API key not found or already revoked")
	}
	return nil
}
