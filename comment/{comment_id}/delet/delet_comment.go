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
	ProductID uint `gorm:"primaryKey"`
	PostID    uint `gorm:"primaryKey"`
	CommentID uint
	Content string
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

// DeleteCommentHandler 删除评论的处理函数
func DeleteCommentHandler(ctx context.Context, c *app.RequestContext) {
	productIDStr := c.Query("product_id")
	postIDStr := c.Query("post_id")
	if productIDStr == "" || postIDStr == "" {
		c.JSON(consts.StatusBadRequest, utils.H{
			"info":   "product_id and post_id are required",
			"status": 400,
		})
		return
	}
	var productID, postID uint
	_, err := fmt.Sscanf(productIDStr, "%d", &productID)
	if err != nil {
		c.JSON(consts.StatusBadRequest, utils.H{
			"info":   "invalid product_id format",
			"status": 400,
		})
		return
	}
	_, err = fmt.Sscanf(postIDStr, "%d", &postID)
	if err != nil {
		c.JSON(consts.StatusBadRequest, utils.H{
			"info":   "invalid post_id format",
			"status": 400,
		})
		return
	}
	token := c.Request.Header.Get("Authorization")
	if token == "" {
		c.JSON(consts.StatusUnauthorized, utils.H{
			"info":   "Authorization token is required",
			"status": 401,
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
	result := DB.First(&comment, "product_id =? AND post_id =?", productID, postID)
	if result.Error != nil {
		c.JSON(consts.StatusNotFound, utils.H{
			"info":   fmt.Sprintf("comment not found: %v", result.Error),
			"status": 404,
		})
		return
	}
	// 这里可以添加与product_id和post_id相关的其他逻辑，比如删除该评论对应的文章下的一些统计信息等
	// 先删除评论
	result = DB.Delete(&Comment{}, "product_id =? AND post_id =?", productID, postID)
	if result.Error != nil {
		c.JSON(consts.StatusInternalServerError, utils.H{
			"info":   fmt.Sprintf("failed to delete comment: %v", result.Error),
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
	h := server.New(server.WithHostPorts("127.0.0.1:8014"))
	h.DELETE("/comment/{comment_id}", DeleteCommentHandler)
	h.Spin()
}
