package postgres

import (
	"catalog-sync/internal/domain"
	"context"
	"database/sql"
	"errors"
	"fmt"
)

type ProductRepository struct {
	db *DB
}

func NewProductRepository(db *DB) *ProductRepository {
	return &ProductRepository{db: db}
}

// GetBySKU queries a product by SKU within or outside an active transaction.
func (r *ProductRepository) GetBySKU(ctx context.Context, tx *sql.Tx, sku string) (*domain.Product, error) {
	query := `
		SELECT id, sku, name, description, price_cents, status, supplier_updated_at, withdrawn_at, created_at, updated_at
		FROM products
		WHERE sku = $1
	`
	var row *sql.Row
	if tx != nil {
		row = tx.QueryRowContext(ctx, query, sku)
	} else {
		row = r.db.QueryRowContext(ctx, query, sku)
	}

	var p domain.Product
	var withdrawnAt sql.NullTime
	err := row.Scan(
		&p.ID,
		&p.SKU,
		&p.Name,
		&p.Description,
		&p.PriceCents,
		&p.Status,
		&p.SupplierUpdatedAt,
		&withdrawnAt,
		&p.CreatedAt,
		&p.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil // Not found
		}
		return nil, fmt.Errorf("error querying product by SKU: %w", err)
	}

	if withdrawnAt.Valid {
		p.WithdrawnAt = &withdrawnAt.Time
	}

	return &p, nil
}

// Create inserts a new product within a transaction.
func (r *ProductRepository) Create(ctx context.Context, tx *sql.Tx, p *domain.Product) error {
	query := `
		INSERT INTO products (id, sku, name, description, price_cents, status, supplier_updated_at, withdrawn_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`
	var withdrawnAt *sql.NullTime
	if p.WithdrawnAt != nil {
		withdrawnAt = &sql.NullTime{Time: *p.WithdrawnAt, Valid: true}
	}

	_, err := tx.ExecContext(ctx, query,
		p.ID, p.SKU, p.Name, p.Description, p.PriceCents, p.Status,
		p.SupplierUpdatedAt, withdrawnAt, p.CreatedAt, p.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to insert product: %w", err)
	}
	return nil
}

// Update updates an existing product within a transaction.
func (r *ProductRepository) Update(ctx context.Context, tx *sql.Tx, p *domain.Product) error {
	query := `
		UPDATE products
		SET name = $1, description = $2, price_cents = $3, status = $4,
		    supplier_updated_at = $5, withdrawn_at = $6, updated_at = $7
		WHERE id = $8
	`
	var withdrawnAt *sql.NullTime
	if p.WithdrawnAt != nil {
		withdrawnAt = &sql.NullTime{Time: *p.WithdrawnAt, Valid: true}
	}

	_, err := tx.ExecContext(ctx, query,
		p.Name, p.Description, p.PriceCents, p.Status,
		p.SupplierUpdatedAt, withdrawnAt, p.UpdatedAt, p.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update product: %w", err)
	}
	return nil
}

// ListProducts retrieves products with optional status filter and search query.
func (r *ProductRepository) ListProducts(ctx context.Context, status string, search string, limit, offset int) ([]domain.Product, int, error) {
	where := "WHERE 1=1"
	args := []interface{}{}
	argIdx := 1

	if status != "" && status != "ALL" {
		where += fmt.Sprintf(" AND status = $%d", argIdx)
		args = append(args, status)
		argIdx++
	}

	if search != "" {
		where += fmt.Sprintf(" AND (name ILIKE $%d OR sku ILIKE $%d)", argIdx, argIdx)
		args = append(args, "%"+search+"%")
		argIdx++
	}

	// Count total
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM products %s", where)
	var total int
	err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count products: %w", err)
	}

	// Fetch data
	dataQuery := fmt.Sprintf(`
		SELECT id, sku, name, description, price_cents, status, supplier_updated_at, withdrawn_at, created_at, updated_at
		FROM products
		%s
		ORDER BY updated_at DESC, sku ASC
		LIMIT $%d OFFSET $%d
	`, where, argIdx, argIdx+1)

	args = append(args, limit, offset)
	rows, err := r.db.QueryContext(ctx, dataQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list products: %w", err)
	}
	defer rows.Close()

	products := make([]domain.Product, 0)
	for rows.Next() {
		var p domain.Product
		var withdrawnAt sql.NullTime
		if err := rows.Scan(
			&p.ID, &p.SKU, &p.Name, &p.Description, &p.PriceCents, &p.Status,
			&p.SupplierUpdatedAt, &withdrawnAt, &p.CreatedAt, &p.UpdatedAt,
		); err != nil {
			return nil, 0, err
		}
		if withdrawnAt.Valid {
			p.WithdrawnAt = &withdrawnAt.Time
		}
		products = append(products, p)
	}

	return products, total, nil
}
