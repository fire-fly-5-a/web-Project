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
	"strconv"
	"time"
)

// OrderItem 定义订单内容中的单个商品项结构体
type OrderItem struct {
	ID        uint `gorm:"primaryKey"`
	OrderID   uint
	ProductID uint
	Quantity  uint
}

// Order 定义订单结构体
type Order struct {
	OrderID    uint `gorm:"primaryKey"`
	UserID     uint
	Address    string
	Total      float64
	CreatedAt  time.Time
	OrderItems []OrderItem `gorm:"foreignKey:OrderID"`
}

// OrderRequest 定义下单请求结构体
type OrderRequest struct {
	UserID  uint        `json:"user_id"`
	Orders  []OrderItem `json:"orders"`
	Address string      `json:"address"`
	Total   float64     `json:"total"`
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
	err = DB.AutoMigrate(&Order{}, &OrderItem{})
	if err != nil {
		return fmt.Errorf("failed to auto - migrate database: %w", err)
	}
	return nil
}

// PlaceOrderHandler 下单处理函数
func PlaceOrderHandler(ctx context.Context, c *app.RequestContext) {
	token := c.Request.Header.Get("Authorization")
	if token == "" {
		c.JSON(consts.StatusUnauthorized, utils.H{
			"info":   "Authorization token is required",
			"status": 401,
		})
		return
	}

	var req OrderRequest
	err := c.Bind(&req)
	if err != nil {
		c.JSON(consts.StatusBadRequest, utils.H{
			"info":   fmt.Sprintf("failed to bind request: %v", err),
			"status": 400,
		})
		return
	}

	if req.UserID == 0 {
		c.JSON(consts.StatusBadRequest, utils.H{
			"info":   "user_id is required",
			"status": 400,
		})
		return
	}

	if len(req.Orders) == 0 {
		c.JSON(consts.StatusBadRequest, utils.H{
			"info":   "orders is required",
			"status": 400,
		})
		return
	}

	if req.Address == "" {
		c.JSON(consts.StatusBadRequest, utils.H{
			"info":   "address is required",
			"status": 400,
		})
		return
	}

	if req.Total <= 0 {
		c.JSON(consts.StatusBadRequest, utils.H{
			"info":   "total should be greater than 0",
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

	newOrder := Order{
		UserID:    req.UserID,
		Address:   req.Address,
		Total:     req.Total,
		CreatedAt: time.Now(),
	}
	result := DB.Create(&newOrder)
	if result.Error != nil {
		c.JSON(consts.StatusInternalServerError, utils.H{
			"info":   fmt.Sprintf("failed to create order: %v", result.Error),
			"status": 500,
		})
		return
	}

	for _, item := range req.Orders {
		// 明确传递字段值
		orderItem := OrderItem{
			OrderID:   newOrder.OrderID,
			ProductID: item.ProductID,
			Quantity:  item.Quantity,
		}
		result = DB.Create(&orderItem)
		if result.Error != nil {
			c.JSON(consts.StatusInternalServerError, utils.H{
				"info":   fmt.Sprintf("failed to create order item: %v", result.Error),
				"status": 500,
			})
			return
		}
	}

	c.JSON(consts.StatusOK, utils.H{
		"info":     "success",
		"status":   10000,
		"order_id": strconv.Itoa(int(newOrder.OrderID)),
	})
}

func main() {
	err := InitDB()
	if err != nil {
		fmt.Printf("Failed to initialize database: %v", err)
		return
	}
	h := server.New(server.WithHostPorts("127.0.0.1:8017"))
	h.POST("/operate/order", PlaceOrderHandler)
	h.Spin()
}
