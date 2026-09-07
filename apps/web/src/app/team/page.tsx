'use client';

import React, { useState } from 'react';
import { DashboardShell } from '@/components/DashboardShell';
import {
  Users,
  Plus,
  Trash2,
  Shield,
  Mail,
  CheckCircle2,
  Clock,
  UserCheck,
  Search,
  X,
} from 'lucide-react';

interface Member {
  id: string;
  name: string;
  email: string;
  role: 'owner' | 'admin' | 'manager' | 'developer' | 'viewer';
  status: 'active' | 'invited';
  joined_at: string;
}

const initialMembers: Member[] = [
  {
    id: 'mem-1',
    name: 'Mizanur Rahman',
    email: 'admin@hostvra.com',
    role: 'owner',
    status: 'active',
    joined_at: '2026-08-10',
  },
  {
    id: 'mem-2',
    name: 'DevOps Automated Pipeline',
    email: 'ci-runner@hostvra.internal',
    role: 'admin',
    status: 'active',
    joined_at: '2026-08-15',
  },
  {
    id: 'mem-3',
    name: 'Sarah Chen',
    email: 'sarah.c@techcorp.io',
    role: 'developer',
    status: 'active',
    joined_at: '2026-09-01',
  },
];

export default function TeamPage() {
  const [members, setMembers] = useState<Member[]>(initialMembers);
  const [showInviteModal, setShowInviteModal] = useState(false);
  const [search, setSearch] = useState('');

  // Invite state
  const [inviteEmail, setInviteEmail] = useState('');
  const [inviteName, setInviteName] = useState('');
  const [inviteRole, setInviteRole] = useState<'admin' | 'manager' | 'developer' | 'viewer'>('developer');

  const filteredMembers = members.filter(
    (m) =>
      m.name.toLowerCase().includes(search.toLowerCase()) ||
      m.email.toLowerCase().includes(search.toLowerCase()) ||
      m.role.toLowerCase().includes(search.toLowerCase())
  );

  const handleInvite = (e: React.FormEvent) => {
    e.preventDefault();
    const newMember: Member = {
      id: Math.random().toString(36).substring(7),
      name: inviteName || inviteEmail.split('@')[0],
      email: inviteEmail,
      role: inviteRole,
      status: 'invited',
      joined_at: new Date().toISOString().split('T')[0],
    };
    setMembers([...members, newMember]);
    setShowInviteModal(false);
    setInviteEmail('');
    setInviteName('');
  };

  const handleRemove = (id: string) => {
    setMembers(members.filter((m) => m.id !== id));
  };

  return (
    <DashboardShell>
      <div className="space-y-6 animate-fadeIn max-w-7xl mx-auto pb-12">
        {/* Page Header */}
        <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
          <div>
            <h1 className="text-2xl font-bold text-slate-950 dark:text-white tracking-tight flex items-center gap-2">
              <Users className="w-7 h-7 text-indigo-600 dark:text-indigo-400" />
              Team & Organization Members
            </h1>
            <p className="text-sm text-slate-600 dark:text-slate-400 mt-1">
              Role-based access control (RBAC) across server nodes, clusters, and website scopes.
            </p>
          </div>
          <button
            onClick={() => setShowInviteModal(true)}
            className="inline-flex items-center gap-2 px-4 py-2.5 rounded-xl bg-indigo-600 hover:bg-indigo-500 text-white text-sm font-semibold shadow-lg shadow-indigo-600/25 transition-all"
          >
            <Plus className="w-4 h-4" />
            Invite Teammate
          </button>
        </div>

        {/* Roles overview cards */}
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
          <div className="bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-800 rounded-2xl p-4 shadow-xs">
            <div className="flex items-center justify-between text-xs font-bold text-purple-600 dark:text-purple-400 mb-1">
              <span>Owner & Admin</span>
              <Shield className="w-4 h-4" />
            </div>
            <p className="text-xs text-slate-600 dark:text-slate-400 leading-relaxed">
              Unrestricted control over billing, root daemons, licenses, and clustering.
            </p>
          </div>
          <div className="bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-800 rounded-2xl p-4 shadow-xs">
            <div className="flex items-center justify-between text-xs font-bold text-amber-600 dark:text-amber-400 mb-1">
              <span>Manager</span>
              <UserCheck className="w-4 h-4" />
            </div>
            <p className="text-xs text-slate-600 dark:text-slate-400 leading-relaxed">
              Can deploy vhosts, databases, manage backups, and configure webservers.
            </p>
          </div>
          <div className="bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-800 rounded-2xl p-4 shadow-xs">
            <div className="flex items-center justify-between text-xs font-bold text-cyan-600 dark:text-cyan-400 mb-1">
              <span>Developer</span>
              <Mail className="w-4 h-4" />
            </div>
            <p className="text-xs text-slate-600 dark:text-slate-400 leading-relaxed">
              SSH terminal access, SFTP file management, and runtime environment logs.
            </p>
          </div>
          <div className="bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-800 rounded-2xl p-4 shadow-xs">
            <div className="flex items-center justify-between text-xs font-bold text-slate-600 dark:text-slate-400 mb-1">
              <span>Viewer</span>
              <Clock className="w-4 h-4" />
            </div>
            <p className="text-xs text-slate-600 dark:text-slate-400 leading-relaxed">
              Read-only metrics, audit logs, and monitoring telemetry inspector.
            </p>
          </div>
        </div>

        {/* Filter Bar */}
        <div className="flex flex-col sm:flex-row items-center justify-between gap-4 bg-white dark:bg-[#10141d] border border-slate-200 dark:border-surface-800 rounded-2xl p-3 shadow-xs">
          <div className="flex items-center gap-3 w-full sm:w-80 bg-slate-50 dark:bg-[#121824] border border-slate-300 dark:border-surface-700 rounded-xl px-3.5 py-2 shadow-xs focus-within:border-[#20a53a] focus-within:ring-2 focus-within:ring-[#20a53a]/20 transition-all">
            <Search className="w-4 h-4 text-slate-400 dark:text-slate-500 flex-shrink-0" />
            <input
              type="text"
              placeholder="Search members by name or email..."
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              className="w-full bg-transparent text-xs font-medium text-slate-950 dark:text-white placeholder:text-slate-400 focus:outline-none"
            />
            {search && (
              <button onClick={() => setSearch('')} className="text-slate-400 hover:text-slate-600 dark:hover:text-white p-0.5">
                <X className="w-3.5 h-3.5" />
              </button>
            )}
          </div>
          <div className="text-xs font-semibold text-slate-600 dark:text-slate-400 px-2 self-end sm:self-center">
            Showing {filteredMembers.length} of {members.length} team members
          </div>
        </div>

        {/* Members Table */}
        <div className="bg-white dark:bg-[#10141d] border border-slate-200 dark:border-surface-800 rounded-2xl overflow-hidden shadow-xs dark:shadow-xl">
          <div className="overflow-x-auto">
            <table className="w-full text-left text-sm border-collapse">
              <thead>
                <tr className="border-b border-slate-200 dark:border-surface-800 bg-slate-50 dark:bg-[#121824] text-slate-700 dark:text-slate-300 text-[11px] font-bold uppercase tracking-wider">
                  <th className="px-6 py-3.5">User</th>
                  <th className="px-6 py-3.5">Assigned Role</th>
                  <th className="px-6 py-3.5">Status</th>
                  <th className="px-6 py-3.5">Joined</th>
                  <th className="px-6 py-3.5 text-right">Actions</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-200/80 dark:divide-surface-800/80">
                {filteredMembers.map((m) => (
                  <tr key={m.id} className="hover:bg-slate-50 dark:hover:bg-[#151d2d] transition-colors">
                    <td className="px-6 py-4">
                      <div>
                        <div className="font-bold text-slate-950 dark:text-white text-sm">{m.name}</div>
                        <div className="text-slate-600 dark:text-slate-400 font-mono text-xs">{m.email}</div>
                      </div>
                    </td>
                    <td className="px-6 py-4">
                      <span
                        className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-[11px] font-bold uppercase ${
                          m.role === 'owner'
                            ? 'bg-purple-500/10 text-purple-600 dark:text-purple-400 border border-purple-500/20'
                            : m.role === 'admin'
                            ? 'bg-rose-500/10 text-rose-600 dark:text-rose-400 border border-rose-500/20'
                            : m.role === 'manager'
                            ? 'bg-amber-500/10 text-amber-600 dark:text-amber-400 border border-amber-500/20'
                            : m.role === 'developer'
                            ? 'bg-cyan-500/10 text-cyan-600 dark:text-cyan-400 border border-cyan-500/20'
                            : 'bg-slate-500/10 text-slate-600 dark:text-slate-400 border border-slate-500/20'
                        }`}
                      >
                        {m.role}
                      </span>
                    </td>
                    <td className="px-6 py-4">
                      {m.status === 'active' ? (
                        <span className="inline-flex items-center gap-1.5 text-emerald-600 dark:text-emerald-400 font-semibold text-xs">
                          <CheckCircle2 className="w-3.5 h-3.5" /> Active
                        </span>
                      ) : (
                        <span className="inline-flex items-center gap-1.5 text-amber-600 dark:text-amber-400 font-semibold text-xs">
                          <Clock className="w-3.5 h-3.5" /> Invitation Sent
                        </span>
                      )}
                    </td>
                    <td className="px-6 py-4 text-xs font-medium text-slate-600 dark:text-slate-400 font-mono">{m.joined_at}</td>
                    <td className="px-6 py-4 text-right">
                      {m.role !== 'owner' && (
                        <button
                          onClick={() => handleRemove(m.id)}
                          className="p-1.5 rounded-lg border border-slate-300 dark:border-surface-700 text-slate-600 dark:text-slate-400 hover:text-rose-600 dark:hover:text-rose-400 hover:bg-slate-100 dark:hover:bg-surface-800 transition-colors"
                          title="Revoke Access"
                        >
                          <Trash2 className="w-3.5 h-3.5" />
                        </button>
                      )}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>

        {/* Invite Member Modal */}
        {showInviteModal && (
          <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/70 backdrop-blur-sm p-4">
            <div className="bg-surface-900 border border-surface-800 rounded-xl max-w-md w-full p-6 shadow-2xl space-y-4">
              <h3 className="text-lg font-bold text-white flex items-center gap-2">
                <Mail className="w-5 h-5 text-brand-400" />
                Invite Team Collaborator
              </h3>

              <form onSubmit={handleInvite} className="space-y-3.5 text-sm">
                <div>
                  <label className="block text-xs font-semibold text-slate-300 mb-1">Full Name</label>
                  <input
                    type="text"
                    placeholder="e.g. Alex Taylor"
                    value={inviteName}
                    onChange={(e) => setInviteName(e.target.value)}
                    className="w-full px-3.5 py-2 rounded-lg bg-surface-950 border border-surface-800 text-white text-sm focus:outline-none focus:border-brand-500"
                  />
                </div>

                <div>
                  <label className="block text-xs font-semibold text-slate-300 mb-1">Email Address</label>
                  <input
                    type="email"
                    required
                    placeholder="colleague@company.com"
                    value={inviteEmail}
                    onChange={(e) => setInviteEmail(e.target.value)}
                    className="w-full px-3.5 py-2 rounded-lg bg-surface-950 border border-surface-800 text-white font-mono text-sm focus:outline-none focus:border-brand-500"
                  />
                </div>

                <div>
                  <label className="block text-xs font-semibold text-slate-300 mb-1">Role Permissions</label>
                  <select
                    value={inviteRole}
                    onChange={(e) => setInviteRole(e.target.value as any)}
                    className="w-full px-3.5 py-2 rounded-lg bg-surface-950 border border-surface-800 text-white text-sm focus:outline-none focus:border-brand-500 capitalize"
                  >
                    <option value="admin">Admin (Full server & team privileges)</option>
                    <option value="manager">Manager (Websites, databases, backups)</option>
                    <option value="developer">Developer (Sites & runtime editing)</option>
                    <option value="viewer">Viewer (Read-only monitoring)</option>
                  </select>
                </div>

                <div className="flex items-center justify-end gap-3 pt-3">
                  <button
                    type="button"
                    onClick={() => setShowInviteModal(false)}
                    className="px-4 py-2 rounded-lg text-xs font-semibold text-slate-400 hover:text-white"
                  >
                    Cancel
                  </button>
                  <button
                    type="submit"
                    className="px-4 py-2 rounded-lg bg-brand-500 hover:bg-brand-600 text-white text-xs font-semibold transition-colors"
                  >
                    Send Invitation
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
