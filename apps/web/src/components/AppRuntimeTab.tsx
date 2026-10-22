'use client';

import React, { useState, useEffect, useCallback } from 'react';
import {
  Play,
  Square,
  RotateCcw,
  Cpu,
  RefreshCw,
  Terminal,
  Activity,
  Server,
  Layers,
  Settings,
  Plus,
  Trash2,
  CheckCircle2,
  AlertTriangle,
  Globe,
  HardDrive,
  FileCode,
  Zap,
} from 'lucide-react';
import {
  Website,
  ManagedAppRecord,
  AppProcessStatus,
  RuntimeInfo,
  getManagedApp,
  saveManagedApp,
  controlManagedApp,
  getManagedAppStatus,
  deleteManagedApp,
  detectRuntimes,
} from '@/lib/api';

interface AppRuntimeTabProps {
  website: Website;
  showToast: (msg: string) => void;
}

export function AppRuntimeTab({ website, showToast }: AppRuntimeTabProps) {
  const [app, setApp] = useState<ManagedAppRecord | null>(null);
  const [status, setStatus] = useState<AppProcessStatus | null>(null);
  const [runtimes, setRuntimes] = useState<RuntimeInfo[]>([]);
  const [loading, setLoading] = useState(true);
  const [submitting, setSubmitting] = useState(false);
  const [controlling, setControlling] = useState<string | null>(null);

  // Form state
  const [name, setName] = useState('');
  const [appType, setAppType] = useState<'nodejs' | 'python'>('nodejs');
  const [runtimeVersion, setRuntimeVersion] = useState('20');
  const [entrypoint, setEntrypoint] = useState('index.js');
  const [startCommand, setStartCommand] = useState('');
  const [port, setPort] = useState<number>(3000);
  const [envVars, setEnvVars] = useState<{ key: string; value: string }[]>([]);

  // Logs
  const [logTail, setLogTail] = useState('');
  const [autoRefreshLogs, setAutoRefreshLogs] = useState(false);

  const fetchData = useCallback(async () => {
    setLoading(true);
    try {
      const [appRes, runtimesRes] = await Promise.all([
        getManagedApp(website.id),
        detectRuntimes(),
      ]);

      if (runtimesRes.success && runtimesRes.data) {
        setRuntimes(runtimesRes.data);
      }

      if (appRes.success && appRes.data) {
        const a = appRes.data;
        setApp(a);
        setName(a.name || website.primary_domain);
        setAppType(a.app_type);
        setRuntimeVersion(a.runtime_version || (a.app_type === 'nodejs' ? '20' : '3.11'));
        setEntrypoint(a.entrypoint || (a.app_type === 'nodejs' ? 'index.js' : 'app.py'));
        setStartCommand(a.start_command || '');
        setPort(a.port || 3000);

        if (a.env_vars) {
          const list = Object.entries(a.env_vars).map(([k, v]) => ({ key: k, value: v }));
          setEnvVars(list);
        } else {
          setEnvVars([]);
        }

        // Fetch live metrics
        const statusRes = await getManagedAppStatus(website.id);
        if (statusRes.success && statusRes.data) {
          setStatus(statusRes.data);
          if (statusRes.data.last_log_tail) {
            setLogTail(statusRes.data.last_log_tail);
          }
        }
      } else {
        setApp(null);
        setName(website.primary_domain);
        setAppType('nodejs');
        setRuntimeVersion('20');
        setEntrypoint('index.js');
        setEnvVars([
          { key: 'NODE_ENV', value: 'production' },
          { key: 'PORT', value: '3000' },
        ]);
      }
    } catch (err: any) {
      showToast(err?.message || 'Failed to load app runtime state');
    } finally {
      setLoading(false);
    }
  }, [website.id, website.primary_domain, showToast]);

  useEffect(() => {
    fetchData();
  }, [fetchData]);

  // Periodic log refresh if auto-refresh is on
  useEffect(() => {
    if (!autoRefreshLogs || !app) return;
    const interval = setInterval(async () => {
      const res = await getManagedAppStatus(website.id);
      if (res.success && res.data) {
        setStatus(res.data);
        if (res.data.last_log_tail) {
          setLogTail(res.data.last_log_tail);
        }
      }
    }, 4000);
    return () => clearInterval(interval);
  }, [autoRefreshLogs, app, website.id]);

  const handleSaveApp = async (e: React.FormEvent) => {
    e.preventDefault();
    setSubmitting(true);

    const envMap: Record<string, string> = {};
    envVars.forEach((item) => {
      if (item.key.trim()) {
        envMap[item.key.trim()] = item.value;
      }
    });

    try {
      const res = await saveManagedApp(website.id, {
        name: name.trim() || website.primary_domain,
        app_type: appType,
        runtime_version: runtimeVersion,
        entrypoint: entrypoint.trim(),
        start_command: startCommand.trim(),
        port: port || 0,
        env_vars: envMap,
      });

      if (res.success && res.data) {
        setApp(res.data);
        showToast('Application configuration deployed with systemd supervision.');
        fetchData();
      } else {
        showToast(res.error?.message || 'Failed to save application');
      }
    } catch (err: any) {
      showToast(err?.message || 'Failed to save application');
    } finally {
      setSubmitting(false);
    }
  };

  const handleControl = async (action: 'start' | 'stop' | 'restart') => {
    setControlling(action);
    try {
      const res = await controlManagedApp(website.id, action);
      if (res.success && res.data) {
        showToast(`Process ${action} completed: ${res.data.status}`);
        fetchData();
      } else {
        showToast(res.error?.message || `Failed to ${action} process`);
      }
    } catch (err: any) {
      showToast(err?.message || `Process control error`);
    } finally {
      setControlling(null);
    }
  };

  const handleDelete = async () => {
    if (!confirm('Stop supervision and remove this managed application?')) return;
    try {
      const res = await deleteManagedApp(website.id);
      if (res.success) {
        setApp(null);
        setStatus(null);
        showToast('Application supervision removed.');
      }
    } catch (err: any) {
      showToast(err?.message || 'Failed to remove application');
    }
  };

  const addEnvVar = () => {
    setEnvVars([...envVars, { key: '', value: '' }]);
  };

  const removeEnvVar = (idx: number) => {
    setEnvVars(envVars.filter((_, i) => i !== idx));
  };

  const updateEnvVar = (idx: number, field: 'key' | 'value', val: string) => {
    const updated = [...envVars];
    updated[idx][field] = val;
    setEnvVars(updated);
  };

  const formatUptime = (seconds?: number) => {
    if (!seconds) return '0s';
    if (seconds < 60) return `${seconds}s`;
    if (seconds < 3600) return `${Math.floor(seconds / 60)}m ${seconds % 60}s`;
    return `${Math.floor(seconds / 3600)}h ${Math.floor((seconds % 3600) / 60)}m`;
  };

  if (loading) {
    return (
      <div className="flex flex-col items-center justify-center p-12 text-slate-400">
        <RefreshCw className="w-8 h-8 animate-spin mb-3 text-indigo-500" />
        <p className="text-xs font-mono">Inspecting Node.js & Python runtimes...</p>
      </div>
    );
  }

  return (
    <div className="space-y-6 text-xs">
      {/* Header Banner */}
      <div className="flex items-center justify-between p-4 rounded-xl bg-slate-50 dark:bg-[#131B2E] border border-slate-200 dark:border-slate-800">
        <div className="flex items-center gap-3">
          <div className="w-9 h-9 rounded-lg bg-indigo-500/10 text-indigo-600 dark:text-indigo-400 flex items-center justify-center border border-indigo-500/20">
            <Cpu className="w-5 h-5" />
          </div>
          <div>
            <h3 className="text-sm font-bold text-slate-900 dark:text-white flex items-center gap-2">
              Node.js & Python Process Studio
              <span className="px-2 py-0.5 rounded text-[10px] font-bold bg-indigo-500/15 text-indigo-600 dark:text-indigo-400 border border-indigo-500/30">
                systemd Supervised
              </span>
            </h3>
            <p className="text-slate-500 text-[11px]">
              Production process supervisor with auto-restart, isolated environment variables, and automated reverse proxy.
            </p>
          </div>
        </div>

        {app && (
          <div className="flex items-center gap-2">
            {status?.status === 'running' ? (
              <>
                <button
                  onClick={() => handleControl('restart')}
                  disabled={!!controlling}
                  className="px-3 py-1.5 rounded-lg bg-indigo-600 hover:bg-indigo-700 text-white font-bold text-xs flex items-center gap-1.5 transition shadow-xs cursor-pointer disabled:opacity-50"
                >
                  <RotateCcw className={`w-3.5 h-3.5 ${controlling === 'restart' ? 'animate-spin' : ''}`} />
                  <span>Restart</span>
                </button>
                <button
                  onClick={() => handleControl('stop')}
                  disabled={!!controlling}
                  className="px-3 py-1.5 rounded-lg bg-rose-600 hover:bg-rose-700 text-white font-bold text-xs flex items-center gap-1.5 transition shadow-xs cursor-pointer disabled:opacity-50"
                >
                  <Square className="w-3.5 h-3.5 fill-current" />
                  <span>Stop</span>
                </button>
              </>
            ) : (
              <button
                onClick={() => handleControl('start')}
                disabled={!!controlling}
                className="px-3.5 py-1.5 rounded-lg bg-emerald-600 hover:bg-emerald-700 text-white font-bold text-xs flex items-center gap-1.5 transition shadow-xs cursor-pointer disabled:opacity-50"
              >
                <Play className="w-3.5 h-3.5 fill-current" />
                <span>Start Application</span>
              </button>
            )}

            <button
              onClick={handleDelete}
              className="p-1.5 rounded-lg text-slate-400 hover:text-rose-500 hover:bg-rose-50 dark:hover:bg-rose-950/20 transition cursor-pointer"
              title="Remove Supervised App"
            >
              <Trash2 className="w-4 h-4" />
            </button>
          </div>
        )}
      </div>

      {/* Live Process Metrics Card (if configured) */}
      {app && (
        <div className="grid grid-cols-2 sm:grid-cols-5 gap-3">
          <div className="p-3 rounded-xl bg-slate-50 dark:bg-[#131B2E] border border-slate-200 dark:border-slate-800">
            <span className="text-[10px] uppercase font-bold text-slate-400 block mb-1">Status</span>
            <div className="flex items-center gap-1.5">
              <span className={`w-2 h-2 rounded-full ${status?.status === 'running' ? 'bg-emerald-500 animate-pulse' : 'bg-slate-400'}`} />
              <span className="font-bold text-slate-900 dark:text-white capitalize">{status?.status || 'Stopped'}</span>
            </div>
          </div>

          <div className="p-3 rounded-xl bg-slate-50 dark:bg-[#131B2E] border border-slate-200 dark:border-slate-800">
            <span className="text-[10px] uppercase font-bold text-slate-400 block mb-1">Local Proxy Port</span>
            <span className="font-mono font-bold text-emerald-600 dark:text-emerald-400 text-sm">
              :{status?.port || app.port}
            </span>
          </div>

          <div className="p-3 rounded-xl bg-slate-50 dark:bg-[#131B2E] border border-slate-200 dark:border-slate-800">
            <span className="text-[10px] uppercase font-bold text-slate-400 block mb-1">PID / Uptime</span>
            <span className="font-mono font-bold text-slate-900 dark:text-white">
              {status?.pid ? `#${status.pid} (${formatUptime(status?.uptime_sec)})` : 'None'}
            </span>
          </div>

          <div className="p-3 rounded-xl bg-slate-50 dark:bg-[#131B2E] border border-slate-200 dark:border-slate-800">
            <span className="text-[10px] uppercase font-bold text-slate-400 block mb-1">Memory Usage</span>
            <span className="font-mono font-bold text-slate-900 dark:text-white">
              {status?.memory_mb ? `${status.memory_mb} MB` : '0 MB'}
            </span>
          </div>

          <div className="p-3 rounded-xl bg-slate-50 dark:bg-[#131B2E] border border-slate-200 dark:border-slate-800">
            <span className="text-[10px] uppercase font-bold text-slate-400 block mb-1">CPU %</span>
            <span className="font-mono font-bold text-slate-900 dark:text-white">
              {status?.cpu_percent ? `${status.cpu_percent}%` : '0.0%'}
            </span>
          </div>
        </div>
      )}

      {/* Detected Host Runtimes */}
      <div className="p-3.5 rounded-xl bg-slate-50 dark:bg-[#131B2E] border border-slate-200 dark:border-slate-800 flex items-center justify-between">
        <div className="flex items-center gap-3">
          <Server className="w-4 h-4 text-slate-400" />
          <span className="font-semibold text-slate-700 dark:text-slate-300">Installed Host Runtimes:</span>
          <div className="flex items-center gap-2">
            {runtimes.map((r, i) => (
              <span key={i} className="px-2 py-0.5 rounded font-mono text-[11px] bg-slate-200 dark:bg-[#0B1120] text-slate-800 dark:text-slate-200 border border-slate-300 dark:border-slate-700">
                {r.language === 'nodejs' ? 'Node' : 'Python'} {r.version} ({r.pkg_mgr})
              </span>
            ))}
          </div>
        </div>
        <span className="text-[10px] text-slate-400">Auto-detected from PATH</span>
      </div>

      {/* Main Configuration Form */}
      <form onSubmit={handleSaveApp} className="p-4 rounded-xl bg-slate-50 dark:bg-[#131B2E] border border-slate-200 dark:border-slate-800 space-y-4">
        <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
          <div>
            <label className="block text-slate-700 dark:text-slate-300 font-semibold mb-1">Runtime Ecosystem</label>
            <div className="grid grid-cols-2 gap-2">
              <button
                type="button"
                onClick={() => {
                  setAppType('nodejs');
                  setEntrypoint('index.js');
                  setRuntimeVersion('20');
                }}
                className={`p-2.5 rounded-lg border font-bold text-center cursor-pointer transition ${
                  appType === 'nodejs'
                    ? 'bg-emerald-500/15 border-emerald-500 text-emerald-600 dark:text-emerald-400'
                    : 'bg-white dark:bg-[#0F172A] border-slate-300 dark:border-slate-700 text-slate-700 dark:text-slate-300'
                }`}
              >
                Node.js
              </button>
              <button
                type="button"
                onClick={() => {
                  setAppType('python');
                  setEntrypoint('app.py');
                  setRuntimeVersion('3.11');
                }}
                className={`p-2.5 rounded-lg border font-bold text-center cursor-pointer transition ${
                  appType === 'python'
                    ? 'bg-blue-500/15 border-blue-500 text-blue-600 dark:text-blue-400'
                    : 'bg-white dark:bg-[#0F172A] border-slate-300 dark:border-slate-700 text-slate-700 dark:text-slate-300'
                }`}
              >
                Python
              </button>
            </div>
          </div>

          <div>
            <label className="block text-slate-700 dark:text-slate-300 font-semibold mb-1">Target Version</label>
            <select
              value={runtimeVersion}
              onChange={(e) => setRuntimeVersion(e.target.value)}
              className="w-full p-2 rounded-lg bg-white dark:bg-[#0F172A] border border-slate-300 dark:border-slate-700 text-slate-900 dark:text-slate-100 font-medium"
            >
              {appType === 'nodejs' ? (
                <>
                  <option value="22">Node.js 22 (Current LTS)</option>
                  <option value="20">Node.js 20 (Active LTS)</option>
                  <option value="18">Node.js 18 (Maintenance)</option>
                </>
              ) : (
                <>
                  <option value="3.12">Python 3.12</option>
                  <option value="3.11">Python 3.11</option>
                  <option value="3.10">Python 3.10</option>
                </>
              )}
            </select>
          </div>

          <div>
            <label className="block text-slate-700 dark:text-slate-300 font-semibold mb-1">Application Port (3000..9999)</label>
            <input
              type="number"
              min={3000}
              max={9999}
              value={port}
              onChange={(e) => setPort(parseInt(e.target.value) || 3000)}
              className="w-full p-2 rounded-lg bg-white dark:bg-[#0F172A] border border-slate-300 dark:border-slate-700 text-slate-900 dark:text-slate-100 font-mono text-xs"
            />
          </div>
        </div>

        <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
          <div>
            <label className="block text-slate-700 dark:text-slate-300 font-semibold mb-1">
              Entrypoint File (Relative to document root)
            </label>
            <input
              type="text"
              value={entrypoint}
              onChange={(e) => setEntrypoint(e.target.value)}
              placeholder={appType === 'nodejs' ? 'index.js / server.js' : 'app.py / main.py'}
              className="w-full p-2 rounded-lg bg-white dark:bg-[#0F172A] border border-slate-300 dark:border-slate-700 text-slate-900 dark:text-slate-100 font-mono text-xs"
            />
          </div>

          <div>
            <label className="block text-slate-700 dark:text-slate-300 font-semibold mb-1">
              Custom Start Command (Optional)
            </label>
            <input
              type="text"
              value={startCommand}
              onChange={(e) => setStartCommand(e.target.value)}
              placeholder={appType === 'nodejs' ? 'node server.js or npm start' : 'gunicorn --workers 4 app:app'}
              className="w-full p-2 rounded-lg bg-white dark:bg-[#0F172A] border border-slate-300 dark:border-slate-700 text-slate-900 dark:text-slate-100 font-mono text-xs"
            />
          </div>
        </div>

        {/* Environment Variables Table */}
        <div className="space-y-2 pt-2 border-t border-slate-200 dark:border-slate-800">
          <div className="flex items-center justify-between">
            <label className="font-semibold text-slate-900 dark:text-white flex items-center gap-2">
              <Settings className="w-3.5 h-3.5 text-slate-400" />
              Environment Variables
            </label>
            <button
              type="button"
              onClick={addEnvVar}
              className="px-2 py-1 rounded bg-slate-200 dark:bg-slate-700 hover:bg-slate-300 dark:hover:bg-slate-600 text-slate-800 dark:text-white font-semibold text-[10px] flex items-center gap-1 transition cursor-pointer"
            >
              <Plus className="w-3 h-3" />
              <span>Add Variable</span>
            </button>
          </div>

          {envVars.length === 0 ? (
            <p className="text-[11px] text-slate-400 italic">No environment variables defined.</p>
          ) : (
            <div className="space-y-2">
              {envVars.map((item, idx) => (
                <div key={idx} className="flex items-center gap-2 font-mono">
                  <input
                    type="text"
                    value={item.key}
                    onChange={(e) => updateEnvVar(idx, 'key', e.target.value)}
                    placeholder="KEY"
                    className="flex-1 p-1.5 rounded bg-white dark:bg-[#0F172A] border border-slate-300 dark:border-slate-700 text-slate-900 dark:text-slate-100 text-[11px]"
                  />
                  <span className="text-slate-400">=</span>
                  <input
                    type="text"
                    value={item.value}
                    onChange={(e) => updateEnvVar(idx, 'value', e.target.value)}
                    placeholder="VALUE"
                    className="flex-1 p-1.5 rounded bg-white dark:bg-[#0F172A] border border-slate-300 dark:border-slate-700 text-slate-900 dark:text-emerald-400 text-[11px]"
                  />
                  <button
                    type="button"
                    onClick={() => removeEnvVar(idx)}
                    className="p-1.5 text-slate-400 hover:text-rose-500 transition cursor-pointer"
                  >
                    <Trash2 className="w-3.5 h-3.5" />
                  </button>
                </div>
              ))}
            </div>
          )}
        </div>

        <div className="flex justify-end pt-3">
          <button
            type="submit"
            disabled={submitting}
            className="px-4 py-2 rounded-lg bg-indigo-600 hover:bg-indigo-700 text-white font-bold transition shadow-xs cursor-pointer disabled:opacity-50 flex items-center gap-1.5"
          >
            {submitting ? <RefreshCw className="w-3.5 h-3.5 animate-spin" /> : <Zap className="w-3.5 h-3.5" />}
            <span>{app ? 'Save & Restart Application' : 'Deploy Supervised Application'}</span>
          </button>
        </div>
      </form>

      {/* Live Application Log Output */}
      {app && (
        <div className="rounded-xl bg-[#090D16] border border-slate-800 overflow-hidden">
          <div className="px-4 py-2.5 border-b border-slate-800 bg-[#0F172A] flex items-center justify-between">
            <span className="font-mono text-[11px] font-bold text-slate-300 flex items-center gap-2">
              <Terminal className="w-3.5 h-3.5 text-emerald-400" />
              Live Console Output (app.log)
            </span>
            <label className="flex items-center gap-1.5 text-[10px] text-slate-400 cursor-pointer select-none">
              <input
                type="checkbox"
                checked={autoRefreshLogs}
                onChange={(e) => setAutoRefreshLogs(e.target.checked)}
                className="w-3 h-3 rounded"
              />
              <span>Auto-refresh</span>
            </label>
          </div>
          <div className="p-3 font-mono text-[11px] text-slate-300 max-h-48 overflow-y-auto whitespace-pre-wrap select-text">
            {logTail || 'No output recorded in app.log yet.'}
          </div>
        </div>
      )}
    </div>
  );
}
