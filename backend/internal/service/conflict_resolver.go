package service

import (
	"catalog-sync/internal/domain"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

var (
	ErrEmptySKU     = errors.New("SKU cannot be empty")
	ErrInvalidPrice = errors.New("price cannot be negative")
)

type ResolutionAction string

const (
	ActionCreate         ResolutionAction = "CREATE"
	ActionUpdate         ResolutionAction = "UPDATE"
	ActionWithdraw       ResolutionAction = "WITHDRAW"
	ActionSkipOutdated   ResolutionAction = "SKIP_OUTDATED"
	ActionSkipIdentical  ResolutionAction = "SKIP_IDENTICAL"
	ActionRejectConflict ResolutionAction = "REJECT_CONFLICT"
)

type ResolutionResult struct {
	Action         ResolutionAction
	TargetProduct  *domain.Product
	Diff           domain.ProductDiff
	DiffJSON       json.RawMessage
	ChangeType     domain.ChangeType
	SkipReason     string
	ValidationError error
}

type ConflictResolver struct{}

func NewConflictResolver() *ConflictResolver {
	return &ConflictResolver{}
}

// ValidateItem checks for corrupted or invalid incoming supplier data.
func (r *ConflictResolver) ValidateItem(item domain.SupplierProductItem) error {
	if item.SKU == "" {
		return ErrEmptySKU
	}
	if item.PriceCents < 0 {
		return fmt.Errorf("%w: received %d cents", ErrInvalidPrice, item.PriceCents)
	}
	return nil
}

// Resolve compares an incoming supplier item against the currently stored product.
func (r *ConflictResolver) Resolve(item domain.SupplierProductItem, existing *domain.Product, now time.Time) ResolutionResult {
	if err := r.ValidateItem(item); err != nil {
		return ResolutionResult{
			Action:          ActionRejectConflict,
			ValidationError: err,
		}
	}

	// 1. Product is brand new to local catalog
	if existing == nil {
		status := domain.ProductStatusActive
		var withdrawnAt *time.Time
		if item.IsWithdrawn {
			status = domain.ProductStatusWithdrawn
			withdrawnAt = &item.UpdatedAt
		}

		newProduct := &domain.Product{
			ID:                uuid.NewString(),
			SKU:               item.SKU,
			Name:              item.Name,
			Description:       item.Description,
			PriceCents:        item.PriceCents,
			Status:            status,
			SupplierUpdatedAt: item.UpdatedAt,
			WithdrawnAt:       withdrawnAt,
			CreatedAt:         now,
			UpdatedAt:         now,
		}

		diff := domain.ProductDiff{
			"status":      domain.DiffField{Old: nil, New: string(status)},
			"name":        domain.DiffField{Old: nil, New: item.Name},
			"price_cents": domain.DiffField{Old: nil, New: item.PriceCents},
		}
		diffJSON, _ := json.Marshal(diff)

		return ResolutionResult{
			Action:        ActionCreate,
			TargetProduct: newProduct,
			Diff:          diff,
			DiffJSON:      diffJSON,
			ChangeType:    domain.ChangeTypeCreated,
		}
	}

	// 2. Existing product found: Check Out-of-order timestamp rule
	// "older information must not replace newer information"
	if !item.UpdatedAt.After(existing.SupplierUpdatedAt) {
		return ResolutionResult{
			Action:     ActionSkipOutdated,
			ChangeType: domain.ChangeTypeSkippedOutdated,
			SkipReason: fmt.Sprintf("Incoming timestamp (%s) is not newer than existing timestamp (%s)",
				item.UpdatedAt.Format(time.RFC3339), existing.SupplierUpdatedAt.Format(time.RFC3339)),
		}
	}

	// 3. Check Withdrawn Protection
	// "older information must not replace newer information or restore a withdrawn product"
	if existing.Status == domain.ProductStatusWithdrawn && !item.IsWithdrawn {
		// If product is currently withdrawn, check if incoming item was timestamped before or equal to withdrawal
		withdrawnTime := existing.SupplierUpdatedAt
		if existing.WithdrawnAt != nil {
			withdrawnTime = *existing.WithdrawnAt
		}
		if !item.UpdatedAt.After(withdrawnTime) {
			return ResolutionResult{
				Action:     ActionSkipOutdated,
				ChangeType: domain.ChangeTypeSkippedOutdated,
				SkipReason: fmt.Sprintf("Out-of-order update cannot restore withdrawn product (item timestamp: %s, withdrawn: %s)",
					item.UpdatedAt.Format(time.RFC3339), withdrawnTime.Format(time.RFC3339)),
			}
		}
	}

	// 4. Calculate Diffs
	diff := make(domain.ProductDiff)

	if existing.Name != item.Name {
		diff["name"] = domain.DiffField{Old: existing.Name, New: item.Name}
	}
	if existing.Description != item.Description {
		diff["description"] = domain.DiffField{Old: existing.Description, New: item.Description}
	}
	if existing.PriceCents != item.PriceCents {
		diff["price_cents"] = domain.DiffField{Old: existing.PriceCents, New: item.PriceCents}
	}

	targetStatus := domain.ProductStatusActive
	if item.IsWithdrawn {
		targetStatus = domain.ProductStatusWithdrawn
	}

	if existing.Status != targetStatus {
		diff["status"] = domain.DiffField{Old: string(existing.Status), New: string(targetStatus)}
	}

	// If no fields changed at all
	if len(diff) == 0 {
		return ResolutionResult{
			Action:     ActionSkipIdentical,
			ChangeType: domain.ChangeTypeSkippedIdentical,
			SkipReason: "Item content is identical to stored product",
		}
	}

	// Build updated product entity
	updatedProduct := *existing
	updatedProduct.Name = item.Name
	updatedProduct.Description = item.Description
	updatedProduct.PriceCents = item.PriceCents
	updatedProduct.Status = targetStatus
	updatedProduct.SupplierUpdatedAt = item.UpdatedAt
	updatedProduct.UpdatedAt = now

	changeType := domain.ChangeTypeUpdated
	action := ActionUpdate

	if targetStatus == domain.ProductStatusWithdrawn && existing.Status != domain.ProductStatusWithdrawn {
		updatedProduct.WithdrawnAt = &item.UpdatedAt
		changeType = domain.ChangeTypeWithdrawn
		action = ActionWithdraw
	}

	diffJSON, _ := json.Marshal(diff)

	return ResolutionResult{
		Action:        action,
		TargetProduct: &updatedProduct,
		Diff:          diff,
		DiffJSON:      diffJSON,
		ChangeType:    changeType,
	}
}
