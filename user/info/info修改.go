package main

import (
	"bytes"
	"context"
	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/common/utils"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
	"github.com/dgrijalva/jwt-go"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"log"
)

// User 定义用户信息结构体，只包含 ID 和 Email 字段
type User struct {
	ID       uint   `gorm:"primaryKey"`
	Username string `json:"username" gorm:"unique"`
	Email    string `json:"email" gorm:"unique"`
}

// UpdateUserResponse 定义更新用户信息的响应结构体
type UpdateUserResponse struct {
	Info   string `json:"info"`
	Status int    `json:"status"`
}

// 假设的 JWT 密钥，实际应用中应妥善保管
var jwtKey = []byte("your_secret_key")

// 验证 JWT Token
func validateToken(tokenString string) bool {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return jwtKey, nil
	})
	return err == nil && token.Valid
}

// JWTAuthorization 中间件用于验证 JWT Token
func JWTAuthorization() app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		// 从请求头中获取 Authorization
		authHeader := c.GetHeader("Authorization")
		if len(authHeader) < 7 || !bytes.Equal(authHeader[:7], []byte("Bearer ")) {
			c.JSON(consts.StatusUnauthorized, utils.H{
				"info":   "Invalid token format",
				"status": 10005,
			})
			return
		}
		tokenString := string(authHeader[7:])
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			return jwtKey, nil
		})
		if err != nil || !token.Valid {
			c.JSON(consts.StatusUnauthorized, utils.H{
				"info":   "Unauthorized",
				"status": 10005,
			})
			return
		}
		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			c.JSON(consts.StatusUnauthorized, utils.H{
				"info":   "Invalid JWT claims",
				"status": 10005,
			})
			return
		}
		username, ok := claims["sub"].(string)
		if !ok {
			c.JSON(consts.StatusUnauthorized, utils.H{
				"info":   "Username not found in JWT claims",
				"status": 10005,
			})
			return
		}
		// 将用户名存储在RequestContext的扩展字段中
		c.Set("username", username)
		c.Next(ctx)
	}
}

func main() {
	h := server.New(server.WithHostPorts("127.0.0.1:8004"))

	// 数据库连接，这里的 dsn 需要根据实际情况修改
	dsn := "root:123456@tcp(127.0.0.1:3306)/MySQL?charset=utf8mb4&parseTime=True&loc=Local"
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}
	// 自动迁移模式
	db.AutoMigrate(&User{})

	// 注册 PUT /user/info 路由，并使用 JWTAuthorization 中间件
	h.PUT("/user/info", JWTAuthorization(), func(ctx context.Context, c *app.RequestContext) {
		var updateUser User
		// 解析请求体中的 JSON 数据
		if err := c.Bind(&updateUser); err != nil {
			c.JSON(consts.StatusBadRequest, utils.H{
				"info":   "Invalid request body",
				"status": 10001,
			})
			return
		}
		// 从扩展字段中获取用户名
		username, ok := c.Get("username")
		if !ok {
			c.JSON(consts.StatusBadRequest, utils.H{
				"info":   "Username not found in context",
				"status": 10006,
			})
			return
		}
		var user User
		result := db.Where("username =?", username).First(&user)
		if result.Error != nil {
			c.JSON(consts.StatusNotFound, utils.H{
				"info":   "User not found",
				"status": 10002,
			})
			return
		}

		// 开始事务
		tx := db.Begin()
		if tx.Error != nil {
			c.JSON(consts.StatusInternalServerError, utils.H{
				"info":   "Database transaction start error",
				"status": 10003,
			})
			return
		}
		// 只更新 Email 字段
		if updateUser.Email != "" {
			user.Email = updateUser.Email
		}
		result = tx.Save(&user)
		if result.Error != nil {
			tx.Rollback()
			log.Printf("Failed to update user information: %v", result.Error)
			c.JSON(consts.StatusInternalServerError, utils.H{
				"info":   "Failed to update user information",
				"status": 10003,
			})
			return
		}
		// 提交事务
		if err := tx.Commit().Error; err != nil {
			log.Printf("Database transaction commit error: %v", err)
			c.JSON(consts.StatusInternalServerError, utils.H{
				"info":   "Database transaction commit error",
				"status": 10003,
			})
			return
		}

		// 返回成功响应
		c.JSON(consts.StatusOK, UpdateUserResponse{
			Info:   "success",
			Status: 10000,
		})
	})

	// 启动服务器
	h.Spin()
}
