package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type Claims struct {
	UserID       string   `json:"uid"`
	Username     string   `json:"uname"`
	Permissions  []string `json:"perms"`
	WarehouseIDs []string `json:"whs"`
	Clients      []string `json:"clients"`
	jwt.RegisteredClaims
}

type TokenPair struct {
	AccessToken  string
	RefreshToken string
	ExpiresIn    int64
}

type Manager struct {
	secret     []byte
	accessTTL  time.Duration
	refreshTTL time.Duration
}

func NewManager(secret string, accessTTL, refreshTTL time.Duration) *Manager {
	return &Manager{secret: []byte(secret), accessTTL: accessTTL, refreshTTL: refreshTTL}
}

func (m *Manager) IssueAccess(userID, username string, perms, whs, clients []string) (string, int64, error) {
	now := time.Now()
	exp := now.Add(m.accessTTL)
	claims := Claims{
		UserID:       userID,
		Username:     username,
		Permissions:  perms,
		WarehouseIDs: whs,
		Clients:      clients,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        uuid.NewString(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(exp),
			Issuer:    "watcher",
		},
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	s, err := t.SignedString(m.secret)
	return s, int64(m.accessTTL.Seconds()), err
}

func (m *Manager) ParseAccess(token string) (*Claims, error) {
	parsed, err := jwt.ParseWithClaims(token, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		if t.Method != jwt.SigningMethodHS256 {
			return nil, errors.New("unexpected signing method")
		}
		return m.secret, nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := parsed.Claims.(*Claims)
	if !ok || !parsed.Valid {
		return nil, errors.New("invalid token")
	}
	return claims, nil
}

func (m *Manager) RefreshTTL() time.Duration { return m.refreshTTL }
