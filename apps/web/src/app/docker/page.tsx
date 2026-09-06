'use client';

import React, { useState } from 'react';
import {
  Container,
  Play,
  Square,
  RotateCw,
  FileText,
  Server as ServerIcon,
  RefreshCw,
  X,
  CheckCircle2,
  AlertCircle,
} from 'lucide-react';
import { DashboardShell } from '@/components/DashboardShell';

interface ContainerData {
  id: string;
  name: string;
  image: string;
  state: 'running' | 'exited' | 'paused';
  status: string;
  ports: string;
}

export default function DockerPage() {
  const [containers, setContainers] = useState<ContainerData[]>([
    {
      id: 'c1',
      name: 'redis-cache',
      image: 'redis:7-alpine',
      state: 'running',
      status: 'Up 3 days',
      ports: '0.0.0.0:6379->6379/tcp',
    },
    {
      id: 'c2',
      name: 'postgres-db',
      image: 'postgres:16-alpine',
      state: 'running',
      status: 'Up 3 days',
      ports: '0.0.0.0:5432->5432/tcp',
    },
    {
      id: 'c3',
      name: 'rabbitmq-queue',
      image: 'rabbitmq:3-management',
      state: 'exited',
      status: 'Exited (0) 2 hours ago',
      ports: '5672/tcp, 15672/tcp',
    },
  ]);
  const [logsModalOpen, setLogsModalOpen] = useState(false);
  const [activeContainer, setActiveContainer] = useState<ContainerData | null>(null);
  const [logsContent, setLogsContent] = useState('');

  const handleAction = (id: string, action: 'start' | 'stop' | 'restart') => {
    setContainers(
      containers.map((c) => {
        if (c.id === id) {
          if (action === 'stop') return { ...c, state: 'exited', status: 'Exited (0) just now' };
          if (action === 'start' || action === 'restart')
            return { ...c, state: 'running', status: 'Up just now' };
        }
        return c;
      })
    );
  };

  const handleOpenLogs = (c: ContainerData) => {
    setActiveContainer(c);
    setLogsContent(
      `[${c.name}] 2026-09-06T09:40:12.102Z INFO Service started\n[${c.name}] 2026-09-06T09:40:12.155Z Ready to accept connections\n[${c.name}] 2026-09-06T09:42:01.320Z Heartbeat OK - Active connections: 4`
    );
    setLogsModalOpen(true);
  };

  return (
    <DashboardShell>
      <div className="space-y-8">
        <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
          <div>
            <h1 className="text-2xl font-bold tracking-tight text-white">Docker Engine & Containers</h1>
            <p className="text-sm text-slate-400 mt-1">
              Inspect and control Docker containers, images, and compose workloads.
            </p>
          </div>
          <div className="flex items-center gap-3">
            <span className="flex items-center gap-2 px-3 py-1.5 rounded-lg bg-surface-900 border border-surface-800 text-xs text-slate-300 font-medium">
              <span className="w-2 h-2 rounded-full bg-emerald-500 animate-pulse" />
              Docker v27.0 Active
            </span>
          </div>
        </div>

        {/* Containers Table */}
        <div className="bg-surface-900 border border-surface-800 rounded-2xl overflow-hidden shadow-xl">
          <table className="w-full text-left text-sm">
            <thead>
              <tr className="border-b border-surface-800 bg-surface-950/40 text-slate-400 text-xs uppercase tracking-wider">
                <th className="px-6 py-3.5 font-semibold">Container Name</th>
                <th className="px-6 py-3.5 font-semibold">Image</th>
                <th className="px-6 py-3.5 font-semibold">State</th>
                <th className="px-6 py-3.5 font-semibold">Port Mappings</th>
                <th className="px-6 py-3.5 font-semibold text-right">Actions</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-surface-800/60">
              {containers.map((c) => (
                <tr key={c.id} className="hover:bg-surface-800/30 transition-colors">
                  <td className="px-6 py-4 font-bold text-white flex items-center gap-2.5">
                    <Container className="w-4 h-4 text-indigo-400" />
                    <span>{c.name}</span>
                  </td>
                  <td className="px-6 py-4 font-mono text-xs text-slate-300">{c.image}</td>
                  <td className="px-6 py-4">
                    <span
                      className={`inline-flex items-center gap-1.5 px-2.5 py-0.5 rounded-full text-[11px] font-semibold capitalize ${
                        c.state === 'running'
                          ? 'bg-emerald-500/10 text-emerald-400 border border-emerald-500/20'
                          : 'bg-slate-500/10 text-slate-400 border border-slate-500/20'
                      }`}
                    >
                      <span
                        className={`w-1.5 h-1.5 rounded-full ${
                          c.state === 'running' ? 'bg-emerald-500' : 'bg-slate-500'
                        }`}
                      />
                      {c.status}
                    </span>
                  </td>
                  <td className="px-6 py-4 font-mono text-xs text-slate-400">{c.ports}</td>
                  <td className="px-6 py-4 text-right">
                    <div className="flex items-center justify-end gap-2">
                      {c.state === 'running' ? (
                        <button
                          onClick={() => handleAction(c.id, 'stop')}
                          title="Stop"
                          className="p-1.5 rounded-lg border border-surface-700 text-slate-400 hover:text-amber-400 hover:bg-surface-800 transition-colors"
                        >
                          <Square className="w-3.5 h-3.5" />
                        </button>
                      ) : (
                        <button
                          onClick={() => handleAction(c.id, 'start')}
                          title="Start"
                          className="p-1.5 rounded-lg border border-surface-700 text-slate-400 hover:text-emerald-400 hover:bg-surface-800 transition-colors"
                        >
                          <Play className="w-3.5 h-3.5" />
                        </button>
                      )}
                      <button
                        onClick={() => handleAction(c.id, 'restart')}
                        title="Restart"
                        className="p-1.5 rounded-lg border border-surface-700 text-slate-400 hover:text-indigo-400 hover:bg-surface-800 transition-colors"
                      >
                        <RotateCw className="w-3.5 h-3.5" />
                      </button>
                      <button
                        onClick={() => handleOpenLogs(c)}
                        title="View Logs"
                        className="p-1.5 rounded-lg border border-surface-700 text-slate-400 hover:text-white hover:bg-surface-800 transition-colors"
                      >
                        <FileText className="w-3.5 h-3.5" />
                      </button>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>

        {/* Container Logs Modal */}
        {logsModalOpen && activeContainer && (
          <div className="fixed inset-0 z-50 bg-black/80 backdrop-blur-sm flex items-center justify-center p-4">
            <div className="w-full max-w-3xl bg-surface-900 border border-surface-700 rounded-2xl shadow-2xl flex flex-col h-[70vh] overflow-hidden">
              <div className="px-6 py-4 border-b border-surface-800 flex items-center justify-between bg-surface-950/60">
                <div className="flex items-center gap-2">
                  <Container className="w-5 h-5 text-indigo-400" />
                  <h3 className="font-bold text-white text-sm">Logs: {activeContainer.name}</h3>
                </div>
                <button
                  onClick={() => setLogsModalOpen(false)}
                  className="p-1.5 rounded-lg text-slate-400 hover:text-white hover:bg-surface-800 transition-colors"
                >
                  <X className="w-5 h-5" />
                </button>
              </div>
              <div className="flex-1 p-4 bg-[#0d1117] overflow-y-auto font-mono text-xs text-slate-300 whitespace-pre-wrap leading-relaxed">
                {logsContent}
              </div>
            </div>
          </div>
        )}
      </div>
    </DashboardShell>
  );
}
