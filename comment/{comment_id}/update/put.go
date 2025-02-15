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

// Comment 定义评论结构体
type Comment struct {
	ProductID uint `gorm:"primaryKey"`
	PostID    uint
	CommentID uint
	Content   string
}

// UpdateCommentRequest 定义更新评论的请求结构体
type UpdateCommentRequest struct {
	PostID  uint   `json:"post_id"`
	Content string `json:"content"`
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

func getAndValidateRequestBody(c *app.RequestContext) (uint, string, error) {
	var req UpdateCommentRequest
	if err := c.BindJSON(&req); err != nil {
		log.Printf("Error binding JSON: %v", err)
		return 0, "", fmt.Errorf("Invalid request body format")
	}

	if req.PostID == 0 {
		log.Println("post_id in request body is required")
		return 0, "", fmt.Errorf("post_id in request body is required")
	}
	if req.Content == "" {
		log.Println("content in request body is required")
		return 0, "", fmt.Errorf("content in request body is required")
	}

	return req.PostID, req.Content, nil
}

// UpdateCommentHandler 更新评论的处理函数
func UpdateCommentHandler(ctx context.Context, c *app.RequestContext) {
	token := c.Request.Header.Get("Authorization")
	if token == "" {
		c.JSON(consts.StatusUnauthorized, utils.H{
			"info":   "Authorization token is required",
			"status": 401,
		})
		return
	}

	postID, content, err := getAndValidateRequestBody(c)
	if err != nil {
		c.JSON(consts.StatusBadRequest, utils.H{
			"info":   fmt.Sprintf("Invalid request body: %v", err),
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
	result := DB.First(&comment, "post_id =?", postID)
	if result.Error != nil {
		c.JSON(consts.StatusNotFound, utils.H{
			"info":   fmt.Sprintf("comment not found: %v", result.Error),
			"status": 404,
		})
		return
	}
	// 将请求中获取的post_id赋值给comment_id
	comment.CommentID = postID
	comment.Content = content
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
	h := server.New(server.WithHostPorts("127.0.0.1:8015"))
	h.PUT("/comment/{comment_id}", UpdateCommentHandler)
	h.Spin()
}
