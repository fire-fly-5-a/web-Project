package main

import (
	"context"
	"fmt"
	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/common/utils"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
	"github.com/dgrijalva/jwt-go"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"log"
	"time"
)

// User 定义用户结构体
type User struct {
	ID       int    `gorm:"primaryKey"`
	Username string `gorm:"unique;not null"`
	Password string `gorm:"not null"`
}

// TokenResponse 定义返回的 token 响应结构体
type TokenResponse struct {
	Status int       `json:"status"`
	Info   string    `json:"info"`
	Data   TokenData `json:"data"`
}

// TokenData 定义 token 数据结构体
type TokenData struct {
	RefreshToken string `json:"refresh_token"`
	Token        string `json:"token"`
}

var DB *gorm.DB
var jwtKey = []byte("your_secret_key")

// InitDB 初始化数据库连接
func InitDB() error {
	dsn := "root:123456@tcp(127.0.0.1:3306)/MySQL?charset=utf8mb4&parseTime=True&loc=Local"
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		return fmt.Errorf("failed to connect database: %w", err)
	}
	DB = db
	return nil
}

// generateToken 生成 JWT Token
func generateToken(username string) (string, error) {
	expirationTime := time.Now().Add(2 * time.Hour)
	claims := &jwt.StandardClaims{
		ExpiresAt: expirationTime.Unix(),
		Subject:   username,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(jwtKey)
	if err != nil {
		return "", err
	}
	return tokenString, nil
}

// generateRefreshToken 生成刷新 Token
func generateRefreshToken(username string) (string, error) {
	expirationTime := time.Now().Add(2 * time.Hour)
	claims := &jwt.StandardClaims{
		ExpiresAt: expirationTime.Unix(),
		Subject:   username,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(jwtKey)
	if err != nil {
		return "", err
	}
	return tokenString, nil
}

// validateRefreshToken 验证刷新 Token
func validateRefreshToken(refreshToken string) (string, error) {
	claims := &jwt.StandardClaims{}

	tkn, err := jwt.ParseWithClaims(refreshToken, claims, func(token *jwt.Token) (interface{}, error) {
		return jwtKey, nil
	})

	if err != nil {
		return "", err
	}

	if !tkn.Valid {
		return "", fmt.Errorf("invalid refresh token")
	}

	return claims.Subject, nil
}

func main() {
	err := InitDB()
	if err != nil {
		log.Fatalf("failed to initialize database: %v", err)
	}

	h := server.New(server.WithHostPorts("127.0.0.1:8002"))

	// 注册获取 token 的路由（用户登录）
	h.GET("/user/token", func(ctx context.Context, c *app.RequestContext) {
		var req struct {
			Username string `json:"username"`
			Password string `json:"password"`
		}

		if err := c.BindAndValidate(&req); err != nil {
			c.JSON(consts.StatusBadRequest, utils.H{"code": 400, "message": "Invalid request"})
			return
		}

		// 验证用户名和密码
		var user User
		result := DB.Where("username = ? AND password = ?", req.Username, req.Password).First(&user)
		if result.Error != nil {
			if result.Error == gorm.ErrRecordNotFound {
				c.JSON(consts.StatusUnauthorized, utils.H{"code": 401, "message": "Invalid username or password"})
			} else {
				c.JSON(consts.StatusInternalServerError, utils.H{"code": 500, "message": "Database error"})
			}
			return
		}

		// 生成 Token 和刷新 Token
		token, err := generateToken(req.Username)
		if err != nil {
			c.JSON(consts.StatusInternalServerError, utils.H{"code": 500, "message": "Failed to generate token"})
			return
		}
		refreshToken, err := generateRefreshToken(req.Username)
		if err != nil {
			c.JSON(consts.StatusInternalServerError, utils.H{"code": 500, "message": "Failed to generate refresh token"})
			return
		}

		// 返回 token
		response := TokenResponse{
			Status: 10000,
			Info:   "success",
			Data: TokenData{
				RefreshToken: refreshToken,
				Token:        token,
			},
		}

		c.JSON(consts.StatusOK, response)
	})

	// 注册刷新 token 的路由
	h.GET("/user/token/refresh", func(ctx context.Context, c *app.RequestContext) {
		refreshToken := c.Query("refresh_token")

		if refreshToken == "" {
			c.JSON(
				consts.StatusBadRequest,
				utils.H{"code": 400, "message": "refresh_token is required"})
			return
		}

		// 验证刷新 Token
		username, err := validateRefreshToken(refreshToken)
		if err != nil {
			c.JSON(consts.StatusUnauthorized, utils.H{"code": 401, "message": "Invalid refresh token"})
			return
		}

		// 生成新的 Token 和刷新 Token
		newToken, err := generateToken(username)
		if err != nil {
			c.JSON(consts.StatusInternalServerError, utils.H{"code": 500, "message": "Failed to generate new token"})
			return
		}
		newRefreshToken, err := generateRefreshToken(username)
		if err != nil {
			c.JSON(consts.StatusInternalServerError, utils.H{"code": 500, "message": "Failed to generate new refresh token"})
			return
		}

		// 返回新的 token
		response := TokenResponse{
			Status: 10000,
			Info:   "success",
			Data: TokenData{
				RefreshToken: newRefreshToken,
				Token:        newToken,
			},
		}
		c.JSON(consts.StatusOK, response)
	})

	h.Run()
}
