package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

var (
	// ErrInvalidToken covers any structural or signature problem with a token.
	ErrInvalidToken = errors.New("auth: invalid token")
	// ErrExpiredToken is returned when the token's exp claim is in the past.
	ErrExpiredToken = errors.New("auth: token expired")
)

type jwtHeader struct {
	Alg string `json:"alg"`
	Typ string `json:"typ"`
}

// Claims are the JWT payload fields used by this service.
type Claims struct {
	UserID string `json:"sub"`
	Email  string `json:"email"`
	Iat    int64  `json:"iat"`
	Exp    int64  `json:"exp"`
}

// GenerateToken creates a signed HS256 JWT for the given user, valid for ttl.
func GenerateToken(userID, email, secret string, ttl time.Duration) (string, error) {
	if secret == "" {
		return "", errors.New("auth: JWT secret must not be empty")
	}

	now := time.Now()
	header := jwtHeader{Alg: "HS256", Typ: "JWT"}
	claims := Claims{
		UserID: userID,
		Email:  email,
		Iat:    now.Unix(),
		Exp:    now.Add(ttl).Unix(),
	}

	headerJSON, err := json.Marshal(header)
	if err != nil {
		return "", err
	}
	claimsJSON, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}

	headerSeg := base64URLEncode(headerJSON)
	claimsSeg := base64URLEncode(claimsJSON)
	signingInput := headerSeg + "." + claimsSeg

	sig := sign(signingInput, secret)
	sigSeg := base64URLEncode(sig)

	return signingInput + "." + sigSeg, nil
}

// ParseToken validates the signature and expiry of a JWT and returns its claims.
func ParseToken(token, secret string) (*Claims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, ErrInvalidToken
	}

	headerSeg, claimsSeg, sigSeg := parts[0], parts[1], parts[2]

	headerJSON, err := base64URLDecode(headerSeg)
	if err != nil {
		return nil, ErrInvalidToken
	}
	var header jwtHeader
	if err := json.Unmarshal(headerJSON, &header); err != nil {
		return nil, ErrInvalidToken
	}
	if header.Alg != "HS256" {
		return nil, fmt.Errorf("%w: unsupported algorithm %q", ErrInvalidToken, header.Alg)
	}

	expectedSig := sign(headerSeg+"."+claimsSeg, secret)
	actualSig, err := base64URLDecode(sigSeg)
	if err != nil {
		return nil, ErrInvalidToken
	}
	if subtle.ConstantTimeCompare(expectedSig, actualSig) != 1 {
		return nil, ErrInvalidToken
	}

	claimsJSON, err := base64URLDecode(claimsSeg)
	if err != nil {
		return nil, ErrInvalidToken
	}
	var claims Claims
	if err := json.Unmarshal(claimsJSON, &claims); err != nil {
		return nil, ErrInvalidToken
	}

	if time.Now().Unix() > claims.Exp {
		return nil, ErrExpiredToken
	}

	return &claims, nil
}

func sign(input, secret string) []byte {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(input))
	return mac.Sum(nil)
}

func base64URLEncode(b []byte) string {
	return base64.RawURLEncoding.EncodeToString(b)
}

func base64URLDecode(s string) ([]byte, error) {
	return base64.RawURLEncoding.DecodeString(s)
}
