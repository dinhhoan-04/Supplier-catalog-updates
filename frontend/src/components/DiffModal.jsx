import React from 'react';
import { X, ArrowRight } from 'lucide-react';

export default function DiffModal({ log, onClose }) {
  if (!log) return null;

  let diffData = null;
  try {
    diffData = typeof log.diff_data === 'string' ? JSON.parse(log.diff_data) : log.diff_data;
  } catch (e) {
    diffData = log.diff_data;
  }

  const formatValue = (field, val) => {
    if (val === null || val === undefined) return <span style={{ color: 'var(--text-muted)' }}>none</span>;
    if (field === 'price_cents') {
      return `$${(val / 100).toFixed(2)}`;
    }
    return String(val);
  };

  return (
    <div className="modal-overlay" onClick={onClose}>
      <div className="modal-content" onClick={(e) => e.stopPropagation()}>
        <div className="modal-header">
          <div>
            <h3 style={{ fontSize: '1.1rem', fontWeight: 700 }}>Item Change Details</h3>
            <p style={{ color: 'var(--text-secondary)', fontSize: '0.8rem', marginTop: '0.2rem' }}>
              SKU: <span className="code-badge">{log.sku}</span> • Type: <strong style={{ color: '#818cf8' }}>{log.change_type}</strong>
            </p>
          </div>
          <button
            onClick={onClose}
            style={{ background: 'transparent', border: 'none', color: 'var(--text-secondary)', cursor: 'pointer' }}
          >
            <X size={20} />
          </button>
        </div>

        <div className="modal-body">
          {diffData?.reason ? (
            <div style={{ background: 'rgba(255, 255, 255, 0.04)', padding: '1rem', borderRadius: '8px' }}>
              <div style={{ fontSize: '0.8rem', color: 'var(--text-muted)', marginBottom: '0.25rem' }}>RESOLUTION NOTE:</div>
              <p style={{ color: 'var(--text-primary)', fontSize: '0.9rem' }}>{diffData.reason}</p>
            </div>
          ) : diffData && Object.keys(diffData).length > 0 ? (
            <div>
              {Object.entries(diffData).map(([fieldName, change]) => (
                <div key={fieldName} className="diff-row">
                  <span className="diff-field-name">{fieldName.replace('_', ' ')}</span>
                  <div className="diff-values">
                    <span className="diff-old">{formatValue(fieldName, change?.old)}</span>
                    <ArrowRight size={14} color="var(--text-muted)" />
                    <span className="diff-new">{formatValue(fieldName, change?.new)}</span>
                  </div>
                </div>
              ))}
            </div>
          ) : (
            <p style={{ color: 'var(--text-muted)', fontSize: '0.875rem' }}>No field diff available for this change.</p>
          )}

          <div style={{ marginTop: '1.5rem', textAlign: 'right' }}>
            <button className="btn btn-secondary" onClick={onClose}>
              Close
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}
