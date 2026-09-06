'use client';

import React, { useState } from 'react';
import {
  Clock,
  Plus,
  Play,
  Trash2,
  Power,
  RefreshCw,
  Search,
  CheckCircle2,
  X,
  Terminal,
} from 'lucide-react';
import { DashboardShell } from '@/components/DashboardShell';

interface CronJobItem {
  id: string;
  schedule: string;
  command: string;
  system_user: string;
  description: string;
  is_enabled: boolean;
  last_run_at?: string;
}

export default function CronPage() {
  const [jobs, setJobs] = useState<CronJobItem[]>([
    {
      id: 'cron-1',
      schedule: '0 3 * * *',
      command: 'php /var/www/mycoolapp.com/artisan schedule:run',
      system_user: 'www-data',
      description: 'Application Scheduled Tasks',
      is_enabled: true,
      last_run_at: '2026-09-06T03:00:00Z',
    },
    {
      id: 'cron-2',
      schedule: '0 4 * * 0',
      command: '/usr/local/bin/hostvra-backup-db',
      system_user: 'root',
      description: 'Weekly Database Snapshot',
      is_enabled: true,
      last_run_at: '2026-09-01T04:00:00Z',
    },
  ]);
  const [modalOpen, setModalOpen] = useState(false);
  const [schedule, setSchedule] = useState('0 2 * * *');
  const [command, setCommand] = useState('');
  const [description, setDescription] = useState('');
  const [systemUser, setSystemUser] = useState('www-data');
  const [runOutput, setRunOutput] = useState<string | null>(null);

  const handleAddJob = (e: React.FormEvent) => {
    e.preventDefault();
    if (!command) return;

    const newJob: CronJobItem = {
      id: 'cron-' + Date.now(),
      schedule,
      command,
      system_user: systemUser,
      description: description || 'Custom scheduled task',
      is_enabled: true,
    };

    setJobs([...jobs, newJob]);
    setModalOpen(false);
    setCommand('');
    setDescription('');
  };

  const handleToggle = (id: string) => {
    setJobs(
      jobs.map((j) => (j.id === id ? { ...j, is_enabled: !j.is_enabled } : j))
    );
  };

  const handleDelete = (id: string) => {
    if (confirm('Delete this cron job?')) {
      setJobs(jobs.filter((j) => j.id !== id));
    }
  };

  const handleRunNow = (cmd: string) => {
    setRunOutput(`$ ${cmd}\n[Hostvra Runner] Command dispatched successfully.\nStatus: Process exited with return code 0.\nTimestamp: ${new Date().toISOString()}`);
  };

  return (
    <DashboardShell>
      <div className="space-y-8">
        <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
          <div>
            <h1 className="text-2xl font-bold tracking-tight text-white">Scheduled Cron Jobs</h1>
            <p className="text-sm text-slate-400 mt-1">
              Configure, test, and monitor automated Linux background tasks with 5-field cron syntax.
            </p>
          </div>
          <button
            onClick={() => setModalOpen(true)}
            className="inline-flex items-center gap-2 px-4 py-2.5 rounded-xl bg-indigo-600 hover:bg-indigo-500 text-white text-sm font-semibold shadow-lg shadow-indigo-600/25 transition-all"
          >
            <Plus className="w-4 h-4" />
            Add Cron Job
          </button>
        </div>

        {/* Cron Table */}
        <div className="bg-surface-900 border border-surface-800 rounded-2xl overflow-hidden shadow-xl">
          <table className="w-full text-left text-sm">
            <thead>
              <tr className="border-b border-surface-800 bg-surface-950/40 text-slate-400 text-xs uppercase tracking-wider">
                <th className="px-6 py-3.5 font-semibold">Schedule Expression</th>
                <th className="px-6 py-3.5 font-semibold">Command & Description</th>
                <th className="px-6 py-3.5 font-semibold">User</th>
                <th className="px-6 py-3.5 font-semibold">Status</th>
                <th className="px-6 py-3.5 font-semibold text-right">Actions</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-surface-800/60">
              {jobs.map((job) => (
                <tr key={job.id} className="hover:bg-surface-800/30 transition-colors">
                  <td className="px-6 py-4">
                    <span className="font-mono text-xs px-2.5 py-1 rounded bg-surface-800 text-indigo-300 font-semibold border border-surface-700">
                      {job.schedule}
                    </span>
                  </td>
                  <td className="px-6 py-4">
                    <div className="font-mono text-xs text-white truncate max-w-md">{job.command}</div>
                    <div className="text-xs text-slate-400 mt-0.5">{job.description}</div>
                  </td>
                  <td className="px-6 py-4 font-mono text-xs text-slate-300">{job.system_user}</td>
                  <td className="px-6 py-4">
                    <span
                      className={`inline-flex items-center px-2 py-0.5 rounded-full text-[11px] font-semibold ${
                        job.is_enabled
                          ? 'bg-emerald-500/10 text-emerald-400 border border-emerald-500/20'
                          : 'bg-slate-500/10 text-slate-400 border border-slate-500/20'
                      }`}
                    >
                      {job.is_enabled ? 'Enabled' : 'Disabled'}
                    </span>
                  </td>
                  <td className="px-6 py-4 text-right">
                    <div className="flex items-center justify-end gap-2">
                      <button
                        onClick={() => handleRunNow(job.command)}
                        title="Run Immediately"
                        className="p-1.5 rounded-lg border border-surface-700 text-slate-400 hover:text-emerald-400 hover:bg-surface-800 transition-colors"
                      >
                        <Play className="w-3.5 h-3.5" />
                      </button>
                      <button
                        onClick={() => handleToggle(job.id)}
                        title={job.is_enabled ? 'Disable' : 'Enable'}
                        className="p-1.5 rounded-lg border border-surface-700 text-slate-400 hover:text-amber-400 hover:bg-surface-800 transition-colors"
                      >
                        <Power className="w-3.5 h-3.5" />
                      </button>
                      <button
                        onClick={() => handleDelete(job.id)}
                        title="Delete"
                        className="p-1.5 rounded-lg border border-surface-700 text-slate-400 hover:text-rose-400 hover:bg-surface-800 transition-colors"
                      >
                        <Trash2 className="w-3.5 h-3.5" />
                      </button>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>

        {/* Execution Output Drawer */}
        {runOutput && (
          <div className="p-4 rounded-2xl bg-surface-950 border border-surface-800 shadow-xl relative">
            <button
              onClick={() => setRunOutput(null)}
              className="absolute top-3 right-3 text-slate-400 hover:text-white"
            >
              <X className="w-4 h-4" />
            </button>
            <div className="flex items-center gap-2 mb-2 text-xs font-semibold text-slate-300">
              <Terminal className="w-4 h-4 text-indigo-400" />
              <span>Manual Execution Output</span>
            </div>
            <pre className="font-mono text-xs text-slate-300 whitespace-pre-wrap">{runOutput}</pre>
          </div>
        )}

        {/* Add Cron Modal */}
        {modalOpen && (
          <div className="fixed inset-0 z-50 bg-black/70 backdrop-blur-sm flex items-center justify-center p-4">
            <div className="w-full max-w-md bg-surface-900 border border-surface-700 rounded-2xl shadow-2xl p-6 relative">
              <button
                onClick={() => setModalOpen(false)}
                className="absolute top-5 right-5 p-1 rounded-lg text-slate-400 hover:text-white hover:bg-surface-800 transition-colors"
              >
                <X className="w-5 h-5" />
              </button>

              <div className="flex items-center gap-3 mb-5">
                <div className="w-10 h-10 rounded-xl bg-indigo-500/10 border border-indigo-500/20 flex items-center justify-center text-indigo-400">
                  <Clock className="w-5 h-5" />
                </div>
                <div>
                  <h2 className="text-lg font-bold text-white">Add Cron Job</h2>
                  <p className="text-xs text-slate-400">Schedule automatic execution</p>
                </div>
              </div>

              <form onSubmit={handleAddJob} className="space-y-4">
                <div>
                  <div className="flex justify-between items-center mb-1.5">
                    <label className="text-xs font-semibold uppercase tracking-wider text-slate-300">
                      Cron Schedule Expression
                    </label>
                    <select
                      onChange={(e) => setSchedule(e.target.value)}
                      className="text-[11px] bg-surface-950 border border-surface-700 text-indigo-400 rounded-lg px-2 py-0.5"
                    >
                      <option value="0 2 * * *">Daily at 2:00 AM</option>
                      <option value="*/15 * * * *">Every 15 Minutes</option>
                      <option value="0 * * * *">Every Hour</option>
                      <option value="0 0 * * 0">Weekly (Sunday)</option>
                    </select>
                  </div>
                  <input
                    type="text"
                    required
                    value={schedule}
                    onChange={(e) => setSchedule(e.target.value)}
                    placeholder="0 2 * * *"
                    className="w-full px-3.5 py-2.5 rounded-xl bg-surface-950 border border-surface-700 text-slate-100 font-mono text-sm focus:outline-none focus:border-indigo-500"
                  />
                </div>

                <div>
                  <label className="block text-xs font-semibold uppercase tracking-wider text-slate-300 mb-1.5">
                    Command to Run
                  </label>
                  <textarea
                    required
                    value={command}
                    onChange={(e) => setCommand(e.target.value)}
                    placeholder="php /var/www/site/artisan schedule:run"
                    className="w-full px-3.5 py-2.5 rounded-xl bg-surface-950 border border-surface-700 text-slate-100 font-mono text-xs focus:outline-none focus:border-indigo-500 h-20 resize-none"
                  />
                </div>

                <div className="grid grid-cols-2 gap-3">
                  <div>
                    <label className="block text-xs font-semibold uppercase tracking-wider text-slate-300 mb-1.5">
                      System User
                    </label>
                    <input
                      type="text"
                      value={systemUser}
                      onChange={(e) => setSystemUser(e.target.value)}
                      className="w-full px-3.5 py-2 rounded-xl bg-surface-950 border border-surface-700 text-slate-100 font-mono text-xs focus:outline-none focus:border-indigo-500"
                    />
                  </div>
                  <div>
                    <label className="block text-xs font-semibold uppercase tracking-wider text-slate-300 mb-1.5">
                      Description
                    </label>
                    <input
                      type="text"
                      value={description}
                      onChange={(e) => setDescription(e.target.value)}
                      placeholder="e.g. Cache clean"
                      className="w-full px-3.5 py-2 rounded-xl bg-surface-950 border border-surface-700 text-slate-100 text-xs focus:outline-none focus:border-indigo-500"
                    />
                  </div>
                </div>

                <div className="flex justify-end gap-2 pt-2">
                  <button
                    type="button"
                    onClick={() => setModalOpen(false)}
                    className="px-4 py-2 rounded-xl text-xs text-slate-400 hover:text-white transition-colors"
                  >
                    Cancel
                  </button>
                  <button
                    type="submit"
                    className="px-5 py-2 rounded-xl bg-indigo-600 hover:bg-indigo-500 text-white text-xs font-semibold shadow-md transition-all"
                  >
                    Save Cron Job
                  </button>
                </div>
              </form>
            </div>
          </div>
        )}
      </div>
    </DashboardShell>
  );
}
