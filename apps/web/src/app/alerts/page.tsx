'use client';

import React, { useState } from 'react';
import { DashboardShell } from '@/components/DashboardShell';
import {
  BellRing,
  AlertTriangle,
  CheckCircle2,
  Plus,
  Zap,
  Sliders,
  Radio,
  Send,
  ShieldAlert,
  Server,
  Activity,
} from 'lucide-react';

interface Incident {
  id: string;
  server_name: string;
  rule_name: string;
  severity: 'critical' | 'warning' | 'info';
  status: 'firing' | 'resolved';
  value: number;
  threshold: number;
  message: string;
  started_at: string;
}

interface AlertRule {
  id: string;
  name: string;
  type: 'cpu' | 'memory' | 'disk';
  threshold: number;
  severity: 'critical' | 'warning' | 'info';
  enabled: boolean;
}

const initialIncidents: Incident[] = [
  {
    id: 'inc-9482',
    server_name: 'prod-edge-01',
    rule_name: 'High CPU Usage',
    severity: 'critical',
    status: 'firing',
    value: 94.2,
    threshold: 90.0,
    message: 'Server prod-edge-01: CPU load spiked to 94.2%',
    started_at: '12 mins ago',
  },
  {
    id: 'inc-9411',
    server_name: 'prod-db-replica',
    rule_name: 'High Memory Pressure',
    severity: 'warning',
    status: 'resolved',
    value: 78.4,
    threshold: 85.0,
    message: 'Memory normalized back to 78.4%',
    started_at: '2 hours ago',
  },
];

const initialRules: AlertRule[] = [
  { id: '1', name: 'Server CPU Spike', type: 'cpu', threshold: 90.0, severity: 'critical', enabled: true },
  { id: '2', name: 'RAM Exhaustion Warning', type: 'memory', threshold: 85.0, severity: 'warning', enabled: true },
  { id: '3', name: 'Disk Space Alert', type: 'disk', threshold: 90.0, severity: 'critical', enabled: true },
];

export default function AlertsPage() {
  const [incidents, setIncidents] = useState<Incident[]>(initialIncidents);
  const [rules, setRules] = useState<AlertRule[]>(initialRules);
  const [showAddRule, setShowAddRule] = useState(false);
  const [testSent, setTestSent] = useState(false);

  // Form state
  const [ruleName, setRuleName] = useState('');
  const [ruleType, setRuleType] = useState<'cpu' | 'memory' | 'disk'>('cpu');
  const [ruleThreshold, setRuleThreshold] = useState(90);
  const [ruleSeverity, setRuleSeverity] = useState<'critical' | 'warning' | 'info'>('critical');

  const handleAddRule = (e: React.FormEvent) => {
    e.preventDefault();
    const newRule: AlertRule = {
      id: Math.random().toString(36).substring(7),
      name: ruleName,
      type: ruleType,
      threshold: ruleThreshold,
      severity: ruleSeverity,
      enabled: true,
    };
    setRules([...rules, newRule]);
    setShowAddRule(false);
    setRuleName('');
  };

  const handleTriggerTest = () => {
    setTestSent(true);
    const mockIncident: Incident = {
      id: `inc-${Math.floor(Math.random() * 9000 + 1000)}`,
      server_name: 'prod-edge-01',
      rule_name: 'Synthetic Pipeline Test',
      severity: 'info',
      status: 'firing',
      value: 99.9,
      threshold: 90.0,
      message: 'Synthetic test incident dispatched to active webhooks and audit logs',
      started_at: 'Just now',
    };
    setIncidents([mockIncident, ...incidents]);
    setTimeout(() => setTestSent(false), 3000);
  };

  return (
    <DashboardShell>
      <div className="space-y-6">
        {/* Page Header */}
        <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
          <div>
            <h1 className="text-2xl font-bold text-white tracking-tight flex items-center gap-2">
              <BellRing className="w-7 h-7 text-brand-400" />
              Fleet Alerts & Incidents
            </h1>
            <p className="text-sm text-slate-400 mt-1">
              Multi-server hardware threshold monitoring, automated anomaly detection, and webhook notifications.
            </p>
          </div>
          <div className="flex items-center gap-3">
            <button
              onClick={handleTriggerTest}
              disabled={testSent}
              className="inline-flex items-center gap-2 px-3.5 py-2 rounded-lg bg-surface-800 hover:bg-surface-700 text-white text-xs font-semibold transition-colors border border-surface-700"
            >
              <Send className="w-3.5 h-3.5 text-cyan-400" />
              {testSent ? 'Dispatched Test Alert!' : 'Send Test Alert'}
            </button>
            <button
              onClick={() => setShowAddRule(true)}
              className="inline-flex items-center gap-2 px-4 py-2 rounded-lg bg-brand-500 hover:bg-brand-600 text-white text-xs font-semibold transition-colors shadow-lg shadow-brand-500/20"
            >
              <Plus className="w-4 h-4" />
              New Alert Rule
            </button>
          </div>
        </div>

        {/* Live Incident Status Banner */}
        <div className="grid grid-cols-1 md:grid-cols-3 gap-5">
          <div className="bg-surface-900 border border-surface-800 rounded-xl p-5 flex items-start gap-4">
            <div className="p-3 rounded-lg bg-rose-500/10 text-rose-400 border border-rose-500/20">
              <AlertTriangle className="w-6 h-6" />
            </div>
            <div>
              <div className="text-xs font-semibold text-slate-400 uppercase tracking-wider">Active Firing</div>
              <div className="text-2xl font-black text-white mt-1">
                {incidents.filter((i) => i.status === 'firing').length}
              </div>
              <div className="text-xs text-rose-400 mt-1">Requires attention</div>
            </div>
          </div>

          <div className="bg-surface-900 border border-surface-800 rounded-xl p-5 flex items-start gap-4">
            <div className="p-3 rounded-lg bg-emerald-500/10 text-emerald-400 border border-emerald-500/20">
              <CheckCircle2 className="w-6 h-6" />
            </div>
            <div>
              <div className="text-xs font-semibold text-slate-400 uppercase tracking-wider">Resolved Today</div>
              <div className="text-2xl font-black text-white mt-1">
                {incidents.filter((i) => i.status === 'resolved').length}
              </div>
              <div className="text-xs text-emerald-400 mt-1">Returned to normal threshold</div>
            </div>
          </div>

          <div className="bg-surface-900 border border-surface-800 rounded-xl p-5 flex items-start gap-4">
            <div className="p-3 rounded-lg bg-brand-500/10 text-brand-400 border border-brand-500/20">
              <Radio className="w-6 h-6" />
            </div>
            <div>
              <div className="text-xs font-semibold text-slate-400 uppercase tracking-wider">Notification Target</div>
              <div className="text-base font-bold text-white mt-1">Slack & Discord Webhook</div>
              <div className="text-xs text-brand-400 mt-1">Telemetry heartbeat every 10s</div>
            </div>
          </div>
        </div>

        {/* Incidents Stream */}
        <div className="bg-surface-900 border border-surface-800 rounded-xl overflow-hidden">
          <div className="px-6 py-4 border-b border-surface-800 flex items-center justify-between">
            <h2 className="text-sm font-semibold text-white uppercase tracking-wider flex items-center gap-2">
              <Activity className="w-4 h-4 text-brand-400" />
              Incidents & Trigger History
            </h2>
            <span className="text-xs font-mono text-slate-400">{incidents.length} events logged</span>
          </div>

          <div className="divide-y divide-surface-800/60">
            {incidents.map((inc) => (
              <div key={inc.id} className="p-5 flex flex-col sm:flex-row sm:items-center justify-between gap-4 hover:bg-surface-800/20 transition-colors">
                <div className="flex items-start gap-3.5">
                  <div
                    className={`mt-0.5 p-2 rounded-lg ${
                      inc.status === 'firing'
                        ? 'bg-rose-500/10 text-rose-400 border border-rose-500/20'
                        : 'bg-emerald-500/10 text-emerald-400 border border-emerald-500/20'
                    }`}
                  >
                    {inc.status === 'firing' ? <AlertTriangle className="w-4 h-4" /> : <CheckCircle2 className="w-4 h-4" />}
                  </div>
                  <div>
                    <div className="flex items-center gap-2">
                      <span className="font-bold text-white text-sm">{inc.rule_name}</span>
                      <span
                        className={`px-2 py-0.5 rounded text-[10px] font-bold uppercase tracking-wider ${
                          inc.severity === 'critical'
                            ? 'bg-rose-500/20 text-rose-400'
                            : inc.severity === 'warning'
                            ? 'bg-amber-500/20 text-amber-400'
                            : 'bg-blue-500/20 text-blue-400'
                        }`}
                      >
                        {inc.severity}
                      </span>
                      <span className="text-xs font-mono text-slate-400">on {inc.server_name}</span>
                    </div>
                    <p className="text-xs text-slate-300 mt-1 font-mono">{inc.message}</p>
                  </div>
                </div>

                <div className="flex items-center gap-4 text-xs font-mono">
                  <div className="text-right">
                    <div className="text-white font-bold">{inc.value}% / {inc.threshold}%</div>
                    <div className="text-slate-500">{inc.started_at}</div>
                  </div>
                  <span
                    className={`px-2.5 py-1 rounded-full text-[11px] font-bold uppercase ${
                      inc.status === 'firing'
                        ? 'bg-rose-500/10 text-rose-400 border border-rose-500/20'
                        : 'bg-emerald-500/10 text-emerald-400 border border-emerald-500/20'
                    }`}
                  >
                    {inc.status}
                  </span>
                </div>
              </div>
            ))}
          </div>
        </div>

        {/* Active Rules List */}
        <div className="bg-surface-900 border border-surface-800 rounded-xl overflow-hidden">
          <div className="px-6 py-4 border-b border-surface-800 flex items-center justify-between">
            <h2 className="text-sm font-semibold text-white uppercase tracking-wider flex items-center gap-2">
              <Sliders className="w-4 h-4 text-brand-400" />
              Configured Threshold Rules
            </h2>
            <span className="text-xs font-mono text-slate-400">{rules.length} active rules</span>
          </div>

          <div className="overflow-x-auto">
            <table className="w-full text-left text-sm text-slate-300 font-mono text-xs">
              <thead className="bg-surface-950/60 uppercase font-semibold text-slate-400 border-b border-surface-800">
                <tr>
                  <th className="px-6 py-3.5 font-sans">Rule Name</th>
                  <th className="px-6 py-3.5 font-sans">Metric Target</th>
                  <th className="px-6 py-3.5 font-sans">Threshold</th>
                  <th className="px-6 py-3.5 font-sans">Severity</th>
                  <th className="px-6 py-3.5 font-sans text-right">Status</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-surface-800/60">
                {rules.map((r) => (
                  <tr key={r.id} className="hover:bg-surface-800/30 transition-colors">
                    <td className="px-6 py-4 font-sans font-bold text-white">{r.name}</td>
                    <td className="px-6 py-4 uppercase text-brand-400">{r.type}</td>
                    <td className="px-6 py-4 font-bold text-white">&gt; {r.threshold}%</td>
                    <td className="px-6 py-4 uppercase text-slate-300">{r.severity}</td>
                    <td className="px-6 py-4 text-right">
                      <span className="px-2 py-0.5 rounded text-[11px] font-semibold bg-emerald-500/10 text-emerald-400 border border-emerald-500/20 font-sans">
                        Active
                      </span>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>

        {/* Create Rule Modal */}
        {showAddRule && (
          <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/70 backdrop-blur-sm p-4">
            <div className="bg-surface-900 border border-surface-800 rounded-xl max-w-md w-full p-6 shadow-2xl space-y-4">
              <h3 className="text-lg font-bold text-white flex items-center gap-2">
                <Plus className="w-5 h-5 text-brand-400" />
                Add Threshold Alert Rule
              </h3>

              <form onSubmit={handleAddRule} className="space-y-3.5 text-sm">
                <div>
                  <label className="block text-xs font-semibold text-slate-300 mb-1">Rule Name</label>
                  <input
                    type="text"
                    required
                    placeholder="e.g. Critical CPU Pressure"
                    value={ruleName}
                    onChange={(e) => setRuleName(e.target.value)}
                    className="w-full px-3.5 py-2 rounded-lg bg-surface-950 border border-surface-800 text-white text-sm focus:outline-none focus:border-brand-500"
                  />
                </div>

                <div>
                  <label className="block text-xs font-semibold text-slate-300 mb-1">Metric</label>
                  <div className="grid grid-cols-3 gap-2">
                    {(['cpu', 'memory', 'disk'] as const).map((m) => (
                      <button
                        type="button"
                        key={m}
                        onClick={() => setRuleType(m)}
                        className={`py-1.5 rounded text-xs font-bold border uppercase transition-all ${
                          ruleType === m
                            ? 'bg-brand-500/20 border-brand-500 text-brand-400'
                            : 'bg-surface-800 border-surface-700 text-slate-400'
                        }`}
                      >
                        {m}
                      </button>
                    ))}
                  </div>
                </div>

                <div>
                  <label className="block text-xs font-semibold text-slate-300 mb-1">Threshold Percentage (%)</label>
                  <input
                    type="number"
                    min={1}
                    max={100}
                    value={ruleThreshold}
                    onChange={(e) => setRuleThreshold(Number(e.target.value))}
                    className="w-full px-3.5 py-2 rounded-lg bg-surface-950 border border-surface-800 text-white font-mono text-sm focus:outline-none focus:border-brand-500"
                  />
                </div>

                <div>
                  <label className="block text-xs font-semibold text-slate-300 mb-1">Severity</label>
                  <select
                    value={ruleSeverity}
                    onChange={(e) => setRuleSeverity(e.target.value as any)}
                    className="w-full px-3.5 py-2 rounded-lg bg-surface-950 border border-surface-800 text-white text-sm focus:outline-none focus:border-brand-500 capitalize"
                  >
                    <option value="critical">Critical</option>
                    <option value="warning">Warning</option>
                    <option value="info">Info</option>
                  </select>
                </div>

                <div className="flex items-center justify-end gap-3 pt-3">
                  <button
                    type="button"
                    onClick={() => setShowAddRule(false)}
                    className="px-4 py-2 rounded-lg text-xs font-semibold text-slate-400 hover:text-white"
                  >
                    Cancel
                  </button>
                  <button
                    type="submit"
                    className="px-4 py-2 rounded-lg bg-brand-500 hover:bg-brand-600 text-white text-xs font-semibold transition-colors"
                  >
                    Save Rule
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
