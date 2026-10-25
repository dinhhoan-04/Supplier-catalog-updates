const BASE_URL = '/api';

export const api = {
  async getSyncStatus() {
    const res = await fetch(`${BASE_URL}/sync/status`);
    if (!res.ok) throw new Error('Failed to fetch status');
    return res.json();
  },

  async startSync(scenario = 'standard') {
    const res = await fetch(`${BASE_URL}/sync/start`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ scenario }),
    });
    if (!res.ok) {
      const err = await res.json();
      throw new Error(err.error || 'Failed to start sync');
    }
    return res.json();
  },

  async resumeSync(jobId) {
    const res = await fetch(`${BASE_URL}/sync/resume`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ job_id: jobId }),
    });
    if (!res.ok) {
      const err = await res.json();
      throw new Error(err.error || 'Failed to resume sync');
    }
    return res.json();
  },

  async simulateInterrupt() {
    const res = await fetch(`${BASE_URL}/sync/simulate-interrupt`, {
      method: 'POST',
    });
    if (!res.ok) {
      const err = await res.json();
      throw new Error(err.error || 'Failed to simulate interrupt');
    }
    return res.json();
  },

  async getChanges(jobId, limit = 100, offset = 0) {
    const query = new URLSearchParams({ limit, offset });
    if (jobId) query.append('job_id', jobId);
    const res = await fetch(`${BASE_URL}/sync/changes?${query}`);
    if (!res.ok) throw new Error('Failed to fetch changes');
    return res.json();
  },

  async getIssues(jobId, resolved = null) {
    const query = new URLSearchParams();
    if (jobId) query.append('job_id', jobId);
    if (resolved !== null) query.append('resolved', resolved);
    const res = await fetch(`${BASE_URL}/sync/issues?${query}`);
    if (!res.ok) throw new Error('Failed to fetch issues');
    return res.json();
  },

  async resolveIssue(issueId) {
    const res = await fetch(`${BASE_URL}/sync/issues/${issueId}/resolve`, {
      method: 'POST',
    });
    if (!res.ok) throw new Error('Failed to resolve issue');
    return res.json();
  },

  async getProducts(status = 'ALL', search = '', limit = 50, offset = 0) {
    const query = new URLSearchParams({ limit, offset });
    if (status && status !== 'ALL') query.append('status', status);
    if (search) query.append('search', search);
    const res = await fetch(`${BASE_URL}/products?${query}`);
    if (!res.ok) throw new Error('Failed to fetch products');
    return res.json();
  },

  async resetCatalog() {
    const res = await fetch(`${BASE_URL}/catalog/reset`, {
      method: 'POST',
    });
    if (!res.ok) throw new Error('Failed to reset catalog');
    return res.json();
  },

  subscribeSSE(onEvent) {
    const eventSource = new EventSource(`${BASE_URL}/sync/events`);

    eventSource.addEventListener('ping', (e) => {
      // Keepalive
    });

    const eventNames = [
      'job_started',
      'batch_progress',
      'job_completed',
      'job_interrupted',
      'job_resumed',
      'job_failed',
      'catalog_reset',
    ];

    eventNames.forEach((name) => {
      eventSource.addEventListener(name, (e) => {
        try {
          const data = JSON.parse(e.data);
          onEvent(name, data);
        } catch (err) {
          console.error('Error parsing SSE payload', err);
        }
      });
    });

    return () => {
      eventSource.close();
    };
  },
};
