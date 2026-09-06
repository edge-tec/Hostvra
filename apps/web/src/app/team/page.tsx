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
    id: '1',
    name: 'Root Administrator',
    email: 'admin@hostvra.com',
    role: 'owner',
    status: 'active',
    joined_at: '2 months ago',
  },
  {
    id: '2',
    name: 'Sarah Chen',
    email: 'sarah.devops@hostvra.internal',
    role: 'admin',
    status: 'active',
    joined_at: '3 weeks ago',
  },
  {
    id: '3',
    name: 'Alex Morgan',
    email: 'alex.dev@partner.io',
    role: 'developer',
    status: 'active',
    joined_at: '5 days ago',
  },
];

export default function TeamPage() {
  const [members, setMembers] = useState<Member[]>(initialMembers);
  const [showInviteModal, setShowInviteModal] = useState(false);

  // Invite state
  const [inviteEmail, setInviteEmail] = useState('');
  const [inviteName, setInviteName] = useState('');
  const [inviteRole, setInviteRole] = useState<'admin' | 'manager' | 'developer' | 'viewer'>('developer');

  const handleInvite = (e: React.FormEvent) => {
    e.preventDefault();
    const newMember: Member = {
      id: Math.random().toString(36).substring(7),
      name: inviteName || inviteEmail.split('@')[0],
      email: inviteEmail,
      role: inviteRole,
      status: 'invited',
      joined_at: 'Pending Invitation',
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
      <div className="space-y-6">
        {/* Page Header */}
        <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
          <div>
            <h1 className="text-2xl font-bold text-white tracking-tight flex items-center gap-2">
              <Users className="w-7 h-7 text-brand-400" />
              Team & Organization Members
            </h1>
            <p className="text-sm text-slate-400 mt-1">
              Collaborative access management with granular Role-Based Access Control (RBAC).
            </p>
          </div>
          <button
            onClick={() => setShowInviteModal(true)}
            className="inline-flex items-center gap-2 px-4 py-2 rounded-lg bg-brand-500 hover:bg-brand-600 text-white text-xs font-semibold transition-colors shadow-lg shadow-brand-500/20"
          >
            <Plus className="w-4 h-4" />
            Invite Member
          </button>
        </div>

        {/* Roles Info Cards */}
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
          <div className="bg-surface-900 border border-surface-800 rounded-xl p-4">
            <div className="flex items-center gap-2 text-rose-400 font-bold text-xs uppercase">
              <Shield className="w-4 h-4" /> Owner / Admin
            </div>
            <p className="text-xs text-slate-400 mt-2">
              Full unconstrained access to servers, databases, security, billing, and member management.
            </p>
          </div>
          <div className="bg-surface-900 border border-surface-800 rounded-xl p-4">
            <div className="flex items-center gap-2 text-amber-400 font-bold text-xs uppercase">
              <Shield className="w-4 h-4" /> Manager
            </div>
            <p className="text-xs text-slate-400 mt-2">
              Manage websites, databases, SSL certificates, cron jobs, and backup snapshots.
            </p>
          </div>
          <div className="bg-surface-900 border border-surface-800 rounded-xl p-4">
            <div className="flex items-center gap-2 text-cyan-400 font-bold text-xs uppercase">
              <Shield className="w-4 h-4" /> Developer
            </div>
            <p className="text-xs text-slate-400 mt-2">
              Read-write access to websites and web roots, view server metrics and service logs.
            </p>
          </div>
          <div className="bg-surface-900 border border-surface-800 rounded-xl p-4">
            <div className="flex items-center gap-2 text-slate-400 font-bold text-xs uppercase">
              <Shield className="w-4 h-4" /> Viewer
            </div>
            <p className="text-xs text-slate-400 mt-2">
              Read-only metrics, audit logs, and monitoring telemetry inspector.
            </p>
          </div>
        </div>

        {/* Members Table */}
        <div className="bg-surface-900 border border-surface-800 rounded-xl overflow-hidden">
          <div className="px-6 py-4 border-b border-surface-800 flex items-center justify-between">
            <h2 className="text-sm font-semibold text-white uppercase tracking-wider">Active Roster</h2>
            <span className="text-xs font-mono text-slate-400">{members.length} team accounts</span>
          </div>

          <div className="overflow-x-auto">
            <table className="w-full text-left text-sm text-slate-300 font-sans">
              <thead className="bg-surface-950/60 text-xs uppercase font-semibold text-slate-400 border-b border-surface-800">
                <tr>
                  <th className="px-6 py-3.5">User</th>
                  <th className="px-6 py-3.5">Assigned Role</th>
                  <th className="px-6 py-3.5">Status</th>
                  <th className="px-6 py-3.5">Joined</th>
                  <th className="px-6 py-3.5 text-right">Actions</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-surface-800/60 text-xs">
                {members.map((m) => (
                  <tr key={m.id} className="hover:bg-surface-800/30 transition-colors">
                    <td className="px-6 py-4">
                      <div>
                        <div className="font-bold text-white text-sm">{m.name}</div>
                        <div className="text-slate-400 font-mono text-xs">{m.email}</div>
                      </div>
                    </td>
                    <td className="px-6 py-4">
                      <span
                        className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-[11px] font-bold uppercase ${
                          m.role === 'owner'
                            ? 'bg-purple-500/10 text-purple-400 border border-purple-500/20'
                            : m.role === 'admin'
                            ? 'bg-rose-500/10 text-rose-400 border border-rose-500/20'
                            : m.role === 'manager'
                            ? 'bg-amber-500/10 text-amber-400 border border-amber-500/20'
                            : m.role === 'developer'
                            ? 'bg-cyan-500/10 text-cyan-400 border border-cyan-500/20'
                            : 'bg-slate-500/10 text-slate-400 border border-slate-500/20'
                        }`}
                      >
                        {m.role}
                      </span>
                    </td>
                    <td className="px-6 py-4">
                      {m.status === 'active' ? (
                        <span className="inline-flex items-center gap-1 text-emerald-400 font-medium">
                          <CheckCircle2 className="w-3.5 h-3.5" /> Active
                        </span>
                      ) : (
                        <span className="inline-flex items-center gap-1 text-amber-400 font-medium">
                          <Clock className="w-3.5 h-3.5" /> Invitation Sent
                        </span>
                      )}
                    </td>
                    <td className="px-6 py-4 text-slate-400">{m.joined_at}</td>
                    <td className="px-6 py-4 text-right">
                      {m.role !== 'owner' && (
                        <button
                          onClick={() => handleRemove(m.id)}
                          className="p-1.5 rounded-lg hover:bg-rose-500/10 text-slate-400 hover:text-rose-400 transition-colors"
                          title="Revoke Access"
                        >
                          <Trash2 className="w-4 h-4" />
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
