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
)

// Comment 定义评论结构体
type Comment struct {
	ProductID   uint `gorm:"primaryKey"`
	PostID      uint
	CommentID   uint
	Content     string
	PraiseCount uint
}

// PraiseRequest 点赞点踩请求结构体
type PraiseRequest struct {
	Model     int  `form:"model"`
	CommentID uint `form:"comment_id"`
}

var DB *gorm.DB

// 初始化数据库连接
func InitDB() error {
	dsn := "root:123456@tcp(127.0.0.1:3306)/MySQL?charset=utf8mb4&parseTime=True&loc=Local"
	var err error
	DB, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		return fmt.Errorf("failed to connect database: %w", err)
	}
	// 自动迁移表结构
	err = DB.AutoMigrate(&Comment{})
	if err != nil {
		return fmt.Errorf("failed to auto - migrate database: %w", err)
	}
	return nil
}

// PraiseCommentHandler 点赞点踩评论处理函数
func PraiseCommentHandler(ctx context.Context, c *app.RequestContext) {
	token := c.Request.Header.Get("Authorization")
	if token == "" {
		c.JSON(consts.StatusUnauthorized, utils.H{
			"info":   "Authorization token is required",
			"status": 401,
		})
		return
	}

	var req PraiseRequest
	err := c.Bind(&req)
	if err != nil {
		c.JSON(consts.StatusBadRequest, utils.H{
			"info":   fmt.Sprintf("failed to bind request: %v", err),
			"status": 400,
		})
		return
	}

	if req.Model != 1 && req.Model != 2 {
		c.JSON(consts.StatusBadRequest, utils.H{
			"info":   "invalid model value, should be 1 for praise or 2 for dislike",
			"status": 400,
		})
		return
	}

	if req.CommentID == 0 {
		c.JSON(consts.StatusBadRequest, utils.H{
			"info":   "comment_id is required",
			"status": 400,
		})
		return
	}

	if DB == nil {
		c.JSON(consts.StatusInternalServerError, utils.H{
			"info":   "Database connection is nil",
			"status": 500,
		})
		return
	}

	var comment Comment
	result := DB.First(&comment, "user_id =?", req.CommentID)
	if result.Error != nil {
		c.JSON(consts.StatusNotFound, utils.H{
			"info":   fmt.Sprintf("comment not found: %v", result.Error),
			"status": 404,
		})
		return
	}

	if req.Model == 1 {
		comment.PraiseCount++
	}
	result = DB.Save(&comment)
	if result.Error != nil {
		c.JSON(consts.StatusInternalServerError, utils.H{
			"info":   fmt.Sprintf("failed to update comment: %v", result.Error),
			"status": 500,
		})
		return
	}

	c.JSON(consts.StatusOK, utils.H{
		"info":   "success",
		"status": 10000,
	})
}

func main() {
	err := InitDB()
	if err != nil {
		fmt.Printf("Failed to initialize database: %v", err)
		return
	}
	h := server.New(server.WithHostPorts("127.0.0.1:8016"))
	h.PUT("/comment/praise", PraiseCommentHandler)
	h.Spin()
}
