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

// Product 定义商品结构体
type Product struct {
	ProductID   string  `json:"product_id"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Type        string  `json:"type"`
	CommentNum  int     `json:"comment_num"`
	Price       float64 `json:"price"`
	IsAddedCart bool    `json:"is_addedCart"`
	Cover       string  `json:"cover"`
	PublishTime string  `json:"publish_time"`
	Link        string  `json:"link"`
}

// ProductListResponse 定义商品列表响应结构体
type ProductListResponse struct {
	Status int    `json:"status"`
	Info   string `json:"info"`
	Data   struct {
		Products []Product `json:"products"`
	} `json:"data"`
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
	h := server.New(server.WithHostPorts("127.0.0.1:8005"))
	// 注册 GET /product/list 路由
	h.GET("/product/list", func(ctx context.Context, c *app.RequestContext) {
		var products []Product
		// 这里假设 Product 结构体与数据库表结构对应，从数据库查询数据
		result := DB.Find(&products)
		if result.Error != nil {
			c.JSON(consts.StatusInternalServerError, utils.H{
				"status": 10001,
				"info":   "Database query error",
			})
			return
		}
		resp := ProductListResponse{
			Status: 10000,
			Info:   "success",
			Data: struct {
				Products []Product `json:"products"`
			}{Products: products},
		}
		c.JSON(consts.StatusOK, resp)
	})
	h.Spin()
}
