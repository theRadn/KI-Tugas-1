package middleware

import (
	"net/http"
	"os"
	"strings"

	"github.com/theRadn/KI-Tugas-1/utils"

	"github.com/gin-gonic/gin"
)

// Detail kegagalan token tidak dikirim ke klien supaya tidak membantu
// orang yang mencoba memalsukan token.
const pesanTokenTidakValid = "Sesi login tidak valid atau kedaluwarsa, silakan masuk kembali"

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Silakan masuk terlebih dahulu"})
			c.Abort()
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": pesanTokenTidakValid})
			c.Abort()
			return
		}

		tokenString := parts[1]
		secret := os.Getenv("JWT_SECRET")

		claims, err := utils.VerifyToken(tokenString, secret)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": pesanTokenTidakValid})
			c.Abort()
			return
		}

		tokenUse, ok := claims["token_use"].(string)
		if !ok || tokenUse != "access" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": pesanTokenTidakValid,
			})
			c.Abort()
			return
		}

		subRaw, exists := claims["sub"]
		if !exists || subRaw == nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": pesanTokenTidakValid})
			return
		}

		subStr, ok := subRaw.(string)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": pesanTokenTidakValid})
			return
		}

		c.Set("userID", subStr)
		c.Set("userRole", claims["role"])
		c.Next()
	}
}
