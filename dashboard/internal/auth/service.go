package auth

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/gofurry/easyhash"
	"github.com/gofurry/l4d2-plugin-stats/dashboard/internal/store"
	"github.com/golang-jwt/jwt/v5"
)

const (
	BcryptCost       = 12
	AdminTokenTTL    = 8 * time.Hour
	SteamIdentityTTL = 30 * 24 * time.Hour
	SetupTokenTTL    = 30 * time.Minute
)

var (
	ErrNotConfigured = errors.New("administrator is not configured")
	ErrInvalidLogin  = errors.New("invalid username or password")
	ErrInvalidToken  = errors.New("invalid token")
	ErrSetupToken    = errors.New("invalid or expired setup token")
)

type Claims struct {
	TokenType string `json:"typ"`
	Version   int64  `json:"ver,omitempty"`
	SteamID   string `json:"steam_id,omitempty"`
	jwt.RegisteredClaims
}

type Service struct {
	store        store.DashboardStore
	mu           sync.RWMutex
	setupToken   string
	setupExpires time.Time
	setupUsed    bool
}

func New(ctx context.Context, dashboard store.DashboardStore) (*Service, string, error) {
	configured, err := dashboard.AdminConfigured(ctx)
	if err != nil {
		return nil, "", err
	}
	s := &Service{store: dashboard}
	if configured {
		return s, "", nil
	}
	token, err := randomToken(24)
	if err != nil {
		return nil, "", err
	}
	s.setupToken = token
	s.setupExpires = time.Now().Add(SetupTokenTTL)
	return s, token, nil
}

func (s *Service) SetupStatus(ctx context.Context) (bool, time.Time, error) {
	configured, err := s.store.AdminConfigured(ctx)
	if err != nil {
		return false, time.Time{}, err
	}
	s.mu.RLock()
	expires := s.setupExpires
	s.mu.RUnlock()
	return !configured, expires, nil
}

func (s *Service) Setup(ctx context.Context, setupToken, username, password string) error {
	s.mu.Lock()
	if s.setupUsed || time.Now().After(s.setupExpires) || subtle.ConstantTimeCompare([]byte(setupToken), []byte(s.setupToken)) != 1 {
		s.mu.Unlock()
		return ErrSetupToken
	}
	s.setupUsed = true
	s.mu.Unlock()
	hash, err := easyhash.CreateBcrypt(BcryptCost, password)
	if err != nil {
		s.restoreSetup()
		return fmt.Errorf("hash administrator password: %w", err)
	}
	secret, err := randomToken(32)
	if err != nil {
		s.restoreSetup()
		return err
	}
	if err := s.store.CreateAdmin(ctx, username, hash, secret); err != nil {
		s.restoreSetup()
		return err
	}
	s.mu.Lock()
	s.setupToken = ""
	s.setupExpires = time.Time{}
	s.mu.Unlock()
	return nil
}

func (s *Service) restoreSetup() { s.mu.Lock(); s.setupUsed = false; s.mu.Unlock() }

func (s *Service) Login(ctx context.Context, username, password string) (string, error) {
	admin, err := s.store.Admin(ctx)
	if err != nil {
		return "", err
	}
	if admin == nil {
		return "", ErrNotConfigured
	}
	if subtle.ConstantTimeCompare([]byte(username), []byte(admin.Username)) != 1 || !easyhash.VerifyBcrypt(password, admin.PasswordHash) {
		return "", ErrInvalidLogin
	}
	return sign(admin.JWTSecret, Claims{TokenType: "admin", Version: admin.TokenVersion, RegisteredClaims: jwt.RegisteredClaims{Subject: "admin"}}, AdminTokenTTL)
}

func (s *Service) ValidateAdmin(ctx context.Context, raw string) (*store.AdminAccount, error) {
	admin, err := s.store.Admin(ctx)
	if err != nil {
		return nil, err
	}
	if admin == nil {
		return nil, ErrNotConfigured
	}
	claims, err := parse(raw, admin.JWTSecret)
	if err != nil || claims.TokenType != "admin" || claims.Subject != "admin" || claims.Version != admin.TokenVersion {
		return nil, ErrInvalidToken
	}
	return admin, nil
}

func (s *Service) SignSteamIdentity(ctx context.Context, steamID string) (string, error) {
	admin, err := s.store.Admin(ctx)
	if err != nil {
		return "", err
	}
	if admin == nil {
		return "", ErrNotConfigured
	}
	return sign(admin.JWTSecret, Claims{TokenType: "steam_identity", SteamID: steamID, RegisteredClaims: jwt.RegisteredClaims{Subject: steamID}}, SteamIdentityTTL)
}

func (s *Service) ValidateSteamIdentity(ctx context.Context, raw string) (string, error) {
	admin, err := s.store.Admin(ctx)
	if err != nil {
		return "", err
	}
	if admin == nil {
		return "", ErrNotConfigured
	}
	claims, err := parse(raw, admin.JWTSecret)
	if err != nil || claims.TokenType != "steam_identity" || claims.SteamID == "" || claims.Subject != claims.SteamID {
		return "", ErrInvalidToken
	}
	return claims.SteamID, nil
}

func (s *Service) ChangeUsername(ctx context.Context, username string) (string, error) {
	if err := s.store.UpdateAdminUsername(ctx, username); err != nil {
		return "", err
	}
	admin, err := s.store.Admin(ctx)
	if err != nil {
		return "", err
	}
	return sign(admin.JWTSecret, Claims{TokenType: "admin", Version: admin.TokenVersion, RegisteredClaims: jwt.RegisteredClaims{Subject: "admin"}}, AdminTokenTTL)
}

func (s *Service) ChangePassword(ctx context.Context, current, next string) (string, error) {
	admin, err := s.store.Admin(ctx)
	if err != nil {
		return "", err
	}
	if admin == nil || !easyhash.VerifyBcrypt(current, admin.PasswordHash) {
		return "", ErrInvalidLogin
	}
	hash, err := easyhash.CreateBcrypt(BcryptCost, next)
	if err != nil {
		return "", err
	}
	if err := s.store.UpdateAdminPassword(ctx, hash); err != nil {
		return "", err
	}
	admin, err = s.store.Admin(ctx)
	if err != nil {
		return "", err
	}
	return sign(admin.JWTSecret, Claims{TokenType: "admin", Version: admin.TokenVersion, RegisteredClaims: jwt.RegisteredClaims{Subject: "admin"}}, AdminTokenTTL)
}

func sign(secret string, claims Claims, ttl time.Duration) (string, error) {
	now := time.Now().UTC()
	claims.IssuedAt = jwt.NewNumericDate(now)
	claims.ExpiresAt = jwt.NewNumericDate(now.Add(ttl))
	claims.NotBefore = jwt.NewNumericDate(now.Add(-time.Second))
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
}

func parse(raw, secret string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(raw, claims, func(token *jwt.Token) (any, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, ErrInvalidToken
		}
		return []byte(secret), nil
	}, jwt.WithExpirationRequired(), jwt.WithValidMethods([]string{"HS256"}))
	if err != nil || !token.Valid {
		return nil, ErrInvalidToken
	}
	return claims, nil
}

func RandomToken(bytes int) (string, error) { return randomToken(bytes) }
func randomToken(size int) (string, error) {
	b := make([]byte, size)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
