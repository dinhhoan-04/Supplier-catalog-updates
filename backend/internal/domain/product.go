package domain

import (
	"time"
)

type ProductStatus string

const (
	ProductStatusActive    ProductStatus = "ACTIVE"
	ProductStatusWithdrawn ProductStatus = "WITHDRAWN"
)

// Product represents the internal catalog state stored in the shop's database.
type Product struct {
	ID                string        `json:"id"`
	SKU               string        `json:"sku"`
	Name              string        `json:"name"`
	Description       string        `json:"description"`
	PriceCents        int64         `json:"price_cents"`
	Status            ProductStatus `json:"status"`
	SupplierUpdatedAt time.Time     `json:"supplier_updated_at"`
	WithdrawnAt       *time.Time    `json:"withdrawn_at,omitempty"`
	CreatedAt         time.Time     `json:"created_at"`
	UpdatedAt         time.Time     `json:"updated_at"`
}

// SupplierProductItem represents a raw product entry delivered in a supplier batch.
type SupplierProductItem struct {
	SKU         string    `json:"sku"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	PriceCents  int64     `json:"price_cents"`
	IsWithdrawn bool      `json:"is_withdrawn"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// SupplierBatch represents a batch of products from the supplier.
type SupplierBatch struct {
	BatchIndex int                   `json:"batch_index"`
	TotalItems int                   `json:"total_items"`
	Items      []SupplierProductItem `json:"items"`
}
