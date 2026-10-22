'use client';

import React, { useState, useEffect, useCallback } from 'react';
import {
  GitBranch,
  Play,
  RotateCcw,
  Copy,
  Check,
  CheckCircle2,
  AlertTriangle,
  RefreshCw,
  ExternalLink,
  Shield,
  Terminal,
  Clock,
  User,
  Trash2,
  FileCode,
  Zap,
  Lock,
} from 'lucide-react';
import {
  Website,
  GitRepositoryRecord,
  GitDeploymentRecord,
  getGitRepo,
  saveGitRepo,
  deleteGitRepo,
  triggerGitDeploy,
  listGitDeployments,
  rollbackGitDeployment,
} from '@/lib/api';

interface GitCicdTabProps {
  website: Website;
  showToast: (msg: string) => void;
}

export function GitCicdTab({ website, showToast }: GitCicdTabProps) {
  const [repo, setRepo] = useState<GitRepositoryRecord | null>(null);
  const [deployments, setDeployments] = useState<GitDeploymentRecord[]>([]);
  const [loading, setLoading] = useState(true);
  const [submitting, setSubmitting] = useState(false);
  const [deploying, setDeploying] = useState(false);
  const [rollingBack, setRollingBack] = useState<string | null>(null);

  // Form state
  const [provider, setProvider] = useState<'github' | 'gitlab' | 'custom'>('github');
  const [repoUrl, setRepoUrl] = useState('');
  const [branch, setBranch] = useState('main');
  const [autoDeploy, setAutoDeploy] = useState(true);
  const [buildCommand, setBuildCommand] = useState('npm install && npm run build');
  const [keepReleases, setKeepReleases] = useState(5);

  // Copy state
  const [copiedKey, setCopiedKey] = useState(false);
  const [copiedWebhook, setCopiedWebhook] = useState(false);
  const [copiedSecret, setCopiedSecret] = useState(false);

  // View logs modal
  const [selectedLogs, setSelectedLogs] = useState<{ id: string; commit: string; logs: string } | null>(null);

  const fetchData = useCallback(async () => {
    setLoading(true);
    try {
      const [repoRes, depsRes] = await Promise.all([
        getGitRepo(website.id),
        listGitDeployments(website.id),
      ]);

      if (repoRes.success && repoRes.data) {
        setRepo(repoRes.data);
        setProvider(repoRes.data.provider || 'github');
        setRepoUrl(repoRes.data.repo_url);
        setBranch(repoRes.data.branch || 'main');
        setAutoDeploy(repoRes.data.auto_deploy);
        setBuildCommand(repoRes.data.build_command || '');
        setKeepReleases(repoRes.data.release_keep_count || 5);
      } else {
        setRepo(null);
      }

      if (depsRes.success && depsRes.data) {
        setDeployments(depsRes.data);
      }
    } catch (err: any) {
      showToast(err?.message || 'Failed to load Git repository');
    } finally {
      setLoading(false);
    }
  }, [website.id, showToast]);

  useEffect(() => {
    fetchData();
  }, [fetchData]);

  const handleSaveRepo = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!repoUrl.trim()) {
      showToast('Please enter a Git repository URL');
      return;
    }

    setSubmitting(true);
    try {
      const res = await saveGitRepo(website.id, {
        provider,
        repo_url: repoUrl.trim(),
        branch: branch.trim() || 'main',
        auto_deploy: autoDeploy,
        build_command: buildCommand.trim(),
        release_keep_count: keepReleases,
      });

      if (res.success && res.data) {
        setRepo(res.data);
        showToast('Git repository linked successfully with deploy key.');
      } else {
        showToast(res.error?.message || 'Failed to save Git configuration');
      }
    } catch (err: any) {
      showToast(err?.message || 'Failed to save Git configuration');
    } finally {
      setSubmitting(false);
    }
  };

  const handleTriggerDeploy = async () => {
    setDeploying(true);
    try {
      const res = await triggerGitDeploy(website.id, branch);
      if (res.success && res.data) {
        showToast(`Deployment ${res.data.status === 'success' ? 'succeeded' : 'completed'}`);
        fetchData();
      } else {
        showToast(res.error?.message || 'Deployment failed');
      }
    } catch (err: any) {
      showToast(err?.message || 'Deployment error');
    } finally {
      setDeploying(false);
    }
  };

  const handleRollback = async (dep: GitDeploymentRecord) => {
    if (!confirm(`Are you sure you want to rollback to release ${dep.commit_hash}?`)) return;
    setRollingBack(dep.id);
    try {
      const res = await rollbackGitDeployment(website.id, dep.id);
      if (res.success) {
        showToast(`Rolled back successfully to release ${dep.commit_hash}`);
        fetchData();
      } else {
        showToast(res.error?.message || 'Rollback failed');
      }
    } catch (err: any) {
      showToast(err?.message || 'Rollback error');
    } finally {
      setRollingBack(null);
    }
  };

  const handleDelete = async () => {
    if (!confirm('Unlink this Git repository? Deployments will no longer run automatically.')) return;
    try {
      const res = await deleteGitRepo(website.id);
      if (res.success) {
        setRepo(null);
        setDeployments([]);
        showToast('Git integration unlinked.');
      }
    } catch (err: any) {
      showToast(err?.message || 'Failed to unlink repository');
    }
  };

  const copyToClipboard = (text: string, type: 'key' | 'webhook' | 'secret') => {
    navigator.clipboard.writeText(text);
    if (type === 'key') {
      setCopiedKey(true);
      setTimeout(() => setCopiedKey(false), 2000);
    } else if (type === 'webhook') {
      setCopiedWebhook(true);
      setTimeout(() => setCopiedWebhook(false), 2000);
    } else {
      setCopiedSecret(true);
      setTimeout(() => setCopiedSecret(false), 2000);
    }
    showToast('Copied to clipboard!');
  };

  const webhookUrl = typeof window !== 'undefined'
    ? `${window.location.origin}/api/v1/websites/${website.id}/git/webhook`
    : `/api/v1/websites/${website.id}/git/webhook`;

  if (loading) {
    return (
      <div className="flex flex-col items-center justify-center p-12 text-slate-400">
        <RefreshCw className="w-8 h-8 animate-spin mb-3 text-emerald-500" />
        <p className="text-xs font-mono">Loading Git CI/CD repository status...</p>
      </div>
    );
  }

  return (
    <div className="space-y-6 text-xs">
      {/* Header Banner */}
      <div className="flex items-center justify-between p-4 rounded-xl bg-slate-50 dark:bg-[#131B2E] border border-slate-200 dark:border-slate-800">
        <div className="flex items-center gap-3">
          <div className="w-9 h-9 rounded-lg bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 flex items-center justify-center border border-emerald-500/20">
            <GitBranch className="w-5 h-5" />
          </div>
          <div>
            <h3 className="text-sm font-bold text-slate-900 dark:text-white flex items-center gap-2">
              Git CI/CD & Atomic Releases
              <span className="px-2 py-0.5 rounded text-[10px] font-bold bg-emerald-500/15 text-emerald-600 dark:text-emerald-400 border border-emerald-500/30">
                Zero Downtime
              </span>
            </h3>
            <p className="text-slate-500 text-[11px]">
              Continuous deployment from GitHub or GitLab with ed25519 deploy keys and 1-click atomic rollback.
            </p>
          </div>
        </div>

        {repo && (
          <div className="flex items-center gap-2">
            <button
              onClick={handleTriggerDeploy}
              disabled={deploying}
              className="px-3.5 py-1.5 rounded-lg bg-emerald-600 hover:bg-emerald-700 text-white font-bold text-xs flex items-center gap-1.5 transition shadow-xs cursor-pointer disabled:opacity-50"
            >
              {deploying ? <RefreshCw className="w-3.5 h-3.5 animate-spin" /> : <Play className="w-3.5 h-3.5" />}
              <span>{deploying ? 'Deploying...' : 'Deploy Now'}</span>
            </button>
            <button
              onClick={handleDelete}
              className="p-1.5 rounded-lg text-slate-400 hover:text-rose-500 hover:bg-rose-50 dark:hover:bg-rose-950/20 transition cursor-pointer"
              title="Unlink Repository"
            >
              <Trash2 className="w-4 h-4" />
            </button>
          </div>
        )}
      </div>

      {/* Configuration Form */}
      <form onSubmit={handleSaveRepo} className="p-4 rounded-xl bg-slate-50 dark:bg-[#131B2E] border border-slate-200 dark:border-slate-800 space-y-4">
        <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
          <div>
            <label className="block text-slate-700 dark:text-slate-300 font-semibold mb-1">Git Provider</label>
            <select
              value={provider}
              onChange={(e) => setProvider(e.target.value as any)}
              className="w-full p-2 rounded-lg bg-white dark:bg-[#0F172A] border border-slate-300 dark:border-slate-700 text-slate-900 dark:text-slate-100 font-medium"
            >
              <option value="github">GitHub</option>
              <option value="gitlab">GitLab</option>
              <option value="custom">Custom / Self-Hosted Git</option>
            </select>
          </div>

          <div className="sm:col-span-2">
            <label className="block text-slate-700 dark:text-slate-300 font-semibold mb-1">Repository URL (SSH or HTTPS)</label>
            <input
              type="text"
              value={repoUrl}
              onChange={(e) => setRepoUrl(e.target.value)}
              placeholder="git@github.com:username/repository.git"
              className="w-full p-2 rounded-lg bg-white dark:bg-[#0F172A] border border-slate-300 dark:border-slate-700 text-slate-900 dark:text-slate-100 font-mono text-xs"
            />
          </div>
        </div>

        <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
          <div>
            <label className="block text-slate-700 dark:text-slate-300 font-semibold mb-1">Deploy Branch</label>
            <input
              type="text"
              value={branch}
              onChange={(e) => setBranch(e.target.value)}
              placeholder="main"
              className="w-full p-2 rounded-lg bg-white dark:bg-[#0F172A] border border-slate-300 dark:border-slate-700 text-slate-900 dark:text-slate-100 font-mono text-xs"
            />
          </div>

          <div>
            <label className="block text-slate-700 dark:text-slate-300 font-semibold mb-1">Keep Previous Releases</label>
            <input
              type="number"
              min={1}
              max={20}
              value={keepReleases}
              onChange={(e) => setKeepReleases(parseInt(e.target.value) || 5)}
              className="w-full p-2 rounded-lg bg-white dark:bg-[#0F172A] border border-slate-300 dark:border-slate-700 text-slate-900 dark:text-slate-100 font-mono text-xs"
            />
          </div>

          <div className="flex items-center pt-5">
            <label className="flex items-center gap-2 cursor-pointer select-none">
              <input
                type="checkbox"
                checked={autoDeploy}
                onChange={(e) => setAutoDeploy(e.target.checked)}
                className="w-4 h-4 rounded text-emerald-600 focus:ring-emerald-500"
              />
              <span className="font-semibold text-slate-800 dark:text-slate-200">Auto-deploy on Webhook Push</span>
            </label>
          </div>
        </div>

        <div>
          <label className="block text-slate-700 dark:text-slate-300 font-semibold mb-1">
            Build & Install Command (Executed in release folder before symlink switch)
          </label>
          <input
            type="text"
            value={buildCommand}
            onChange={(e) => setBuildCommand(e.target.value)}
            placeholder="npm install && npm run build"
            className="w-full p-2 rounded-lg bg-white dark:bg-[#0F172A] border border-slate-300 dark:border-slate-700 text-slate-900 dark:text-emerald-400 font-mono text-xs"
          />
        </div>

        <div className="flex justify-end pt-2">
          <button
            type="submit"
            disabled={submitting}
            className="px-4 py-2 rounded-lg bg-emerald-600 hover:bg-emerald-700 text-white font-bold transition shadow-xs cursor-pointer disabled:opacity-50 flex items-center gap-1.5"
          >
            {submitting ? <RefreshCw className="w-3.5 h-3.5 animate-spin" /> : <Check className="w-3.5 h-3.5" />}
            <span>{repo ? 'Update Repository Configuration' : 'Save & Generate Deploy Key'}</span>
          </button>
        </div>
      </form>

      {/* Deploy Key & Webhook Details (shown once repo is created) */}
      {repo && (
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
          {/* Deploy Key Box */}
          <div className="p-4 rounded-xl bg-slate-50 dark:bg-[#131B2E] border border-slate-200 dark:border-slate-800 space-y-2">
            <div className="flex items-center justify-between">
              <span className="font-bold text-slate-900 dark:text-white flex items-center gap-1.5">
                <Lock className="w-3.5 h-3.5 text-emerald-500" />
                Ed25519 Public Deploy Key
              </span>
              <button
                type="button"
                onClick={() => copyToClipboard(repo.deploy_key_pub, 'key')}
                className="px-2 py-1 rounded bg-slate-200 dark:bg-slate-700 hover:bg-slate-300 dark:hover:bg-slate-600 text-slate-800 dark:text-white font-semibold text-[10px] flex items-center gap-1 transition cursor-pointer"
              >
                {copiedKey ? <Check className="w-3 h-3 text-emerald-500" /> : <Copy className="w-3 h-3" />}
                <span>{copiedKey ? 'Copied' : 'Copy Key'}</span>
              </button>
            </div>
            <textarea
              readOnly
              rows={3}
              value={repo.deploy_key_pub}
              className="w-full p-2 rounded-lg bg-white dark:bg-[#0F172A] border border-slate-300 dark:border-slate-700 text-slate-600 dark:text-slate-400 font-mono text-[11px] resize-none"
            />
            <p className="text-[10px] text-slate-500">
              Add this key to your repository <strong>Settings → Deploy Keys</strong> (read-only access).
            </p>
          </div>

          {/* Webhook URL & Secret Box */}
          <div className="p-4 rounded-xl bg-slate-50 dark:bg-[#131B2E] border border-slate-200 dark:border-slate-800 space-y-2">
            <div className="flex items-center justify-between">
              <span className="font-bold text-slate-900 dark:text-white flex items-center gap-1.5">
                <Shield className="w-3.5 h-3.5 text-indigo-500" />
                Signed Webhook URL & Secret
              </span>
              <button
                type="button"
                onClick={() => copyToClipboard(webhookUrl, 'webhook')}
                className="px-2 py-1 rounded bg-slate-200 dark:bg-slate-700 hover:bg-slate-300 dark:hover:bg-slate-600 text-slate-800 dark:text-white font-semibold text-[10px] flex items-center gap-1 transition cursor-pointer"
              >
                {copiedWebhook ? <Check className="w-3 h-3 text-emerald-500" /> : <Copy className="w-3 h-3" />}
                <span>{copiedWebhook ? 'Copied URL' : 'Copy URL'}</span>
              </button>
            </div>
            <input
              type="text"
              readOnly
              value={webhookUrl}
              className="w-full p-2 rounded-lg bg-white dark:bg-[#0F172A] border border-slate-300 dark:border-slate-700 text-slate-600 dark:text-slate-400 font-mono text-[11px]"
            />
            <div className="flex items-center justify-between text-[11px] text-slate-500 pt-1">
              <span>Secret: <code className="bg-slate-200 dark:bg-slate-800 px-1 py-0.5 rounded font-mono">{repo.webhook_secret}</code></span>
              <button
                type="button"
                onClick={() => copyToClipboard(repo.webhook_secret, 'secret')}
                className="text-emerald-600 dark:text-emerald-400 hover:underline cursor-pointer"
              >
                {copiedSecret ? 'Copied Secret' : 'Copy Secret'}
              </button>
            </div>
            <p className="text-[10px] text-slate-500">
              Configure in GitHub/GitLab with payload type <code>application/json</code> and push events.
            </p>
          </div>
        </div>
      )}

      {/* Deployment History Table */}
      <div className="rounded-xl bg-slate-50 dark:bg-[#131B2E] border border-slate-200 dark:border-slate-800 overflow-hidden">
        <div className="px-4 py-3 border-b border-slate-200 dark:border-slate-800 flex items-center justify-between">
          <h4 className="font-bold text-slate-900 dark:text-white flex items-center gap-2">
            <Clock className="w-4 h-4 text-slate-400" />
            Deployment History & Rollback Timeline
          </h4>
          <button
            onClick={fetchData}
            className="p-1 rounded text-slate-400 hover:text-slate-700 dark:hover:text-white transition cursor-pointer"
            title="Refresh Deployments"
          >
            <RefreshCw className="w-3.5 h-3.5" />
          </button>
        </div>

        {deployments.length === 0 ? (
          <div className="p-8 text-center text-slate-400">
            <Terminal className="w-8 h-8 mx-auto mb-2 opacity-30" />
            <p className="font-medium">No deployments triggered yet.</p>
            <p className="text-[11px] text-slate-500 mt-1">
              Click &quot;Deploy Now&quot; above or push to your configured branch.
            </p>
          </div>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full text-left text-xs">
              <thead className="bg-slate-100 dark:bg-[#0B1120] text-slate-500 uppercase text-[10px] font-bold">
                <tr>
                  <th className="px-4 py-2.5">Commit</th>
                  <th className="px-4 py-2.5">Message</th>
                  <th className="px-4 py-2.5">Author</th>
                  <th className="px-4 py-2.5">Status</th>
                  <th className="px-4 py-2.5">Duration</th>
                  <th className="px-4 py-2.5">Date</th>
                  <th className="px-4 py-2.5 text-right">Actions</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-200 dark:divide-slate-800 font-mono">
                {deployments.map((dep, idx) => (
                  <tr key={dep.id} className="hover:bg-slate-100/50 dark:hover:bg-slate-800/30 transition">
                    <td className="px-4 py-3 font-bold text-emerald-600 dark:text-emerald-400 flex items-center gap-1.5">
                      <span>{dep.commit_hash || 'HEAD'}</span>
                      {idx === 0 && dep.status === 'success' && (
                        <span className="px-1.5 py-0.2 rounded text-[9px] bg-emerald-500/20 text-emerald-600 dark:text-emerald-400 border border-emerald-500/30 font-sans">
                          Active
                        </span>
                      )}
                    </td>
                    <td className="px-4 py-3 font-sans truncate max-w-[200px]" title={dep.commit_message}>
                      {dep.commit_message || 'Automated deployment'}
                    </td>
                    <td className="px-4 py-3 font-sans text-slate-500 truncate max-w-[120px]">
                      {dep.author || 'Hostvra'}
                    </td>
                    <td className="px-4 py-3 font-sans">
                      <span className={`px-2 py-0.5 rounded text-[10px] font-bold inline-flex items-center gap-1 ${
                        dep.status === 'success'
                          ? 'bg-emerald-500/15 text-emerald-600 dark:text-emerald-400 border border-emerald-500/30'
                          : dep.status === 'failed'
                          ? 'bg-rose-500/15 text-rose-600 dark:text-rose-400 border border-rose-500/30'
                          : dep.status === 'rolled_back'
                          ? 'bg-amber-500/15 text-amber-600 dark:text-amber-400 border border-amber-500/30'
                          : 'bg-blue-500/15 text-blue-600 dark:text-blue-400 border border-blue-500/30'
                      }`}>
                        {dep.status === 'success' && <CheckCircle2 className="w-2.5 h-2.5" />}
                        {dep.status === 'failed' && <AlertTriangle className="w-2.5 h-2.5" />}
                        {dep.status === 'rolled_back' && <RotateCcw className="w-2.5 h-2.5" />}
                        {dep.status}
                      </span>
                    </td>
                    <td className="px-4 py-3 text-slate-500">
                      {dep.duration_seconds ? `${dep.duration_seconds}s` : '<1s'}
                    </td>
                    <td className="px-4 py-3 text-slate-500">
                      {dep.created_at ? dep.created_at.replace('T', ' ').slice(0, 19) : '-'}
                    </td>
                    <td className="px-4 py-3 text-right font-sans space-x-2">
                      {dep.build_logs && (
                        <button
                          onClick={() => setSelectedLogs({ id: dep.id, commit: dep.commit_hash, logs: dep.build_logs })}
                          className="px-2 py-1 rounded bg-slate-200 dark:bg-slate-700 hover:bg-slate-300 dark:hover:bg-slate-600 text-slate-700 dark:text-white font-medium text-[11px] transition cursor-pointer"
                        >
                          Logs
                        </button>
                      )}
                      {idx !== 0 && dep.status === 'success' && (
                        <button
                          onClick={() => handleRollback(dep)}
                          disabled={rollingBack === dep.id}
                          className="px-2 py-1 rounded bg-amber-500/15 hover:bg-amber-500/25 text-amber-600 dark:text-amber-400 border border-amber-500/30 font-bold text-[11px] transition cursor-pointer disabled:opacity-50 inline-flex items-center gap-1"
                        >
                          <RotateCcw className="w-3 h-3" />
                          <span>{rollingBack === dep.id ? 'Rolling back...' : 'Rollback'}</span>
                        </button>
                      )}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>

      {/* Build Logs Modal */}
      {selectedLogs && (
        <div className="fixed inset-0 z-60 bg-black/70 backdrop-blur-xs flex items-center justify-center p-4">
          <div className="w-full max-w-3xl bg-[#0F172A] border border-slate-700 rounded-xl shadow-2xl flex flex-col overflow-hidden max-h-[80vh]">
            <div className="flex items-center justify-between px-4 py-3 border-b border-slate-800 bg-[#0B1120]">
              <span className="font-bold text-white flex items-center gap-2">
                <Terminal className="w-4 h-4 text-emerald-400" />
                Build Logs - Release {selectedLogs.commit}
              </span>
              <button
                onClick={() => setSelectedLogs(null)}
                className="text-slate-400 hover:text-white cursor-pointer"
              >
                ✕
              </button>
            </div>
            <div className="p-4 overflow-y-auto font-mono text-[11px] text-slate-300 bg-[#090D16] space-y-1 whitespace-pre-wrap select-text">
              {selectedLogs.logs}
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
