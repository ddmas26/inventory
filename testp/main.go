package main

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// Secret key for signing JWTs (store securely in production, e.g., env variable)
var jwtSecret = []byte("23894743098790er8fhasdkf;ljasldkjf")

// Mock user database
var users = map[string]struct {
	Password string
	Role     string
}{
	"alice": {"password123", "admin"},
	"bob":   {"password456", "user"},
}

// Claims struct for JWT payload
type Claims struct {
	UserID string `json:"sub"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

// Login handler to generate JWT
func loginHandler(c *gin.Context) {
	type LoginRequest struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}
	// Validate credentials
	user, exists := users[req.Username]
	if !exists || user.Password != req.Password {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}
	// Create JWT
	claims := &Claims{
		UserID: req.Username,
		Role:   user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(jwtSecret)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not generate token"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"token": tokenString})
}

// Middleware to authenticate JWT
func authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header missing or invalid"})
			c.Abort()
			return
		}
		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		claims := &Claims{}
		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return jwtSecret, nil
		})
		if err != nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			c.Abort()
			return
		}
		// Attach claims to context for downstream use
		c.Set("user", claims)
		c.Next()
	}
}

// Middleware to enforce admin role
func adminOnly() gin.HandlerFunc {
	return func(c *gin.Context) {
		user, exists := c.Get("user")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
			c.Abort()
			return
		}
		claims, ok := user.(*Claims)
		if !ok || claims.Role != "admin" {
			c.JSON(http.StatusForbidden, gin.H{"error": "Admin access required"})
			c.Abort()
			return
		}
		c.Next()
	}
}
func main() {
	r := gin.Default()
	// Public route: Login to get JWT
	r.POST("/login", loginHandler)
	// Protected routes
	protected := r.Group("/api")
	protected.Use(authMiddleware())
	{
		protected.GET("/user", func(c *gin.Context) {
			user := c.MustGet("user").(*Claims)
			c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("Welcome, %s!", user.UserID)})
		})
		// Admin-only route
		protected.GET("/admin", adminOnly(), func(c *gin.Context) {
			user := c.MustGet("user").(*Claims)
			c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("Admin access granted, %s!", user.UserID)})
		})
	}
	r.Run(":9080")
}
