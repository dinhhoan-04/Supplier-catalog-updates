package domain

import (
	"encoding/json"
	"time"
)

type IssueErrorCode string

const (
	ErrCodeInvalidPrice        IssueErrorCode = "INVALID_PRICE"
	ErrCodeMissingSKU          IssueErrorCode = "MISSING_SKU"
	ErrCodeCorruptData         IssueErrorCode = "CORRUPT_DATA"
	ErrCodeNegativeQuantity    IssueErrorCode = "NEGATIVE_QUANTITY"
	ErrCodeIllegalReactivation IssueErrorCode = "ILLEGAL_REACTIVATION"
	ErrCodeSupplierTimeout     IssueErrorCode = "SUPPLIER_TIMEOUT"
)

type CatalogIssue struct {
	ID         int64           `json:"id"`
	JobID      string          `json:"job_id"`
	BatchIndex int             `json:"batch_index"`
	SKU        *string         `json:"sku,omitempty"`
	ErrorCode  IssueErrorCode  `json:"error_code"`
	Message    string          `json:"message"`
	RawItem    json.RawMessage `json:"raw_item,omitempty"`
	Resolved   bool            `json:"resolved"`
	CreatedAt  time.Time       `json:"created_at"`
}
