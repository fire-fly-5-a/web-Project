package main

import (
	"context"
	"fmt"
	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"net/http"
)

// Comment 定义评论结构体
type Comment struct {
	PostID      string `json:"post_id"`
	PublishTime string `json:"publish_time"`
	Content     string `json:"content"`
	UserID      string `json:"user_id"`
	Avatar      string `json:"avatar"`
	Nickname    string `json:"nickname"`
	PraiseCount int    `json:"praise_count"`
	IsPraised   int    `json:"is_praised"`
	ProductID   string `json:"product_id"`
}

// CommentResponse 定义获取评论的响应结构体
type CommentResponse struct {
	Status   int       `json:"status"`
	Info     string    `json:"info"`
	Comments []Comment `json:"comments"`
}

var DB *gorm.DB

func InitDB() error {
	dsn := "root:123456@tcp(127.0.0.1:3306)/MySQL?charset=utf8mb4&parseTime=True&loc=Local"
	var err error
	DB, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		return fmt.Errorf("failed to connect database: %w", err)
	}
	return nil
}

func main() {
	if err := InitDB(); err != nil {
		fmt.Printf("Database initialization failed: %v\n", err)
		return
	}
	h := server.New(server.WithHostPorts("127.0.0.1:8011"))
	h.GET("/comment/{product_id}", func(ctx context.Context, c *app.RequestContext) {
		productID := c.Query("product_id")
		if productID == "" {
			c.JSON(http.StatusBadRequest, CommentResponse{
				Status: 10001,
				Info:   "product_id is required",
			})
			return
		}
		var comments []Comment
		result := DB.Table("comments").Where("product_id =?", productID).Find(&comments)
		if result.Error != nil {
			c.JSON(http.StatusInternalServerError, CommentResponse{
				Status: 10002,
				Info:   "Failed to query comments",
			})
			return
		}
		resp := CommentResponse{
			Status:   10000,
			Info:     "success",
			Comments: comments,
		}
		c.JSON(http.StatusOK, resp)
	})
	if err := h.Run(); err != nil {
		fmt.Printf("Server run failed: %v\n", err)
	}
}
