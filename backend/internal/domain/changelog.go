package domain

import (
	"encoding/json"
	"time"
)

type ChangeType string

const (
	ChangeTypeCreated          ChangeType = "CREATED"
	ChangeTypeUpdated          ChangeType = "UPDATED"
	ChangeTypeWithdrawn        ChangeType = "WITHDRAWN"
	ChangeTypeSkippedOutdated   ChangeType = "SKIPPED_OUTDATED"
	ChangeTypeSkippedIdentical  ChangeType = "SKIPPED_IDENTICAL"
)

type DiffField struct {
	Old interface{} `json:"old"`
	New interface{} `json:"new"`
}

type ProductDiff map[string]DiffField

type ChangeLog struct {
	ID         int64           `json:"id"`
	JobID      string          `json:"job_id"`
	BatchIndex int             `json:"batch_index"`
	SKU        string          `json:"sku"`
	ChangeType ChangeType      `json:"change_type"`
	DiffData   json.RawMessage `json:"diff_data,omitempty"`
	CreatedAt  time.Time       `json:"created_at"`
}
