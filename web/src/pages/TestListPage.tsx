import { useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { fetchTests, createTest, runTest, stopTest } from '../api/client.ts';
import type { TestDefinition, TestRun } from '../api/types.ts';
import StatusBadge from '../components/StatusBadge.tsx';

interface CreateFormState {
  name: string;
  scenario_yaml: string;
}

interface RunFormState {
  testId: string;
  vusers: number;
  duration: number;
}

export default function TestListPage() {
  const navigate = useNavigate();
  const [tests, setTests] = useState<TestDefinition[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const [showCreate, setShowCreate] = useState(false);
  const [createForm, setCreateForm] = useState<CreateFormState>({ name: '', scenario_yaml: '' });
  const [creating, setCreating] = useState(false);

  const [runForm, setRunForm] = useState<RunFormState | null>(null);
  const [activeRuns, setActiveRuns] = useState<Record<string, TestRun>>({});

  useEffect(() => {
    loadTests();
  }, []);

  async function loadTests() {
    try {
      setLoading(true);
      const data = await fetchTests();
      setTests(data || []);
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load tests');
    } finally {
      setLoading(false);
    }
  }

  async function handleCreate() {
    if (!createForm.name.trim()) return;
    try {
      setCreating(true);
      await createTest(createForm.name, createForm.scenario_yaml);
      setCreateForm({ name: '', scenario_yaml: '' });
      setShowCreate(false);
      await loadTests();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to create test');
    } finally {
      setCreating(false);
    }
  }

  async function handleRun() {
    if (!runForm) return;
    try {
      const run = await runTest(runForm.testId, runForm.vusers, runForm.duration);
      setActiveRuns((prev) => ({ ...prev, [runForm.testId]: run }));
      setRunForm(null);
      navigate(`/runs/${run.id}`);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to start test run');
    }
  }

  async function handleStop(testId: string) {
    const run = activeRuns[testId];
    if (!run) return;
    try {
      await stopTest(run.id);
      setActiveRuns((prev) => {
        const next = { ...prev };
        delete next[testId];
        return next;
      });
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to stop test');
    }
  }

  return (
    <div className="page">
      <div className="page-header">
        <h1>Tests</h1>
        <button className="btn btn-primary" onClick={() => setShowCreate(true)}>
          + Create Test
        </button>
      </div>

      {error && (
        <div className="alert alert-error">
          {error}
          <button className="alert-dismiss" onClick={() => setError(null)}>x</button>
        </div>
      )}

      {showCreate && (
        <div className="card create-form">
          <h3>New Test</h3>
          <div className="form-group">
            <label>Name</label>
            <input
              type="text"
              value={createForm.name}
              onChange={(e) => setCreateForm((f) => ({ ...f, name: e.target.value }))}
              placeholder="e.g. API Load Test"
            />
          </div>
          <div className="form-group">
            <label>Scenario YAML</label>
            <textarea
              rows={8}
              value={createForm.scenario_yaml}
              onChange={(e) => setCreateForm((f) => ({ ...f, scenario_yaml: e.target.value }))}
              placeholder="scenarios:&#10;  - name: GET /api/health&#10;    method: GET&#10;    url: http://target:8080/api/health"
            />
          </div>
          <div className="form-actions">
            <button className="btn btn-primary" onClick={handleCreate} disabled={creating}>
              {creating ? 'Creating...' : 'Create'}
            </button>
            <button className="btn btn-secondary" onClick={() => setShowCreate(false)}>
              Cancel
            </button>
          </div>
        </div>
      )}

      {runForm && (
        <div className="card create-form">
          <h3>Run Test</h3>
          <div className="form-group">
            <label>Virtual Users</label>
            <input
              type="number"
              min={1}
              value={runForm.vusers}
              onChange={(e) => setRunForm((f) => f ? { ...f, vusers: Number(e.target.value) } : f)}
            />
          </div>
          <div className="form-group">
            <label>Duration (seconds)</label>
            <input
              type="number"
              min={1}
              value={runForm.duration}
              onChange={(e) => setRunForm((f) => f ? { ...f, duration: Number(e.target.value) } : f)}
            />
          </div>
          <div className="form-actions">
            <button className="btn btn-primary" onClick={handleRun}>Start</button>
            <button className="btn btn-secondary" onClick={() => setRunForm(null)}>Cancel</button>
          </div>
        </div>
      )}

      {loading ? (
        <div className="loading">Loading tests...</div>
      ) : tests.length === 0 ? (
        <div className="empty-state">
          <p>No tests yet. Create your first test to get started.</p>
        </div>
      ) : (
        <div className="card">
          <table className="table">
            <thead>
              <tr>
                <th>Name</th>
                <th>ID</th>
                <th>Created</th>
                <th>Status</th>
                <th>Actions</th>
              </tr>
            </thead>
            <tbody>
              {tests.map((test) => {
                const isRunning = !!activeRuns[test.id];
                return (
                  <tr key={test.id}>
                    <td className="text-bright">{test.name}</td>
                    <td className="text-mono">{test.id.substring(0, 8)}</td>
                    <td>{new Date(test.created_at).toLocaleDateString()}</td>
                    <td>
                      <StatusBadge status={isRunning ? 'running' : 'idle'} />
                    </td>
                    <td className="actions-cell">
                      {isRunning ? (
                        <>
                          <button
                            className="btn btn-sm btn-danger"
                            onClick={() => handleStop(test.id)}
                          >
                            Stop
                          </button>
                          <button
                            className="btn btn-sm btn-secondary"
                            onClick={() => navigate(`/runs/${activeRuns[test.id].id}`)}
                          >
                            View
                          </button>
                        </>
                      ) : (
                        <button
                          className="btn btn-sm btn-primary"
                          onClick={() => setRunForm({ testId: test.id, vusers: 10, duration: 60 })}
                        >
                          Run
                        </button>
                      )}
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}
