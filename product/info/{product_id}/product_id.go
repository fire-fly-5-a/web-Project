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
	h := server.New(server.WithHostPorts("127.0.0.1:8009"))
	h.GET("/product/info/{product_id}", func(ctx context.Context, c *app.RequestContext) {
		productId := c.Param("product_id")
		if productId == "" {
			productId = c.Query("product_id")
		}
		if productId == "" {
			c.JSON(http.StatusBadRequest, ProductInfoResponse{
				Status: 10001,
				Info:   "product_id is required",
			})
			return
		}
		var product Product
		result := DB.Where("product_id =?", productId).First(&product)
		if result.Error != nil {
			c.JSON(http.StatusInternalServerError, ProductInfoResponse{
				Status: 10002,
				Info:   "Failed to query product info",
			})
			return
		}
		resp := ProductInfoResponse{
			Status: 10000,
			Info:   "success",
			Data:   product,
		}
		c.JSON(http.StatusOK, resp)
	})
	if err := h.Run(); err != nil {
		fmt.Printf("Server run failed: %v\n", err)
	}
}
