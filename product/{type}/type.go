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

// ProductInfoResponse 定义获取单个商品信息的响应结构体
type ProductInfoResponse struct {
	Status int     `json:"status"`
	Info   string  `json:"info"`
	Data   Product `json:"data"`
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
	h := server.New(server.WithHostPorts("127.0.0.1:8010"))
	h.GET("/product/{type}", func(ctx context.Context, c *app.RequestContext) {
		productType := c.Param("type")
		if productType == "" {
			productType = c.Query("type")
		}
		if productType == "" {
			c.JSON(http.StatusBadRequest, ProductListResponse{
				Status: 10001,
				Info:   "type is required",
			})
			return
		}
		var products []Product
		result := DB.Where("type =?", productType).Find(&products)
		if result.Error != nil {
			c.JSON(http.StatusInternalServerError, ProductListResponse{
				Status: 10002,
				Info:   "Failed to query product list",
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
		c.JSON(http.StatusOK, resp)
	})
	if err := h.Run(); err != nil {
		fmt.Printf("Server run failed: %v\n", err)
	}
}
