'use client';

import React, { useEffect, useState, useCallback } from 'react';
import Link from 'next/link';
import { CheckCircle2, AlertTriangle, XCircle, RefreshCw, Activity, Server, ArrowLeft } from 'lucide-react';
import { apiFetch } from '@/lib/api';

interface ServiceProbe {
  name: string;
  category: string;
  status: 'operational' | 'degraded' | 'outage';
  latency_ms: number;
  details: string;
}

interface StatusPayload {
  overall_status: string;
  checked_at: string;
  uptime_seconds: number;
  memory_mb: number;
  goroutines: number;
  services: ServiceProbe[];
}

export default function PublicStatusPage() {
  const [data, setData] = useState<StatusPayload | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const loadStatus = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const res = await apiFetch<StatusPayload>('/api/v1/public/status');
      if (res.data) setData(res.data);
      else setError(res.error?.message || 'Unable to reach status probe endpoint');
    } catch (e: any) {
      setError(e.message || 'Control plane unreachable');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    loadStatus();
    const timer = setInterval(loadStatus, 30000);
    return () => clearInterval(timer);
  }, [loadStatus]);

  const formatUptime = (secs: number) => {
    const d = Math.floor(secs / 86400);
    const h = Math.floor((secs % 86400) / 3600);
    const m = Math.floor((secs % 3600) / 60);
    return d > 0 ? `${d}d ${h}h ${m}m` : `${h}h ${m}m ${secs % 60}s`;
  };

  return (
    <div className="min-h-screen bg-slate-50 dark:bg-slate-950 text-slate-900 dark:text-slate-100 py-10 px-4">
      <div className="max-w-4xl mx-auto space-y-6">
        <div className="flex items-center justify-between flex-wrap gap-3">
          <div className="flex items-center gap-3">
            <div className="p-2.5 rounded-2xl bg-emerald-500/10 border border-emerald-500/20 text-emerald-600 dark:text-emerald-400">
              <Activity className="w-6 h-6" />
            </div>
            <div>
              <h1 className="text-2xl font-extrabold tracking-tight">Hostvra System Status</h1>
              <p className="text-xs text-slate-500">Live real-time telemetry from production probes (no fabricated uptime numbers)</p>
            </div>
          </div>
          <div className="flex items-center gap-2">
            <button
              onClick={loadStatus}
              disabled={loading}
              className="px-3.5 py-2 rounded-xl bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 text-xs font-bold flex items-center gap-1.5 cursor-pointer hover:bg-slate-100 dark:hover:bg-slate-800"
            >
              <RefreshCw className={`w-3.5 h-3.5 ${loading ? 'animate-spin' : ''}`} />
              <span>Refresh Probes</span>
            </button>
            <Link
              href="/"
              className="px-3.5 py-2 rounded-xl bg-emerald-600 hover:bg-emerald-500 text-white text-xs font-bold flex items-center gap-1.5"
            >
              <ArrowLeft className="w-3.5 h-3.5" />
              <span>Home</span>
            </Link>
          </div>
        </div>

        {error && (
          <div className="p-5 rounded-2xl bg-rose-50 dark:bg-rose-950/30 border border-rose-200 dark:border-rose-800 text-rose-700 dark:text-rose-300 text-sm font-semibold flex items-center gap-3">
            <XCircle className="w-5 h-5 shrink-0" />
            <span>{error}</span>
          </div>
        )}

        {data && (
          <>
            <div className={`p-6 rounded-3xl border shadow-sm flex items-center justify-between flex-wrap gap-4 ${
              data.overall_status === 'operational'
                ? 'bg-emerald-50 dark:bg-emerald-950/20 border-emerald-200 dark:border-emerald-800/60'
                : 'bg-amber-50 dark:bg-amber-950/20 border-amber-200 dark:border-amber-800/60'
            }`}>
              <div className="flex items-center gap-3">
                {data.overall_status === 'operational' ? (
                  <CheckCircle2 className="w-8 h-8 text-emerald-600 dark:text-emerald-400" />
                ) : (
                  <AlertTriangle className="w-8 h-8 text-amber-600 dark:text-amber-400" />
                )}
                <div>
                  <h2 className="text-lg font-bold capitalize">
                    {data.overall_status === 'operational' ? 'All Monitored Systems Operational' : `System Status: ${data.overall_status}`}
                  </h2>
                  <p className="text-xs text-slate-600 dark:text-slate-400">
                    Last checked: {new Date(data.checked_at).toLocaleString()} • Live Process Uptime: <strong>{formatUptime(data.uptime_seconds)}</strong>
                  </p>
                </div>
              </div>
              <div className="text-xs font-mono text-slate-500 bg-white dark:bg-slate-900 px-3 py-1.5 rounded-xl border border-slate-200 dark:border-slate-800">
                RAM: {data.memory_mb} MB • Workers: {data.goroutines}
              </div>
            </div>

            <div className="bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 rounded-3xl overflow-hidden shadow-sm divide-y divide-slate-100 dark:divide-slate-800">
              {data.services.map((svc, idx) => (
                <div key={idx} className="p-5 flex items-center justify-between flex-wrap gap-3">
                  <div className="flex items-center gap-3">
                    <Server className="w-4 h-4 text-slate-400" />
                    <div>
                      <div className="font-bold text-sm">{svc.name}</div>
                      <div className="text-xs text-slate-500">{svc.category} — {svc.details}</div>
                    </div>
                  </div>
                  <div className="flex items-center gap-3">
                    <span className="text-[11px] font-mono text-slate-400">{svc.latency_ms} ms</span>
                    <span className={`px-2.5 py-1 rounded-full text-[11px] font-bold uppercase ${
                      svc.status === 'operational'
                        ? 'bg-emerald-100 text-emerald-700 dark:bg-emerald-500/15 dark:text-emerald-400'
                        : svc.status === 'degraded'
                        ? 'bg-amber-100 text-amber-700'
                        : 'bg-rose-100 text-rose-700'
                    }`}>
                      {svc.status}
                    </span>
                  </div>
                </div>
              ))}
            </div>
          </>
        )}
      </div>
    </div>
  );
}
