package auth

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const defaultExpiry = 24 * time.Hour
const tokenIssuer = "featureflag-api"

// jwtClaims is the JWT standard claims plus our custom fields.
type jwtClaims struct {
	jwt.RegisteredClaims
	Role string `json:"role"`
}

// IssueToken signs a new JWT with sub=userID, role=role and exp.
// Uses HMAC-SHA256 with the given secret.
func IssueToken(userID, role string, secret []byte, expiry time.Duration) (string, error) {
	if expiry <= 0 {
		expiry = defaultExpiry
	}
	now := time.Now()
	claims := jwtClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			ExpiresAt: jwt.NewNumericDate(now.Add(expiry)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    tokenIssuer,
		},
		Role: role,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(secret)
}

// ParseAndValidate parses the token string and validates signature and exp.
// Returns our Claims or an error.
func ParseAndValidate(tokenString string, secret []byte) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &jwtClaims{}, func(token *jwt.Token) (interface{}, error) {
		return secret, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}), jwt.WithIssuer(tokenIssuer))
	if err != nil {
		return nil, err
	}
	jc, ok := token.Claims.(*jwtClaims)
	if !ok || !token.Valid {
		return nil, jwt.ErrTokenInvalidClaims
	}
	if jc.Subject == "" || jc.Role == "" {
		return nil, jwt.ErrTokenInvalidClaims
	}
	return &Claims{Sub: jc.Subject, Role: jc.Role}, nil
}
