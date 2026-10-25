import React from 'react';
import { AlertCircle, CheckCircle2, FileCode, Check } from 'lucide-react';

export default function IssuesTab({ issues, onResolve }) {
  const unresolved = (issues || []).filter((i) => !i.resolved);
  const resolved = (issues || []).filter((i) => i.resolved);

  return (
    <div>
      <div style={{ marginBottom: '1.25rem' }}>
        <h3 style={{ fontSize: '1.1rem', fontWeight: 700, color: unresolved.length > 0 ? '#f87171' : 'var(--text-primary)' }}>
          Problems Still Needing Attention ({unresolved.length})
        </h3>
        <p style={{ color: 'var(--text-secondary)', fontSize: '0.85rem' }}>
          Corrupted batches, invalid prices, or schema errors flagged by the sync engine for staff review
        </p>
      </div>

      {issues?.length === 0 ? (
        <div style={{ textAlign: 'center', padding: '3.5rem 1rem', color: 'var(--text-muted)' }}>
          <CheckCircle2 size={36} color="#10b981" style={{ margin: '0 auto 0.75rem' }} />
          <p style={{ fontWeight: 600, color: 'var(--text-primary)' }}>All clear! No problems detected.</p>
          <p style={{ fontSize: '0.85rem' }}>All items in recent updates passed validation successfully.</p>
        </div>
      ) : (
        <div style={{ display: 'flex', flexDirection: 'column', gap: '1rem' }}>
          {issues.map((issue) => {
            let rawParsed = null;
            try {
              rawParsed = typeof issue.raw_item === 'string' ? JSON.parse(issue.raw_item) : issue.raw_item;
            } catch (e) {
              rawParsed = issue.raw_item;
            }

            return (
              <div
                key={issue.id}
                className="glass-panel"
                style={{
                  padding: '1.25rem',
                  borderLeft: `4px solid ${issue.resolved ? '#10b981' : '#ef4444'}`,
                  background: issue.resolved ? 'rgba(16, 185, 129, 0.04)' : 'var(--bg-card)',
                }}
              >
                <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', flexWrap: 'wrap', gap: '0.75rem' }}>
                  <div>
                    <div style={{ display: 'flex', alignItems: 'center', gap: '0.5rem', marginBottom: '0.35rem' }}>
                      <span className="badge badge-failed">{issue.error_code}</span>
                      <span className="code-badge">Batch #{issue.batch_index}</span>
                      {issue.sku ? (
                        <span className="code-badge" style={{ color: '#f8fafc' }}>SKU: {issue.sku}</span>
                      ) : (
                        <span className="code-badge" style={{ color: '#fbbf24' }}>No SKU</span>
                      )}
                      {issue.resolved && <span className="badge badge-completed">Resolved</span>}
                    </div>
                    <div style={{ fontSize: '0.95rem', fontWeight: 600, color: '#f8fafc' }}>
                      {issue.message}
                    </div>
                  </div>

                  {!issue.resolved && (
                    <button
                      className="btn btn-secondary"
                      style={{ fontSize: '0.75rem', padding: '0.4rem 0.8rem' }}
                      onClick={() => onResolve(issue.id)}
                    >
                      <Check size={14} color="#10b981" />
                      Mark Resolved
                    </button>
                  )}
                </div>

                {/* Raw Item Payload Inspector */}
                {rawParsed && (
                  <div style={{ marginTop: '0.85rem' }}>
                    <div style={{ fontSize: '0.75rem', color: 'var(--text-muted)', marginBottom: '0.35rem', display: 'flex', alignItems: 'center', gap: '0.35rem' }}>
                      <FileCode size={13} /> Raw Supplier Payload:
                    </div>
                    <pre
                      style={{
                        background: 'rgba(0, 0, 0, 0.45)',
                        padding: '0.65rem 0.85rem',
                        borderRadius: '6px',
                        fontSize: '0.78rem',
                        fontFamily: 'var(--font-mono)',
                        overflowX: 'auto',
                        border: '1px solid rgba(255, 255, 255, 0.05)',
                        color: '#94a3b8',
                      }}
                    >
                      {JSON.stringify(rawParsed, null, 2)}
                    </pre>
                  </div>
                )}
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
}
