import React from 'react';
import { PackagePlus, Edit3, Archive, FastForward, AlertCircle, Layers } from 'lucide-react';

export default function MetricsCards({ job }) {
  const metrics = [
    {
      label: 'New Products Added',
      value: job?.created_count || 0,
      icon: <PackagePlus size={18} color="#10b981" />,
      color: '#10b981',
      description: 'First time catalog imports',
    },
    {
      label: 'Details Changed',
      value: job?.updated_count || 0,
      icon: <Edit3 size={18} color="#6366f1" />,
      color: '#6366f1',
      description: 'Price & description updates',
    },
    {
      label: 'Withdrawn From Sale',
      value: job?.withdrawn_count || 0,
      icon: <Archive size={18} color="#94a3b8" />,
      color: '#94a3b8',
      description: 'Discontinued / recalled items',
    },
    {
      label: 'Skipped (Out-of-Order / Dups)',
      value: job?.skipped_count || 0,
      icon: <FastForward size={18} color="#06b6d4" />,
      color: '#06b6d4',
      description: 'Stale packets & idempotent re-runs',
    },
    {
      label: 'Problems Needing Attention',
      value: job?.error_count || 0,
      icon: <AlertCircle size={18} color={job?.error_count > 0 ? '#ef4444' : '#64748b'} />,
      color: job?.error_count > 0 ? '#ef4444' : '#64748b',
      description: 'Validation & corrupt supplier data',
    },
  ];

  return (
    <div className="metrics-grid">
      {metrics.map((m, idx) => (
        <div key={idx} className="metric-card glass-panel">
          <div className="metric-label">
            <span>{m.label}</span>
            {m.icon}
          </div>
          <div className="metric-value" style={{ color: m.color }}>
            {m.value}
          </div>
          <div style={{ fontSize: '0.75rem', color: 'var(--text-muted)', marginTop: '0.25rem' }}>
            {m.description}
          </div>
        </div>
      ))}
    </div>
  );
}
