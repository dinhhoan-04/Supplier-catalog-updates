package service_test

import (
	"catalog-sync/internal/domain"
	"catalog-sync/internal/service"
	"testing"
	"time"
)

func TestConflictResolver_NewProduct(t *testing.T) {
	resolver := service.NewConflictResolver()
	now := time.Now().UTC()
	t0 := now.Add(-1 * time.Hour)

	item := domain.SupplierProductItem{
		SKU:         "SKU-TEST-01",
		Name:        "Test Item",
		Description: "A great test item",
		PriceCents:  5000,
		IsWithdrawn: false,
		UpdatedAt:   t0,
	}

	res := resolver.Resolve(item, nil, now)

	if res.Action != service.ActionCreate {
		t.Fatalf("Expected ActionCreate, got %s", res.Action)
	}
	if res.TargetProduct.SKU != "SKU-TEST-01" {
		t.Fatalf("Expected SKU-TEST-01, got %s", res.TargetProduct.SKU)
	}
	if res.TargetProduct.Status != domain.ProductStatusActive {
		t.Fatalf("Expected ACTIVE status, got %s", res.TargetProduct.Status)
	}
}

func TestConflictResolver_OlderInformationMustNotReplaceNewer(t *testing.T) {
	resolver := service.NewConflictResolver()
	now := time.Now().UTC()
	tNew := now.Add(-10 * time.Minute)
	tOld := now.Add(-2 * time.Hour)

	existing := &domain.Product{
		ID:                "prod-1",
		SKU:               "SKU-TEST-02",
		Name:              "Modern Version",
		PriceCents:        10000,
		Status:            domain.ProductStatusActive,
		SupplierUpdatedAt: tNew,
	}

	// Incoming item has older timestamp tOld < tNew
	incomingOld := domain.SupplierProductItem{
		SKU:         "SKU-TEST-02",
		Name:        "Old Stale Name",
		PriceCents:  8000,
		IsWithdrawn: false,
		UpdatedAt:   tOld,
	}

	res := resolver.Resolve(incomingOld, existing, now)

	if res.Action != service.ActionSkipOutdated {
		t.Fatalf("Expected ActionSkipOutdated for older update, got %s", res.Action)
	}
	if res.ChangeType != domain.ChangeTypeSkippedOutdated {
		t.Fatalf("Expected ChangeTypeSkippedOutdated, got %s", res.ChangeType)
	}
}

func TestConflictResolver_OlderInformationMustNotRestoreWithdrawnProduct(t *testing.T) {
	resolver := service.NewConflictResolver()
	now := time.Now().UTC()
	tWithdrawn := now.Add(-30 * time.Minute)
	tOldActive := now.Add(-2 * time.Hour) // Sent before withdrawal occurred

	existing := &domain.Product{
		ID:                "prod-withdrawn",
		SKU:               "SKU-WITHDRAWN-01",
		Name:              "Recalled Product",
		PriceCents:        5000,
		Status:            domain.ProductStatusWithdrawn,
		SupplierUpdatedAt: tWithdrawn,
		WithdrawnAt:       &tWithdrawn,
	}

	// Out-of-order packet arrives where product is still active, but timestamp is older than withdrawal
	incomingStaleActive := domain.SupplierProductItem{
		SKU:         "SKU-WITHDRAWN-01",
		Name:        "Recalled Product",
		PriceCents:  5000,
		IsWithdrawn: false, // Trying to restore active!
		UpdatedAt:   tOldActive,
	}

	res := resolver.Resolve(incomingStaleActive, existing, now)

	if res.Action != service.ActionSkipOutdated {
		t.Fatalf("Expected ActionSkipOutdated when out-of-order update attempts to restore withdrawn product, got %s", res.Action)
	}
}

func TestConflictResolver_IdempotencyRepeatedImport(t *testing.T) {
	resolver := service.NewConflictResolver()
	now := time.Now().UTC()
	tSame := now.Add(-1 * time.Hour)

	existing := &domain.Product{
		ID:                "prod-same",
		SKU:               "SKU-SAME-01",
		Name:              "Identical Product",
		Description:       "Identical Description",
		PriceCents:        4500,
		Status:            domain.ProductStatusActive,
		SupplierUpdatedAt: tSame,
	}

	// Incoming item has same timestamp and same content
	incomingSame := domain.SupplierProductItem{
		SKU:         "SKU-SAME-01",
		Name:        "Identical Product",
		Description: "Identical Description",
		PriceCents:  4500,
		IsWithdrawn: false,
		UpdatedAt:   tSame,
	}

	res := resolver.Resolve(incomingSame, existing, now)

	if res.Action != service.ActionSkipOutdated && res.Action != service.ActionSkipIdentical {
		t.Fatalf("Expected skip action for identical re-import, got %s", res.Action)
	}
}

func TestConflictResolver_ValidationRejectsCorruptItem(t *testing.T) {
	resolver := service.NewConflictResolver()
	now := time.Now().UTC()

	// Missing SKU
	res1 := resolver.Resolve(domain.SupplierProductItem{
		SKU:        "",
		PriceCents: 1000,
		UpdatedAt:  now,
	}, nil, now)

	if res1.Action != service.ActionRejectConflict {
		t.Fatalf("Expected ActionRejectConflict for missing SKU, got %s", res1.Action)
	}

	// Negative Price
	res2 := resolver.Resolve(domain.SupplierProductItem{
		SKU:        "SKU-ERR-01",
		PriceCents: -500,
		UpdatedAt:  now,
	}, nil, now)

	if res2.Action != service.ActionRejectConflict {
		t.Fatalf("Expected ActionRejectConflict for negative price, got %s", res2.Action)
	}
}
