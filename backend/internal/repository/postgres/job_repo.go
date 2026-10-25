package postgres

import (
	"catalog-sync/internal/domain"
	"context"
	"database/sql"
	"errors"
	"fmt"
)

type JobRepository struct {
	db *DB
}

func NewJobRepository(db *DB) *JobRepository {
	return &JobRepository{db: db}
}

// CreateJob registers a new sync job record.
func (r *JobRepository) CreateJob(ctx context.Context, job *domain.SyncJob) error {
	query := `
		INSERT INTO catalog_sync_jobs (
			id, status, scenario, total_batches, processed_batches, total_items,
			created_count, updated_count, withdrawn_count, skipped_count, error_count,
			started_at, finished_at, interrupted_at, error_summary
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
	`
	_, err := r.db.ExecContext(ctx, query,
		job.ID, job.Status, job.Scenario, job.TotalBatches, job.ProcessedBatches, job.TotalItems,
		job.CreatedCount, job.UpdatedCount, job.WithdrawnCount, job.SkippedCount, job.ErrorCount,
		job.StartedAt, job.FinishedAt, job.InterruptedAt, job.ErrorSummary,
	)
	if err != nil {
		return fmt.Errorf("failed to create sync job: %w", err)
	}
	return nil
}

// UpdateJobStatus updates the status and statistics of an ongoing or completed job.
func (r *JobRepository) UpdateJobStatus(ctx context.Context, job *domain.SyncJob) error {
	query := `
		UPDATE catalog_sync_jobs
		SET status = $1, processed_batches = $2, total_items = $3,
		    created_count = $4, updated_count = $5, withdrawn_count = $6,
		    skipped_count = $7, error_count = $8, finished_at = $9,
		    interrupted_at = $10, error_summary = $11
		WHERE id = $12
	`
	_, err := r.db.ExecContext(ctx, query,
		job.Status, job.ProcessedBatches, job.TotalItems,
		job.CreatedCount, job.UpdatedCount, job.WithdrawnCount,
		job.SkippedCount, job.ErrorCount, job.FinishedAt,
		job.InterruptedAt, job.ErrorSummary, job.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update sync job: %w", err)
	}
	return nil
}

// GetJobByID finds a job by ID.
func (r *JobRepository) GetJobByID(ctx context.Context, jobID string) (*domain.SyncJob, error) {
	query := `
		SELECT id, status, scenario, total_batches, processed_batches, total_items,
		       created_count, updated_count, withdrawn_count, skipped_count, error_count,
		       started_at, finished_at, interrupted_at, error_summary
		FROM catalog_sync_jobs
		WHERE id = $1
	`
	row := r.db.QueryRowContext(ctx, query, jobID)
	return scanJob(row)
}

// GetLatestJob returns the most recent sync job.
func (r *JobRepository) GetLatestJob(ctx context.Context) (*domain.SyncJob, error) {
	query := `
		SELECT id, status, scenario, total_batches, processed_batches, total_items,
		       created_count, updated_count, withdrawn_count, skipped_count, error_count,
		       started_at, finished_at, interrupted_at, error_summary
		FROM catalog_sync_jobs
		ORDER BY started_at DESC
		LIMIT 1
	`
	row := r.db.QueryRowContext(ctx, query)
	return scanJob(row)
}

func scanJob(row *sql.Row) (*domain.SyncJob, error) {
	var j domain.SyncJob
	var finishedAt, interruptedAt sql.NullTime
	var errSummary sql.NullString

	err := row.Scan(
		&j.ID, &j.Status, &j.Scenario, &j.TotalBatches, &j.ProcessedBatches, &j.TotalItems,
		&j.CreatedCount, &j.UpdatedCount, &j.WithdrawnCount, &j.SkippedCount, &j.ErrorCount,
		&j.StartedAt, &finishedAt, &interruptedAt, &errSummary,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("error scanning job: %w", err)
	}

	if finishedAt.Valid {
		j.FinishedAt = &finishedAt.Time
	}
	if interruptedAt.Valid {
		j.InterruptedAt = &interruptedAt.Time
	}
	if errSummary.Valid {
		j.ErrorSummary = &errSummary.String
	}

	return &j, nil
}

// AddCheckpoint records that a batch was committed in the given transaction.
func (r *JobRepository) AddCheckpoint(ctx context.Context, tx *sql.Tx, jobID string, batchIndex, itemsCount int) error {
	query := `
		INSERT INTO job_batch_checkpoints (job_id, batch_index, items_count, processed_at, status)
		VALUES ($1, $2, $3, NOW(), 'COMMITTED')
		ON CONFLICT (job_id, batch_index) DO NOTHING
	`
	_, err := tx.ExecContext(ctx, query, jobID, batchIndex, itemsCount)
	if err != nil {
		return fmt.Errorf("failed to save checkpoint: %w", err)
	}
	return nil
}

// GetProcessedBatchIndices returns all batch indices successfully completed for this job.
func (r *JobRepository) GetProcessedBatchIndices(ctx context.Context, jobID string) (map[int]bool, error) {
	query := `SELECT batch_index FROM job_batch_checkpoints WHERE job_id = $1`
	rows, err := r.db.QueryContext(ctx, query, jobID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch checkpoints: %w", err)
	}
	defer rows.Close()

	indices := make(map[int]bool)
	for rows.Next() {
		var idx int
		if err := rows.Scan(&idx); err != nil {
			return nil, err
		}
		indices[idx] = true
	}
	return indices, nil
}

// RecordChangeLog logs an item change within the transaction.
func (r *JobRepository) RecordChangeLog(ctx context.Context, tx *sql.Tx, log *domain.ChangeLog) error {
	query := `
		INSERT INTO catalog_change_logs (job_id, batch_index, sku, change_type, diff_data, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	_, err := tx.ExecContext(ctx, query,
		log.JobID, log.BatchIndex, log.SKU, log.ChangeType, log.DiffData, log.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to record changelog: %w", err)
	}
	return nil
}

// ListChangeLogs returns the changelogs for a specific job.
func (r *JobRepository) ListChangeLogs(ctx context.Context, jobID string, limit, offset int) ([]domain.ChangeLog, int, error) {
	var total int
	countQuery := `SELECT COUNT(*) FROM catalog_change_logs WHERE job_id = $1`
	if err := r.db.QueryRowContext(ctx, countQuery, jobID).Scan(&total); err != nil {
		return nil, 0, err
	}

	query := `
		SELECT id, job_id, batch_index, sku, change_type, diff_data, created_at
		FROM catalog_change_logs
		WHERE job_id = $1
		ORDER BY id ASC
		LIMIT $2 OFFSET $3
	`
	rows, err := r.db.QueryContext(ctx, query, jobID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	logs := make([]domain.ChangeLog, 0)
	for rows.Next() {
		var l domain.ChangeLog
		var diff []byte
		if err := rows.Scan(&l.ID, &l.JobID, &l.BatchIndex, &l.SKU, &l.ChangeType, &diff, &l.CreatedAt); err != nil {
			return nil, 0, err
		}
		if diff != nil {
			l.DiffData = diff
		}
		logs = append(logs, l)
	}

	return logs, total, nil
}

// RecordIssue logs a problem needing attention.
func (r *JobRepository) RecordIssue(ctx context.Context, tx *sql.Tx, issue *domain.CatalogIssue) error {
	query := `
		INSERT INTO catalog_update_issues (job_id, batch_index, sku, error_code, message, raw_item, resolved, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	var err error
	if tx != nil {
		_, err = tx.ExecContext(ctx, query,
			issue.JobID, issue.BatchIndex, issue.SKU, issue.ErrorCode, issue.Message, issue.RawItem, issue.Resolved, issue.CreatedAt,
		)
	} else {
		_, err = r.db.ExecContext(ctx, query,
			issue.JobID, issue.BatchIndex, issue.SKU, issue.ErrorCode, issue.Message, issue.RawItem, issue.Resolved, issue.CreatedAt,
		)
	}
	if err != nil {
		return fmt.Errorf("failed to record issue: %w", err)
	}
	return nil
}

// ListIssues returns problems needing attention for a job or all active unresolved problems.
func (r *JobRepository) ListIssues(ctx context.Context, jobID string, resolved *bool) ([]domain.CatalogIssue, error) {
	where := "WHERE 1=1"
	args := []interface{}{}
	argIdx := 1

	if jobID != "" {
		where += fmt.Sprintf(" AND job_id = $%d", argIdx)
		args = append(args, jobID)
		argIdx++
	}

	if resolved != nil {
		where += fmt.Sprintf(" AND resolved = $%d", argIdx)
		args = append(args, *resolved)
		argIdx++
	}

	query := fmt.Sprintf(`
		SELECT id, job_id, batch_index, sku, error_code, message, raw_item, resolved, created_at
		FROM catalog_update_issues
		%s
		ORDER BY id DESC
	`, where)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	issues := make([]domain.CatalogIssue, 0)
	for rows.Next() {
		var iss domain.CatalogIssue
		var raw []byte
		var sku sql.NullString
		if err := rows.Scan(&iss.ID, &iss.JobID, &iss.BatchIndex, &sku, &iss.ErrorCode, &iss.Message, &raw, &iss.Resolved, &iss.CreatedAt); err != nil {
			return nil, err
		}
		if sku.Valid {
			iss.SKU = &sku.String
		}
		if raw != nil {
			iss.RawItem = raw
		}
		issues = append(issues, iss)
	}
	return issues, nil
}

// ResolveIssue marks a problem as resolved.
func (r *JobRepository) ResolveIssue(ctx context.Context, issueID int64) error {
	query := `UPDATE catalog_update_issues SET resolved = TRUE WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, issueID)
	return err
}

// MarkStaleRunningJobsAsInterrupted on startup finds any jobs that were left in RUNNING status and marks them INTERRUPTED.
func (r *JobRepository) MarkStaleRunningJobsAsInterrupted(ctx context.Context) (int64, error) {
	query := `
		UPDATE catalog_sync_jobs
		SET status = 'INTERRUPTED', interrupted_at = NOW(), error_summary = 'Application was restarted while update was in progress'
		WHERE status = 'RUNNING'
	`
	res, err := r.db.ExecContext(ctx, query)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}
