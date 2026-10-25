import React, { useState } from 'react';
import { Eye, Filter, ArrowUpDown } from 'lucide-react';

export default function ChangesTab({ changes, onSelectDiff }) {
  const [filterType, setFilterType] = useState('ALL');

  const filtered = (changes || []).filter((item) => {
    if (filterType === 'ALL') return true;
    return item.change_type === filterType;
  });

  const getChangeBadge = (type) => {
    switch (type) {
      case 'CREATED':
        return <span className="badge badge-completed">New Product</span>;
      case 'UPDATED':
        return <span className="badge badge-running">Details Changed</span>;
      case 'WITHDRAWN':
        return <span className="badge badge-withdrawn">Withdrawn</span>;
      case 'SKIPPED_OUTDATED':
        return <span className="badge badge-interrupted">Out-Of-Order Skipped</span>;
      case 'SKIPPED_IDENTICAL':
        return <span className="badge badge-idle">Identical Skipped</span>;
      default:
        return <span className="badge">{type}</span>;
    }
  };

  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '1.25rem', flexWrap: 'wrap', gap: '0.75rem' }}>
        <div>
          <h3 style={{ fontSize: '1.1rem', fontWeight: 700 }}>What Changed (Audit Log)</h3>
          <p style={{ color: 'var(--text-secondary)', fontSize: '0.85rem' }}>
            Showing {filtered.length} recorded events from the latest update
          </p>
        </div>

        <div style={{ display: 'flex', alignItems: 'center', gap: '0.5rem' }}>
          <Filter size={16} color="var(--text-secondary)" />
          <select
            className="select-custom"
            value={filterType}
            onChange={(e) => setFilterType(e.target.value)}
          >
            <option value="ALL">All Changes ({changes?.length || 0})</option>
            <option value="CREATED">New Products</option>
            <option value="UPDATED">Updated Details</option>
            <option value="WITHDRAWN">Withdrawn Products</option>
            <option value="SKIPPED_OUTDATED">Out-of-Order (Protected)</option>
            <option value="SKIPPED_IDENTICAL">Identical (Deduplicated)</option>
          </select>
        </div>
      </div>

      {filtered.length === 0 ? (
        <div style={{ textAlign: 'center', padding: '3rem 1rem', color: 'var(--text-muted)' }}>
          No change records found matching this filter.
        </div>
      ) : (
        <div className="table-responsive">
          <table>
            <thead>
              <tr>
                <th>Batch</th>
                <th>SKU</th>
                <th>Action Taken</th>
                <th>Timestamp</th>
                <th style={{ textAlign: 'right' }}>Details</th>
              </tr>
            </thead>
            <tbody>
              {filtered.map((log) => (
                <tr key={log.id}>
                  <td>
                    <span className="code-badge">Batch #{log.batch_index}</span>
                  </td>
                  <td>
                    <span className="code-badge" style={{ fontWeight: 600, color: '#f8fafc' }}>
                      {log.sku}
                    </span>
                  </td>
                  <td>{getChangeBadge(log.change_type)}</td>
                  <td style={{ color: 'var(--text-muted)', fontSize: '0.8rem' }}>
                    {new Date(log.created_at).toLocaleTimeString()}
                  </td>
                  <td style={{ textAlign: 'right' }}>
                    <button
                      className="btn btn-secondary"
                      style={{ padding: '0.35rem 0.75rem', fontSize: '0.75rem' }}
                      onClick={() => onSelectDiff(log)}
                    >
                      <Eye size={14} />
                      View Diff
                    </button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}
