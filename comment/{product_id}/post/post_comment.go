package main

import (
	"bytes"
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

// Comment 定义评论结构体
type Comment struct {
	ProductID string `gorm:"primaryKey;not null"`
	Content   string `gorm:"not null"`
}

// CommentRequest 定义请求体结构体
type CommentRequest struct {
	ProductID string `json:"product_id"`
	Content   string `json:"content"`
}

var DB *gorm.DB

// 初始化数据库连接
func InitDB() error {
	dsn := "root:123456@tcp(127.0.0.1:3306)/MySQL?charset=utf8mb4&parseTime=True&loc=Local"
	var err error
	DB, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Printf("Failed to connect database: %v", err)
		return fmt.Errorf("failed to connect database: %w", err)
	}
	// 自动迁移表结构
	err = DB.AutoMigrate(&Comment{})
	if err != nil {
		log.Printf("Failed to auto - migrate database: %v", err)
		return fmt.Errorf("failed to auto - migrate database: %w", err)
	}
	return nil
}

// 定义验证 JWT Token 的密钥
var jwtKey = []byte("your_secret_key")

// 验证 JWT Token 并解析用户名
func validateAndParseUsername(tokenString string) (string, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return jwtKey, nil
	})
	if err != nil {
		return "", err
	}
	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		if expiration, ok := claims["exp"].(float64); ok {
			if int64(expiration) < time.Now().Unix() {
				return "", fmt.Errorf("token has expired")
			}
		}
		username, ok := claims["sub"].(string)
		if !ok {
			return "", fmt.Errorf("username claim not found in token")
		}
		return username, nil
	}
	return "", fmt.Errorf("invalid token")
}

// JWTAuthorization 中间件用于验证 JWT Token 并解析用户名
func JWTAuthorization() app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		authHeader := c.GetHeader("Authorization")
		if len(authHeader) < 7 || !bytes.Equal(authHeader[:7], []byte("Bearer ")) {
			log.Println("Invalid token format")
			c.JSON(consts.StatusUnauthorized, utils.H{
				"info":   "Invalid token format",
				"status": 10005,
			})
			return
		}
		tokenString := authHeader[7:]
		username, err := validateAndParseUsername(string(tokenString))
		if err != nil {
			log.Printf("Token validation failed: %v", err)
			c.JSON(consts.StatusUnauthorized, utils.H{
				"info":   "Unauthorized",
				"status": 10005,
			})
			return
		}
		c.Set("username", username)
		c.Next(ctx)
	}
}

// 从请求体获取并验证参数
func getAndValidateRequestBody(c *app.RequestContext) (string, string, error) {
	var req CommentRequest
	if err := c.BindJSON(&req); err != nil {
		log.Printf("Error binding JSON: %v", err)
		return "", "", fmt.Errorf("Invalid request body format")
	}

	if req.ProductID == "" {
		log.Println("product_id in request body is required")
		return "", "", fmt.Errorf("product_id in request body is required")
	}
	if req.Content == "" {
		log.Println("content in request body is required")
		return "", "", fmt.Errorf("content in request body is required")
	}

	return req.ProductID, req.Content, nil
}

// 创建评论
func createComment(productID string, content string) (Comment, error) {
	comment := Comment{
		ProductID: productID,
		Content:   content,
	}
	result := DB.Create(&comment)
	if result.Error != nil {
		log.Printf("Failed to create comment: %v", result.Error)
		return Comment{}, result.Error
	}
	return comment, nil
}

func main() {
	if err := InitDB(); err != nil {
		log.Printf("Database initialization failed: %v\n", err)
		return
	}
	h := server.New(server.WithHostPorts("127.0.0.1:8012"))

	h.POST("/comment/{product_id}", JWTAuthorization(), func(ctx context.Context, c *app.RequestContext) {
		productID, content, err := getAndValidateRequestBody(c)
		if err != nil {
			c.JSON(consts.StatusBadRequest, utils.H{
				"info":   err.Error(),
				"status": 10001,
			})
			return
		}

		comment, err := createComment(productID, content)
		if err != nil {
			c.JSON(consts.StatusInternalServerError, utils.H{
				"info":   "Failed to create comment",
				"status": 10002,
			})
			return
		}

		c.JSON(consts.StatusOK, utils.H{
			"info":   "success",
			"status": 10000,
			"data":   comment,
		})
	})

	if err := h.Run(); err != nil {
		log.Fatal(err)
	}
}
