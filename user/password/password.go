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
)

// 修改密码请求结构体
type ChangePasswordRequest struct {
	OldPassword string `json:"old_password"`
	NewPassword string `json:"new_password"`
}

// User 数据库用户模型
type User struct {
	ID       uint   `gorm:"primaryKey"`
	Username string `gorm:"unique;not null"`
	Password string `gorm:"not null"`
}

// 密钥，用于 JWT 签名和验证
var jwtKey = []byte("your_secret_key")

func ChangePassword(ctx context.Context, c *app.RequestContext) {
	// 从请求头获取 Authorization
	authorization := c.Request.Header.Get("Authorization")
	if authorization == "" {
		c.JSON(consts.StatusBadRequest, utils.H{
			"info":   "Authorization header is required",
			"status": 10001,
		})
		return
	}

	// 解析 token
	tokenString := authorization[len("Bearer "):]
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return jwtKey, nil
	})
	if err != nil || !token.Valid {
		c.JSON(consts.StatusUnauthorized, utils.H{
			"info":   "Invalid token",
			"status": 10003,
		})
		return
	}

	// 从 token 中获取用户名
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		c.JSON(consts.StatusUnauthorized, utils.H{
			"info":   "Invalid token claims",
			"status": 10008,
		})
		return
	}
	username, ok := claims["sub"].(string)
	if ok != true {
		c.JSON(consts.StatusUnauthorized, utils.H{
			"info":   "Username not found in token",
			"status": 10009,
		})
		return
	}

	var req ChangePasswordRequest
	// 绑定请求参数
	erro := c.BindAndValidate(&req)
	fmt.Println(erro)
	if erro != nil {
		c.JSON(consts.StatusBadRequest, utils.H{
			"info":   "Invalid request parameters",
			"status": 10002,
		})
		return
	}

	// 打开数据库连接
	dsn := "root:123456@tcp(127.0.0.1:3306)/MySQL?charset=utf8mb4&parseTime=True&loc=Local"
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		c.JSON(consts.StatusInternalServerError, utils.H{
			"info":   "Database connection error",
			"status": 10004,
		})
		return
	}

	// 根据从 token 解析出的用户名查找用户
	var user User
	if err := db.Where("username = ?", username).First(&user).Error; err != nil {
		c.JSON(consts.StatusNotFound, utils.H{
			"info":   "User not found",
			"status": 10005,
		})
		return
	}

	// 验证旧密码
	if user.Password != req.OldPassword {
		c.JSON(consts.StatusBadRequest, utils.H{
			"info":   "Old password is incorrect",
			"status": 10006,
		})
		return
	}

	// 更新新密码
	user.Password = req.NewPassword
	if err := db.Save(&user).Error; err != nil {
		c.JSON(consts.StatusInternalServerError, utils.H{
			"info":   "Failed to update password",
			"status": 10007,
		})
		return
	}

	c.JSON(consts.StatusOK, utils.H{
		"info":   "success",
		"status": 10000,
	})
}

func main() {
	h := server.New(server.WithHostPorts("127.0.0.1:8003"))
	h.PUT("/user/password", ChangePassword)
	h.Spin()
}
