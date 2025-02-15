package main

import (
	"bytes"
	"context"
	"fmt"
	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/common/utils"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
	"github.com/dgrijalva/jwt-go"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"log"
	"strconv"
)

// 定义购物车结构体
type Cart struct {
	gorm.Model
	UserID    uint   `gorm:"not null"`
	ProductID string `gorm:"not null"`
}

// 定义商品结构体
type Product struct {
	ProductID string  `gorm:"type:varchar(255);index"`
	Name      string  `json:"name"`
	Type      string  `json:"type"`
	Price     float64 `json:"price"`
	Cover     string  `json:"cover"`
	Link      string  `json:"link"`
	Num       int     `json:"num"`
}

// 定义响应结构体
type CartProductsResponse struct {
	Status int      `json:"status"`
	Info   string   `json:"info"`
	Data   CartData `json:"data"`
}

type CartData struct {
	Products []Product `json:"products"`
	Account  int       `json:"account"`
}

// 定义验证 JWT Token 的密钥
var jwtKey = []byte("your_secret_key")
var DB *gorm.DB

// 验证 JWT Token 并解析用户名
func validateAndParseUsername(tokenString string) (string, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return jwtKey, nil
	})
	if err != nil {
		return "", err
	}
	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		username, ok := claims["sub"].(string)
		if !ok {
			return "", fmt.Errorf("username claim not found in token")
		}
		return username, nil
	}
	return "", fmt.Errorf("invalid token")
}

// JWTAuthorization 中间件用于验证 JWT Token 并解析用户名
func JWTAuthorization() app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		authHeader := c.GetHeader("Authorization")
		if len(authHeader) < 7 || !bytes.Equal(authHeader[:7], []byte("Bearer ")) {
			c.JSON(consts.StatusUnauthorized, utils.H{
				"info":   "Invalid token format",
				"status": 10005,
			})
			return
		}
		tokenString := authHeader[7:]
		username, err := validateAndParseUsername(string(tokenString))
		if err != nil {
			c.JSON(consts.StatusUnauthorized, utils.H{
				"info":   "Unauthorized",
				"status": 10005,
			})
			return
		}
		c.Set("username", username) // 将用户名存储到上下文
		c.Next(ctx)
	}
}

// 初始化数据库连接
func InitDB() error {
	dsn := "root:123456@tcp(127.0.0.1:3306)/MySQL?charset=utf8mb4&parseTime=True&loc=Local"
	var err error
	DB, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		return fmt.Errorf("failed to connect database: %w", err)
	}
	// 自动迁移表结构
	err = DB.AutoMigrate(&Cart{}, &Product{})
	if err != nil {
		log.Printf("Failed to migrate database table: %v\n", err)
		return fmt.Errorf("failed to migrate database table: %w", err)
	}
	return nil
}

func main() {
	if err := InitDB(); err != nil {
		log.Printf("Database initialization failed: %v\n", err)
		return
	}
	fmt.Println("Database initialized successfully.")

	h := server.New(server.WithHostPorts("127.0.0.1:8008"))

	// 获取购物车商品列表接口
	h.GET("/product/cart", JWTAuthorization(), func(ctx context.Context, c *app.RequestContext) {
		USERidStr := c.Param("user_id")
		if USERidStr == "" {
			USERidStr = c.Query("user_id")
		}
		userid, err := strconv.Atoi(USERidStr)
		if err != nil {
			c.JSON(consts.StatusBadRequest, utils.H{
				"info":   "Invalid id parameter",
				"status": 10001,
			})
			return
		}

		var cartItems []Cart
		result := DB.Where("user_id =?", userid).Find(&cartItems)
		if result.Error != nil {
			c.JSON(consts.StatusInternalServerError, utils.H{
				"info":   "Failed to query cart items",
				"status": 10002,
			})
			return
		}

		var products []Product
		productIDs := make([]string, 0, len(cartItems))
		for _, cartItem := range cartItems {
			productIDs = append(productIDs, cartItem.ProductID)
		}

		if len(productIDs) > 0 {
			result = DB.Table("products").Where("product_id IN?", productIDs).Find(&products)
			if result.Error != nil {
				c.JSON(consts.StatusInternalServerError, utils.H{
					"info":   "Failed to query product details",
					"status": 10002,
				})
				return
			}
		}

		account := 0
		for _, product := range products {
			account += int(product.Price) // 累加价格作为金额
		}

		resp := CartProductsResponse{
			Status: 10000,
			Info:   "success",
			Data: CartData{
				Products: products,
				Account:  account,
			},
		}
		c.JSON(consts.StatusOK, resp)
	})

	if err := h.Run(); err != nil {
		log.Fatal(err)
	}
}
