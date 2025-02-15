package main

import (
	"bytes"
	"context"
	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/common/utils"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
	"github.com/dgrijalva/jwt-go"
)

// Product 定义商品结构体
type Product struct {
	ProductID     string  `json:"product_id"`
	Name          string  `json:"name"`
	Description   string  `json:"description"`
	CommentNum    int     `json:"comment_num"`
	Type          string  `json:"type"`
	Price         float64 `json:"price"`
	IsAddedCart   bool    `json:"is_addedCart"`
	Cover         string  `json:"cover"`
	PublishTime   string  `json:"publish_time"`
	Link          string  `json:"link"`
	ProductNumber string  `json:"product_number"`
}

// ProductListResponse 定义商品列表响应结构体
type ProductListResponse struct {
	Status int    `json:"status"`
	Info   string `json:"info"`
	Data   struct {
		Products []Product `json:"products"`
	} `json:"data"`
}

var jwtKey = []byte("your_secret_key")

// 模拟的商品数据
var mockProducts = []Product{
	{
		ProductID:     "1",
		Name:          "傲慢与偏见",
		Description:   "一本书",
		CommentNum:    35,
		Type:          "book",
		Price:         9.80,
		IsAddedCart:   true,
		Cover:         "http://127.0.0.1/picture_url1",
		PublishTime:   "1980-11-07",
		Link:          "http://127.0.0.1/test1",
		ProductNumber: "",
	},
	{
		ProductID:     "2",
		Name:          "T-shirt",
		Description:   "一件短袖",
		CommentNum:    100,
		Type:          "clothes",
		Price:         88.88,
		IsAddedCart:   false,
		Cover:         "http://127.0.0.1/picture_url2",
		PublishTime:   "1980-11-07",
		Link:          "http://127.0.0.1/test2",
		ProductNumber: "",
	},
}

// 验证 JWT Token
func validateToken(tokenString string) bool {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return jwtKey, nil
	})
	return err == nil && token.Valid
}

// JWTAuthorization 中间件用于验证 JWT Token
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
		if !validateToken(string(tokenString)) {
			c.JSON(consts.StatusUnauthorized, utils.H{
				"info":   "Unauthorized",
				"status": 10005,
			})
			return
		}
		c.Next(ctx)
	}
}

func processProducts(products []Product, hasValidAuth bool) []Product {
	if !hasValidAuth {
		for i := range products {
			products[i].IsAddedCart = false
		}
	}
	return products
}

func main() {
	h := server.New(server.WithHostPorts("127.0.0.1:8007"))

	h.GET("/book/search", JWTAuthorization(), func(ctx context.Context, c *app.RequestContext) {
		productName := c.Query("product_name")
		if productName == "" {
			c.JSON(consts.StatusBadRequest, utils.H{
				"status": 10001,
				"info":   "product_name is required",
			})
			return
		}

		var filteredProducts []Product
		for _, product := range mockProducts {
			if product.Name == productName {
				filteredProducts = append(filteredProducts, product)
			}
		}

		authHeader := c.GetHeader("Authorization")
		hasValidAuth := len(authHeader) >= 7 && bytes.Equal(authHeader[:7], []byte("Bearer ")) && validateToken(string(authHeader[7:]))
		processedProducts := processProducts(filteredProducts, hasValidAuth)

		resp := ProductListResponse{
			Status: 10000,
			Info:   "success",
			Data: struct {
				Products []Product `json:"products"`
			}{Products: processedProducts},
		}
		c.JSON(consts.StatusOK, resp)
	})

	h.Spin()
}
