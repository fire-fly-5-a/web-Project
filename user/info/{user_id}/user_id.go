package main

import (
	"context"
	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/common/utils"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// User 定义用户信息结构体，只包含需要的字段
type User struct {
	Username string `json:"nickname"`
	Email    string `json:"email"`
}

// UserInfoResponse 定义返回的用户信息响应结构体
type UserInfoResponse struct {
	Status int    `json:"status"`
	Info   string `json:"info"`
	Data   struct {
		User User `json:"user"`
	} `json:"data"`
}

func main() {
	h := server.New(server.WithHostPorts("127.0.0.1:8006"))

	h.GET("/user/info/{user_id}", func(ctx context.Context, c *app.RequestContext) {
		authHeaderBytes := c.GetHeader("Authorization")
		authHeader := string(authHeaderBytes)
		if authHeader == "" {
			c.JSON(consts.StatusUnauthorized, utils.H{
				"status": 10005,
				"info":   "Missing authorization token",
			})
			return
		}

		// 数据库连接，这里的dsn需要根据实际情况修改
		dsn := "root:123456@tcp(127.0.0.1:3306)/MySQL?charset=utf8mb4&parseTime=True&loc=Local"
		db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
		if err != nil {
			c.JSON(consts.StatusInternalServerError, utils.H{
				"status": 10004,
				"info":   "Database connection error",
			})
			return
		}

		// 获取用户ID
		// 优先从路径参数获取user_id
		userID := c.Param("user_id")
		if userID == "" {
			// 如果路径参数中未获取到，则从查询参数中获取
			userID = c.Query("user_id")
		}
		if userID == "" {
			c.JSON(consts.StatusBadRequest, utils.H{
				"status": 10001,
				"info":   "Missing user ID",
			})
			return
		}

		// 模拟从数据库查询用户信息，将username赋值给Nickname
		var user User
		result := db.Table("users").Select("username", "email").Where("id =?", userID).First(&user)
		if result.Error != nil {
			if result.Error == gorm.ErrRecordNotFound {
				c.JSON(consts.StatusNotFound, utils.H{
					"status": 10002,
					"info":   "User not found",
				})
			} else {
				c.JSON(consts.StatusInternalServerError, utils.H{
					"status": 10003,
					"info":   "Database query error",
				})
			}
			return
		}

		// 创建返回结构体
		resp := UserInfoResponse{
			Status: 10000,
			Info:   "success",
			Data: struct {
				User User `json:"user"`
			}{User: user},
		}

		// 返回JSON响应
		c.JSON(consts.StatusOK, resp)
	})

	// 启动服务器
	h.Spin()
}
