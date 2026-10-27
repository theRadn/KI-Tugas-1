package utils

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

func GenerateToken(userID uuid.UUID, role string) (string, error) {
	return generateJWT(userID, role, "access", os.Getenv("JWT_SECRET"), 15*time.Minute)
}

func GenerateRefreshToken(userID uuid.UUID, role string) (string, error) {
	return generateJWT(userID, role, "refresh", os.Getenv("JWT_REFRESH_SECRET"), 7*24*time.Hour)
}

func generateJWT(
	userID uuid.UUID,
	role string,
	tokenUse string,
	secret string,
	duration time.Duration,
) (string, error) {
	if secret == "" {
		return "", errors.New("JWT signing secret is not configured")
	}

	now := time.Now()

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":       userID.String(),
		"role":      role,
		"token_use": tokenUse,
		"jti":       uuid.NewString(),
		"iat":       jwt.NewNumericDate(now),
		"exp":       jwt.NewNumericDate(now.Add(duration)),
	})

	return token.SignedString([]byte(secret))
}

func VerifyToken(tokenStr string, secret string) (jwt.MapClaims, error) {
	if secret == "" {
		return nil, errors.New("JWT signing secret is not configured")
	}

	token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("unexpected signing algorithm")
		}
		return []byte(secret), nil
	})
	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, errors.New("invalid token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, errors.New("invalid token claims")
	}

	return claims, nil
}
