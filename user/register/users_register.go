package main

import (
	"context"
	"fmt"
	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/common/utils"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"log"
)

type User struct {
	Username string `gorm:"uniqueIndex;not_null"`
	Password string `gorm:"not_null"`
}

var DB *gorm.DB

func InitDB() error {
	dsn := "root:123456@tcp(127.0.0.1:3306)/MySQL?charset=utf8mb4&parseTime=True&loc=Local"
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		return fmt.Errorf("failed to connect database: %w", err)
	}

	// 自动迁移
	err = db.AutoMigrate(&User{})
	if err != nil {
		return fmt.Errorf("failed to auto migrate: %w", err)
	}

	DB = db
	return nil
}

func Register(ctx context.Context, c *app.RequestContext) {
	var req struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
	}

	if err := c.BindAndValidate(&req); err != nil {
		c.JSON(consts.StatusBadRequest, utils.H{"error": err.Error()})
		return
	}

	// 额外验证：确保用户名不为空字符串
	if req.Username == "" {
		c.JSON(consts.StatusBadRequest, utils.H{"error": "用户名不能为空"})
		return
	}

	err := DB.Transaction(func(tx *gorm.DB) error {
		// 检查用户名是否已存在
		var count int64
		result := tx.Model(&User{}).Where("username = ?", req.Username).Count(&count)
		if result.Error != nil {
			return result.Error
		}
		if count > 0 {
			return fmt.Errorf("username already exists")
		}

		// 创建新用户
		newUser := User{
			Username: req.Username,
			Password: req.Password,
		}
		result = tx.Create(&newUser)
		if result.Error != nil {
			return fmt.Errorf("failed to create user: %w", result.Error)
		}
		return nil
	})

	if err != nil {
		if err.Error() == "username already exists" {
			c.JSON(consts.StatusConflict, utils.H{"error": err.Error()})
		} else {
			c.JSON(consts.StatusInternalServerError, utils.H{"error": err.Error()})
		}
		return
	}

	c.JSON(consts.StatusOK, utils.H{
		"status": 10000,
		"info":   "success",
	})
}
func RegisterRoutes(r *server.Hertz) {
	r.POST("/user/register", Register)
}

func main() {
	err := InitDB()
	if err != nil {
		log.Fatalf("failed to initialize database: %v", err)
	}
	h := server.New(server.WithHostPorts("127.0.0.1:8001"))
	RegisterRoutes(h)
	h.Run()
}
