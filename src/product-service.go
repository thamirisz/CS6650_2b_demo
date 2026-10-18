package main

import (
	"net/http"
	"strconv"
	"sync"

	"github.com/gin-gonic/gin"
)

// Product matches components/schemas/Product in api.yaml.
type Product struct {
	ProductID    int32  `json:"product_id"`
	SKU          string `json:"sku"`
	Manufacturer string `json:"manufacturer"`
	CategoryID   int32  `json:"category_id"`
	Weight       int32  `json:"weight"`
	SomeOtherID  int32  `json:"some_other_id"`
}

// productInput is used only for reading the request body.
// Pointers let us tell "field missing" apart from "field is 0".
type productInput struct {
	ProductID    *int32  `json:"product_id" binding:"required,min=1"`
	SKU          *string `json:"sku" binding:"required,min=1,max=100"`
	Manufacturer *string `json:"manufacturer" binding:"required,min=1,max=200"`
	CategoryID   *int32  `json:"category_id" binding:"required,min=1"`
	Weight       *int32  `json:"weight" binding:"required,min=0"`
	SomeOtherID  *int32  `json:"some_other_id" binding:"required,min=1"`
}

// ErrorResponse matches components/schemas/Error in api.yaml.
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// productStore keeps products in memory.
// The lock is needed because Gin runs each request in its own goroutine.
type productStore struct {
	mu   sync.RWMutex
	data map[int32]Product
}

func newProductStore() *productStore {
	return &productStore{data: make(map[int32]Product)}
}

func (s *productStore) get(id int32) (Product, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	p, ok := s.data[id]
	return p, ok
}

func (s *productStore) save(p Product) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[p.ProductID] = p
}

func sendError(c *gin.Context, status int, code, message, details string) {
	c.AbortWithStatusJSON(status, ErrorResponse{Error: code, Message: message, Details: details})
}

// parseID turns the path value into an int32 that is >= 1 (spec: format int32, minimum 1).
func parseID(raw string) (int32, bool) {
	n, err := strconv.ParseInt(raw, 10, 32)
	if err != nil || n < 1 {
		return 0, false
	}
	return int32(n), true
}

// GET /products/{productId} -> 200, 404, 500
func getProduct(store *productStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, ok := parseID(c.Param("productId"))
		if !ok {
			// The spec lists no 400 for GET, so a bad ID is treated as "not found".
			sendError(c, http.StatusNotFound, "NOT_FOUND", "Product not found",
				"Product ID must be a positive integer")
			return
		}
		p, found := store.get(id)
		if !found {
			sendError(c, http.StatusNotFound, "NOT_FOUND", "Product not found", "")
			return
		}
		c.JSON(http.StatusOK, p)
	}
}

// POST /products/{productId}/details -> 204, 400, 404, 500
func addProductDetails(store *productStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, ok := parseID(c.Param("productId"))
		if !ok {
			sendError(c, http.StatusBadRequest, "INVALID_INPUT", "The provided input data is invalid",
				"Product ID must be a positive integer")
			return
		}

		var in productInput
		if err := c.ShouldBindJSON(&in); err != nil {
			sendError(c, http.StatusBadRequest, "INVALID_INPUT", "The provided input data is invalid", err.Error())
			return
		}
		if *in.ProductID != id {
			sendError(c, http.StatusBadRequest, "INVALID_INPUT", "The provided input data is invalid",
				"product_id in body must match productId in path")
			return
		}

		if _, found := store.get(id); !found {
			sendError(c, http.StatusNotFound, "NOT_FOUND", "Product not found", "")
			return
		}

		store.save(Product{
			ProductID:    *in.ProductID,
			SKU:          *in.SKU,
			Manufacturer: *in.Manufacturer,
			CategoryID:   *in.CategoryID,
			Weight:       *in.Weight,
			SomeOtherID:  *in.SomeOtherID,
		})
		c.Status(http.StatusNoContent)
	}
}

// simulateError lets us show a 500 on purpose: send header "X-Simulate-Error: true".
// For demo only.
func simulateError() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.GetHeader("X-Simulate-Error") == "true" {
			panic("simulated server failure")
		}
		c.Next()
	}
}

func main() {
	store := newProductStore()
	// Seed data: the spec has no "create product" route, so POST updates existing products.
	store.save(Product{ProductID: 1, SKU: "ABC-123-XYZ", Manufacturer: "Acme Corporation", CategoryID: 456, Weight: 1250, SomeOtherID: 789})
	store.save(Product{ProductID: 2, SKU: "DEF-456-UVW", Manufacturer: "Globex", CategoryID: 100, Weight: 500, SomeOtherID: 10})

	r := gin.New()
	r.Use(gin.Logger())
	// Any panic becomes a 500 with the Error schema.
	r.Use(gin.CustomRecovery(func(c *gin.Context, err any) {
		sendError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error", "")
	}))
	r.Use(simulateError())

	// Server URL in api.yaml ends in /v1, so all paths live under /v1.
	v1 := r.Group("/v1")
	v1.GET("/products/:productId", getProduct(store))
	v1.POST("/products/:productId/details", addProductDetails(store))

	r.Run(":8080")
}
