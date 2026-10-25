import React from 'react';
import { Play, RotateCw, PauseCircle, CheckCircle2, AlertOctagon } from 'lucide-react';

export default function SyncControl({ job, onStart, onResume, onInterrupt, loading }) {
  const isRunning = job?.status === 'RUNNING';
  const isInterrupted = job?.status === 'INTERRUPTED';

  const totalBatches = job?.total_batches || 0;
  const processedBatches = job?.processed_batches || 0;
  const progressPercent = totalBatches > 0 ? Math.min(100, Math.round((processedBatches / totalBatches) * 100)) : 0;

  return (
    <div className="glass-panel" style={{ padding: '1.5rem', marginBottom: '2rem' }}>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', flexWrap: 'wrap', gap: '1rem' }}>
        <div>
          <h2 style={{ fontSize: '1.15rem', fontWeight: 700, marginBottom: '0.25rem' }}>Catalog Sync Pipeline</h2>
          <p style={{ color: 'var(--text-secondary)', fontSize: '0.85rem' }}>
            {isRunning && `Synchronizing batch ${processedBatches} of ${totalBatches}...`}
            {isInterrupted && `Update paused at batch ${processedBatches}/${totalBatches}. Staff can resume without losing completed work.`}
            {job?.status === 'COMPLETED' && `Update finished. All ${totalBatches} batches processed successfully.`}
            {(!job || job.status === 'IDLE') && 'Ready to trigger supplier catalog update.'}
          </p>
        </div>

        <div style={{ display: 'flex', gap: '0.75rem', flexWrap: 'wrap' }}>
          {isInterrupted && (
            <button
              className="btn btn-warning"
              onClick={onResume}
              disabled={loading}
              title="Resume update from last committed checkpoint"
            >
              <RotateCw size={16} />
              Resume Interrupted Sync
            </button>
          )}

          {!isRunning ? (
            <button
              className="btn btn-primary"
              onClick={() => onStart('standard')}
              disabled={loading}
            >
              <Play size={16} fill="currentColor" />
              {job?.status === 'COMPLETED' ? 'Run Sync Again (Test Idempotency)' : 'Start Update'}
            </button>
          ) : (
            <button
              className="btn btn-danger"
              onClick={onInterrupt}
              disabled={loading}
              title="Simulate server crash or network drop mid-batch"
            >
              <PauseCircle size={16} />
              Simulate Outage / Crash
            </button>
          )}
        </div>
      </div>

      {/* Progress Bar */}
      {(isRunning || isInterrupted || job?.status === 'COMPLETED') && (
        <div className="progress-bar-container" style={{ marginTop: '1.25rem' }}>
          <div style={{ display: 'flex', justifyContent: 'space-between', fontSize: '0.8rem', color: 'var(--text-secondary)' }}>
            <span>
              <strong>Batches Committed:</strong> {processedBatches} / {totalBatches}
            </span>
            <span>
              <strong>{progressPercent}% Complete</strong>
            </span>
          </div>

          <div className="progress-track">
            <div
              className="progress-fill"
              style={{
                width: `${progressPercent}%`,
                background: isInterrupted
                  ? 'linear-gradient(90deg, #f59e0b, #d97706)'
                  : undefined,
              }}
            ></div>
          </div>

          {job?.error_summary && isInterrupted && (
            <div style={{ marginTop: '0.75rem', fontSize: '0.8rem', color: '#fbbf24', display: 'flex', alignItems: 'center', gap: '0.5rem' }}>
              <AlertOctagon size={14} />
              <span>{job.error_summary}</span>
            </div>
          )}
        </div>
      )}
    </div>
  );
}
