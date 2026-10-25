package service

import (
	"catalog-sync/internal/domain"
	"catalog-sync/internal/handler"
	"catalog-sync/internal/mocksupplier"
	"catalog-sync/internal/repository/postgres"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

var (
	ErrSyncAlreadyRunning   = errors.New("a sync job is already in progress")
	ErrJobNotInterrupted    = errors.New("only interrupted jobs can be resumed")
	ErrJobNotFound          = errors.New("job not found")
)

type SyncEngine struct {
	db               *postgres.DB
	productRepo      *postgres.ProductRepository
	jobRepo          *postgres.JobRepository
	resolver         *ConflictResolver
	supplier         *mocksupplier.MockSupplier
	sseHub           *handler.SSEHub
	activeCancelFunc context.CancelFunc
	activeJobID      string
	mu               sync.Mutex
}

func NewSyncEngine(
	db *postgres.DB,
	productRepo *postgres.ProductRepository,
	jobRepo *postgres.JobRepository,
	resolver *ConflictResolver,
	supplier *mocksupplier.MockSupplier,
	sseHub *handler.SSEHub,
) *SyncEngine {
	return &SyncEngine{
		db:          db,
		productRepo: productRepo,
		jobRepo:     jobRepo,
		resolver:    resolver,
		supplier:    supplier,
		sseHub:      sseHub,
	}
}

// StartNewSync creates and kicks off a new sync job.
func (e *SyncEngine) StartNewSync(scenario string) (*domain.SyncJob, error) {
	e.mu.Lock()
	if e.activeCancelFunc != nil {
		e.mu.Unlock()
		return nil, ErrSyncAlreadyRunning
	}

	batches := e.supplier.GetBatches(scenario)
	totalItems := 0
	for _, b := range batches {
		totalItems += len(b.Items)
	}

	job := &domain.SyncJob{
		ID:               uuid.NewString(),
		Status:           domain.JobStatusRunning,
		Scenario:         scenario,
		TotalBatches:     len(batches),
		ProcessedBatches: 0,
		TotalItems:       totalItems,
		StartedAt:        time.Now().UTC(),
	}

	ctx, cancel := context.WithCancel(context.Background())
	e.activeCancelFunc = cancel
	e.activeJobID = job.ID
	e.mu.Unlock()

	if err := e.jobRepo.CreateJob(ctx, job); err != nil {
		e.clearActive()
		return nil, fmt.Errorf("failed to create job: %w", err)
	}

	e.sseHub.Broadcast("job_started", job)

	go e.runJob(ctx, job, batches)

	return job, nil
}

// ResumeSync continues an interrupted sync job from its last completed checkpoint.
func (e *SyncEngine) ResumeSync(jobID string) (*domain.SyncJob, error) {
	e.mu.Lock()
	if e.activeCancelFunc != nil {
		e.mu.Unlock()
		return nil, ErrSyncAlreadyRunning
	}

	job, err := e.jobRepo.GetJobByID(context.Background(), jobID)
	if err != nil {
		e.mu.Unlock()
		return nil, err
	}
	if job == nil {
		e.mu.Unlock()
		return nil, ErrJobNotFound
	}
	if job.Status != domain.JobStatusInterrupted {
		e.mu.Unlock()
		return nil, ErrJobNotInterrupted
	}

	job.Status = domain.JobStatusRunning
	job.InterruptedAt = nil
	job.ErrorSummary = nil

	ctx, cancel := context.WithCancel(context.Background())
	e.activeCancelFunc = cancel
	e.activeJobID = job.ID
	e.mu.Unlock()

	if err := e.jobRepo.UpdateJobStatus(ctx, job); err != nil {
		e.clearActive()
		return nil, fmt.Errorf("failed to resume job: %w", err)
	}

	batches := e.supplier.GetBatches(job.Scenario)
	e.sseHub.Broadcast("job_resumed", job)

	go e.runJob(ctx, job, batches)

	return job, nil
}

// SimulateInterruption cancels the active sync mid-stream to verify crash recovery.
func (e *SyncEngine) SimulateInterruption() error {
	e.mu.Lock()
	cancel := e.activeCancelFunc
	e.mu.Unlock()

	if cancel == nil {
		return errors.New("no active sync job is currently running to interrupt")
	}

	cancel() // Triggers context cancellation in worker loop
	return nil
}

func (e *SyncEngine) clearActive() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.activeCancelFunc = nil
	e.activeJobID = ""
}

func (e *SyncEngine) runJob(ctx context.Context, job *domain.SyncJob, batches []domain.SupplierBatch) {
	defer e.clearActive()

	// 1. Fetch checkpoints to know which batches were already committed
	checkpoints, err := e.jobRepo.GetProcessedBatchIndices(context.Background(), job.ID)
	if err != nil {
		e.failJob(job, fmt.Sprintf("failed to read checkpoints: %v", err))
		return
	}

	// 2. Iterate batches sequentially
	for _, batch := range batches {
		// Check for early interruption / cancellation
		select {
		case <-ctx.Done():
			e.interruptJob(job, "Job was interrupted by staff or application shutdown")
			return
		default:
		}

		// Skip if already processed in a previous attempt (Checkpointing)
		if checkpoints[batch.BatchIndex] {
			continue
		}

		// Small delay (350ms) to make batch progression visible in the UI
		time.Sleep(350 * time.Millisecond)

		// Process batch inside a dedicated database transaction
		if err := e.processBatch(ctx, job, batch); err != nil {
			if errors.Is(err, context.Canceled) {
				e.interruptJob(job, "Job was interrupted during batch processing")
				return
			}
			e.failJob(job, fmt.Sprintf("Error in batch %d: %v", batch.BatchIndex, err))
			return
		}

		// Update job progress
		job.ProcessedBatches++
		if err := e.jobRepo.UpdateJobStatus(context.Background(), job); err != nil {
			fmt.Printf("Warning: failed to update job status: %v\n", err)
		}

		// Broadcast batch completion event
		e.sseHub.Broadcast("batch_progress", map[string]interface{}{
			"job_id":            job.ID,
			"processed_batches": job.ProcessedBatches,
			"total_batches":     job.TotalBatches,
			"created_count":     job.CreatedCount,
			"updated_count":     job.UpdatedCount,
			"withdrawn_count":   job.WithdrawnCount,
			"skipped_count":     job.SkippedCount,
			"error_count":       job.ErrorCount,
		})
	}

	// 3. Mark job as COMPLETED
	now := time.Now().UTC()
	job.Status = domain.JobStatusCompleted
	job.FinishedAt = &now
	_ = e.jobRepo.UpdateJobStatus(context.Background(), job)

	e.sseHub.Broadcast("job_completed", job)
}

func (e *SyncEngine) processBatch(ctx context.Context, job *domain.SyncJob, batch domain.SupplierBatch) error {
	tx, err := e.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	now := time.Now().UTC()

	for _, item := range batch.Items {
		// Check cancellation
		if ctx.Err() != nil {
			return ctx.Err()
		}

		// 1. Fetch current product from local DB (if exists)
		var existing *domain.Product
		if item.SKU != "" {
			existing, err = e.productRepo.GetBySKU(ctx, tx, item.SKU)
			if err != nil {
				return fmt.Errorf("failed to query product for SKU %s: %w", item.SKU, err)
			}
		}

		// 2. Resolve conflict, versioning, out-of-order, and tombstone rules
		resolution := e.resolver.Resolve(item, existing, now)

		// 3. Execute resolution action
		switch resolution.Action {
		case ActionCreate:
			if err := e.productRepo.Create(ctx, tx, resolution.TargetProduct); err != nil {
				return fmt.Errorf("failed to create product %s: %w", item.SKU, err)
			}
			job.CreatedCount++
			_ = e.jobRepo.RecordChangeLog(ctx, tx, &domain.ChangeLog{
				JobID:      job.ID,
				BatchIndex: batch.BatchIndex,
				SKU:        item.SKU,
				ChangeType: domain.ChangeTypeCreated,
				DiffData:   resolution.DiffJSON,
				CreatedAt:  now,
			})

		case ActionUpdate:
			if err := e.productRepo.Update(ctx, tx, resolution.TargetProduct); err != nil {
				return fmt.Errorf("failed to update product %s: %w", item.SKU, err)
			}
			job.UpdatedCount++
			_ = e.jobRepo.RecordChangeLog(ctx, tx, &domain.ChangeLog{
				JobID:      job.ID,
				BatchIndex: batch.BatchIndex,
				SKU:        item.SKU,
				ChangeType: domain.ChangeTypeUpdated,
				DiffData:   resolution.DiffJSON,
				CreatedAt:  now,
			})

		case ActionWithdraw:
			if err := e.productRepo.Update(ctx, tx, resolution.TargetProduct); err != nil {
				return fmt.Errorf("failed to withdraw product %s: %w", item.SKU, err)
			}
			job.WithdrawnCount++
			_ = e.jobRepo.RecordChangeLog(ctx, tx, &domain.ChangeLog{
				JobID:      job.ID,
				BatchIndex: batch.BatchIndex,
				SKU:        item.SKU,
				ChangeType: domain.ChangeTypeWithdrawn,
				DiffData:   resolution.DiffJSON,
				CreatedAt:  now,
			})

		case ActionSkipOutdated:
			job.SkippedCount++
			_ = e.jobRepo.RecordChangeLog(ctx, tx, &domain.ChangeLog{
				JobID:      job.ID,
				BatchIndex: batch.BatchIndex,
				SKU:        item.SKU,
				ChangeType: domain.ChangeTypeSkippedOutdated,
				DiffData:   []byte(fmt.Sprintf(`{"reason": %q}`, resolution.SkipReason)),
				CreatedAt:  now,
			})

		case ActionSkipIdentical:
			job.SkippedCount++
			_ = e.jobRepo.RecordChangeLog(ctx, tx, &domain.ChangeLog{
				JobID:      job.ID,
				BatchIndex: batch.BatchIndex,
				SKU:        item.SKU,
				ChangeType: domain.ChangeTypeSkippedIdentical,
				DiffData:   []byte(fmt.Sprintf(`{"reason": %q}`, resolution.SkipReason)),
				CreatedAt:  now,
			})

		case ActionRejectConflict:
			// "problems still need attention"
			job.ErrorCount++
			rawJSON, _ := json.Marshal(item)
			var skuPtr *string
			if item.SKU != "" {
				s := item.SKU
				skuPtr = &s
			}
			errCode := domain.ErrCodeCorruptData
			if errors.Is(resolution.ValidationError, ErrEmptySKU) {
				errCode = domain.ErrCodeMissingSKU
			} else if errors.Is(resolution.ValidationError, ErrInvalidPrice) {
				errCode = domain.ErrCodeInvalidPrice
			}

			_ = e.jobRepo.RecordIssue(ctx, tx, &domain.CatalogIssue{
				JobID:      job.ID,
				BatchIndex: batch.BatchIndex,
				SKU:        skuPtr,
				ErrorCode:  errCode,
				Message:    resolution.ValidationError.Error(),
				RawItem:    rawJSON,
				Resolved:   false,
				CreatedAt:  now,
			})
		}
	}

	// 4. Record Checkpoint for this batch
	if err := e.jobRepo.AddCheckpoint(ctx, tx, job.ID, batch.BatchIndex, len(batch.Items)); err != nil {
		return fmt.Errorf("failed to save batch checkpoint: %w", err)
	}

	// 5. Commit batch transaction
	return tx.Commit()
}

func (e *SyncEngine) interruptJob(job *domain.SyncJob, reason string) {
	now := time.Now().UTC()
	job.Status = domain.JobStatusInterrupted
	job.InterruptedAt = &now
	job.ErrorSummary = &reason
	_ = e.jobRepo.UpdateJobStatus(context.Background(), job)
	e.sseHub.Broadcast("job_interrupted", job)
}

func (e *SyncEngine) failJob(job *domain.SyncJob, reason string) {
	now := time.Now().UTC()
	job.Status = domain.JobStatusFailed
	job.FinishedAt = &now
	job.ErrorSummary = &reason
	_ = e.jobRepo.UpdateJobStatus(context.Background(), job)
	e.sseHub.Broadcast("job_failed", job)
}
