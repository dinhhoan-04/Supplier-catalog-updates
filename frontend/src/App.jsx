import React, { useState, useEffect, useCallback } from 'react';
import Header from './components/Header';
import SyncControl from './components/SyncControl';
import MetricsCards from './components/MetricsCards';
import ChangesTab from './components/ChangesTab';
import IssuesTab from './components/IssuesTab';
import CatalogTab from './components/CatalogTab';
import DiffModal from './components/DiffModal';
import { api } from './services/api';
import { Layers, AlertCircle, ShoppingBag, History } from 'lucide-react';

export default function App() {
  const [job, setJob] = useState(null);
  const [changes, setChanges] = useState([]);
  const [issues, setIssues] = useState([]);
  const [activeTab, setActiveTab] = useState('changes');
  const [selectedDiff, setSelectedDiff] = useState(null);
  const [loading, setLoading] = useState(false);
  const [isResetting, setIsResetting] = useState(false);
  const [notification, setNotification] = useState(null);

  const showNotification = (message, type = 'info') => {
    setNotification({ message, type });
    setTimeout(() => setNotification(null), 5000);
  };

  const fetchStatusAndData = useCallback(async () => {
    try {
      const statusData = await api.getSyncStatus();
      if (statusData && statusData.id) {
        setJob(statusData);

        const [changesRes, issuesRes] = await Promise.all([
          api.getChanges(statusData.id),
          api.getIssues(statusData.id),
        ]);
        setChanges(changesRes.items || []);
        setIssues(issuesRes.items || []);
      } else {
        setJob(null);
        setChanges([]);
        setIssues([]);
      }
    } catch (err) {
      console.error('Failed to load status:', err);
    }
  }, []);

  useEffect(() => {
    fetchStatusAndData();

    // Subscribe to real-time Server-Sent Events
    const unsubscribe = api.subscribeSSE((eventType, data) => {
      console.log('SSE Event:', eventType, data);

      if (eventType === 'job_started' || eventType === 'job_resumed') {
        setJob(data);
        showNotification(`Sync update ${eventType === 'job_resumed' ? 'resumed' : 'started'}...`, 'info');
      } else if (eventType === 'batch_progress') {
        setJob((prev) => ({
          ...prev,
          processed_batches: data.processed_batches,
          total_batches: data.total_batches,
          created_count: data.created_count,
          updated_count: data.updated_count,
          withdrawn_count: data.withdrawn_count,
          skipped_count: data.skipped_count,
          error_count: data.error_count,
        }));
        // Refresh changelog & issues as batches commit
        if (data.job_id) {
          api.getChanges(data.job_id).then((res) => setChanges(res.items || []));
          api.getIssues(data.job_id).then((res) => setIssues(res.items || []));
        }
      } else if (eventType === 'job_completed') {
        setJob(data);
        showNotification('Update completed successfully!', 'success');
        if (data.id) {
          api.getChanges(data.id).then((res) => setChanges(res.items || []));
          api.getIssues(data.id).then((res) => setIssues(res.items || []));
        }
      } else if (eventType === 'job_interrupted') {
        setJob(data);
        showNotification('Update was interrupted! You can resume from the last committed checkpoint.', 'warning');
      } else if (eventType === 'job_failed') {
        setJob(data);
        showNotification(`Update failed: ${data.error_summary}`, 'danger');
      } else if (eventType === 'catalog_reset') {
        setJob(null);
        setChanges([]);
        setIssues([]);
        showNotification('Local catalog reset clean.', 'info');
      }
    });

    return () => {
      unsubscribe();
    };
  }, [fetchStatusAndData]);

  const handleStartSync = async (scenario) => {
    setLoading(true);
    try {
      const res = await api.startSync(scenario);
      setJob(res);
      showNotification('Catalog update pipeline started', 'info');
    } catch (err) {
      showNotification(err.message, 'danger');
    } finally {
      setLoading(false);
    }
  };

  const handleResumeSync = async () => {
    if (!job?.id) return;
    setLoading(true);
    try {
      const res = await api.resumeSync(job.id);
      setJob(res);
      showNotification('Resuming update from last checkpoint...', 'warning');
    } catch (err) {
      showNotification(err.message, 'danger');
    } finally {
      setLoading(false);
    }
  };

  const handleInterrupt = async () => {
    try {
      await api.simulateInterrupt();
      showNotification('Simulating outage: interrupting active update...', 'warning');
    } catch (err) {
      showNotification(err.message, 'danger');
    }
  };

  const handleResetCatalog = async () => {
    if (!confirm('Are you sure you want to clear the local catalog database?')) return;
    setIsResetting(true);
    try {
      await api.resetCatalog();
      setJob(null);
      setChanges([]);
      setIssues([]);
    } catch (err) {
      showNotification(err.message, 'danger');
    } finally {
      setIsResetting(false);
    }
  };

  const handleResolveIssue = async (issueId) => {
    try {
      await api.resolveIssue(issueId);
      setIssues((prev) =>
        prev.map((i) => (i.id === issueId ? { ...i, resolved: true } : i))
      );
      showNotification('Problem marked as resolved', 'success');
    } catch (err) {
      showNotification(err.message, 'danger');
    }
  };

  const unresolvedCount = issues.filter((i) => !i.resolved).length;

  return (
    <div className="container">
      {/* Toast Notification */}
      {notification && (
        <div
          style={{
            position: 'fixed',
            bottom: '2rem',
            right: '2rem',
            padding: '0.85rem 1.25rem',
            borderRadius: '8px',
            background:
              notification.type === 'success'
                ? '#065f46'
                : notification.type === 'warning'
                ? '#92400e'
                : notification.type === 'danger'
                ? '#991b1b'
                : '#312e81',
            color: '#ffffff',
            boxShadow: 'var(--shadow-md)',
            zIndex: 1000,
            fontSize: '0.875rem',
            fontWeight: 600,
            display: 'flex',
            alignItems: 'center',
            gap: '0.5rem',
          }}
        >
          {notification.message}
        </div>
      )}

      {/* Header */}
      <Header
        status={job}
        onReset={handleResetCatalog}
        isResetting={isResetting}
      />

      {/* Control Panel */}
      <SyncControl
        job={job}
        onStart={handleStartSync}
        onResume={handleResumeSync}
        onInterrupt={handleInterrupt}
        loading={loading}
      />

      {/* Metrics Cards */}
      <MetricsCards job={job} />

      {/* Main Tabs Navigation */}
      <div className="glass-panel" style={{ padding: '1.5rem' }}>
        <div className="tabs-nav">
          <button
            className={`tab-btn ${activeTab === 'changes' ? 'active' : ''}`}
            onClick={() => setActiveTab('changes')}
          >
            <History size={16} />
            What Changed
            <span className="tab-count">{changes.length}</span>
          </button>

          <button
            className={`tab-btn ${activeTab === 'issues' ? 'active' : ''}`}
            onClick={() => setActiveTab('issues')}
          >
            <AlertCircle size={16} color={unresolvedCount > 0 ? '#f87171' : undefined} />
            Problems Needing Attention
            {unresolvedCount > 0 && (
              <span className="tab-count" style={{ background: '#ef4444', color: 'white' }}>
                {unresolvedCount}
              </span>
            )}
          </button>

          <button
            className={`tab-btn ${activeTab === 'catalog' ? 'active' : ''}`}
            onClick={() => setActiveTab('catalog')}
          >
            <ShoppingBag size={16} />
            Current Catalog
          </button>
        </div>

        {/* Tab Contents */}
        {activeTab === 'changes' && (
          <ChangesTab changes={changes} onSelectDiff={setSelectedDiff} />
        )}

        {activeTab === 'issues' && (
          <IssuesTab issues={issues} onResolve={handleResolveIssue} />
        )}

        {activeTab === 'catalog' && (
          <CatalogTab onFetchProducts={api.getProducts} />
        )}
      </div>

      {/* Diff Inspector Modal */}
      {selectedDiff && (
        <DiffModal log={selectedDiff} onClose={() => setSelectedDiff(null)} />
      )}
    </div>
  );
}
