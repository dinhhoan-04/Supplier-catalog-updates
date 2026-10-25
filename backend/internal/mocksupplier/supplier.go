package mocksupplier

import (
	"catalog-sync/internal/domain"
	"fmt"
	"time"
)

type MockSupplier struct{}

func NewMockSupplier() *MockSupplier {
	return &MockSupplier{}
}

// GetBatches returns a deterministic sequence of batches designed to demonstrate all challenge requirements.
func (s *MockSupplier) GetBatches(scenario string) []domain.SupplierBatch {
	// Base timestamps
	t0 := time.Date(2026, 10, 20, 10, 0, 0, 0, time.UTC)
	t1 := t0.Add(2 * time.Hour) // t1 > t0
	t2 := t0.Add(4 * time.Hour) // t2 > t1
	t3 := t0.Add(6 * time.Hour) // t3 > t2

	batches := []domain.SupplierBatch{
		// BATCH 0: Initial import (New products)
		{
			BatchIndex: 0,
			TotalItems: 5,
			Items: []domain.SupplierProductItem{
				{
					SKU:         "SKU-HEADPHONE-01",
					Name:        "Wireless Noise-Canceling Headphones",
					Description: "Premium over-ear wireless headphones with ANC",
					PriceCents:  19900,
					IsWithdrawn: false,
					UpdatedAt:   t0,
				},
				{
					SKU:         "SKU-KEYBOARD-02",
					Name:        "Mechanical Gaming Keyboard",
					Description: "RGB mechanical keyboard with blue tactile switches",
					PriceCents:  8900,
					IsWithdrawn: false,
					UpdatedAt:   t0,
				},
				{
					SKU:         "SKU-MUG-03",
					Name:        "Ceramic Artisan Coffee Mug",
					Description: "Handmade 350ml ceramic mug for coffee lovers",
					PriceCents:  1500,
					IsWithdrawn: false,
					UpdatedAt:   t0,
				},
				{
					SKU:         "SKU-CHAIR-04",
					Name:        "Ergonomic Mesh Office Chair",
					Description: "Breathable lumbar support desk chair",
					PriceCents:  24900,
					IsWithdrawn: false,
					UpdatedAt:   t0,
				},
				{
					SKU:         "SKU-MOUSE-05",
					Name:        "Wireless Ergonomic Vertical Mouse",
					Description: "Comfort grip mouse to prevent wrist strain",
					PriceCents:  4500,
					IsWithdrawn: false,
					UpdatedAt:   t0,
				},
			},
		},

		// BATCH 1: Price and description updates (Newer timestamps t1)
		{
			BatchIndex: 1,
			TotalItems: 3,
			Items: []domain.SupplierProductItem{
				{
					SKU:         "SKU-HEADPHONE-01",
					Name:        "Wireless Noise-Canceling Headphones (Gen 2)",
					Description: "Updated 40h battery life and enhanced ANC",
					PriceCents:  17900, // Price drop from 199.00 to 179.00
					IsWithdrawn: false,
					UpdatedAt:   t1,
				},
				{
					SKU:         "SKU-MUG-03",
					Name:        "Ceramic Artisan Coffee Mug (Limited Edition)",
					Description: "Handmade 350ml ceramic mug in Matte Charcoal finish",
					PriceCents:  1800, // Price increase from 15.00 to 18.00
					IsWithdrawn: false,
					UpdatedAt:   t1,
				},
				{
					SKU:         "SKU-CABLE-06",
					Name:        "Braided USB-C Fast Charging Cable (2m)",
					Description: "Durable nylon braided 100W PD charging cable",
					PriceCents:  1200, // New product introduced in batch 1
					IsWithdrawn: false,
					UpdatedAt:   t1,
				},
			},
		},

		// BATCH 2: Products withdrawn from sale (t2)
		{
			BatchIndex: 2,
			TotalItems: 2,
			Items: []domain.SupplierProductItem{
				{
					SKU:         "SKU-KEYBOARD-02",
					Name:        "Mechanical Gaming Keyboard",
					Description: "Discontinued by manufacturer",
					PriceCents:  8900,
					IsWithdrawn: true, // Marked withdrawn at t2
					UpdatedAt:   t2,
				},
				{
					SKU:         "SKU-CHAIR-04",
					Name:        "Ergonomic Mesh Office Chair",
					Description: "Recalled due to supplier fabric shortage",
					PriceCents:  24900,
					IsWithdrawn: true, // Marked withdrawn at t2
					UpdatedAt:   t2,
				},
			},
		},

		// BATCH 3: Out-of-order updates (Old timestamp t0 sent after t1/t2)
		// MUST NOT replace newer info or restore withdrawn product!
		{
			BatchIndex: 3,
			TotalItems: 2,
			Items: []domain.SupplierProductItem{
				{
					// Attempt to overwrite HEADPHONE (currently at t1 with 17900) with old t0 price 19900
					SKU:         "SKU-HEADPHONE-01",
					Name:        "Wireless Noise-Canceling Headphones",
					Description: "Old out-of-order description",
					PriceCents:  19900,
					IsWithdrawn: false,
					UpdatedAt:   t0, // Older than t1!
				},
				{
					// Attempt to RESTORE KEYBOARD-02 (withdrawn at t2) with old active record from t0
					SKU:         "SKU-KEYBOARD-02",
					Name:        "Mechanical Gaming Keyboard",
					Description: "Old active record arriving out-of-order",
					PriceCents:  8900,
					IsWithdrawn: false, // Attempting illegal reactivation
					UpdatedAt:   t0,    // Older than t2!
				},
			},
		},

		// BATCH 4: Duplicate / Repeated import (Testing Idempotency)
		// Exact same payload as Batch 1 - must not create duplicates or alter state
		{
			BatchIndex: 4,
			TotalItems: 2,
			Items: []domain.SupplierProductItem{
				{
					SKU:         "SKU-HEADPHONE-01",
					Name:        "Wireless Noise-Canceling Headphones (Gen 2)",
					Description: "Updated 40h battery life and enhanced ANC",
					PriceCents:  17900,
					IsWithdrawn: false,
					UpdatedAt:   t1,
				},
				{
					SKU:         "SKU-CABLE-06",
					Name:        "Braided USB-C Fast Charging Cable (2m)",
					Description: "Durable nylon braided 100W PD charging cable",
					PriceCents:  1200,
					IsWithdrawn: false,
					UpdatedAt:   t1,
				},
			},
		},

		// BATCH 5: Problems needing attention (Invalid data, corrupted payload)
		{
			BatchIndex: 5,
			TotalItems: 3,
			Items: []domain.SupplierProductItem{
				{
					SKU:         "SKU-MONITOR-07",
					Name:        "Ultrawide 34-inch Curved Gaming Monitor",
					Description: "144Hz 1ms IPS Display with HDR",
					PriceCents:  -49900, // INVALID: Negative price!
					IsWithdrawn: false,
					UpdatedAt:   t3,
				},
				{
					SKU:         "", // INVALID: Missing SKU!
					Name:        "Mystery Product without SKU",
					Description: "Broken payload from vendor API",
					PriceCents:  2500,
					IsWithdrawn: false,
					UpdatedAt:   t3,
				},
				{
					SKU:         "SKU-MONITOR-08",
					Name:        "4K 27-inch Color Calibrated Monitor",
					Description: "Professional monitor for graphic designers",
					PriceCents:  39900, // Valid item in the same batch
					IsWithdrawn: false,
					UpdatedAt:   t3,
				},
			},
		},
	}

	return batches
}

// FetchBatch simulates network call to supplier with latency and failure options.
func (s *MockSupplier) FetchBatch(batches []domain.SupplierBatch, batchIndex int) (*domain.SupplierBatch, error) {
	if batchIndex < 0 || batchIndex >= len(batches) {
		return nil, fmt.Errorf("batch index %d out of bounds (total: %d)", batchIndex, len(batches))
	}
	return &batches[batchIndex], nil
}
