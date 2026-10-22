'use client';

import React, { useState, useEffect, useCallback } from 'react';
import {
  Zap,
  Server,
  Database,
  RefreshCw,
  Trash2,
  Play,
  CheckCircle2,
  AlertTriangle,
  Activity,
  Lock,
  Layers,
  Gauge,
  Sliders,
  Check,
  Info,
} from 'lucide-react';
import {
  Website,
  WebsiteCacheConfig,
  CacheServiceStatus,
  CacheMetrics,
  CacheBenchmarkResult,
  getCacheServices,
  getWebsiteCacheConfig,
  saveWebsiteCacheConfig,
  disableWebsiteCache,
  getWebsiteCacheMetrics,
  flushWebsiteCache,
  benchmarkWebsiteCache,
  toggleWordPressObjectCache,
} from '@/lib/api';

interface CachingTabProps {
  website: Website;
  showToast: (msg: string) => void;
}

export function CachingTab({ website, showToast }: CachingTabProps) {
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [flushing, setFlushing] = useState(false);
  const [benchmarking, setBenchmarking] = useState(false);
  const [refreshingMetrics, setRefreshingMetrics] = useState(false);

  // Daemon status on host
  const [services, setServices] = useState<CacheServiceStatus[]>([]);

  // Configuration state
  const [cacheType, setCacheType] = useState<'redis' | 'memcached' | 'none'>('none');
  const [connectionType, setConnectionType] = useState<'socket' | 'tcp'>('socket');
  const [socketPath, setSocketPath] = useState('');
  const [host, setHost] = useState('127.0.0.1');
  const [port, setPort] = useState(6379);
  const [password, setPassword] = useState('');
  const [maxMemoryMB, setMaxMemoryMB] = useState(256);
  const [evictionPolicy, setEvictionPolicy] = useState('allkeys-lru');
  const [wpObjectCacheEnabled, setWpObjectCacheEnabled] = useState(false);
  const [status, setStatus] = useState<string>('disabled');

  // Real-time telemetry
  const [metrics, setMetrics] = useState<CacheMetrics | null>(null);
  const [benchmarkResult, setBenchmarkResult] = useState<CacheBenchmarkResult | null>(null);

  const loadData = useCallback(async () => {
    setLoading(true);
    try {
      // 1. Load host daemon availability
      const svcRes = await getCacheServices();
      if (svcRes.success && svcRes.data) {
        setServices(svcRes.data);
      }

      // 2. Load website cache config
      const cfgRes = await getWebsiteCacheConfig(website.id);
      if (cfgRes.success && cfgRes.data) {
        const c = cfgRes.data;
        setCacheType(c.cache_type || 'none');
        setConnectionType(c.connection_type || 'socket');
        setSocketPath(c.socket_path || `/var/run/redis/redis-${website.id}.sock`);
        setHost(c.host || '127.0.0.1');
        setPort(c.port || 6379);
        setPassword(c.password || '');
        setMaxMemoryMB(c.max_memory_mb || 256);
        setEvictionPolicy(c.eviction_policy || 'allkeys-lru');
        setWpObjectCacheEnabled(!!c.wp_object_cache_enabled);
        setStatus(c.status || 'disabled');

        if (c.cache_type && c.cache_type !== 'none') {
          // Fetch metrics
          const mRes = await getWebsiteCacheMetrics(website.id);
          if (mRes.success && mRes.data) {
            setMetrics(mRes.data);
          }
        }
      }
    } catch {
      showToast('Failed to load caching telemetry');
    } finally {
      setLoading(false);
    }
  }, [website.id, showToast]);

  useEffect(() => {
    loadData();
  }, [loadData]);

  const handleRefreshMetrics = async () => {
    if (cacheType === 'none') return;
    setRefreshingMetrics(true);
    try {
      const mRes = await getWebsiteCacheMetrics(website.id);
      if (mRes.success && mRes.data) {
        setMetrics(mRes.data);
        showToast('Live metrics refreshed');
      }
    } catch {
      showToast('Could not fetch telemetry');
    } finally {
      setRefreshingMetrics(false);
    }
  };

  const handleSaveConfig = async () => {
    setSaving(true);
    try {
      const payload: Partial<WebsiteCacheConfig> = {
        website_id: website.id,
        cache_type: cacheType,
        connection_type: connectionType,
        socket_path: socketPath,
        host: host,
        port: port,
        password: password,
        max_memory_mb: maxMemoryMB,
        eviction_policy: evictionPolicy,
        wp_object_cache_enabled: wpObjectCacheEnabled,
      };

      const res = await saveWebsiteCacheConfig(website.id, payload);
      if (res.success && res.data) {
        setStatus(res.data.status);
        showToast('Caching engine configuration applied successfully');
        handleRefreshMetrics();
      } else {
        showToast(res.error?.message || 'Failed to update caching configuration');
      }
    } catch {
      showToast('Error communicating with caching engine');
    } finally {
      setSaving(false);
    }
  };

  const handleDisableCache = async () => {
    if (!confirm('Are you sure you want to completely disable caching and WordPress object-cache constants for this website?')) {
      return;
    }
    setSaving(true);
    try {
      const res = await disableWebsiteCache(website.id);
      if (res.success) {
        setCacheType('none');
        setStatus('disabled');
        setWpObjectCacheEnabled(false);
        setMetrics(null);
        setBenchmarkResult(null);
        showToast('Caching disabled and configurations removed');
      } else {
        showToast(res.error?.message || 'Failed to disable caching');
      }
    } catch {
      showToast('Error communicating with caching engine');
    } finally {
      setSaving(false);
    }
  };

  const handleFlush = async () => {
    setFlushing(true);
    try {
      const res = await flushWebsiteCache(website.id);
      if (res.success) {
        showToast('Cache memory keyspace flushed successfully');
        handleRefreshMetrics();
      } else {
        showToast(res.error?.message || 'Failed to flush cache');
      }
    } catch {
      showToast('Error executing cache flush');
    } finally {
      setFlushing(false);
    }
  };

  const handleBenchmark = async () => {
    setBenchmarking(true);
    setBenchmarkResult(null);
    try {
      const res = await benchmarkWebsiteCache(website.id, 30);
      if (res.success && res.data) {
        setBenchmarkResult(res.data);
        showToast('Benchmark probe completed');
      } else {
        showToast(res.error?.message || 'Benchmark failed');
      }
    } catch {
      showToast('Error running latency benchmark');
    } finally {
      setBenchmarking(false);
    }
  };

  const handleToggleWP = async (enabled: boolean) => {
    try {
      const res = await toggleWordPressObjectCache(website.id, enabled);
      if (res.success && res.data) {
        setWpObjectCacheEnabled(enabled);
        showToast(enabled ? 'WordPress Object Cache enabled in wp-config.php' : 'WordPress Object Cache disabled');
      } else {
        showToast(res.error?.message || 'Failed to update WordPress object cache');
      }
    } catch {
      showToast('Error updating WordPress object cache');
    }
  };

  if (loading) {
    return (
      <div className="flex flex-col items-center justify-center p-12 text-slate-500">
        <RefreshCw className="w-8 h-8 animate-spin text-emerald-500 mb-3" />
        <span className="text-sm font-medium">Inspecting caching subsystem & host daemons...</span>
      </div>
    );
  }

  return (
    <div className="space-y-6 text-slate-800 dark:text-slate-100 pb-8">
      {/* 1. Header Banner & Host Daemon Status */}
      <div className="p-4 rounded-xl bg-gradient-to-r from-emerald-500/10 via-cyan-500/10 to-transparent border border-emerald-500/20 flex flex-col md:flex-row items-start md:items-center justify-between gap-4">
        <div>
          <div className="flex items-center gap-2">
            <Zap className="w-5 h-5 text-emerald-500" />
            <h3 className="text-sm font-bold tracking-tight">Enterprise In-Memory Caching Studio</h3>
            <span
              className={`px-2 py-0.5 text-xs font-semibold rounded-full border ${
                status === 'active'
                  ? 'bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 border-emerald-500/30'
                  : 'bg-slate-500/10 text-slate-500 border-slate-500/20'
              }`}
            >
              {status === 'active' ? '● Active Caching' : '○ Disabled'}
            </span>
          </div>
          <p className="text-xs text-slate-500 dark:text-slate-400 mt-1">
            Supercharge site responsiveness and eliminate SQL bottlenecks using isolated Redis or Memcached instances.
          </p>
        </div>

        {/* Host daemons pill */}
        <div className="flex items-center gap-2 text-xs">
          {services.map((svc) => (
            <div
              key={svc.service_type}
              className={`flex items-center gap-1.5 px-2.5 py-1 rounded-lg border text-xs font-medium ${
                svc.installed && svc.running
                  ? 'bg-emerald-50 dark:bg-emerald-950/30 border-emerald-200 dark:border-emerald-800/60 text-emerald-700 dark:text-emerald-300'
                  : svc.installed
                  ? 'bg-amber-50 dark:bg-amber-950/30 border-amber-200 dark:border-amber-800/60 text-amber-700 dark:text-amber-300'
                  : 'bg-slate-100 dark:bg-slate-800/50 border-slate-200 dark:border-slate-700 text-slate-500'
              }`}
            >
              <Server className="w-3.5 h-3.5" />
              <span className="capitalize">{svc.service_type}</span>
              <span className="text-[10px] opacity-80">
                {svc.installed && svc.running ? `v${svc.version || 'Active'}` : svc.installed ? 'Stopped' : 'Not installed'}
              </span>
            </div>
          ))}
        </div>
      </div>

      {/* 2. Caching Engine Selector */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-3">
        {/* NONE */}
        <div
          onClick={() => setCacheType('none')}
          className={`p-3.5 rounded-xl border cursor-pointer transition flex items-start gap-3 ${
            cacheType === 'none'
              ? 'bg-slate-100 dark:bg-slate-800/80 border-slate-400 dark:border-slate-600 shadow-xs'
              : 'bg-white dark:bg-slate-900/50 border-slate-200 dark:border-slate-800 hover:border-slate-300 dark:hover:border-slate-700'
          }`}
        >
          <div className="p-2 rounded-lg bg-slate-100 dark:bg-slate-800 text-slate-500 mt-0.5">
            <Layers className="w-4 h-4" />
          </div>
          <div className="flex-1 min-w-0">
            <div className="flex items-center justify-between">
              <span className="text-xs font-bold">No Caching</span>
              {cacheType === 'none' && <Check className="w-4 h-4 text-slate-600 dark:text-slate-300" />}
            </div>
            <p className="text-[11px] text-slate-500 mt-0.5">Direct queries hitting backend database without in-memory layer.</p>
          </div>
        </div>

        {/* REDIS */}
        <div
          onClick={() => {
            setCacheType('redis');
            if (port === 11211) setPort(6379);
          }}
          className={`p-3.5 rounded-xl border cursor-pointer transition flex items-start gap-3 ${
            cacheType === 'redis'
              ? 'bg-rose-500/5 border-rose-500/50 dark:border-rose-500/60 shadow-xs ring-1 ring-rose-500/20'
              : 'bg-white dark:bg-slate-900/50 border-slate-200 dark:border-slate-800 hover:border-rose-300 dark:hover:border-rose-900/40'
          }`}
        >
          <div className="p-2 rounded-lg bg-rose-50 dark:bg-rose-950/40 text-rose-600 dark:text-rose-400 mt-0.5">
            <Zap className="w-4 h-4" />
          </div>
          <div className="flex-1 min-w-0">
            <div className="flex items-center justify-between">
              <span className="text-xs font-bold text-rose-600 dark:text-rose-400">Redis Engine</span>
              {cacheType === 'redis' && <Check className="w-4 h-4 text-rose-600 dark:text-rose-400" />}
            </div>
            <p className="text-[11px] text-slate-500 mt-0.5">Rich data structures, high hit rates, WordPress object-cache recommended.</p>
          </div>
        </div>

        {/* MEMCACHED */}
        <div
          onClick={() => {
            setCacheType('memcached');
            if (port === 6379) setPort(11211);
          }}
          className={`p-3.5 rounded-xl border cursor-pointer transition flex items-start gap-3 ${
            cacheType === 'memcached'
              ? 'bg-cyan-500/5 border-cyan-500/50 dark:border-cyan-500/60 shadow-xs ring-1 ring-cyan-500/20'
              : 'bg-white dark:bg-slate-900/50 border-slate-200 dark:border-slate-800 hover:border-cyan-300 dark:hover:border-cyan-900/40'
          }`}
        >
          <div className="p-2 rounded-lg bg-cyan-50 dark:bg-cyan-950/40 text-cyan-600 dark:text-cyan-400 mt-0.5">
            <Database className="w-4 h-4" />
          </div>
          <div className="flex-1 min-w-0">
            <div className="flex items-center justify-between">
              <span className="text-xs font-bold text-cyan-600 dark:text-cyan-400">Memcached Engine</span>
              {cacheType === 'memcached' && <Check className="w-4 h-4 text-cyan-600 dark:text-cyan-400" />}
            </div>
            <p className="text-[11px] text-slate-500 mt-0.5">Simple multithreaded key-value memory store with low footprint.</p>
          </div>
        </div>
      </div>

      {cacheType !== 'none' && (
        <>
          {/* 3. Live Telemetry & Control Panel */}
          <div className="p-4 rounded-xl bg-slate-50 dark:bg-slate-900/60 border border-slate-200 dark:border-slate-800 space-y-4">
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-2">
                <Activity className="w-4 h-4 text-emerald-500" />
                <h4 className="text-xs font-bold uppercase tracking-wider text-slate-600 dark:text-slate-300">
                  Live Engine Telemetry & Actions
                </h4>
              </div>
              <div className="flex items-center gap-2">
                <button
                  type="button"
                  onClick={handleRefreshMetrics}
                  disabled={refreshingMetrics}
                  className="px-2.5 py-1 text-xs font-medium rounded-lg border border-slate-300 dark:border-slate-700 hover:bg-white dark:hover:bg-slate-800 text-slate-700 dark:text-slate-300 flex items-center gap-1.5 transition cursor-pointer"
                >
                  <RefreshCw className={`w-3 h-3 ${refreshingMetrics ? 'animate-spin' : ''}`} />
                  <span>Refresh</span>
                </button>
                <button
                  type="button"
                  onClick={handleFlush}
                  disabled={flushing}
                  className="px-2.5 py-1 text-xs font-medium rounded-lg bg-rose-50 dark:bg-rose-950/40 border border-rose-300 dark:border-rose-800/60 text-rose-700 dark:text-rose-300 hover:bg-rose-100 dark:hover:bg-rose-900/40 flex items-center gap-1.5 transition cursor-pointer"
                >
                  <Trash2 className="w-3 h-3" />
                  <span>{flushing ? 'Flushing...' : 'Flush Cache'}</span>
                </button>
                <button
                  type="button"
                  onClick={handleBenchmark}
                  disabled={benchmarking}
                  className="px-2.5 py-1 text-xs font-medium rounded-lg bg-emerald-600 hover:bg-emerald-700 text-white flex items-center gap-1.5 transition cursor-pointer shadow-xs"
                >
                  <Gauge className="w-3 h-3" />
                  <span>{benchmarking ? 'Probing...' : 'Run Benchmark'}</span>
                </button>
              </div>
            </div>

            {/* Metrics cards */}
            <div className="grid grid-cols-2 md:grid-cols-4 gap-3 text-xs">
              <div className="p-3 rounded-lg bg-white dark:bg-slate-800/80 border border-slate-200 dark:border-slate-700">
                <span className="text-slate-500 block text-[11px]">Hit Rate</span>
                <span className="text-lg font-extrabold text-emerald-600 dark:text-emerald-400">
                  {metrics?.hit_rate_percent !== undefined ? `${metrics.hit_rate_percent}%` : '--'}
                </span>
                <div className="w-full bg-slate-200 dark:bg-slate-700 h-1.5 rounded-full overflow-hidden mt-1.5">
                  <div
                    className="bg-emerald-500 h-full rounded-full transition-all duration-500"
                    style={{ width: `${metrics?.hit_rate_percent || 0}%` }}
                  />
                </div>
              </div>

              <div className="p-3 rounded-lg bg-white dark:bg-slate-800/80 border border-slate-200 dark:border-slate-700">
                <span className="text-slate-500 block text-[11px]">Memory Used</span>
                <span className="text-lg font-extrabold text-slate-800 dark:text-slate-100">
                  {metrics?.used_memory_human || (metrics?.used_memory_bytes ? `${Math.round(metrics.used_memory_bytes / 1024 / 1024)} MB` : '--')}
                </span>
                <span className="text-[10px] text-slate-400 block mt-0.5">Cap: {maxMemoryMB} MB</span>
              </div>

              <div className="p-3 rounded-lg bg-white dark:bg-slate-800/80 border border-slate-200 dark:border-slate-700">
                <span className="text-slate-500 block text-[11px]">Active Clients</span>
                <span className="text-lg font-extrabold text-slate-800 dark:text-slate-100">
                  {metrics?.connected_clients ?? '--'}
                </span>
                <span className="text-[10px] text-slate-400 block mt-0.5">Total Keys: {metrics?.total_keys ?? '--'}</span>
              </div>

              <div className="p-3 rounded-lg bg-white dark:bg-slate-800/80 border border-slate-200 dark:border-slate-700">
                <span className="text-slate-500 block text-[11px]">Hits / Misses</span>
                <span className="text-lg font-extrabold text-slate-800 dark:text-slate-100">
                  {metrics?.hits ?? 0} <span className="text-slate-400 text-xs font-normal">/ {metrics?.misses ?? 0}</span>
                </span>
                <span className="text-[10px] text-slate-400 block mt-0.5">
                  Uptime: {metrics?.uptime_seconds ? `${Math.round(metrics.uptime_seconds / 3600)}h` : '--'}
                </span>
              </div>
            </div>

            {/* Benchmark Output Pill */}
            {benchmarkResult && (
              <div className="p-3 rounded-lg bg-emerald-500/10 border border-emerald-500/30 flex items-center justify-between text-xs">
                <div className="flex items-center gap-2">
                  <CheckCircle2 className="w-4 h-4 text-emerald-500" />
                  <span className="font-semibold text-emerald-700 dark:text-emerald-300">
                    Benchmark Probe: {benchmarkResult.operations} Ops in {benchmarkResult.total_duration_ms}ms
                  </span>
                </div>
                <div className="flex items-center gap-4 text-emerald-800 dark:text-emerald-200">
                  <span>Avg Latency: <strong>{benchmarkResult.avg_latency_ms} ms</strong></span>
                  <span>Min/Max: <strong>{benchmarkResult.min_latency_ms} / {benchmarkResult.max_latency_ms} ms</strong></span>
                  <span>Throughput: <strong>{benchmarkResult.ops_per_sec.toLocaleString()} ops/s</strong></span>
                </div>
              </div>
            )}
          </div>

          {/* 4. WordPress Object Cache 1-Click Card */}
          <div className="p-4 rounded-xl bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 flex items-center justify-between gap-4">
            <div className="space-y-1">
              <div className="flex items-center gap-2">
                <span className="text-xs font-bold">WordPress Object Cache (wp-config.php)</span>
                {wpObjectCacheEnabled ? (
                  <span className="px-2 py-0.5 text-[10px] font-bold rounded-full bg-emerald-100 dark:bg-emerald-950/40 text-emerald-700 dark:text-emerald-400 border border-emerald-300 dark:border-emerald-800">
                    ENABLED
                  </span>
                ) : (
                  <span className="px-2 py-0.5 text-[10px] font-medium rounded-full bg-slate-100 dark:bg-slate-800 text-slate-500">
                    DISABLED
                  </span>
                )}
              </div>
              <p className="text-xs text-slate-500">
                Injects <code className="px-1 py-0.5 bg-slate-100 dark:bg-slate-800 rounded text-[11px]">WP_CACHE</code> and socket definitions directly into WordPress. Eliminates query overhead on every page view.
              </p>
            </div>

            <button
              type="button"
              onClick={() => handleToggleWP(!wpObjectCacheEnabled)}
              className={`px-3 py-1.5 rounded-lg text-xs font-bold transition flex items-center gap-1.5 cursor-pointer ${
                wpObjectCacheEnabled
                  ? 'bg-rose-50 dark:bg-rose-950/40 text-rose-700 dark:text-rose-300 border border-rose-300 dark:border-rose-800 hover:bg-rose-100'
                  : 'bg-emerald-600 hover:bg-emerald-700 text-white shadow-xs'
              }`}
            >
              <Zap className="w-3.5 h-3.5" />
              <span>{wpObjectCacheEnabled ? 'Disable Object Cache' : '1-Click Enable for WordPress'}</span>
            </button>
          </div>

          {/* 5. Advanced Settings (Socket vs TCP, Memory Limits, Eviction) */}
          <div className="p-4 rounded-xl bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 space-y-4 text-xs">
            <div className="flex items-center gap-2 border-b border-slate-200 dark:border-slate-800 pb-2">
              <Sliders className="w-4 h-4 text-slate-500" />
              <h4 className="font-bold text-slate-700 dark:text-slate-200 uppercase tracking-wider text-[11px]">
                Connection & Memory Parameters
              </h4>
            </div>

            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              {/* Connection Type */}
              <div className="space-y-1.5">
                <label className="font-semibold text-slate-700 dark:text-slate-300 block">Connection Architecture</label>
                <div className="grid grid-cols-2 gap-2">
                  <button
                    type="button"
                    onClick={() => setConnectionType('socket')}
                    className={`p-2 rounded-lg border text-left cursor-pointer transition ${
                      connectionType === 'socket'
                        ? 'border-emerald-500 bg-emerald-50/50 dark:bg-emerald-950/20 text-emerald-800 dark:text-emerald-300 font-bold'
                        : 'border-slate-200 dark:border-slate-700 hover:bg-slate-50 dark:hover:bg-slate-800'
                    }`}
                  >
                    <span>Isolated Unix Socket</span>
                    <span className="block text-[10px] text-slate-400 font-normal mt-0.5">Zero TCP overhead, maximum security</span>
                  </button>
                  <button
                    type="button"
                    onClick={() => setConnectionType('tcp')}
                    className={`p-2 rounded-lg border text-left cursor-pointer transition ${
                      connectionType === 'tcp'
                        ? 'border-emerald-500 bg-emerald-50/50 dark:bg-emerald-950/20 text-emerald-800 dark:text-emerald-300 font-bold'
                        : 'border-slate-200 dark:border-slate-700 hover:bg-slate-50 dark:hover:bg-slate-800'
                    }`}
                  >
                    <span>127.0.0.1 Loopback</span>
                    <span className="block text-[10px] text-slate-400 font-normal mt-0.5">Strict localhost TCP binding</span>
                  </button>
                </div>
              </div>

              {/* Memory Limit */}
              <div className="space-y-1.5">
                <div className="flex justify-between">
                  <label className="font-semibold text-slate-700 dark:text-slate-300">Memory Cap (MB)</label>
                  <span className="font-bold text-emerald-600 dark:text-emerald-400">{maxMemoryMB} MB</span>
                </div>
                <input
                  type="range"
                  min="64"
                  max="2048"
                  step="64"
                  value={maxMemoryMB}
                  onChange={(e) => setMaxMemoryMB(parseInt(e.target.value))}
                  className="w-full accent-emerald-500 cursor-pointer"
                />
                <div className="flex justify-between text-[10px] text-slate-400">
                  <span>64 MB</span>
                  <span>512 MB</span>
                  <span>1024 MB</span>
                  <span>2048 MB</span>
                </div>
              </div>

              {/* Socket Path or TCP details */}
              {connectionType === 'socket' ? (
                <div className="space-y-1.5 md:col-span-2">
                  <label className="font-semibold text-slate-700 dark:text-slate-300">Dedicated Socket File Path</label>
                  <input
                    type="text"
                    value={socketPath}
                    onChange={(e) => setSocketPath(e.target.value)}
                    className="w-full p-2 text-xs font-mono rounded-lg border border-slate-300 dark:border-slate-700 bg-slate-50 dark:bg-slate-800/80 text-slate-900 dark:text-slate-100"
                    placeholder="/var/run/redis/redis-site.sock"
                  />
                  <span className="text-[11px] text-slate-400 flex items-center gap-1">
                    <Lock className="w-3 h-3 text-emerald-500" />
                    Automated chmod 770 file permission isolation applied.
                  </span>
                </div>
              ) : (
                <>
                  <div className="space-y-1.5">
                    <label className="font-semibold text-slate-700 dark:text-slate-300">Loopback Host</label>
                    <input
                      type="text"
                      disabled
                      value="127.0.0.1 (Strict Localhost Only)"
                      className="w-full p-2 text-xs font-mono rounded-lg border border-slate-200 dark:border-slate-700 bg-slate-100 dark:bg-slate-800 text-slate-500 cursor-not-allowed"
                    />
                  </div>
                  <div className="space-y-1.5">
                    <label className="font-semibold text-slate-700 dark:text-slate-300">Port</label>
                    <input
                      type="number"
                      value={port}
                      onChange={(e) => setPort(parseInt(e.target.value) || 6379)}
                      className="w-full p-2 text-xs font-mono rounded-lg border border-slate-300 dark:border-slate-700 bg-white dark:bg-slate-800 text-slate-900 dark:text-slate-100"
                    />
                  </div>
                </>
              )}

              {/* Eviction Policy */}
              <div className="space-y-1.5">
                <label className="font-semibold text-slate-700 dark:text-slate-300">Eviction Policy</label>
                <select
                  value={evictionPolicy}
                  onChange={(e) => setEvictionPolicy(e.target.value)}
                  className="w-full p-2 text-xs rounded-lg border border-slate-300 dark:border-slate-700 bg-white dark:bg-slate-800 text-slate-900 dark:text-slate-100"
                >
                  <option value="allkeys-lru">allkeys-lru (Evict least recently used - Recommended)</option>
                  <option value="volatile-lru">volatile-lru (Evict least recently used with TTL)</option>
                  <option value="allkeys-lfu">allkeys-lfu (Evict least frequently used)</option>
                  <option value="volatile-ttl">volatile-ttl (Evict keys with shortest TTL)</option>
                  <option value="noeviction">noeviction (Return error when memory limit reached)</option>
                </select>
              </div>

              {/* Password Auth */}
              <div className="space-y-1.5">
                <label className="font-semibold text-slate-700 dark:text-slate-300">Password / Token (Optional)</label>
                <input
                  type="password"
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  placeholder="Leave empty for passwordless socket"
                  className="w-full p-2 text-xs font-mono rounded-lg border border-slate-300 dark:border-slate-700 bg-white dark:bg-slate-800 text-slate-900 dark:text-slate-100"
                />
              </div>
            </div>
          </div>
        </>
      )}

      {/* 6. Footer Actions */}
      <div className="flex items-center justify-between pt-4 border-t border-slate-200 dark:border-slate-800">
        <div>
          {cacheType !== 'none' && (
            <button
              type="button"
              onClick={handleDisableCache}
              disabled={saving}
              className="px-3 py-1.5 rounded-lg border border-rose-300 dark:border-rose-800/80 text-rose-600 dark:text-rose-400 hover:bg-rose-50 dark:hover:bg-rose-950/30 text-xs font-bold transition cursor-pointer"
            >
              Disable Caching
            </button>
          )}
        </div>

        <button
          type="button"
          onClick={handleSaveConfig}
          disabled={saving}
          className="px-5 py-2 rounded-lg bg-emerald-600 hover:bg-emerald-700 active:bg-emerald-800 text-white text-xs font-bold transition shadow-xs flex items-center gap-2 cursor-pointer disabled:opacity-50"
        >
          {saving ? (
            <>
              <RefreshCw className="w-3.5 h-3.5 animate-spin" />
              <span>Applying Configuration...</span>
            </>
          ) : (
            <span>Save & Apply Caching Engine</span>
          )}
        </button>
      </div>
    </div>
  );
}
