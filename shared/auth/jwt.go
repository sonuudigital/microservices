package auth

import (
	"crypto/ecdsa"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type JWTManager struct {
	privateKey *ecdsa.PrivateKey
	publicKey  *ecdsa.PublicKey
	issuer     string
	audience   string
	ttl        time.Duration
	leeway     time.Duration
	now        func() time.Time
}

type Claims struct {
	Email string `json:"email"`
	jwt.RegisteredClaims
}

func NewJWTManager(privateKeyPEM, publicKeyPEM []byte, issuer, audience string, ttl time.Duration) (*JWTManager, error) {
	privateKey, err := jwt.ParseECPrivateKeyFromPEM(privateKeyPEM)
	if err != nil {
		return nil, fmt.Errorf("could not parse private key: %w", err)
	}

	publicKey, err := jwt.ParseECPublicKeyFromPEM(publicKeyPEM)
	if err != nil {
		return nil, fmt.Errorf("could not parse public key: %w", err)
	}

	return &JWTManager{
		privateKey: privateKey,
		publicKey:  publicKey,
		issuer:     issuer,
		audience:   audience,
		ttl:        ttl,
		leeway:     30 * time.Second,
		now:        time.Now,
	}, nil
}

func (j *JWTManager) WithLeeway(d time.Duration) *JWTManager {
	if d >= 0 {
		j.leeway = d
	}
	return j
}

func (j *JWTManager) WithNowFunc(f func() time.Time) *JWTManager {
	if f != nil {
		j.now = f
	}
	return j
}

func (j *JWTManager) GenerateToken(userID, email string) (string, error) {
	now := j.now()
	claims := Claims{
		Email: email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(j.ttl)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    j.issuer,
			Audience:  []string{j.audience},
			Subject:   userID,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodES256, claims)
	signedToken, err := token.SignedString(j.privateKey)
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return signedToken, nil
}

func (j *JWTManager) ValidateToken(tokenString string) (*Claims, error) {
	tokenString = sanitizeBearer(tokenString)

	parser := jwt.NewParser(
		jwt.WithValidMethods([]string{jwt.SigningMethodES256.Alg()}),
		jwt.WithIssuer(j.issuer),
		jwt.WithAudience(j.audience),
		jwt.WithLeeway(j.leeway),
		jwt.WithTimeFunc(j.now),
	)

	token, err := parser.ParseWithClaims(tokenString, &Claims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodECDSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return j.publicKey, nil
	})
	if err != nil {
		return nil, classifyJWTError(err)
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, jwt.ErrTokenInvalidClaims
	}

	return claims, nil
}

func sanitizeBearer(s string) string {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(strings.ToLower(s), "bearer ") {
		return strings.TrimSpace(s[7:])
	}
	return s
}

func classifyJWTError(err error) error {
	switch {
	case errors.Is(err, jwt.ErrTokenExpired):
		return jwt.ErrTokenExpired
	case errors.Is(err, jwt.ErrTokenNotValidYet):
		return jwt.ErrTokenNotValidYet
	case errors.Is(err, jwt.ErrTokenInvalidIssuer):
		return jwt.ErrTokenInvalidIssuer
	case errors.Is(err, jwt.ErrTokenInvalidAudience):
		return jwt.ErrTokenInvalidAudience
	default:
		return err
	}
}
