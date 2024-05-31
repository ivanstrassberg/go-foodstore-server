package main

import (
	"time"
)

type Product struct {
	ID          int     `json:"id"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Price       float64 `json:"price"`
	Stock       int     `json:"stock"`
	Rating      float64 `json:"rating"`
	Category_ID int     `json:"category_id"`
}
type ReqProduct struct {
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Price       float64 `json:"price"`
	Stock       int     `json:"stock"`
	Category_ID int     `json:"category_id"`
}

type Customer struct {
	ID              int       `json:"id"`
	Email           string    `json:"email"`
	PasswordHash    string    `json:"password_hash"`
	DeliveryAddress string    `json:"delivery_address"`
	CreatedAt       time.Time `json:"created_at"`
}

type CustomerLR struct {
	Email        string `json:"email"`
	PasswordHash string `json:"password_hash"`
}

type Order struct {
	ID         int       `json:"id"`
	CustomerID int       `json:"customer_id"`
	Total      float64   `json:"total"`
	Status     int       `json:"status"`
	IsCash     bool      `json:"is_cash"`
	CreatedAt  time.Time `json:"created_at"`
}

type Cart struct {
	ID         int `json:"id"`
	CustomerID int `json:"customer_id"`
}

type CartProduct struct {
	CartID    int `json:"cart_id"`
	ProductID int `json:"customer_id"`
}

type Category struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type ReqCategory struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type AuthKey struct {
	Header string `json:"Header"`
	Token  string `json:"Token"`
}

type CheckoutReq struct {
	ProductID int `json:"productID"`
	Quantity  int `json:"quantity"`
}

// type CheckoutReq struct {
// 	Items []CheckoutItem //`json:"cartItems"`
// }
