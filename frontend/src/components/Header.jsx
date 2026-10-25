import React from 'react';
import { RefreshCw, Store, AlertTriangle, Play, RotateCcw } from 'lucide-react';

export default function Header({ status, onReset, isResetting }) {
  const getStatusBadge = () => {
    switch (status?.status) {
      case 'RUNNING':
        return (
          <span className="badge badge-running">
            <span style={{ width: 8, height: 8, borderRadius: '50%', background: '#6366f1', display: 'inline-block' }}></span>
            Sync In Progress
          </span>
        );
      case 'COMPLETED':
        return <span className="badge badge-completed">✓ Sync Completed</span>;
      case 'INTERRUPTED':
        return (
          <span className="badge badge-interrupted">
            <AlertTriangle size={14} /> Update Interrupted (Ready to Resume)
          </span>
        );
      case 'FAILED':
        return <span className="badge badge-failed">✕ Sync Failed</span>;
      default:
        return <span className="badge badge-idle">Ready / Idle</span>;
    }
  };

  return (
    <header className="app-header">
      <div className="header-title-group">
        <h1>
          <Store className="text-indigo-400" size={32} />
          Supplier Catalog Sync Center
        </h1>
        <p>Reliable, fault-tolerant batch synchronization with out-of-order & tombstone protection</p>
      </div>

      <div style={{ display: 'flex', alignItems: 'center', gap: '1rem' }}>
        {getStatusBadge()}

        <button
          className="btn btn-secondary"
          onClick={onReset}
          disabled={isResetting || status?.status === 'RUNNING'}
          title="Clear local database to re-run demo scenarios from scratch"
        >
          <RotateCcw size={16} className={isResetting ? 'animate-spin' : ''} />
          Reset Demo
        </button>
      </div>
    </header>
  );
}
