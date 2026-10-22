'use client';

import React, { useState, useEffect, useCallback } from 'react';
import {
  Server as ServerIcon,
  Plus,
  Copy,
  Check,
  Terminal,
  Activity,
  Cpu,
  HardDrive,
  Clock,
  Shield,
  RefreshCw,
  Search,
  X,
  AlertTriangle,
  ArrowRightLeft,
  Trash2,
  KeyRound,
  Play,
  RotateCcw,
  CheckCircle2,
  XCircle,
  Layers,
  Globe,
  Database,
  ArrowRight,
  ChevronRight,
  Info,
} from 'lucide-react';
import { DashboardShell } from '@/components/DashboardShell';
import {
  apiFetch,
  Server as ServerModel,
  EnrollmentTokenResponse,
  ServerWorkloadSummary,
  CrossNodeMigration,
  PreflightCheckResponse,
  Website,
  Database as DatabaseModel,
  toggleServerMaintenance,
  drainServerNode,
  getServerWorkloads,
  deleteServer,
  rotateServerAgentKey,
  preflightMigration,
  createCrossNodeMigration,
  listCrossNodeMigrations,
  cutoverMigration,
  rollbackMigration,
} from '@/lib/api';

export default function ServersPage() {
  const [activeTab, setActiveTab] = useState<'nodes' | 'migrations'>('nodes');
  const [servers, setServers] = useState<ServerModel[]>([]);
  const [websites, setWebsites] = useState<Website[]>([]);
  const [databases, setDatabases] = useState<DatabaseModel[]>([]);
  const [migrations, setMigrations] = useState<CrossNodeMigration[]>([]);
  const [workloadsMap, setWorkloadsMap] = useState<Record<string, ServerWorkloadSummary>>({});
  const [loading, setLoading] = useState(true);
  const [searchQuery, setSearchQuery] = useState('');

  // Enrollment Modal State
  const [enrollModalOpen, setEnrollModalOpen] = useState(false);
  const [enrollLabel, setEnrollLabel] = useState('');
  const [enrollTokenData, setEnrollTokenData] = useState<EnrollmentTokenResponse | null>(null);
  const [generatingToken, setGeneratingToken] = useState(false);
  const [copied, setCopied] = useState(false);

  // Key Rotation Modal State
  const [rotateKeyModal, setRotateKeyModal] = useState<{ open: boolean; server: ServerModel | null; newKey?: string }>({
    open: false,
    server: null,
  });

  // Drain / Workload Modal State
  const [drainModal, setDrainModal] = useState<{ open: boolean; server: ServerModel | null; summary?: ServerWorkloadSummary }>({
    open: false,
    server: null,
  });

  // Delete Server Modal State
  const [deleteModal, setDeleteModal] = useState<{ open: boolean; server: ServerModel | null; summary?: ServerWorkloadSummary; force: boolean }>({
    open: false,
    server: null,
    force: false,
  });

  // Migration Modal State
  const [newMigrationModal, setNewMigrationModal] = useState(false);
  const [migSourceId, setMigSourceId] = useState('');
  const [migDestId, setMigDestId] = useState('');
  const [migWebsiteId, setMigWebsiteId] = useState('');
  const [migDatabaseId, setMigDatabaseId] = useState('');
  const [preflightData, setPreflightData] = useState<PreflightCheckResponse | null>(null);
  const [preflightLoading, setPreflightLoading] = useState(false);
  const [migratingLoading, setMigratingLoading] = useState(false);

  // View Logs Modal State
  const [viewLogsMigration, setViewLogsMigration] = useState<CrossNodeMigration | null>(null);

  // Action status notification
  const [toast, setToast] = useState<{ message: string; type: 'success' | 'error' } | null>(null);

  const showToast = (message: string, type: 'success' | 'error' = 'success') => {
    setToast({ message, type });
    setTimeout(() => setToast(null), 4000);
  };

  const fetchFleetData = useCallback(async () => {
    setLoading(true);
    try {
      const [serversRes, sitesRes, dbsRes, migsRes] = await Promise.all([
        apiFetch<ServerModel[]>('/api/v1/servers'),
        apiFetch<Website[]>('/api/v1/websites'),
        apiFetch<DatabaseModel[]>('/api/v1/databases'),
        listCrossNodeMigrations(),
      ]);

      if (serversRes.success && serversRes.data) {
        setServers(serversRes.data);

        // Fetch workload summaries for each server
        const summaryPromises = serversRes.data.map(async (s) => {
          const wRes = await getServerWorkloads(s.id);
          if (wRes.success && wRes.data) {
            return { id: s.id, summary: wRes.data };
          }
          return null;
        });
        const summaries = await Promise.all(summaryPromises);
        const map: Record<string, ServerWorkloadSummary> = {};
        summaries.forEach((item) => {
          if (item) map[item.id] = item.summary;
        });
        setWorkloadsMap(map);
      }

      if (sitesRes.success && sitesRes.data) setWebsites(sitesRes.data);
      if (dbsRes.success && dbsRes.data) setDatabases(dbsRes.data);
      if (migsRes.success && migsRes.data) setMigrations(migsRes.data);
    } catch {
      showToast('Failed to load server fleet telemetry', 'error');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchFleetData();
  }, [fetchFleetData]);

  // Handle Maintenance Toggle
  const handleToggleMaintenance = async (server: ServerModel) => {
    const nextState = !server.is_maintenance;
    const res = await toggleServerMaintenance(server.id, nextState);
    if (res.success) {
      showToast(`${server.name} maintenance mode ${nextState ? 'enabled' : 'disabled'}`);
      fetchFleetData();
    } else {
      showToast(res.error?.message || 'Failed to toggle maintenance', 'error');
    }
  };

  // Handle Drain Node
  const handleExecuteDrain = async (server: ServerModel, nextState: boolean) => {
    const res = await drainServerNode(server.id, nextState);
    if (res.success) {
      showToast(`Node ${server.name} draining ${nextState ? 'activated' : 'deactivated'}`);
      setDrainModal({ open: false, server: null });
      fetchFleetData();
    } else {
      showToast(res.error?.message || 'Failed to update draining state', 'error');
    }
  };

  // Handle Rotate Agent Key
  const handleRotateKey = async (server: ServerModel) => {
    const res = await rotateServerAgentKey(server.id);
    if (res.success && res.data) {
      setRotateKeyModal({ open: true, server, newKey: res.data.agent_key });
      showToast(`Agent mutual auth key rotated for ${server.name}`);
    } else {
      showToast(res.error?.message || 'Failed to rotate key', 'error');
    }
  };

  // Handle Delete Server
  const handleDeleteServer = async (server: ServerModel, force: boolean) => {
    const res = await deleteServer(server.id, force);
    if (res.success) {
      showToast(`Server ${server.name} decommissioned successfully`);
      setDeleteModal({ open: false, server: null, force: false });
      fetchFleetData();
    } else {
      showToast(res.error?.message || 'Failed to decommission server', 'error');
    }
  };

  // Run Preflight Check
  const handleRunPreflight = async () => {
    if (!migSourceId || !migDestId) {
      showToast('Select both source and destination nodes', 'error');
      return;
    }
    if (migSourceId === migDestId) {
      showToast('Source and destination nodes must be different', 'error');
      return;
    }
    setPreflightLoading(true);
    setPreflightData(null);
    const res = await preflightMigration({
      source_server_id: migSourceId,
      destination_server_id: migDestId,
      website_id: migWebsiteId || undefined,
      database_id: migDatabaseId || undefined,
    });
    setPreflightLoading(false);
    if (res.success && res.data) {
      setPreflightData(res.data);
    } else {
      showToast(res.error?.message || 'Preflight check failed', 'error');
    }
  };

  // Dispatch Migration Job
  const handleStartMigration = async () => {
    setMigratingLoading(true);
    const res = await createCrossNodeMigration({
      source_server_id: migSourceId,
      destination_server_id: migDestId,
      website_id: migWebsiteId || undefined,
      database_id: migDatabaseId || undefined,
    });
    setMigratingLoading(false);
    if (res.success && res.data) {
      showToast('Cross-node migration dispatched successfully');
      setNewMigrationModal(false);
      setPreflightData(null);
      setMigSourceId('');
      setMigDestId('');
      setMigWebsiteId('');
      setMigDatabaseId('');
      setActiveTab('migrations');
      fetchFleetData();
    } else {
      showToast(res.error?.message || 'Failed to dispatch migration', 'error');
    }
  };

  // Handle Cutover
  const handleCutover = async (migId: string) => {
    const res = await cutoverMigration(migId);
    if (res.success) {
      showToast('Atomic traffic cutover executed successfully');
      fetchFleetData();
    } else {
      showToast(res.error?.message || 'Cutover failed', 'error');
    }
  };

  // Handle Rollback
  const handleRollback = async (migId: string) => {
    const res = await rollbackMigration(migId);
    if (res.success) {
      showToast('Migration rolled back safely');
      fetchFleetData();
    } else {
      showToast(res.error?.message || 'Rollback failed', 'error');
    }
  };

  // Handle Generate Enrollment Token
  const handleGenerateEnrollment = async (e: React.FormEvent) => {
    e.preventDefault();
    setGeneratingToken(true);
    const res = await apiFetch<EnrollmentTokenResponse>('/api/v1/servers/enrollment-tokens', {
      method: 'POST',
      body: JSON.stringify({
        label: enrollLabel || 'Production Node',
        expiration_minutes: 120,
      }),
    });
    setGeneratingToken(false);
    if (res.success && res.data) {
      setEnrollTokenData(res.data);
    } else {
      showToast(res.error?.message || 'Failed to generate enrollment token', 'error');
    }
  };

  const handleCopyCommand = (text: string) => {
    navigator.clipboard.writeText(text);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  // Filtered servers
  const filteredServers = servers.filter(
    (s) =>
      s.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
      s.hostname.toLowerCase().includes(searchQuery.toLowerCase()) ||
      s.ip_address.includes(searchQuery)
  );

  // Aggregated Telemetry
  const totalCores = servers.reduce((acc, s) => acc + (s.cpu_cores || 0), 0);
  const totalRAMGB = servers.reduce((acc, s) => acc + (s.ram_total_mb || 0) / 1024, 0);
  const onlineCount = servers.filter((s) => s.status === 'online').length;
  const maintenanceCount = servers.filter((s) => s.is_maintenance || s.status === 'maintenance').length;
  const drainingCount = servers.filter((s) => s.draining).length;
  const totalWorkloads = Object.values(workloadsMap).reduce((acc, w) => acc + (w.active_workloads || 0), 0);

  return (
    <DashboardShell>
      <div className="space-y-6">
        {/* Toast Notification */}
        {toast && (
          <div
            className={`fixed top-4 right-4 z-50 flex items-center gap-2.5 px-4 py-3 rounded-xl shadow-lg border text-sm font-medium transition-all ${
              toast.type === 'success'
                ? 'bg-emerald-50 border-emerald-200 text-emerald-800'
                : 'bg-rose-50 border-rose-200 text-rose-800'
            }`}
          >
            {toast.type === 'success' ? (
              <CheckCircle2 className="w-4 h-4 text-[#16A34A] flex-shrink-0" />
            ) : (
              <XCircle className="w-4 h-4 text-rose-600 flex-shrink-0" />
            )}
            <span>{toast.message}</span>
          </div>
        )}

        {/* Header */}
        <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4 border-b border-slate-200 pb-5">
          <div>
            <h1 className="text-2xl font-bold tracking-tight text-slate-900">Multi-Server Fleet Management</h1>
            <p className="text-sm text-slate-500 mt-1">
              Enterprise cluster orchestration, node draining, telemetry monitoring, and zero-downtime cross-node migrations.
            </p>
          </div>
          <div className="flex items-center gap-3">
            <button
              onClick={fetchFleetData}
              className="p-2.5 rounded-xl border border-slate-200 text-slate-600 hover:text-slate-900 hover:bg-slate-50 transition-colors"
              title="Refresh Fleet Telemetry"
            >
              <RefreshCw className={`w-4 h-4 ${loading ? 'animate-spin' : ''}`} />
            </button>
            <button
              onClick={() => {
                setMigSourceId('');
                setMigDestId('');
                setMigWebsiteId('');
                setMigDatabaseId('');
                setPreflightData(null);
                setNewMigrationModal(true);
              }}
              className="inline-flex items-center gap-2 px-3.5 py-2.5 rounded-xl border border-slate-300 bg-white hover:bg-slate-50 text-slate-700 text-sm font-semibold shadow-xs transition-all"
            >
              <ArrowRightLeft className="w-4 h-4 text-slate-500" />
              Migrate Workload
            </button>
            <button
              onClick={() => {
                setEnrollModalOpen(true);
                setEnrollTokenData(null);
                setEnrollLabel('');
              }}
              className="inline-flex items-center gap-2 px-4 py-2.5 rounded-xl bg-[#16A34A] hover:bg-[#15803D] text-white text-sm font-semibold shadow-xs transition-all"
            >
              <Plus className="w-4 h-4" />
              Enroll Server
            </button>
          </div>
        </div>

        {/* Fleet Aggregation Metrics */}
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
          <div className="bg-white border border-slate-200 rounded-2xl p-5 shadow-xs flex items-center justify-between">
            <div>
              <p className="text-xs font-semibold text-slate-500 uppercase tracking-wider">Fleet Nodes</p>
              <p className="text-2xl font-bold text-slate-900 mt-1">{servers.length}</p>
              <p className="text-xs text-emerald-600 font-medium mt-1">
                {onlineCount} Online • {servers.length - onlineCount} Offline
              </p>
            </div>
            <div className="w-12 h-12 rounded-xl bg-emerald-50 border border-emerald-100 flex items-center justify-center text-[#16A34A]">
              <ServerIcon className="w-6 h-6" />
            </div>
          </div>

          <div className="bg-white border border-slate-200 rounded-2xl p-5 shadow-xs flex items-center justify-between">
            <div>
              <p className="text-xs font-semibold text-slate-500 uppercase tracking-wider">Cluster Hardware</p>
              <p className="text-2xl font-bold text-slate-900 mt-1">{totalCores} vCPU</p>
              <p className="text-xs text-slate-500 font-medium mt-1">
                {totalRAMGB.toFixed(0)} GB Total Cluster RAM
              </p>
            </div>
            <div className="w-12 h-12 rounded-xl bg-blue-50 border border-blue-100 flex items-center justify-center text-blue-600">
              <Cpu className="w-6 h-6" />
            </div>
          </div>

          <div className="bg-white border border-slate-200 rounded-2xl p-5 shadow-xs flex items-center justify-between">
            <div>
              <p className="text-xs font-semibold text-slate-500 uppercase tracking-wider">Hosted Workloads</p>
              <p className="text-2xl font-bold text-slate-900 mt-1">{totalWorkloads}</p>
              <p className="text-xs text-slate-500 font-medium mt-1">
                {websites.length} Websites • {databases.length} Databases
              </p>
            </div>
            <div className="w-12 h-12 rounded-xl bg-indigo-50 border border-indigo-100 flex items-center justify-center text-indigo-600">
              <Layers className="w-6 h-6" />
            </div>
          </div>

          <div className="bg-white border border-slate-200 rounded-2xl p-5 shadow-xs flex items-center justify-between">
            <div>
              <p className="text-xs font-semibold text-slate-500 uppercase tracking-wider">Node Status States</p>
              <p className="text-2xl font-bold text-slate-900 mt-1">
                {maintenanceCount} <span className="text-xs font-normal text-slate-500">Maint</span> / {drainingCount}{' '}
                <span className="text-xs font-normal text-slate-500">Drain</span>
              </p>
              <p className="text-xs text-slate-500 font-medium mt-1">
                {migrations.filter((m) => m.status !== 'completed' && m.status !== 'rolled_back').length} Active Migrations
              </p>
            </div>
            <div className="w-12 h-12 rounded-xl bg-amber-50 border border-amber-100 flex items-center justify-center text-amber-600">
              <Shield className="w-6 h-6" />
            </div>
          </div>
        </div>

        {/* Tab Navigation */}
        <div className="border-b border-slate-200 flex gap-6">
          <button
            onClick={() => setActiveTab('nodes')}
            className={`pb-3 text-sm font-semibold flex items-center gap-2 border-b-2 transition-all ${
              activeTab === 'nodes'
                ? 'border-[#16A34A] text-[#16A34A]'
                : 'border-transparent text-slate-500 hover:text-slate-800'
            }`}
          >
            <ServerIcon className="w-4 h-4" />
            Cluster Nodes ({servers.length})
          </button>
          <button
            onClick={() => setActiveTab('migrations')}
            className={`pb-3 text-sm font-semibold flex items-center gap-2 border-b-2 transition-all ${
              activeTab === 'migrations'
                ? 'border-[#16A34A] text-[#16A34A]'
                : 'border-transparent text-slate-500 hover:text-slate-800'
            }`}
          >
            <ArrowRightLeft className="w-4 h-4" />
            Cross-Node Migrations ({migrations.length})
          </button>
        </div>

        {/* TAB 1: CLUSTER NODES */}
        {activeTab === 'nodes' && (
          <div className="space-y-4">
            {/* Search Filter Bar */}
            <div className="flex flex-col sm:flex-row items-center justify-between gap-4 bg-white border border-slate-200 rounded-2xl p-3 shadow-xs">
              <div className="flex items-center gap-3 w-full sm:w-96 bg-slate-50 border border-slate-200 rounded-xl px-3.5 py-2 focus-within:border-[#16A34A] focus-within:ring-2 focus-within:ring-[#16A34A]/10 transition-all">
                <Search className="w-4 h-4 text-slate-400 flex-shrink-0" />
                <input
                  type="text"
                  placeholder="Search by server name, hostname, or IP..."
                  value={searchQuery}
                  onChange={(e) => setSearchQuery(e.target.value)}
                  className="w-full bg-transparent text-xs font-medium text-slate-900 placeholder:text-slate-400 focus:outline-none"
                />
                {searchQuery && (
                  <button onClick={() => setSearchQuery('')} className="text-slate-400 hover:text-slate-600 p-0.5">
                    <X className="w-3.5 h-3.5" />
                  </button>
                )}
              </div>
              <div className="text-xs font-semibold text-slate-500 px-2 self-end sm:self-center">
                Showing {filteredServers.length} of {servers.length} nodes
              </div>
            </div>

            {/* Nodes Grid */}
            {loading ? (
              <div className="py-20 text-center text-slate-400">
                <div className="w-8 h-8 border-2 border-[#16A34A] border-t-transparent rounded-full animate-spin mx-auto mb-3" />
                Querying cluster telemetry and status...
              </div>
            ) : filteredServers.length === 0 ? (
              <div className="py-16 px-4 rounded-2xl border border-dashed border-slate-300 text-center bg-slate-50/50">
                <div className="w-14 h-14 rounded-2xl bg-slate-100 flex items-center justify-center mx-auto mb-4 text-slate-400">
                  <ServerIcon className="w-7 h-7" />
                </div>
                <h3 className="text-lg font-bold text-slate-900">No Servers Connected</h3>
                <p className="text-sm text-slate-500 mt-1 max-w-md mx-auto">
                  Enroll your Ubuntu, Debian, or RHEL-based Linux nodes to manage high-availability hosting fleets.
                </p>
                <button
                  onClick={() => setEnrollModalOpen(true)}
                  className="mt-5 inline-flex items-center gap-2 px-5 py-2.5 rounded-xl bg-[#16A34A] hover:bg-[#15803D] text-white text-sm font-semibold shadow-xs transition-all"
                >
                  <Plus className="w-4 h-4" />
                  Enroll First Server
                </button>
              </div>
            ) : (
              <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
                {filteredServers.map((server) => {
                  const summary = workloadsMap[server.id];
                  const isMaintenance = server.is_maintenance || server.status === 'maintenance';
                  const isDraining = server.draining;

                  return (
                    <div
                      key={server.id}
                      className="bg-white border border-slate-200 rounded-2xl p-5 shadow-xs hover:border-slate-300 transition-all flex flex-col justify-between"
                    >
                      <div>
                        {/* Node Header */}
                        <div className="flex items-start justify-between gap-3 mb-3">
                          <div className="flex items-center gap-3">
                            <div className="w-10 h-10 rounded-xl bg-slate-100 border border-slate-200 flex items-center justify-center text-slate-700">
                              <ServerIcon className="w-5 h-5" />
                            </div>
                            <div>
                              <h3 className="font-bold text-base text-slate-900">{server.name}</h3>
                              <p className="text-xs text-slate-500 font-mono">{server.hostname}</p>
                            </div>
                          </div>
                          <div className="flex flex-col items-end gap-1">
                            <span
                              className={`inline-flex items-center gap-1.5 px-2.5 py-0.5 rounded-full text-[11px] font-semibold capitalize ${
                                server.status === 'online'
                                  ? 'bg-emerald-50 text-emerald-700 border border-emerald-200'
                                  : 'bg-slate-100 text-slate-600 border border-slate-200'
                              }`}
                            >
                              <span
                                className={`w-1.5 h-1.5 rounded-full ${
                                  server.status === 'online' ? 'bg-[#16A34A]' : 'bg-slate-400'
                                }`}
                              />
                              {server.status}
                            </span>
                            {isMaintenance && (
                              <span className="inline-flex items-center px-2 py-0.5 rounded-full text-[10px] font-semibold bg-amber-50 text-amber-700 border border-amber-200">
                                Maintenance
                              </span>
                            )}
                            {isDraining && (
                              <span className="inline-flex items-center px-2 py-0.5 rounded-full text-[10px] font-semibold bg-purple-50 text-purple-700 border border-purple-200">
                                Draining
                              </span>
                            )}
                          </div>
                        </div>

                        {/* Hardware & OS Specs */}
                        <div className="space-y-1.5 text-xs py-3 border-y border-slate-100 text-slate-600">
                          <div className="flex justify-between">
                            <span className="text-slate-400">IP Address:</span>
                            <span className="font-mono font-medium text-slate-800">{server.ip_address}</span>
                          </div>
                          <div className="flex justify-between">
                            <span className="text-slate-400">Operating System:</span>
                            <span className="font-medium text-slate-800">
                              {server.os_name} {server.os_version} ({server.architecture})
                            </span>
                          </div>
                          <div className="flex justify-between">
                            <span className="text-slate-400">Hardware Allocation:</span>
                            <span className="font-medium text-slate-800">
                              {server.cpu_cores} vCPU • {(server.ram_total_mb / 1024).toFixed(1)} GB RAM • {server.disk_total_gb} GB Disk
                            </span>
                          </div>
                          <div className="flex justify-between">
                            <span className="text-slate-400">Agent Daemon:</span>
                            <span className="font-mono text-emerald-600">v{server.agent_version}</span>
                          </div>
                        </div>

                        {/* Workload Summary Badges */}
                        <div className="pt-3 pb-1">
                          <p className="text-[11px] font-semibold text-slate-400 uppercase tracking-wider mb-2">
                            Assigned Workloads ({summary ? summary.active_workloads : 0})
                          </p>
                          <div className="grid grid-cols-3 gap-2 text-center">
                            <div className="bg-slate-50 border border-slate-100 rounded-lg py-1.5 px-2">
                              <span className="block text-xs font-bold text-slate-900">
                                {summary ? summary.websites_count : 0}
                              </span>
                              <span className="text-[10px] text-slate-500 font-medium">Websites</span>
                            </div>
                            <div className="bg-slate-50 border border-slate-100 rounded-lg py-1.5 px-2">
                              <span className="block text-xs font-bold text-slate-900">
                                {summary ? summary.databases_count : 0}
                              </span>
                              <span className="text-[10px] text-slate-500 font-medium">Databases</span>
                            </div>
                            <div className="bg-slate-50 border border-slate-100 rounded-lg py-1.5 px-2">
                              <span className="block text-xs font-bold text-slate-900">
                                {summary ? summary.mailboxes_count : 0}
                              </span>
                              <span className="text-[10px] text-slate-500 font-medium">Mailboxes</span>
                            </div>
                          </div>
                        </div>
                      </div>

                      {/* Node Control Toolbar */}
                      <div className="mt-4 pt-3 border-t border-slate-100 space-y-2">
                        <div className="flex items-center justify-between text-[11px] text-slate-400">
                          <span className="flex items-center gap-1">
                            <Clock className="w-3.5 h-3.5" />
                            Uptime: {Math.floor((server.uptime_seconds || 0) / 3600)}h
                          </span>
                          <span className="text-slate-500">
                            Cap: max {server.max_websites || 100} sites
                          </span>
                        </div>

                        <div className="grid grid-cols-4 gap-1.5 pt-1">
                          {/* Maintenance Toggle */}
                          <button
                            onClick={() => handleToggleMaintenance(server)}
                            className={`py-1.5 px-2 rounded-lg text-xs font-semibold border transition-all ${
                              isMaintenance
                                ? 'bg-amber-500 text-white border-amber-600'
                                : 'bg-slate-50 text-slate-700 border-slate-200 hover:bg-slate-100'
                            }`}
                            title={isMaintenance ? 'Disable Maintenance' : 'Enable Maintenance'}
                          >
                            {isMaintenance ? 'In Maint' : 'Maint'}
                          </button>

                          {/* Drain Toggle */}
                          <button
                            onClick={() => setDrainModal({ open: true, server, summary })}
                            className={`py-1.5 px-2 rounded-lg text-xs font-semibold border transition-all ${
                              isDraining
                                ? 'bg-purple-600 text-white border-purple-700'
                                : 'bg-slate-50 text-slate-700 border-slate-200 hover:bg-slate-100'
                            }`}
                            title="Drain workloads from this node"
                          >
                            {isDraining ? 'Draining' : 'Drain'}
                          </button>

                          {/* Rotate Key */}
                          <button
                            onClick={() => handleRotateKey(server)}
                            className="py-1.5 px-2 rounded-lg text-xs font-semibold bg-slate-50 text-slate-700 border border-slate-200 hover:bg-slate-100 transition-all flex items-center justify-center gap-1"
                            title="Rotate Mutual Agent Key"
                          >
                            <KeyRound className="w-3 h-3 text-slate-500" />
                            Key
                          </button>

                          {/* Decommission / Delete */}
                          <button
                            onClick={() => setDeleteModal({ open: true, server, summary, force: false })}
                            className="py-1.5 px-2 rounded-lg text-xs font-semibold bg-rose-50 text-rose-700 border border-rose-200 hover:bg-rose-100 transition-all flex items-center justify-center gap-1"
                            title="Decommission Server"
                          >
                            <Trash2 className="w-3 h-3 text-rose-600" />
                            Drop
                          </button>
                        </div>
                      </div>
                    </div>
                  );
                })}
              </div>
            )}
          </div>
        )}

        {/* TAB 2: CROSS-NODE MIGRATIONS */}
        {activeTab === 'migrations' && (
          <div className="space-y-4">
            <div className="flex items-center justify-between bg-white border border-slate-200 rounded-2xl p-4 shadow-xs">
              <div>
                <h3 className="font-bold text-slate-900 text-base">Cluster Migration Pipelines</h3>
                <p className="text-xs text-slate-500 mt-0.5">
                  Live tracking of snapshot creation, high-throughput network transfer, cryptographic SHA-256 integrity verification, and atomic cutover.
                </p>
              </div>
              <button
                onClick={() => {
                  setMigSourceId('');
                  setMigDestId('');
                  setMigWebsiteId('');
                  setMigDatabaseId('');
                  setPreflightData(null);
                  setNewMigrationModal(true);
                }}
                className="inline-flex items-center gap-2 px-4 py-2 rounded-xl bg-[#16A34A] hover:bg-[#15803D] text-white text-xs font-semibold shadow-xs transition-all"
              >
                <Plus className="w-3.5 h-3.5" />
                New Migration
              </button>
            </div>

            {migrations.length === 0 ? (
              <div className="py-16 px-4 rounded-2xl border border-dashed border-slate-300 text-center bg-slate-50/50">
                <div className="w-14 h-14 rounded-2xl bg-slate-100 flex items-center justify-center mx-auto mb-4 text-slate-400">
                  <ArrowRightLeft className="w-7 h-7" />
                </div>
                <h3 className="text-lg font-bold text-slate-900">No Active or Past Migrations</h3>
                <p className="text-sm text-slate-500 mt-1 max-w-md mx-auto">
                  Execute zero-downtime migrations between nodes with SHA-256 integrity verification and atomic cutovers.
                </p>
                <button
                  onClick={() => setNewMigrationModal(true)}
                  className="mt-5 inline-flex items-center gap-2 px-5 py-2.5 rounded-xl bg-[#16A34A] hover:bg-[#15803D] text-white text-sm font-semibold shadow-xs transition-all"
                >
                  <Plus className="w-4 h-4" />
                  Initiate First Migration
                </button>
              </div>
            ) : (
              <div className="space-y-3">
                {migrations.map((job) => {
                  const srcServer = servers.find((s) => s.id === job.source_server_id);
                  const destServer = servers.find((s) => s.id === job.destination_server_id);
                  const site = websites.find((w) => w.id === job.website_id);
                  const db = databases.find((d) => d.id === job.database_id);

                  return (
                    <div
                      key={job.id}
                      className="bg-white border border-slate-200 rounded-2xl p-5 shadow-xs hover:border-slate-300 transition-all space-y-4"
                    >
                      <div className="flex flex-col md:flex-row md:items-center justify-between gap-3">
                        <div className="flex items-center gap-3">
                          <div className="w-10 h-10 rounded-xl bg-indigo-50 border border-indigo-100 flex items-center justify-center text-indigo-600">
                            <ArrowRightLeft className="w-5 h-5" />
                          </div>
                          <div>
                            <div className="flex items-center gap-2">
                              <span className="font-bold text-sm text-slate-900 font-mono">
                                {site ? site.primary_domain : db ? db.name : 'All Node Workloads'}
                              </span>
                              <span
                                className={`px-2 py-0.5 rounded-full text-[10px] font-semibold uppercase tracking-wider ${
                                  job.status === 'completed'
                                    ? 'bg-emerald-50 text-emerald-700 border border-emerald-200'
                                    : job.status === 'cutover_ready'
                                    ? 'bg-blue-50 text-blue-700 border border-blue-200'
                                    : job.status === 'failed' || job.status === 'rolled_back'
                                    ? 'bg-rose-50 text-rose-700 border border-rose-200'
                                    : 'bg-amber-50 text-amber-700 border border-amber-200'
                                }`}
                              >
                                {job.status.replace('_', ' ')}
                              </span>
                            </div>
                            <p className="text-xs text-slate-500 flex items-center gap-2 mt-0.5">
                              <span>Source: <strong className="text-slate-700">{srcServer?.name || job.source_server_id.slice(0, 8)}</strong></span>
                              <ArrowRight className="w-3 h-3 text-slate-400" />
                              <span>Destination: <strong className="text-slate-700">{destServer?.name || job.destination_server_id.slice(0, 8)}</strong></span>
                            </p>
                          </div>
                        </div>

                        {/* Cutover & Rollback Controls */}
                        <div className="flex items-center gap-2 self-end md:self-center">
                          <button
                            onClick={() => setViewLogsMigration(job)}
                            className="px-3 py-1.5 rounded-lg border border-slate-200 bg-white hover:bg-slate-50 text-xs font-semibold text-slate-700 transition-colors"
                          >
                            Logs ({job.logs ? job.logs.length : 0})
                          </button>

                          {job.status === 'cutover_ready' && (
                            <>
                              <button
                                onClick={() => handleRollback(job.id)}
                                className="px-3 py-1.5 rounded-lg border border-slate-200 bg-white hover:bg-slate-50 text-xs font-semibold text-rose-600 transition-colors flex items-center gap-1"
                              >
                                <RotateCcw className="w-3.5 h-3.5" />
                                Rollback
                              </button>
                              <button
                                onClick={() => handleCutover(job.id)}
                                className="px-3.5 py-1.5 rounded-lg bg-[#16A34A] hover:bg-[#15803D] text-white text-xs font-semibold shadow-xs transition-all flex items-center gap-1"
                              >
                                <Check className="w-3.5 h-3.5" />
                                Cutover Traffic Now
                              </button>
                            </>
                          )}
                        </div>
                      </div>

                      {/* Progress Bar & Step */}
                      <div className="space-y-1.5">
                        <div className="flex justify-between text-xs text-slate-600">
                          <span className="font-medium text-slate-800">{job.current_step}</span>
                          <span className="font-bold">{job.progress}%</span>
                        </div>
                        <div className="w-full h-2 bg-slate-100 rounded-full overflow-hidden">
                          <div
                            className={`h-full transition-all duration-500 rounded-full ${
                              job.status === 'completed'
                                ? 'bg-[#16A34A]'
                                : job.status === 'failed' || job.status === 'rolled_back'
                                ? 'bg-rose-500'
                                : job.status === 'cutover_ready'
                                ? 'bg-blue-600 animate-pulse'
                                : 'bg-[#16A34A]'
                            }`}
                            style={{ width: `${job.progress}%` }}
                          />
                        </div>
                      </div>

                      {/* Cryptographic Verification Details */}
                      {job.source_checksum && (
                        <div className="bg-slate-50 border border-slate-200/80 rounded-xl p-3 text-[11px] font-mono text-slate-600 space-y-1">
                          <div className="flex items-center justify-between">
                            <span className="text-slate-400">Source SHA-256:</span>
                            <span className="text-slate-800 truncate max-w-xs">{job.source_checksum}</span>
                          </div>
                          <div className="flex items-center justify-between">
                            <span className="text-slate-400">Dest SHA-256:</span>
                            <span className="text-slate-800 truncate max-w-xs">{job.destination_checksum}</span>
                          </div>
                          {job.source_checksum === job.destination_checksum && (
                            <div className="text-[#16A34A] font-semibold flex items-center gap-1 pt-1 font-sans">
                              <CheckCircle2 className="w-3.5 h-3.5" />
                              Cryptographic checksums match byte-for-byte. Data integrity certified.
                            </div>
                          )}
                        </div>
                      )}
                    </div>
                  );
                })}
              </div>
            )}
          </div>
        )}

        {/* MODAL 1: NEW MIGRATION & PREFLIGHT CHECK */}
        {newMigrationModal && (
          <div className="fixed inset-0 z-50 bg-black/50 backdrop-blur-xs flex items-center justify-center p-4">
            <div className="w-full max-w-2xl bg-white border border-slate-200 rounded-2xl shadow-xl p-6 relative max-h-[90vh] overflow-y-auto">
              <button
                onClick={() => setNewMigrationModal(false)}
                className="absolute top-5 right-5 p-1 rounded-lg text-slate-400 hover:text-slate-700 hover:bg-slate-100 transition-colors"
              >
                <X className="w-5 h-5" />
              </button>

              <div className="flex items-center gap-3 mb-5">
                <div className="w-10 h-10 rounded-xl bg-emerald-50 border border-emerald-100 flex items-center justify-center text-[#16A34A]">
                  <ArrowRightLeft className="w-5 h-5" />
                </div>
                <div>
                  <h2 className="text-lg font-bold text-slate-900">Cross-Node Migration Orchestrator</h2>
                  <p className="text-xs text-slate-500">Automated preflight compatibility checks and live transfer</p>
                </div>
              </div>

              <div className="space-y-4">
                {/* Source & Destination Nodes */}
                <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
                  <div>
                    <label className="block text-xs font-semibold text-slate-700 uppercase tracking-wider mb-1.5">
                      Source Node
                    </label>
                    <select
                      value={migSourceId}
                      onChange={(e) => {
                        setMigSourceId(e.target.value);
                        setPreflightData(null);
                      }}
                      className="w-full px-3.5 py-2.5 rounded-xl border border-slate-300 bg-white text-slate-900 text-sm focus:outline-none focus:border-[#16A34A]"
                    >
                      <option value="">Select source server...</option>
                      {servers.map((s) => (
                        <option key={s.id} value={s.id}>
                          {s.name} ({s.ip_address})
                        </option>
                      ))}
                    </select>
                  </div>

                  <div>
                    <label className="block text-xs font-semibold text-slate-700 uppercase tracking-wider mb-1.5">
                      Destination Node
                    </label>
                    <select
                      value={migDestId}
                      onChange={(e) => {
                        setMigDestId(e.target.value);
                        setPreflightData(null);
                      }}
                      className="w-full px-3.5 py-2.5 rounded-xl border border-slate-300 bg-white text-slate-900 text-sm focus:outline-none focus:border-[#16A34A]"
                    >
                      <option value="">Select destination server...</option>
                      {servers
                        .filter((s) => s.id !== migSourceId)
                        .map((s) => (
                          <option key={s.id} value={s.id}>
                            {s.name} ({s.ip_address})
                          </option>
                        ))}
                    </select>
                  </div>
                </div>

                {/* Specific Workload Selection (Optional) */}
                <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
                  <div>
                    <label className="block text-xs font-semibold text-slate-700 uppercase tracking-wider mb-1.5">
                      Select Website to Migrate (Optional)
                    </label>
                    <select
                      value={migWebsiteId}
                      onChange={(e) => {
                        setMigWebsiteId(e.target.value);
                        setPreflightData(null);
                      }}
                      className="w-full px-3.5 py-2.5 rounded-xl border border-slate-300 bg-white text-slate-900 text-sm focus:outline-none focus:border-[#16A34A]"
                    >
                      <option value="">Migrate specific website...</option>
                      {websites
                        .filter((w) => !migSourceId || w.server_id === migSourceId)
                        .map((w) => (
                          <option key={w.id} value={w.id}>
                            {w.primary_domain} ({w.web_server_type || 'nginx'})
                          </option>
                        ))}
                    </select>
                  </div>

                  <div>
                    <label className="block text-xs font-semibold text-slate-700 uppercase tracking-wider mb-1.5">
                      Select Database to Migrate (Optional)
                    </label>
                    <select
                      value={migDatabaseId}
                      onChange={(e) => {
                        setMigDatabaseId(e.target.value);
                        setPreflightData(null);
                      }}
                      className="w-full px-3.5 py-2.5 rounded-xl border border-slate-300 bg-white text-slate-900 text-sm focus:outline-none focus:border-[#16A34A]"
                    >
                      <option value="">Migrate specific database...</option>
                      {databases
                        .filter((d) => !migSourceId || d.server_id === migSourceId)
                        .map((d) => (
                          <option key={d.id} value={d.id}>
                            {d.name} ({d.db_type})
                          </option>
                        ))}
                    </select>
                  </div>
                </div>

                {/* Preflight Check Button */}
                <div className="pt-2">
                  <button
                    type="button"
                    onClick={handleRunPreflight}
                    disabled={preflightLoading || !migSourceId || !migDestId}
                    className="w-full py-2.5 px-4 rounded-xl border border-slate-300 bg-slate-50 hover:bg-slate-100 disabled:opacity-50 text-slate-700 text-xs font-semibold transition-all flex items-center justify-center gap-2"
                  >
                    {preflightLoading ? (
                      <div className="w-4 h-4 border-2 border-[#16A34A] border-t-transparent rounded-full animate-spin" />
                    ) : (
                      <Shield className="w-4 h-4 text-slate-600" />
                    )}
                    Run Automated Preflight Compatibility Audit
                  </button>
                </div>

                {/* Preflight Results */}
                {preflightData && (
                  <div className="bg-slate-50 border border-slate-200 rounded-xl p-4 space-y-3">
                    <div className="flex items-center justify-between">
                      <span className="text-xs font-bold text-slate-900 flex items-center gap-1.5">
                        {preflightData.compatible ? (
                          <CheckCircle2 className="w-4 h-4 text-[#16A34A]" />
                        ) : (
                          <XCircle className="w-4 h-4 text-rose-600" />
                        )}
                        Preflight Outcome:{' '}
                        {preflightData.compatible ? 'COMPATIBLE FOR CUTOVER' : 'INCOMPATIBLE'}
                      </span>
                      <span className="text-xs font-semibold text-slate-500">
                        Estimated Payload: {preflightData.estimated_size_mb} MB
                      </span>
                    </div>

                    <div className="space-y-2 pt-1">
                      {preflightData.checks.map((check, idx) => (
                        <div
                          key={idx}
                          className="flex items-start justify-between text-xs py-1 border-b border-slate-200 last:border-0"
                        >
                          <span className="font-medium text-slate-700">{check.name}</span>
                          <span
                            className={`px-2 py-0.5 rounded text-[10px] font-semibold capitalize ${
                              check.status === 'pass'
                                ? 'bg-emerald-100 text-emerald-800'
                                : check.status === 'warn'
                                ? 'bg-amber-100 text-amber-800'
                                : 'bg-rose-100 text-rose-800'
                            }`}
                          >
                            {check.status}: {check.message}
                          </span>
                        </div>
                      ))}
                    </div>
                  </div>
                )}

                {/* Modal Action Buttons */}
                <div className="flex justify-end gap-3 pt-4 border-t border-slate-200">
                  <button
                    type="button"
                    onClick={() => setNewMigrationModal(false)}
                    className="px-4 py-2 rounded-xl text-sm font-medium text-slate-600 hover:text-slate-900 transition-colors"
                  >
                    Cancel
                  </button>
                  <button
                    type="button"
                    disabled={!preflightData || !preflightData.compatible || migratingLoading}
                    onClick={handleStartMigration}
                    className="px-5 py-2.5 rounded-xl bg-[#16A34A] hover:bg-[#15803D] disabled:opacity-50 text-white text-sm font-semibold shadow-xs transition-all flex items-center gap-2"
                  >
                    {migratingLoading ? (
                      <div className="w-4 h-4 border-2 border-white border-t-transparent rounded-full animate-spin" />
                    ) : (
                      <Play className="w-4 h-4" />
                    )}
                    Start Live Migration Pipeline
                  </button>
                </div>
              </div>
            </div>
          </div>
        )}

        {/* MODAL 2: DRAIN NODE MODAL */}
        {drainModal.open && drainModal.server && (
          <div className="fixed inset-0 z-50 bg-black/50 backdrop-blur-xs flex items-center justify-center p-4">
            <div className="w-full max-w-md bg-white border border-slate-200 rounded-2xl shadow-xl p-6 relative">
              <button
                onClick={() => setDrainModal({ open: false, server: null })}
                className="absolute top-5 right-5 p-1 rounded-lg text-slate-400 hover:text-slate-700 hover:bg-slate-100 transition-colors"
              >
                <X className="w-5 h-5" />
              </button>

              <div className="flex items-center gap-3 mb-4">
                <div className="w-10 h-10 rounded-xl bg-purple-50 border border-purple-100 flex items-center justify-center text-purple-700">
                  <Shield className="w-5 h-5" />
                </div>
                <div>
                  <h3 className="text-base font-bold text-slate-900">Node Draining Policy</h3>
                  <p className="text-xs text-slate-500">{drainModal.server.name} ({drainModal.server.ip_address})</p>
                </div>
              </div>

              <div className="space-y-3 text-xs text-slate-600">
                <p>
                  Draining stops the node from accepting new website allocations, database provisions, or mailbox bindings. Existing workloads continue serving active production traffic until migrated.
                </p>

                {drainModal.summary && (
                  <div className="bg-slate-50 border border-slate-200 rounded-xl p-3.5 space-y-1">
                    <p className="font-semibold text-slate-900">Active Workloads on this Node:</p>
                    <p>• Websites: {drainModal.summary.websites_count}</p>
                    <p>• Databases: {drainModal.summary.databases_count}</p>
                    <p>• Mailboxes: {drainModal.summary.mailboxes_count}</p>
                    <p className="font-bold text-slate-800 pt-1">
                      Total Active: {drainModal.summary.active_workloads}
                    </p>
                  </div>
                )}
              </div>

              <div className="flex justify-end gap-3 pt-5 mt-4 border-t border-slate-200">
                <button
                  onClick={() => setDrainModal({ open: false, server: null })}
                  className="px-4 py-2 rounded-xl text-xs font-semibold text-slate-600 hover:text-slate-900"
                >
                  Cancel
                </button>
                <button
                  onClick={() => handleExecuteDrain(drainModal.server!, !drainModal.server!.draining)}
                  className={`px-4 py-2 rounded-xl text-xs font-semibold text-white shadow-xs transition-all ${
                    drainModal.server.draining
                      ? 'bg-slate-700 hover:bg-slate-800'
                      : 'bg-purple-600 hover:bg-purple-700'
                  }`}
                >
                  {drainModal.server.draining ? 'Stop Draining (Resume Provisioning)' : 'Confirm Draining'}
                </button>
              </div>
            </div>
          </div>
        )}

        {/* MODAL 3: DECOMMISSION / DELETE SERVER MODAL */}
        {deleteModal.open && deleteModal.server && (
          <div className="fixed inset-0 z-50 bg-black/50 backdrop-blur-xs flex items-center justify-center p-4">
            <div className="w-full max-w-md bg-white border border-slate-200 rounded-2xl shadow-xl p-6 relative">
              <button
                onClick={() => setDeleteModal({ open: false, server: null, force: false })}
                className="absolute top-5 right-5 p-1 rounded-lg text-slate-400 hover:text-slate-700 hover:bg-slate-100 transition-colors"
              >
                <X className="w-5 h-5" />
              </button>

              <div className="flex items-center gap-3 mb-4">
                <div className="w-10 h-10 rounded-xl bg-rose-50 border border-rose-100 flex items-center justify-center text-rose-700">
                  <AlertTriangle className="w-5 h-5" />
                </div>
                <div>
                  <h3 className="text-base font-bold text-slate-900">Decommission Server Node</h3>
                  <p className="text-xs text-slate-500">{deleteModal.server.name}</p>
                </div>
              </div>

              <div className="space-y-3 text-xs text-slate-600">
                {deleteModal.summary && deleteModal.summary.active_workloads > 0 ? (
                  <div className="p-3 rounded-xl bg-amber-50 border border-amber-200 text-amber-800 space-y-1">
                    <p className="font-bold flex items-center gap-1">
                      <AlertTriangle className="w-4 h-4 text-amber-600" />
                      Active Workloads Warning
                    </p>
                    <p>
                      This server currently hosts <strong>{deleteModal.summary.active_workloads}</strong> active workloads ({deleteModal.summary.websites_count} websites, {deleteModal.summary.databases_count} databases).
                    </p>
                    <p className="pt-1">
                      To prevent catastrophic data loss, please migrate workloads to another node first. If you must forcefully drop this node record, check the box below.
                    </p>
                    <label className="flex items-center gap-2 pt-2 cursor-pointer">
                      <input
                        type="checkbox"
                        checked={deleteModal.force}
                        onChange={(e) => setDeleteModal((prev) => ({ ...prev, force: e.target.checked }))}
                        className="rounded border-amber-300 text-rose-600 focus:ring-rose-500"
                      />
                      <span className="font-semibold text-rose-700">Force decommission node (bypasses safety lock)</span>
                    </label>
                  </div>
                ) : (
                  <p>
                    Are you sure you want to decommission and remove this server from Hostvra? This action will invalidate the mutual agent authentication key.
                  </p>
                )}
              </div>

              <div className="flex justify-end gap-3 pt-5 mt-4 border-t border-slate-200">
                <button
                  onClick={() => setDeleteModal({ open: false, server: null, force: false })}
                  className="px-4 py-2 rounded-xl text-xs font-semibold text-slate-600 hover:text-slate-900"
                >
                  Cancel
                </button>
                <button
                  onClick={() => handleDeleteServer(deleteModal.server!, deleteModal.force)}
                  disabled={deleteModal.summary && deleteModal.summary.active_workloads > 0 && !deleteModal.force}
                  className="px-4 py-2 rounded-xl text-xs font-semibold bg-rose-600 hover:bg-rose-700 disabled:opacity-40 text-white shadow-xs transition-all"
                >
                  Decommission Node
                </button>
              </div>
            </div>
          </div>
        )}

        {/* MODAL 4: KEY ROTATION MODAL */}
        {rotateKeyModal.open && rotateKeyModal.server && (
          <div className="fixed inset-0 z-50 bg-black/50 backdrop-blur-xs flex items-center justify-center p-4">
            <div className="w-full max-w-md bg-white border border-slate-200 rounded-2xl shadow-xl p-6 relative">
              <button
                onClick={() => setRotateKeyModal({ open: false, server: null })}
                className="absolute top-5 right-5 p-1 rounded-lg text-slate-400 hover:text-slate-700 hover:bg-slate-100 transition-colors"
              >
                <X className="w-5 h-5" />
              </button>

              <div className="flex items-center gap-3 mb-4">
                <div className="w-10 h-10 rounded-xl bg-emerald-50 border border-emerald-100 flex items-center justify-center text-[#16A34A]">
                  <KeyRound className="w-5 h-5" />
                </div>
                <div>
                  <h3 className="text-base font-bold text-slate-900">Agent Mutual Auth Key Rotated</h3>
                  <p className="text-xs text-slate-500">{rotateKeyModal.server.name}</p>
                </div>
              </div>

              <div className="space-y-3 text-xs text-slate-600">
                <p>
                  A new high-entropy agent bearer key has been generated. The target node&apos;s daemon configuration at <code>/etc/hostvra/agent.json</code> has been synced automatically.
                </p>

                {rotateKeyModal.newKey && (
                  <div className="relative group">
                    <pre className="p-3 rounded-xl bg-slate-900 font-mono text-xs text-emerald-400 overflow-x-auto select-all">
                      {rotateKeyModal.newKey}
                    </pre>
                    <button
                      onClick={() => handleCopyCommand(rotateKeyModal.newKey!)}
                      className="absolute top-2 right-2 px-2.5 py-1 rounded-md bg-slate-800 hover:bg-slate-700 text-white text-[10px] font-medium flex items-center gap-1 shadow-xs"
                    >
                      {copied ? <Check className="w-3 h-3 text-emerald-400" /> : <Copy className="w-3 h-3" />}
                      <span>{copied ? 'Copied' : 'Copy'}</span>
                    </button>
                  </div>
                )}
              </div>

              <div className="flex justify-end pt-5 mt-4 border-t border-slate-200">
                <button
                  onClick={() => setRotateKeyModal({ open: false, server: null })}
                  className="px-4 py-2 rounded-xl text-xs font-semibold bg-[#16A34A] hover:bg-[#15803D] text-white"
                >
                  Done
                </button>
              </div>
            </div>
          </div>
        )}

        {/* MODAL 5: ENROLLMENT MODAL */}
        {enrollModalOpen && (
          <div className="fixed inset-0 z-50 bg-black/50 backdrop-blur-xs flex items-center justify-center p-4">
            <div className="w-full max-w-xl bg-white border border-slate-200 rounded-2xl shadow-xl p-6 relative">
              <button
                onClick={() => setEnrollModalOpen(false)}
                className="absolute top-5 right-5 p-1 rounded-lg text-slate-400 hover:text-slate-700 hover:bg-slate-100 transition-colors"
              >
                <X className="w-5 h-5" />
              </button>

              <div className="flex items-center gap-3 mb-4">
                <div className="w-10 h-10 rounded-xl bg-emerald-50 border border-emerald-100 flex items-center justify-center text-[#16A34A]">
                  <Terminal className="w-5 h-5" />
                </div>
                <div>
                  <h2 className="text-lg font-bold text-slate-900">Enroll New Server Node</h2>
                  <p className="text-xs text-slate-500">Deploy Hostvra Agent via one-command curl installer</p>
                </div>
              </div>

              {!enrollTokenData ? (
                <form onSubmit={handleGenerateEnrollment} className="space-y-4 pt-2">
                  <div>
                    <label className="block text-xs font-semibold text-slate-700 uppercase tracking-wider mb-1.5">
                      Server Label / Environment
                    </label>
                    <input
                      type="text"
                      required
                      value={enrollLabel}
                      onChange={(e) => setEnrollLabel(e.target.value)}
                      placeholder="e.g. Production Web US-East-1"
                      className="w-full px-3.5 py-2.5 rounded-xl border border-slate-300 text-slate-900 placeholder-slate-400 text-sm focus:outline-none focus:border-[#16A34A]"
                    />
                  </div>

                  <p className="text-xs text-slate-500 leading-relaxed">
                    This generates a cryptographically signed, single-use enrollment token valid for 2 hours. Once the agent executes on the target server, it exchanges this token for a permanent mutual authentication key.
                  </p>

                  <div className="flex justify-end gap-3 pt-3 border-t border-slate-100">
                    <button
                      type="button"
                      onClick={() => setEnrollModalOpen(false)}
                      className="px-4 py-2 rounded-xl text-sm font-medium text-slate-600 hover:text-slate-900"
                    >
                      Cancel
                    </button>
                    <button
                      type="submit"
                      disabled={generatingToken}
                      className="px-5 py-2 rounded-xl bg-[#16A34A] hover:bg-[#15803D] disabled:opacity-50 text-white text-sm font-semibold shadow-xs transition-all flex items-center gap-2"
                    >
                      {generatingToken ? (
                        <div className="w-4 h-4 border-2 border-white border-t-transparent rounded-full animate-spin" />
                      ) : (
                        'Generate Enrollment Command'
                      )}
                    </button>
                  </div>
                </form>
              ) : (
                <div className="space-y-4 pt-2">
                  <div className="p-3.5 rounded-xl bg-emerald-50 border border-emerald-200 text-emerald-800 text-xs flex items-center gap-2">
                    <Check className="w-4 h-4 flex-shrink-0 text-[#16A34A]" />
                    <span>Enrollment token generated successfully. Valid until {new Date(enrollTokenData.expires_at).toLocaleTimeString()}.</span>
                  </div>

                  <div>
                    <label className="block text-xs font-semibold text-slate-700 uppercase tracking-wider mb-2">
                      Run this on your clean Linux server as root:
                    </label>
                    <div className="relative group">
                      <pre className="p-4 rounded-xl bg-slate-900 font-mono text-xs text-emerald-400 overflow-x-auto whitespace-pre-wrap select-all">
                        {enrollTokenData.install_command}
                      </pre>
                      <button
                        onClick={() => handleCopyCommand(enrollTokenData.install_command)}
                        className="absolute top-2.5 right-2.5 px-3 py-1.5 rounded-lg bg-slate-800 hover:bg-slate-700 text-white text-xs font-medium flex items-center gap-1.5 shadow-xs transition-all"
                      >
                        {copied ? <Check className="w-3.5 h-3.5 text-emerald-400" /> : <Copy className="w-3.5 h-3.5" />}
                        <span>{copied ? 'Copied!' : 'Copy'}</span>
                      </button>
                    </div>
                  </div>

                  <div className="p-3.5 rounded-xl bg-slate-50 border border-slate-200 text-[11px] text-slate-600 space-y-1">
                    <p className="font-semibold text-slate-800">Target Requirements:</p>
                    <p>• Supported OS: Ubuntu 22.04/24.04 LTS, Debian 12, AlmaLinux 9, Rocky Linux 9</p>
                    <p>• Architecture: x86_64, arm64</p>
                    <p>• Minimum Hardware: 1 vCPU, 1 GB RAM, 10 GB Disk</p>
                  </div>

                  <div className="flex justify-end pt-2 border-t border-slate-100">
                    <button
                      type="button"
                      onClick={() => {
                        setEnrollModalOpen(false);
                        fetchFleetData();
                      }}
                      className="px-5 py-2 rounded-xl bg-[#16A34A] hover:bg-[#15803D] text-white text-sm font-semibold transition-all"
                    >
                      Done & Close
                    </button>
                  </div>
                </div>
              )}
            </div>
          </div>
        )}

        {/* MODAL 6: VIEW LOGS MODAL */}
        {viewLogsMigration && (
          <div className="fixed inset-0 z-50 bg-black/50 backdrop-blur-xs flex items-center justify-center p-4">
            <div className="w-full max-w-2xl bg-white border border-slate-200 rounded-2xl shadow-xl p-6 relative">
              <button
                onClick={() => setViewLogsMigration(null)}
                className="absolute top-5 right-5 p-1 rounded-lg text-slate-400 hover:text-slate-700 hover:bg-slate-100 transition-colors"
              >
                <X className="w-5 h-5" />
              </button>

              <div className="flex items-center gap-3 mb-4">
                <div className="w-10 h-10 rounded-xl bg-slate-100 border border-slate-200 flex items-center justify-center text-slate-700">
                  <Terminal className="w-5 h-5" />
                </div>
                <div>
                  <h3 className="text-base font-bold text-slate-900">Migration Pipeline Logs</h3>
                  <p className="text-xs text-slate-500 font-mono">Job ID: {viewLogsMigration.id}</p>
                </div>
              </div>

              <div className="bg-slate-900 rounded-xl p-4 font-mono text-xs text-slate-200 max-h-96 overflow-y-auto space-y-1.5">
                {viewLogsMigration.logs && viewLogsMigration.logs.length > 0 ? (
                  viewLogsMigration.logs.map((log, idx) => (
                    <div key={idx} className="flex items-start gap-2">
                      <span className="text-slate-500 select-none flex-shrink-0">
                        {new Date(log.timestamp).toLocaleTimeString()}
                      </span>
                      <span
                        className={`font-semibold flex-shrink-0 ${
                          log.level === 'SUCCESS'
                            ? 'text-emerald-400'
                            : log.level === 'WARN'
                            ? 'text-amber-400'
                            : log.level === 'ERROR'
                            ? 'text-rose-400'
                            : 'text-indigo-400'
                        }`}
                      >
                        [{log.level}]
                      </span>
                      <span className="text-slate-300 break-all">{log.message}</span>
                    </div>
                  ))
                ) : (
                  <p className="text-slate-500 italic">No logs recorded yet</p>
                )}
              </div>

              <div className="flex justify-end pt-4 mt-4 border-t border-slate-200">
                <button
                  onClick={() => setViewLogsMigration(null)}
                  className="px-4 py-2 rounded-xl text-xs font-semibold bg-slate-100 hover:bg-slate-200 text-slate-700"
                >
                  Close
                </button>
              </div>
            </div>
          </div>
        )}
      </div>
    </DashboardShell>
  );
}
