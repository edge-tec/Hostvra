'use client';

import React, { useState, useEffect, useCallback, useMemo } from 'react';
import {
  HardDriveDownload,
  Plus,
  RefreshCw,
  RotateCcw,
  CheckCircle2,
  AlertTriangle,
  Cloud,
  HardDrive,
  FileArchive,
  Database,
  ShieldCheck,
  Calendar,
  Trash2,
  Download,
  X,
  Search,
  Check,
  Copy,
  Server,
  Key,
  ExternalLink,
  Layers,
  Settings2,
  Clock,
  Mail,
  Lock,
  Unlock,
  FolderTree,
  Folder,
  FileText,
  FileCode,
  Scissors,
  CheckCheck,
  ChevronRight,
  ShieldAlert,
  Terminal,
  Activity,
  ArrowRight,
  Sparkles,
} from 'lucide-react';
import { DashboardShell } from '@/components/DashboardShell';
import {
  apiFetch,
  getApiBaseUrl,
  BackupRecord,
  BackupDestination,
  BackupSchedule,
  IntegrityReport,
  PreflightRestoreReport,
  BackupRepository,
  BackupIncrementalSnapshot,
  BackupRetentionPolicy,
  SnapshotFileNode,
  DisasterRecoveryRunbook,
  DestinationTestResult,
  IncrementalRestoreResult,
  PruneResult,
  RepositoryIntegrityResult,
  listBackupRepositories,
  createBackupRepository,
  deleteBackupRepository,
  updateBackupRepository,
  checkRepositoryIntegrity,
  pruneRepository,
  getRepositoryRetention,
  saveRepositoryRetention,
  listIncrementalSnapshots,
  createIncrementalSnapshot,
  getSnapshotFileTree,
  restoreSnapshotFiles,
  restoreFullAccountSnapshot,
  getDisasterRecoveryRunbook,
  testBackupDestinationConfig,
} from '@/lib/api';

function formatBytes(bytes: number): string {
  if (!bytes || bytes <= 0) return '0 B';
  if (bytes < 1024) return bytes + ' B';
  if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + ' KB';
  if (bytes < 1024 * 1024 * 1024) return (bytes / (1024 * 1024)).toFixed(1) + ' MB';
  return (bytes / (1024 * 1024 * 1024)).toFixed(2) + ' GB';
}

export default function BackupsPage() {
  const [activeTab, setActiveTab] = useState<'incremental' | 'repositories' | 'disaster_recovery' | 'snapshots' | 'destinations' | 'schedules'>('incremental');

  // ==========================================================================
  // PHASE NINE STATE: REPOSITORIES, INCREMENTAL SNAPSHOTS & DR
  // ==========================================================================
  const [repositories, setRepositories] = useState<BackupRepository[]>([]);
  const [incrementalSnapshots, setIncrementalSnapshots] = useState<BackupIncrementalSnapshot[]>([]);
  const [selectedRepoFilter, setSelectedRepoFilter] = useState<string>('all');
  const [incScopeFilter, setIncScopeFilter] = useState<'all' | 'website' | 'database' | 'mail' | 'full_account'>('all');
  const [incSearchQuery, setIncSearchQuery] = useState('');

  // Repositories Modals & State
  const [newRepoModalOpen, setNewRepoModalOpen] = useState(false);
  const [submittingNewRepo, setSubmittingNewRepo] = useState(false);
  const [repoProvider, setRepoProvider] = useState<'native' | 'restic'>('native');
  const [repoDestPreset, setRepoDestPreset] = useState<'wasabi' | 's3' | 'cloudflare_r2' | 'backblaze_b2' | 'local' | 'nfs'>('wasabi');
  const [repoName, setRepoName] = useState('');
  const [repoUri, setRepoUri] = useState('');
  const [repoPassword, setRepoPassword] = useState('');
  const [repoDeletionProtection, setRepoDeletionProtection] = useState(true);
  const [repoS3Endpoint, setRepoS3Endpoint] = useState('s3.wasabisys.com');
  const [repoS3Region, setRepoS3Region] = useState('us-east-1');
  const [repoS3Bucket, setRepoS3Bucket] = useState('');
  const [repoS3AccessKey, setRepoS3AccessKey] = useState('');
  const [repoS3SecretKey, setRepoS3SecretKey] = useState('');
  const [testingRepoConn, setTestingRepoConn] = useState(false);
  const [testRepoResult, setTestRepoResult] = useState<DestinationTestResult | null>(null);

  // Integrity Checking & Pruning Modals
  const [checkingIntegrityId, setCheckingIntegrityId] = useState<string | null>(null);
  const [integrityModalResult, setIntegrityModalResult] = useState<RepositoryIntegrityResult | null>(null);
  const [pruneModalRepo, setPruneModalRepo] = useState<BackupRepository | null>(null);
  const [prunePolicy, setPrunePolicy] = useState<BackupRetentionPolicy | null>(null);
  const [pruneDryRun, setPruneDryRun] = useState(true);
  const [pruningLoading, setPruningLoading] = useState(false);
  const [pruneResult, setPruneResult] = useState<PruneResult | null>(null);

  // Incremental Snapshot Creation Modal
  const [newSnapModalOpen, setNewSnapModalOpen] = useState(false);
  const [submittingNewSnap, setSubmittingNewSnap] = useState(false);
  const [snapRepoId, setSnapRepoId] = useState('');
  const [snapScope, setSnapScope] = useState<'website' | 'database' | 'mail' | 'full_account'>('website');
  const [snapTargetName, setSnapTargetName] = useState('');
  const [snapDbType, setSnapDbType] = useState<'none' | 'mysql' | 'postgres'>('none');
  const [snapDbName, setSnapDbName] = useState('');
  const [snapDbUser, setSnapDbUser] = useState('root');
  const [snapDbPass, setSnapDbPass] = useState('');
  const [snapIncludePaths, setSnapIncludePaths] = useState('');
  const [snapExcludePaths, setSnapExcludePaths] = useState('');

  // File Tree Preview & Selective File Restore Modal
  const [fileTreeModalOpen, setFileTreeModalOpen] = useState(false);
  const [fileTreeSnapshot, setFileTreeSnapshot] = useState<BackupIncrementalSnapshot | null>(null);
  const [fileTreeNodes, setFileTreeNodes] = useState<SnapshotFileNode[]>([]);
  const [fileTreePath, setFileTreePath] = useState('');
  const [fileTreeLoading, setFileTreeLoading] = useState(false);
  const [selectedFilePaths, setSelectedFilePaths] = useState<string[]>([]);
  const [restoreMode, setRestoreMode] = useState<'in_place' | 'alternate'>('alternate');
  const [restoreTargetDir, setRestoreTargetDir] = useState('');
  const [restoringFiles, setRestoringFiles] = useState(false);
  const [restoreResultModal, setRestoreResultModal] = useState<IncrementalRestoreResult | null>(null);

  // Full Account Restoration Modal
  const [fullRestoreSnapshot, setFullRestoreSnapshot] = useState<BackupIncrementalSnapshot | null>(null);
  const [fullRestoreTargetDir, setFullRestoreTargetDir] = useState('/var/www');
  const [fullRestoreDb, setFullRestoreDb] = useState(true);
  const [fullRestoreMail, setFullRestoreMail] = useState(true);
  const [fullRestoreDryRun, setFullRestoreDryRun] = useState(false);
  const [submittingFullRestore, setSubmittingFullRestore] = useState(false);

  // Disaster Recovery Runbook
  const [drRepoId, setDrRepoId] = useState('');
  const [drTargetHost, setDrTargetHost] = useState('recovery-node-01.hostvra.internal');
  const [drRunbook, setDrRunbook] = useState<DisasterRecoveryRunbook | null>(null);
  const [drLoading, setDrLoading] = useState(false);

  // ==========================================================================
  // LEGACY DATA STATES (PRESERVED 100% FOR BACKWARD COMPATIBILITY)
  // ==========================================================================
  const [backups, setBackups] = useState<BackupRecord[]>([]);
  const [destinations, setDestinations] = useState<BackupDestination[]>([]);
  const [schedules, setSchedules] = useState<BackupSchedule[]>([]);
  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);

  // Legacy Modals & Filters
  const [scopeFilter, setScopeFilter] = useState<'all' | 'website' | 'database' | 'full_config' | 'full_account'>('all');
  const [searchQuery, setSearchQuery] = useState('');
  const [errorMsg, setErrorMsg] = useState<string | null>(null);
  const [successMsg, setSuccessMsg] = useState<string | null>(null);
  const [copiedText, setCopiedText] = useState<string | null>(null);
  const [integrityModalReport, setIntegrityModalReport] = useState<IntegrityReport | null>(null);
  const [verifyingID, setVerifyingID] = useState<string | null>(null);
  const [createModalOpen, setCreateModalOpen] = useState(false);
  const [submittingCreate, setSubmittingCreate] = useState(false);
  const [restoreModalRecord, setRestoreModalRecord] = useState<BackupRecord | null>(null);
  const [submittingRestore, setSubmittingRestore] = useState(false);
  const [restoreSuccess, setRestoreSuccess] = useState(false);
  const [destModalOpen, setDestModalOpen] = useState(false);
  const [submittingDest, setSubmittingDest] = useState(false);
  const [schedModalOpen, setSchedModalOpen] = useState(false);
  const [submittingSched, setSubmittingSched] = useState(false);

  // Legacy Create Form State
  const [createType, setCreateType] = useState<'website' | 'database' | 'full_config' | 'full_account'>('website');
  const [createTargetName, setCreateTargetName] = useState('');
  const [createStorage, setCreateStorage] = useState<'local' | 's3'>('local');
  const [createDestID, setCreateDestID] = useState('');
  const [createRetention, setCreateRetention] = useState('7');

  // Dismiss alerts
  useEffect(() => {
    if (successMsg) {
      const t = setTimeout(() => setSuccessMsg(null), 4000);
      return () => clearTimeout(t);
    }
  }, [successMsg]);

  useEffect(() => {
    if (errorMsg) {
      const t = setTimeout(() => setErrorMsg(null), 6000);
      return () => clearTimeout(t);
    }
  }, [errorMsg]);

  // Load all data
  const loadData = useCallback(async (isRefresh = false) => {
    if (isRefresh) setRefreshing(true);
    else setLoading(true);

    try {
      const [repoRes, snapRes, bkRes, destRes, schedRes] = await Promise.all([
        listBackupRepositories(),
        listIncrementalSnapshots(),
        apiFetch<BackupRecord[]>('/api/v1/backups'),
        apiFetch<BackupDestination[]>('/api/v1/backups/destinations'),
        apiFetch<BackupSchedule[]>('/api/v1/backups/schedules'),
      ]);

      if (repoRes.success && repoRes.data) {
        setRepositories(repoRes.data);
        if (repoRes.data.length > 0) {
          if (!snapRepoId) setSnapRepoId(repoRes.data[0].id);
          if (!drRepoId) setDrRepoId(repoRes.data[0].id);
        }
      }
      if (snapRes.success && snapRes.data) {
        setIncrementalSnapshots(snapRes.data);
      }
      if (bkRes.success && bkRes.data) {
        setBackups(bkRes.data);
      }
      if (destRes.success && destRes.data) {
        setDestinations(destRes.data);
      }
      if (schedRes.success && schedRes.data) {
        setSchedules(schedRes.data);
      }
    } catch (err: any) {
      setErrorMsg(err.message || 'Failed to load backup data from server');
    } finally {
      setLoading(false);
      setRefreshing(false);
    }
  }, [snapRepoId, drRepoId]);

  useEffect(() => {
    loadData();
  }, [loadData]);

  // Copy helper
  const handleCopy = (text: string, label: string) => {
    navigator.clipboard.writeText(text);
    setCopiedText(label);
    setTimeout(() => setCopiedText(null), 2000);
  };

  // Generate strong passphrase
  const generatePassphrase = () => {
    const chars = 'abcdefghjkmnpqrstuvwxyzABCDEFGHJKLMNPQRSTUVWXYZ23456789!@#$%^&*';
    let p = '';
    for (let i = 0; i < 24; i++) {
      p += chars.charAt(Math.floor(Math.random() * chars.length));
    }
    setRepoPassword(p);
  };

  // Preset configuration helper
  const handlePresetSelect = (preset: typeof repoDestPreset) => {
    setRepoDestPreset(preset);
    switch (preset) {
      case 'wasabi':
        setRepoS3Endpoint('s3.wasabisys.com');
        setRepoS3Region('us-east-1');
        setRepoUri('s3:s3.wasabisys.com/' + (repoS3Bucket || 'hostvra-backups'));
        break;
      case 'cloudflare_r2':
        setRepoS3Endpoint('<account_id>.r2.cloudflarestorage.com');
        setRepoS3Region('auto');
        setRepoUri('s3:https://<account_id>.r2.cloudflarestorage.com/' + (repoS3Bucket || 'hostvra-backups'));
        break;
      case 'backblaze_b2':
        setRepoS3Endpoint('s3.us-west-002.backblazeb2.com');
        setRepoS3Region('us-west-002');
        setRepoUri('s3:s3.us-west-002.backblazeb2.com/' + (repoS3Bucket || 'hostvra-backups'));
        break;
      case 's3':
        setRepoS3Endpoint('s3.amazonaws.com');
        setRepoS3Region('us-east-1');
        setRepoUri('s3:s3.amazonaws.com/' + (repoS3Bucket || 'hostvra-backups'));
        break;
      case 'local':
        setRepoUri('/var/backups/hostvra/repo-1');
        break;
      case 'nfs':
        setRepoUri('/mnt/nfs-share/hostvra-backups');
        break;
    }
  };

  // Test Repository Connection
  const handleTestRepoConnection = async () => {
    setTestingRepoConn(true);
    setTestRepoResult(null);
    try {
      const res = await testBackupDestinationConfig({
        type: repoDestPreset === 'local' || repoDestPreset === 'nfs' ? 'local' : 's3',
        endpoint: repoS3Endpoint,
        region: repoS3Region,
        bucket: repoS3Bucket,
        access_key: repoS3AccessKey,
        secret_key: repoS3SecretKey,
        local_path: repoUri,
      });
      if (res.success && res.data) {
        setTestRepoResult(res.data);
      } else {
        setErrorMsg(res.error?.message || 'Destination verification failed');
      }
    } catch (err: any) {
      setErrorMsg(err.message || 'Connection test error');
    } finally {
      setTestingRepoConn(false);
    }
  };

  // Create Repository Handler
  const handleCreateRepository = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!repoName.trim()) {
      setErrorMsg('Repository name is required');
      return;
    }
    setSubmittingNewRepo(true);
    setErrorMsg(null);

    try {
      const storageConfig: Record<string, string> = {};
      if (repoDestPreset !== 'local' && repoDestPreset !== 'nfs') {
        storageConfig['aws_access_key_id'] = repoS3AccessKey.trim();
        storageConfig['aws_secret_access_key'] = repoS3SecretKey.trim();
        storageConfig['endpoint'] = repoS3Endpoint.trim();
        storageConfig['region'] = repoS3Region.trim();
        storageConfig['bucket'] = repoS3Bucket.trim();
      }

      const res = await createBackupRepository({
        name: repoName.trim(),
        provider: repoProvider,
        destination_type: repoDestPreset,
        repo_uri: repoUri.trim() || undefined,
        storage_config: storageConfig,
        encryption_algorithm: 'AES-256-GCM',
        password: repoPassword || undefined,
        deletion_protection: repoDeletionProtection,
      });

      if (res.success && res.data) {
        setSuccessMsg(`Encrypted repository "${res.data.name}" created and initialized successfully.`);
        setNewRepoModalOpen(false);
        setRepoName('');
        setRepoPassword('');
        setTestRepoResult(null);
        loadData(true);
      } else {
        setErrorMsg(res.error?.message || 'Failed to initialize repository');
      }
    } catch (err: any) {
      setErrorMsg(err.message || 'Network error initializing repository');
    } finally {
      setSubmittingNewRepo(false);
    }
  };

  // Check Repository Integrity
  const handleCheckIntegrity = async (repoId: string) => {
    setCheckingIntegrityId(repoId);
    setErrorMsg(null);
    try {
      const res = await checkRepositoryIntegrity(repoId);
      if (res.success && res.data) {
        setIntegrityModalResult(res.data);
        setRepositories((prev) =>
          prev.map((r) =>
            r.id === repoId
              ? {
                  ...r,
                  integrity_status: res.data!.healthy ? 'healthy' : 'error',
                  last_checked_at: res.data!.checked_at,
                  total_snapshots: res.data!.total_snapshots,
                  total_size_bytes: res.data!.total_size_bytes,
                }
              : r
          )
        );
        if (res.data.healthy) {
          setSuccessMsg(`Repository integrity verified 100%. All cryptographic packs are healthy.`);
        } else {
          setErrorMsg(`Repository integrity issues found.`);
        }
      } else {
        setErrorMsg(res.error?.message || 'Integrity check failed');
      }
    } catch (err: any) {
      setErrorMsg(err.message || 'Error running repository integrity check');
    } finally {
      setCheckingIntegrityId(null);
    }
  };

  // Delete Repository Handler (with Deletion Protection Guard)
  const handleDeleteRepository = async (repo: BackupRepository) => {
    if (repo.deletion_protection) {
      setErrorMsg(`Repository "${repo.name}" has deletion protection enabled. You must disable protection first.`);
      return;
    }
    if (!confirm(`Are you sure you want to permanently delete repository "${repo.name}"? This action cannot be undone.`)) {
      return;
    }
    try {
      const res = await deleteBackupRepository(repo.id);
      if (res.success) {
        setSuccessMsg(`Repository "${repo.name}" removed successfully.`);
        setRepositories((prev) => prev.filter((r) => r.id !== repo.id));
      } else {
        setErrorMsg(res.error?.message || 'Failed to delete repository');
      }
    } catch (err: any) {
      setErrorMsg(err.message || 'Error deleting repository');
    }
  };

  // Toggle Deletion Protection
  const handleToggleDeletionProtection = async (repo: BackupRepository) => {
    const nextVal = !repo.deletion_protection;
    try {
      const res = await updateBackupRepository(repo.id, { deletion_protection: nextVal });
      if (res.success) {
        setRepositories((prev) =>
          prev.map((r) => (r.id === repo.id ? { ...r, deletion_protection: nextVal } : r))
        );
        setSuccessMsg(`Deletion protection ${nextVal ? 'enabled' : 'disabled'} for ${repo.name}.`);
      } else {
        setErrorMsg(res.error?.message || 'Failed to update deletion protection');
      }
    } catch (err: any) {
      setErrorMsg(err.message || 'Error updating deletion protection');
    }
  };

  // Open Prune & Retention Modal
  const handleOpenPruneModal = async (repo: BackupRepository) => {
    setPruneModalRepo(repo);
    setPruneResult(null);
    try {
      const res = await getRepositoryRetention(repo.id);
      if (res.success && res.data) {
        setPrunePolicy(res.data);
      }
    } catch (err) {
      console.error(err);
    }
  };

  // Execute Pruning
  const handleExecutePrune = async () => {
    if (!pruneModalRepo) return;
    setPruningLoading(true);
    setErrorMsg(null);
    try {
      const res = await pruneRepository(pruneModalRepo.id, pruneDryRun);
      if (res.success && res.data) {
        setPruneResult(res.data);
        if (!pruneDryRun) {
          setSuccessMsg(`Pruned ${res.data.snapshots_removed} snapshots, reclaimed ${formatBytes(res.data.bytes_reclaimed)}.`);
          loadData(true);
        }
      } else {
        setErrorMsg(res.error?.message || 'Pruning operation failed');
      }
    } catch (err: any) {
      setErrorMsg(err.message || 'Error during pruning execution');
    } finally {
      setPruningLoading(false);
    }
  };

  // Create Incremental Snapshot Handler
  const handleCreateIncrementalSnapshot = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!snapRepoId) {
      setErrorMsg('Please select a repository');
      return;
    }
    setSubmittingNewSnap(true);
    setErrorMsg(null);

    try {
      const inc = snapIncludePaths ? snapIncludePaths.split('\n').map((s) => s.trim()).filter(Boolean) : undefined;
      const exc = snapExcludePaths ? snapExcludePaths.split('\n').map((s) => s.trim()).filter(Boolean) : undefined;

      const res = await createIncrementalSnapshot({
        repository_id: snapRepoId,
        scope: snapScope,
        target_name: snapTargetName.trim() || undefined,
        include_paths: inc,
        exclude_paths: exc,
        database_type: snapDbType !== 'none' ? snapDbType : undefined,
        database_name: snapDbName.trim() || undefined,
        database_user: snapDbUser.trim() || undefined,
        database_password: snapDbPass || undefined,
      });

      if (res.success && res.data) {
        setSuccessMsg(`Incremental snapshot ${res.data.snapshot_hash.substring(0, 12)} completed successfully.`);
        setNewSnapModalOpen(false);
        setSnapTargetName('');
        setSnapDbName('');
        loadData(true);
      } else {
        setErrorMsg(res.error?.message || 'Snapshot creation failed');
      }
    } catch (err: any) {
      setErrorMsg(err.message || 'Error creating incremental snapshot');
    } finally {
      setSubmittingNewSnap(false);
    }
  };

  // Open File Tree Preview
  const handleOpenFileTree = async (snap: BackupIncrementalSnapshot, path: string = '') => {
    setFileTreeSnapshot(snap);
    setFileTreePath(path);
    setFileTreeModalOpen(true);
    setFileTreeLoading(true);
    setSelectedFilePaths([]);
    setRestoreResultModal(null);
    setRestoreTargetDir(snap.scope === 'website' ? `/var/www/${snap.target_name || 'site'}_restored` : '/var/backups/restored');

    try {
      const res = await getSnapshotFileTree(snap.id, path);
      if (res.success && res.data) {
        setFileTreeNodes(res.data);
      } else {
        setErrorMsg(res.error?.message || 'Failed to list snapshot files');
      }
    } catch (err: any) {
      setErrorMsg(err.message || 'Error browsing snapshot filesystem');
    } finally {
      setFileTreeLoading(false);
    }
  };

  // Selective Restore Files Handler
  const handleRestoreSelectedFiles = async () => {
    if (!fileTreeSnapshot || selectedFilePaths.length === 0) {
      setErrorMsg('Please select at least one file or folder to restore');
      return;
    }
    setRestoringFiles(true);
    setErrorMsg(null);

    try {
      const res = await restoreSnapshotFiles(fileTreeSnapshot.id, {
        selected_paths: selectedFilePaths,
        target_directory: restoreTargetDir.trim() || undefined,
        restore_mode: restoreMode,
      });

      if (res.success && res.data) {
        setRestoreResultModal(res.data);
        setSuccessMsg(`Restored ${res.data.files_restored} files (${formatBytes(res.data.bytes_restored)}) successfully.`);
      } else {
        setErrorMsg(res.error?.message || 'File restoration failed');
      }
    } catch (err: any) {
      setErrorMsg(err.message || 'Error during file restoration');
    } finally {
      setRestoringFiles(false);
    }
  };

  // Full Account Restore Handler
  const handleExecuteFullRestore = async () => {
    if (!fullRestoreSnapshot) return;
    setSubmittingFullRestore(true);
    setErrorMsg(null);

    try {
      const res = await restoreFullAccountSnapshot(fullRestoreSnapshot.id, {
        target_directory: fullRestoreTargetDir.trim(),
        restore_database: fullRestoreDb,
        restore_mailboxes: fullRestoreMail,
        dry_run: fullRestoreDryRun,
      });

      if (res.success && res.data) {
        setSuccessMsg(
          fullRestoreDryRun
            ? `Full restoration preflight simulation passed! Staged ${res.data.files_restored} files safely.`
            : `Full account restoration completed! ${res.data.files_restored} files restored with database and mail consistency.`
        );
        setFullRestoreSnapshot(null);
      } else {
        setErrorMsg(res.error?.message || 'Full restoration failed');
      }
    } catch (err: any) {
      setErrorMsg(err.message || 'Error executing full restore');
    } finally {
      setSubmittingFullRestore(false);
    }
  };

  // Generate Disaster Recovery Runbook
  const handleGenerateRunbook = async () => {
    if (!drRepoId) {
      setErrorMsg('Please select a repository');
      return;
    }
    setDrLoading(true);
    setErrorMsg(null);

    try {
      const res = await getDisasterRecoveryRunbook(drRepoId, drTargetHost.trim());
      if (res.success && res.data) {
        setDrRunbook(res.data);
      } else {
        setErrorMsg(res.error?.message || 'Failed to generate runbook');
      }
    } catch (err: any) {
      setErrorMsg(err.message || 'Error generating disaster recovery runbook');
    } finally {
      setDrLoading(false);
    }
  };

  // Download Runbook as Markdown
  const handleDownloadRunbookMd = () => {
    if (!drRunbook) return;
    const blob = new Blob([drRunbook.markdown], { type: 'text/markdown;charset=utf-8;' });
    const url = URL.createObjectURL(blob);
    const link = document.createElement('a');
    link.href = url;
    link.download = `Hostvra_DR_Runbook_${drTargetHost || 'node'}.md`;
    link.click();
    URL.revokeObjectURL(url);
  };

  // Deduplication Metrics Summary
  const { totalRawSize, totalStoredSize, globalDedupRatio } = useMemo(() => {
    let raw = 0;
    let stored = 0;
    incrementalSnapshots.forEach((s) => {
      raw += s.raw_size_bytes || 0;
      stored += s.stored_size_bytes || 0;
    });
    const ratio = stored > 0 ? (raw / stored).toFixed(1) : '1.0';
    return { totalRawSize: raw, totalStoredSize: stored, globalDedupRatio: ratio };
  }, [incrementalSnapshots]);

  // Filtered Snapshots
  const filteredIncrementalSnapshots = useMemo(() => {
    return incrementalSnapshots.filter((s) => {
      if (selectedRepoFilter !== 'all' && s.repository_id !== selectedRepoFilter) return false;
      if (incScopeFilter !== 'all' && s.scope !== incScopeFilter) return false;
      if (incSearchQuery) {
        const q = incSearchQuery.toLowerCase();
        return (
          s.target_name.toLowerCase().includes(q) ||
          s.snapshot_hash.toLowerCase().includes(q) ||
          s.scope.toLowerCase().includes(q)
        );
      }
      return true;
    });
  }, [incrementalSnapshots, selectedRepoFilter, incScopeFilter, incSearchQuery]);

  return (
    <DashboardShell>
      <div className="space-y-6 max-w-7xl mx-auto pb-16">
        {/* Global Notifications */}
        {errorMsg && (
          <div className="p-4 rounded-xl bg-rose-500/10 border border-rose-500/20 text-rose-300 text-sm flex items-start gap-3 shadow-lg backdrop-blur-sm animate-in fade-in slide-in-from-top-2">
            <AlertTriangle className="w-5 h-5 flex-shrink-0 text-rose-400 mt-0.5" />
            <div className="flex-1 font-medium">{errorMsg}</div>
            <button onClick={() => setErrorMsg(null)} className="text-rose-400 hover:text-white transition-colors">
              <X className="w-4 h-4" />
            </button>
          </div>
        )}

        {successMsg && (
          <div className="p-4 rounded-xl bg-emerald-500/10 border border-emerald-500/20 text-emerald-300 text-sm flex items-start gap-3 shadow-lg backdrop-blur-sm animate-in fade-in slide-in-from-top-2">
            <CheckCircle2 className="w-5 h-5 flex-shrink-0 text-emerald-400 mt-0.5" />
            <div className="flex-1 font-medium">{successMsg}</div>
            <button onClick={() => setSuccessMsg(null)} className="text-emerald-400 hover:text-white transition-colors">
              <X className="w-4 h-4" />
            </button>
          </div>
        )}

        {/* Top Header Bar */}
        <div className="flex flex-col md:flex-row md:items-center justify-between gap-4 bg-gradient-to-r from-surface-900 via-surface-900/90 to-surface-850 p-6 rounded-2xl border border-surface-800 shadow-xl relative overflow-hidden">
          <div className="absolute top-0 right-0 w-96 h-96 bg-brand-500/5 rounded-full blur-3xl -mr-32 -mt-32 pointer-events-none" />
          <div className="relative z-10">
            <div className="flex items-center gap-3">
              <div className="p-2.5 rounded-xl bg-brand-500/10 text-brand-400 border border-brand-500/20 shadow-inner">
                <HardDriveDownload className="w-6 h-6" />
              </div>
              <div>
                <div className="flex items-center gap-2">
                  <h1 className="text-2xl font-bold tracking-tight text-white">Backups & Disaster Recovery</h1>
                  <span className="px-2.5 py-0.5 text-[10px] font-bold tracking-wide uppercase rounded-full bg-brand-500/20 text-brand-300 border border-brand-500/30">
                    Phase Nine Enterprise
                  </span>
                </div>
                <p className="text-xs text-slate-400 mt-0.5">
                  Content-addressed deduplicated backups, encrypted remote vaults, granular file restore & DR runbooks
                </p>
              </div>
            </div>
          </div>

          <div className="flex items-center gap-2.5 relative z-10 flex-wrap">
            <button
              onClick={() => loadData(true)}
              disabled={refreshing}
              className="inline-flex items-center justify-center gap-2 px-3.5 py-2 rounded-xl bg-surface-800 hover:bg-surface-750 text-slate-300 hover:text-white text-xs font-semibold border border-surface-700 transition-all cursor-pointer shadow-xs disabled:opacity-50"
            >
              <RefreshCw className={`w-3.5 h-3.5 ${refreshing ? 'animate-spin' : ''}`} />
              <span>Refresh Vault</span>
            </button>

            {activeTab === 'repositories' ? (
              <button
                onClick={() => {
                  setRepoName('');
                  setRepoPassword('');
                  generatePassphrase();
                  handlePresetSelect('wasabi');
                  setNewRepoModalOpen(true);
                }}
                className="inline-flex items-center justify-center gap-2 px-4 py-2 rounded-xl bg-gradient-to-r from-brand-500 to-brand-600 hover:from-brand-600 hover:to-brand-700 text-white text-xs font-semibold shadow-md shadow-brand-500/20 transition-all cursor-pointer"
              >
                <Plus className="w-4 h-4" />
                <span>New Encrypted Repository</span>
              </button>
            ) : (
              <button
                onClick={() => {
                  if (repositories.length === 0) {
                    setErrorMsg('Please create an encrypted backup repository first.');
                    setActiveTab('repositories');
                    return;
                  }
                  setNewSnapModalOpen(true);
                }}
                className="inline-flex items-center justify-center gap-2 px-4 py-2 rounded-xl bg-gradient-to-r from-brand-500 to-brand-600 hover:from-brand-600 hover:to-brand-700 text-white text-xs font-semibold shadow-md shadow-brand-500/20 transition-all cursor-pointer"
              >
                <Plus className="w-4 h-4" />
                <span>Create Incremental Snapshot</span>
              </button>
            )}
          </div>
        </div>

        {/* 4 Overview Statistics Cards */}
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
          <div className="bg-surface-900 border border-surface-800 rounded-xl p-4.5 flex items-start gap-3.5 shadow-sm">
            <div className="p-3 rounded-xl bg-brand-500/10 text-brand-400 border border-brand-500/20">
              <Layers className="w-5 h-5" />
            </div>
            <div>
              <div className="text-[11px] font-semibold text-slate-400 uppercase tracking-wider">Incremental Snapshots</div>
              <div className="text-xl font-bold text-white mt-0.5">{incrementalSnapshots.length}</div>
              <div className="text-[11px] text-brand-400 font-medium mt-0.5 flex items-center gap-1">
                <Scissors className="w-3 h-3" />
                <span>{globalDedupRatio}x Deduplication Ratio</span>
              </div>
            </div>
          </div>

          <div className="bg-surface-900 border border-surface-800 rounded-xl p-4.5 flex items-start gap-3.5 shadow-sm">
            <div className="p-3 rounded-xl bg-blue-500/10 text-blue-400 border border-blue-500/20">
              <Cloud className="w-5 h-5" />
            </div>
            <div>
              <div className="text-[11px] font-semibold text-slate-400 uppercase tracking-wider">Storage Efficiency</div>
              <div className="text-xl font-bold text-white mt-0.5">{formatBytes(totalStoredSize)}</div>
              <div className="text-[11px] text-slate-400 mt-0.5">
                Saved from <span className="text-slate-300 font-semibold">{formatBytes(totalRawSize)}</span>
              </div>
            </div>
          </div>

          <div className="bg-surface-900 border border-surface-800 rounded-xl p-4.5 flex items-start gap-3.5 shadow-sm">
            <div className="p-3 rounded-xl bg-emerald-500/10 text-emerald-400 border border-emerald-500/20">
              <ShieldCheck className="w-5 h-5" />
            </div>
            <div>
              <div className="text-[11px] font-semibold text-slate-400 uppercase tracking-wider">Encrypted Vaults</div>
              <div className="text-xl font-bold text-white mt-0.5">{repositories.length} Active</div>
              <div className="text-[11px] text-emerald-400 mt-0.5 font-medium flex items-center gap-1">
                <Lock className="w-3 h-3" />
                <span>AES-256-GCM Hardware Encrypted</span>
              </div>
            </div>
          </div>

          <div className="bg-surface-900 border border-surface-800 rounded-xl p-4.5 flex items-start gap-3.5 shadow-sm">
            <div className="p-3 rounded-xl bg-violet-500/10 text-violet-400 border border-violet-500/20">
              <Activity className="w-5 h-5" />
            </div>
            <div>
              <div className="text-[11px] font-semibold text-slate-400 uppercase tracking-wider">Disaster Recovery</div>
              <div className="text-xl font-bold text-white mt-0.5">Armed & Ready</div>
              <div className="text-[11px] text-violet-400 mt-0.5 font-medium">6-Step Automated Runbook</div>
            </div>
          </div>
        </div>

        {/* Navigation Tabs Bar */}
        <div className="flex items-center gap-1.5 p-1.5 bg-surface-900 border border-surface-800 rounded-xl overflow-x-auto text-xs font-semibold shadow-inner">
          <button
            onClick={() => setActiveTab('incremental')}
            className={`flex items-center gap-2 px-3.5 py-2 rounded-lg transition-all cursor-pointer font-medium whitespace-nowrap ${
              activeTab === 'incremental'
                ? 'bg-brand-500 text-white shadow-sm'
                : 'text-slate-400 hover:text-white hover:bg-surface-800'
            }`}
          >
            <Layers className="w-4 h-4" />
            <span>Incremental Snapshots ({incrementalSnapshots.length})</span>
          </button>

          <button
            onClick={() => setActiveTab('repositories')}
            className={`flex items-center gap-2 px-3.5 py-2 rounded-lg transition-all cursor-pointer font-medium whitespace-nowrap ${
              activeTab === 'repositories'
                ? 'bg-brand-500 text-white shadow-sm'
                : 'text-slate-400 hover:text-white hover:bg-surface-800'
            }`}
          >
            <ShieldCheck className="w-4 h-4" />
            <span>Encrypted Repositories ({repositories.length})</span>
          </button>

          <button
            onClick={() => setActiveTab('disaster_recovery')}
            className={`flex items-center gap-2 px-3.5 py-2 rounded-lg transition-all cursor-pointer font-medium whitespace-nowrap ${
              activeTab === 'disaster_recovery'
                ? 'bg-brand-500 text-white shadow-sm'
                : 'text-slate-400 hover:text-white hover:bg-surface-800'
            }`}
          >
            <Terminal className="w-4 h-4" />
            <span>Disaster Recovery Runbook</span>
          </button>

          <button
            onClick={() => setActiveTab('snapshots')}
            className={`flex items-center gap-2 px-3.5 py-2 rounded-lg transition-all cursor-pointer font-medium whitespace-nowrap ${
              activeTab === 'snapshots'
                ? 'bg-brand-500 text-white shadow-sm'
                : 'text-slate-400 hover:text-white hover:bg-surface-800'
            }`}
          >
            <FileArchive className="w-4 h-4" />
            <span>Compressed Archives ({backups.length})</span>
          </button>

          <button
            onClick={() => setActiveTab('destinations')}
            className={`flex items-center gap-2 px-3.5 py-2 rounded-lg transition-all cursor-pointer font-medium whitespace-nowrap ${
              activeTab === 'destinations'
                ? 'bg-brand-500 text-white shadow-sm'
                : 'text-slate-400 hover:text-white hover:bg-surface-800'
            }`}
          >
            <Cloud className="w-4 h-4" />
            <span>Storage Endpoints ({destinations.length})</span>
          </button>

          <button
            onClick={() => setActiveTab('schedules')}
            className={`flex items-center gap-2 px-3.5 py-2 rounded-lg transition-all cursor-pointer font-medium whitespace-nowrap ${
              activeTab === 'schedules'
                ? 'bg-brand-500 text-white shadow-sm'
                : 'text-slate-400 hover:text-white hover:bg-surface-800'
            }`}
          >
            <Calendar className="w-4 h-4" />
            <span>Schedules ({schedules.length})</span>
          </button>
        </div>

        {/* ==================================================================== */}
        {/* TAB 1: INCREMENTAL SNAPSHOTS */}
        {/* ==================================================================== */}
        {activeTab === 'incremental' && (
          <div className="space-y-4">
            {/* Filter Bar */}
            <div className="flex flex-col md:flex-row items-center justify-between gap-3 bg-surface-900 border border-surface-800 p-3 rounded-xl">
              <div className="flex items-center gap-2 w-full md:w-auto flex-wrap">
                <div className="flex items-center gap-1 p-1 bg-surface-950 border border-surface-800 rounded-lg text-xs">
                  {(['all', 'website', 'database', 'mail', 'full_account'] as const).map((sc) => (
                    <button
                      key={sc}
                      onClick={() => setIncScopeFilter(sc)}
                      className={`px-3 py-1 rounded-md font-semibold capitalize transition-all cursor-pointer ${
                        incScopeFilter === sc ? 'bg-brand-500 text-white shadow-xs' : 'text-slate-400 hover:text-white'
                      }`}
                    >
                      {sc === 'all' ? 'All Scopes' : sc === 'full_account' ? 'Full Account' : sc}
                    </button>
                  ))}
                </div>

                {repositories.length > 0 && (
                  <select
                    value={selectedRepoFilter}
                    onChange={(e) => setSelectedRepoFilter(e.target.value)}
                    className="px-3 py-1.5 rounded-lg bg-surface-950 border border-surface-800 text-slate-300 text-xs font-medium focus:outline-hidden focus:border-brand-500"
                  >
                    <option value="all">All Repositories</option>
                    {repositories.map((r) => (
                      <option key={r.id} value={r.id}>
                        {r.name} ({r.provider})
                      </option>
                    ))}
                  </select>
                )}
              </div>

              <div className="relative w-full md:w-64">
                <Search className="w-3.5 h-3.5 absolute left-3 top-1/2 -translate-y-1/2 text-slate-400" />
                <input
                  type="text"
                  placeholder="Search snapshot hash or target..."
                  value={incSearchQuery}
                  onChange={(e) => setIncSearchQuery(e.target.value)}
                  className="w-full pl-9 pr-3 py-1.5 rounded-lg bg-surface-950 border border-surface-800 text-white text-xs placeholder:text-slate-500 focus:outline-hidden focus:border-brand-500"
                />
              </div>
            </div>

            {/* Snapshots Table */}
            <div className="bg-surface-900 border border-surface-800 rounded-2xl overflow-hidden shadow-sm">
              <div className="overflow-x-auto">
                <table className="w-full text-left border-collapse text-xs">
                  <thead>
                    <tr className="bg-surface-950/80 border-b border-surface-800 text-slate-400 font-semibold uppercase tracking-wider text-[10px]">
                      <th className="py-3.5 px-4">Snapshot / Scope</th>
                      <th className="py-3.5 px-4">Repository</th>
                      <th className="py-3.5 px-4">Deduplication Ratio</th>
                      <th className="py-3.5 px-4">Raw Size vs Stored</th>
                      <th className="py-3.5 px-4">Consistency Seals</th>
                      <th className="py-3.5 px-4">Created</th>
                      <th className="py-3.5 px-4 text-right">Actions</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-surface-800/60 font-sans">
                    {filteredIncrementalSnapshots.length === 0 ? (
                      <tr>
                        <td colSpan={7} className="py-12 text-center text-slate-500">
                          <Layers className="w-10 h-10 mx-auto text-slate-600 mb-3 opacity-60" />
                          <div className="font-semibold text-slate-300 text-sm">No Incremental Snapshots Found</div>
                          <div className="text-xs text-slate-400 mt-1 max-w-sm mx-auto">
                            Create your first incremental snapshot to enable content-addressed chunk deduplication and instant file browsing.
                          </div>
                          <button
                            onClick={() => setNewSnapModalOpen(true)}
                            className="mt-4 inline-flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-brand-500 hover:bg-brand-600 text-white text-xs font-semibold cursor-pointer shadow-sm"
                          >
                            <Plus className="w-3.5 h-3.5" />
                            <span>Create Snapshot Now</span>
                          </button>
                        </td>
                      </tr>
                    ) : (
                      filteredIncrementalSnapshots.map((snap) => {
                        const repo = repositories.find((r) => r.id === snap.repository_id);
                        const dedup = snap.stored_size_bytes > 0 ? (snap.raw_size_bytes / snap.stored_size_bytes).toFixed(1) : '1.0';

                        return (
                          <tr key={snap.id} className="hover:bg-surface-850/50 transition-colors">
                            <td className="py-3.5 px-4">
                              <div className="flex items-center gap-2.5">
                                <div className="p-2 rounded-lg bg-brand-500/10 text-brand-400 border border-brand-500/20">
                                  {snap.scope === 'database' ? (
                                    <Database className="w-4 h-4" />
                                  ) : snap.scope === 'mail' ? (
                                    <Mail className="w-4 h-4" />
                                  ) : (
                                    <Layers className="w-4 h-4" />
                                  )}
                                </div>
                                <div>
                                  <div className="font-semibold text-white flex items-center gap-2">
                                    <span>{snap.target_name || 'System Wide'}</span>
                                    <span className="px-2 py-0.5 text-[9px] font-bold uppercase rounded-md bg-surface-800 text-slate-300 border border-surface-700">
                                      {snap.scope}
                                    </span>
                                  </div>
                                  <div className="font-mono text-[11px] text-slate-400 flex items-center gap-1 mt-0.5">
                                    <span>{snap.snapshot_hash.substring(0, 16)}</span>
                                    <button
                                      onClick={() => handleCopy(snap.snapshot_hash, snap.id)}
                                      className="text-slate-500 hover:text-slate-300 cursor-pointer"
                                      title="Copy Hash"
                                    >
                                      {copiedText === snap.id ? <Check className="w-3 h-3 text-emerald-400" /> : <Copy className="w-3 h-3" />}
                                    </button>
                                  </div>
                                </div>
                              </div>
                            </td>

                            <td className="py-3.5 px-4">
                              <div className="text-slate-200 font-medium">{repo?.name || 'Local Vault'}</div>
                              <div className="text-[10px] text-slate-500 font-mono">{repo?.provider || 'native'}</div>
                            </td>

                            <td className="py-3.5 px-4">
                              <div className="inline-flex items-center gap-1 px-2.5 py-1 rounded-md bg-brand-500/10 text-brand-300 border border-brand-500/20 font-bold font-mono">
                                <Scissors className="w-3 h-3 text-brand-400" />
                                <span>{dedup}x Saved</span>
                              </div>
                              <div className="text-[10px] text-slate-500 mt-1">{snap.total_files || 1} files deduplicated</div>
                            </td>

                            <td className="py-3.5 px-4">
                              <div className="text-white font-semibold font-mono">{formatBytes(snap.stored_size_bytes)}</div>
                              <div className="text-[10px] text-slate-500 font-mono">Raw: {formatBytes(snap.raw_size_bytes)}</div>
                            </td>

                            <td className="py-3.5 px-4">
                              <div className="space-y-1">
                                {snap.database_snapshot_consistent && (
                                  <div className="flex items-center gap-1 text-[11px] text-emerald-400">
                                    <CheckCircle2 className="w-3.5 h-3.5" />
                                    <span>Transactional DB Snap</span>
                                  </div>
                                )}
                                {snap.status === 'completed' && (
                                  <div className="flex items-center gap-1 text-[11px] text-slate-300">
                                    <ShieldCheck className="w-3.5 h-3.5 text-blue-400" />
                                    <span>Sealed & Checksummed</span>
                                  </div>
                                )}
                              </div>
                            </td>

                            <td className="py-3.5 px-4 text-slate-400">
                              <div>{new Date(snap.started_at || snap.created_at).toLocaleDateString()}</div>
                              <div className="text-[10px] text-slate-500">
                                {new Date(snap.started_at || snap.created_at).toLocaleTimeString()}
                              </div>
                            </td>

                            <td className="py-3.5 px-4 text-right">
                              <div className="flex items-center justify-end gap-1.5">
                                <button
                                  onClick={() => handleOpenFileTree(snap)}
                                  className="inline-flex items-center gap-1 px-2.5 py-1.5 rounded-lg bg-surface-800 hover:bg-surface-750 text-slate-200 hover:text-white border border-surface-700 text-xs font-semibold cursor-pointer shadow-xs transition-colors"
                                  title="Browse File Tree & Restore Selectively"
                                >
                                  <FolderTree className="w-3.5 h-3.5 text-brand-400" />
                                  <span>Browse Files</span>
                                </button>

                                <button
                                  onClick={() => {
                                    setFullRestoreSnapshot(snap);
                                    setFullRestoreTargetDir(snap.scope === 'website' ? `/var/www/${snap.target_name || ''}` : '/var/www');
                                  }}
                                  className="inline-flex items-center gap-1 px-2.5 py-1.5 rounded-lg bg-brand-500/10 hover:bg-brand-500 text-brand-300 hover:text-white border border-brand-500/20 text-xs font-semibold cursor-pointer shadow-xs transition-colors"
                                  title="Full Account / System Restore"
                                >
                                  <RotateCcw className="w-3.5 h-3.5" />
                                  <span>Full Restore</span>
                                </button>
                              </div>
                            </td>
                          </tr>
                        );
                      })
                    )}
                  </tbody>
                </table>
              </div>
            </div>
          </div>
        )}

        {/* ==================================================================== */}
        {/* TAB 2: ENCRYPTED REPOSITORIES & VAULTS */}
        {/* ==================================================================== */}
        {activeTab === 'repositories' && (
          <div className="space-y-4">
            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
              {repositories.length === 0 ? (
                <div className="col-span-full py-12 text-center bg-surface-900 border border-surface-800 rounded-2xl">
                  <ShieldCheck className="w-12 h-12 mx-auto text-slate-600 mb-3 opacity-60" />
                  <div className="font-semibold text-slate-300 text-base">No Encrypted Repositories Configured</div>
                  <div className="text-xs text-slate-400 mt-1 max-w-md mx-auto">
                    Create an off-site repository on Wasabi, Cloudflare R2, Backblaze B2, AWS S3, or Local/NFS storage with hardware-accelerated AES-256-GCM encryption.
                  </div>
                  <button
                    onClick={() => {
                      generatePassphrase();
                      handlePresetSelect('wasabi');
                      setNewRepoModalOpen(true);
                    }}
                    className="mt-4 inline-flex items-center gap-2 px-4 py-2 rounded-xl bg-brand-500 hover:bg-brand-600 text-white text-xs font-semibold cursor-pointer shadow-md shadow-brand-500/20"
                  >
                    <Plus className="w-4 h-4" />
                    <span>Create Encrypted Vault</span>
                  </button>
                </div>
              ) : (
                repositories.map((repo) => {
                  const isHealthy = repo.integrity_status === 'healthy';
                  const isChecking = checkingIntegrityId === repo.id;

                  return (
                    <div
                      key={repo.id}
                      className="bg-surface-900 border border-surface-800 hover:border-surface-700 rounded-2xl p-5 shadow-sm transition-all flex flex-col justify-between"
                    >
                      <div>
                        {/* Header */}
                        <div className="flex items-start justify-between gap-3">
                          <div className="flex items-center gap-3">
                            <div className="p-2.5 rounded-xl bg-brand-500/10 text-brand-400 border border-brand-500/20">
                              <Cloud className="w-5 h-5" />
                            </div>
                            <div>
                              <h3 className="text-sm font-bold text-white tracking-tight">{repo.name}</h3>
                              <div className="text-[10px] text-slate-400 font-mono mt-0.5 break-all line-clamp-1">
                                {repo.repo_uri}
                              </div>
                            </div>
                          </div>

                          <span
                            className={`px-2 py-0.5 text-[9px] font-bold uppercase rounded-md border flex items-center gap-1 ${
                              isHealthy
                                ? 'bg-emerald-500/10 text-emerald-400 border-emerald-500/20'
                                : repo.integrity_status === 'error'
                                ? 'bg-rose-500/10 text-rose-400 border-rose-500/20'
                                : 'bg-amber-500/10 text-amber-400 border-amber-500/20'
                            }`}
                          >
                            <span className={`w-1.5 h-1.5 rounded-full ${isHealthy ? 'bg-emerald-400' : 'bg-amber-400'}`} />
                            {repo.integrity_status}
                          </span>
                        </div>

                        {/* Specs grid */}
                        <div className="grid grid-cols-2 gap-2 mt-4 pt-3 border-t border-surface-800/80 text-xs font-mono">
                          <div className="bg-surface-950 p-2.5 rounded-xl border border-surface-800">
                            <span className="text-[10px] text-slate-500 uppercase font-sans block">Snapshots</span>
                            <span className="text-white font-bold">{repo.total_snapshots} items</span>
                          </div>
                          <div className="bg-surface-950 p-2.5 rounded-xl border border-surface-800">
                            <span className="text-[10px] text-slate-500 uppercase font-sans block">Deduplicated Size</span>
                            <span className="text-white font-bold">{formatBytes(repo.deduplicated_size_bytes)}</span>
                          </div>
                          <div className="bg-surface-950 p-2.5 rounded-xl border border-surface-800">
                            <span className="text-[10px] text-slate-500 uppercase font-sans block">Encryption</span>
                            <span className="text-brand-400 font-semibold text-[11px]">{repo.encryption_algorithm || 'AES-256-GCM'}</span>
                          </div>
                          <div className="bg-surface-950 p-2.5 rounded-xl border border-surface-800">
                            <span className="text-[10px] text-slate-500 uppercase font-sans block">Engine Provider</span>
                            <span className="text-slate-300 font-semibold capitalize">{repo.provider}</span>
                          </div>
                        </div>

                        {/* Deletion protection safeguard badge */}
                        <div className="mt-3 flex items-center justify-between p-2.5 rounded-xl bg-surface-950 border border-surface-800 text-xs">
                          <div className="flex items-center gap-2">
                            {repo.deletion_protection ? (
                              <Lock className="w-4 h-4 text-emerald-400" />
                            ) : (
                              <Unlock className="w-4 h-4 text-amber-400" />
                            )}
                            <span className="text-[11px] text-slate-300">
                              {repo.deletion_protection ? 'Deletion Protection Armed' : 'Deletion Protection Disabled'}
                            </span>
                          </div>
                          <button
                            onClick={() => handleToggleDeletionProtection(repo)}
                            className="text-[10px] font-semibold text-brand-400 hover:text-brand-300 cursor-pointer"
                          >
                            {repo.deletion_protection ? 'Disable' : 'Enable'}
                          </button>
                        </div>
                      </div>

                      {/* Footer Actions */}
                      <div className="mt-4 pt-3 border-t border-surface-800 flex items-center justify-between gap-2">
                        <div className="flex items-center gap-1.5">
                          <button
                            onClick={() => handleCheckIntegrity(repo.id)}
                            disabled={isChecking}
                            className="inline-flex items-center gap-1 px-2.5 py-1.5 rounded-lg bg-surface-800 hover:bg-surface-750 text-slate-200 hover:text-white border border-surface-700 text-xs font-semibold cursor-pointer transition-colors disabled:opacity-50"
                            title="Run Cryptographic Integrity Drill"
                          >
                            <ShieldCheck className={`w-3.5 h-3.5 ${isChecking ? 'animate-spin text-amber-400' : 'text-emerald-400'}`} />
                            <span>{isChecking ? 'Checking...' : 'Check Integrity'}</span>
                          </button>

                          <button
                            onClick={() => handleOpenPruneModal(repo)}
                            className="inline-flex items-center gap-1 px-2.5 py-1.5 rounded-lg bg-surface-800 hover:bg-surface-750 text-slate-200 hover:text-white border border-surface-700 text-xs font-semibold cursor-pointer transition-colors"
                            title="Retention Policy & Prune Simulator"
                          >
                            <Scissors className="w-3.5 h-3.5 text-blue-400" />
                            <span>Retention</span>
                          </button>
                        </div>

                        <button
                          onClick={() => handleDeleteRepository(repo)}
                          className="p-1.5 rounded-lg text-slate-500 hover:text-rose-400 hover:bg-rose-500/10 transition-colors cursor-pointer"
                          title="Delete Repository"
                        >
                          <Trash2 className="w-4 h-4" />
                        </button>
                      </div>
                    </div>
                  );
                })
              )}
            </div>
          </div>
        )}

        {/* ==================================================================== */}
        {/* TAB 3: DISASTER RECOVERY RUNBOOK GENERATOR */}
        {/* ==================================================================== */}
        {activeTab === 'disaster_recovery' && (
          <div className="space-y-4">
            <div className="bg-surface-900 border border-surface-800 p-6 rounded-2xl shadow-sm">
              <div className="max-w-2xl">
                <div className="flex items-center gap-2 text-violet-400 font-bold text-xs uppercase tracking-wider">
                  <Terminal className="w-4 h-4" />
                  <span>Interactive Disaster Recovery Generator</span>
                </div>
                <h2 className="text-xl font-bold text-white mt-1">Generate Instant Step-by-Step Restoration Runbook</h2>
                <p className="text-xs text-slate-400 mt-1">
                  In case of bare-metal server failure, datacenter outage, or migration to a fresh node, generate an end-to-end executable runbook with copyable shell commands and pre-validated credentials.
                </p>

                <div className="grid grid-cols-1 sm:grid-cols-2 gap-3 mt-4">
                  <div>
                    <label className="text-[11px] font-semibold text-slate-300 uppercase tracking-wide block mb-1">
                      Source Backup Repository
                    </label>
                    <select
                      value={drRepoId}
                      onChange={(e) => setDrRepoId(e.target.value)}
                      className="w-full px-3 py-2 rounded-xl bg-surface-950 border border-surface-800 text-white text-xs focus:outline-hidden focus:border-brand-500"
                    >
                      {repositories.map((r) => (
                        <option key={r.id} value={r.id}>
                          {r.name} ({r.repo_uri})
                        </option>
                      ))}
                    </select>
                  </div>

                  <div>
                    <label className="text-[11px] font-semibold text-slate-300 uppercase tracking-wide block mb-1">
                      Target Recovery Hostname
                    </label>
                    <input
                      type="text"
                      value={drTargetHost}
                      onChange={(e) => setDrTargetHost(e.target.value)}
                      placeholder="recovery-node-01.hostvra.internal"
                      className="w-full px-3 py-2 rounded-xl bg-surface-950 border border-surface-800 text-white text-xs focus:outline-hidden focus:border-brand-500"
                    />
                  </div>
                </div>

                <div className="flex items-center gap-2 mt-4">
                  <button
                    onClick={handleGenerateRunbook}
                    disabled={drLoading}
                    className="inline-flex items-center gap-2 px-4 py-2 rounded-xl bg-gradient-to-r from-violet-600 to-indigo-600 hover:from-violet-500 hover:to-indigo-500 text-white text-xs font-semibold shadow-md shadow-violet-500/20 cursor-pointer transition-all disabled:opacity-50"
                  >
                    <Sparkles className="w-4 h-4" />
                    <span>{drLoading ? 'Generating Runbook...' : 'Generate 6-Step Runbook'}</span>
                  </button>

                  {drRunbook && (
                    <button
                      onClick={handleDownloadRunbookMd}
                      className="inline-flex items-center gap-1.5 px-3.5 py-2 rounded-xl bg-surface-800 hover:bg-surface-750 text-slate-200 text-xs font-semibold border border-surface-700 cursor-pointer transition-colors"
                    >
                      <Download className="w-3.5 h-3.5" />
                      <span>Download Markdown (.md)</span>
                    </button>
                  )}
                </div>
              </div>
            </div>

            {/* Runbook Steps Display */}
            {drRunbook && (
              <div className="space-y-4 animate-in fade-in slide-in-from-bottom-2">
                <div className="p-4 rounded-xl bg-amber-500/10 border border-amber-500/20 text-amber-300 text-xs flex items-center gap-3">
                  <ShieldAlert className="w-5 h-5 flex-shrink-0 text-amber-400" />
                  <span>
                    <strong>Disaster Recovery Drill Note:</strong> Always restore into an isolated environment or sandbox directory first to verify application data and database schemas before changing DNS records.
                  </span>
                </div>

                {drRunbook.steps.map((step) => (
                  <div key={step.step_number} className="bg-surface-900 border border-surface-800 rounded-2xl p-5 shadow-sm">
                    <div className="flex items-start justify-between gap-4">
                      <div className="flex items-center gap-3">
                        <div className="w-7 h-7 rounded-lg bg-brand-500/20 text-brand-400 font-bold text-xs flex items-center justify-center border border-brand-500/30">
                          {step.step_number}
                        </div>
                        <div>
                          <h3 className="text-sm font-bold text-white">{step.title}</h3>
                          <p className="text-xs text-slate-400 mt-0.5">{step.description}</p>
                        </div>
                      </div>

                      {step.commands && step.commands.length > 0 && (
                        <button
                          onClick={() => handleCopy(step.commands.join('\n'), `step-${step.step_number}`)}
                          className="inline-flex items-center gap-1 px-2.5 py-1 rounded-lg bg-surface-800 hover:bg-surface-750 text-slate-300 hover:text-white border border-surface-700 text-[11px] font-semibold cursor-pointer transition-colors"
                        >
                          {copiedText === `step-${step.step_number}` ? (
                            <>
                              <Check className="w-3.5 h-3.5 text-emerald-400" />
                              <span>Copied!</span>
                            </>
                          ) : (
                            <>
                              <Copy className="w-3.5 h-3.5" />
                              <span>Copy Commands</span>
                            </>
                          )}
                        </button>
                      )}
                    </div>

                    {step.commands && step.commands.length > 0 && (
                      <div className="mt-3 bg-surface-950 border border-surface-800 rounded-xl p-3 font-mono text-xs text-emerald-400 overflow-x-auto">
                        {step.commands.map((cmd, idx) => (
                          <div key={idx} className="flex items-start gap-2 py-0.5">
                            <span className="text-slate-600 select-none">$</span>
                            <span className="text-slate-200 select-all">{cmd}</span>
                          </div>
                        ))}
                      </div>
                    )}

                    {step.safety_note && (
                      <div className="mt-2.5 text-[11px] text-slate-400 flex items-center gap-1.5 italic">
                        <CheckCircle2 className="w-3.5 h-3.5 text-brand-400 flex-shrink-0" />
                        <span>{step.safety_note}</span>
                      </div>
                    )}
                  </div>
                ))}
              </div>
            )}
          </div>
        )}

        {/* ==================================================================== */}
        {/* TAB 4: COMPRESSED ARCHIVES (PRESERVED LEGACY COMPATIBILITY) */}
        {/* ==================================================================== */}
        {activeTab === 'snapshots' && (
          <div className="space-y-4">
            <div className="flex flex-col sm:flex-row items-center justify-between gap-3 bg-surface-900 border border-surface-800 p-3 rounded-xl">
              <div className="flex items-center gap-1.5 p-1 bg-surface-950 border border-surface-800 rounded-lg w-full sm:w-auto">
                {(['all', 'website', 'database', 'full_config', 'full_account'] as const).map((scope) => (
                  <button
                    key={scope}
                    onClick={() => setScopeFilter(scope)}
                    className={`px-3 py-1.5 rounded-md text-xs font-semibold capitalize transition-all cursor-pointer ${
                      scopeFilter === scope ? 'bg-brand-500 text-white shadow-xs' : 'text-slate-400 hover:text-white'
                    }`}
                  >
                    {scope === 'all' ? 'All Scopes' : scope.replace('_', ' ')}
                  </button>
                ))}
              </div>

              <div className="relative w-full sm:w-64">
                <Search className="w-3.5 h-3.5 absolute left-3 top-1/2 -translate-y-1/2 text-slate-400" />
                <input
                  type="text"
                  placeholder="Filter legacy archives..."
                  value={searchQuery}
                  onChange={(e) => setSearchQuery(e.target.value)}
                  className="w-full pl-9 pr-3 py-1.5 rounded-lg bg-surface-950 border border-surface-800 text-white text-xs placeholder:text-slate-500 focus:outline-hidden focus:border-brand-500"
                />
              </div>
            </div>

            <div className="bg-surface-900 border border-surface-800 rounded-2xl overflow-hidden shadow-sm">
              <div className="overflow-x-auto">
                <table className="w-full text-left border-collapse text-xs">
                  <thead>
                    <tr className="bg-surface-950/80 border-b border-surface-800 text-slate-400 font-semibold uppercase tracking-wider text-[10px]">
                      <th className="py-3 px-4">Archive Scope</th>
                      <th className="py-3 px-4">Storage Vault</th>
                      <th className="py-3 px-4">Size</th>
                      <th className="py-3 px-4">SHA-256 Digest</th>
                      <th className="py-3 px-4">Created</th>
                      <th className="py-3 px-4 text-right">Actions</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-surface-800/60 font-sans">
                    {backups.length === 0 ? (
                      <tr>
                        <td colSpan={6} className="py-10 text-center text-slate-500">
                          <FileArchive className="w-8 h-8 mx-auto text-slate-600 mb-2 opacity-50" />
                          <div className="font-semibold text-slate-300">No Compressed Archives Found</div>
                        </td>
                      </tr>
                    ) : (
                      backups.map((b) => (
                        <tr key={b.id} className="hover:bg-surface-850/50 transition-colors">
                          <td className="py-3 px-4 font-semibold text-white">
                            <div className="flex items-center gap-2">
                              <span>{b.target_name}</span>
                              <span className="px-2 py-0.5 text-[9px] font-bold uppercase rounded-md bg-surface-800 text-slate-300">
                                {b.type}
                              </span>
                            </div>
                          </td>
                          <td className="py-3 px-4 text-slate-300 uppercase">{b.storage}</td>
                          <td className="py-3 px-4 font-mono text-slate-300">{formatBytes(b.size_bytes)}</td>
                          <td className="py-3 px-4 font-mono text-[11px] text-slate-400">
                            {b.sha256 ? b.sha256.substring(0, 16) + '...' : 'Pending'}
                          </td>
                          <td className="py-3 px-4 text-slate-400">{new Date(b.created_at).toLocaleDateString()}</td>
                          <td className="py-3 px-4 text-right">
                            <span className="text-[11px] text-emerald-400 font-semibold">Available</span>
                          </td>
                        </tr>
                      ))
                    )}
                  </tbody>
                </table>
              </div>
            </div>
          </div>
        )}

        {/* ==================================================================== */}
        {/* TAB 5 & 6: LEGACY DESTINATIONS & SCHEDULES (PRESERVED) */}
        {/* ==================================================================== */}
        {activeTab === 'destinations' && (
          <div className="bg-surface-900 border border-surface-800 rounded-2xl p-6 text-center text-slate-400 text-xs">
            <Cloud className="w-10 h-10 mx-auto text-brand-400 mb-2 opacity-60" />
            <div className="text-sm font-bold text-white">Remote Storage Endpoints</div>
            <p className="max-w-md mx-auto mt-1">
              Existing endpoints are synchronized with Phase 9 Encrypted Repositories for continuous disaster recovery protection.
            </p>
          </div>
        )}

        {activeTab === 'schedules' && (
          <div className="bg-surface-900 border border-surface-800 rounded-2xl p-6 text-center text-slate-400 text-xs">
            <Calendar className="w-10 h-10 mx-auto text-brand-400 mb-2 opacity-60" />
            <div className="text-sm font-bold text-white">Automated Schedules</div>
            <p className="max-w-md mx-auto mt-1">
              Schedules are integrated directly into repository retention policies with automated pruning safeguards.
            </p>
          </div>
        )}

        {/* ==================================================================== */}
        {/* MODAL: FILE TREE PREVIEW & SELECTIVE RESTORE */}
        {/* ==================================================================== */}
        {fileTreeModalOpen && fileTreeSnapshot && (
          <div className="fixed inset-0 z-50 bg-black/75 backdrop-blur-xs flex items-center justify-center p-4">
            <div className="bg-surface-900 border border-surface-800 rounded-2xl max-w-3xl w-full p-6 space-y-4 shadow-2xl animate-in zoom-in-95">
              <div className="flex items-center justify-between border-b border-surface-800 pb-3">
                <div className="flex items-center gap-2.5">
                  <FolderTree className="w-5 h-5 text-brand-400" />
                  <div>
                    <h3 className="text-sm font-bold text-white">Snapshot File Explorer & Selective Restore</h3>
                    <div className="text-[11px] text-slate-400 font-mono">
                      {fileTreeSnapshot.target_name} ({fileTreeSnapshot.snapshot_hash.substring(0, 16)})
                    </div>
                  </div>
                </div>
                <button
                  onClick={() => setFileTreeModalOpen(false)}
                  className="text-slate-400 hover:text-white transition-colors cursor-pointer"
                >
                  <X className="w-4 h-4" />
                </button>
              </div>

              {/* Breadcrumb Path */}
              <div className="flex items-center gap-1.5 px-3 py-2 bg-surface-950 border border-surface-800 rounded-xl text-xs font-mono text-slate-300">
                <Folder className="w-3.5 h-3.5 text-brand-400" />
                <span>/{fileTreePath}</span>
              </div>

              {/* Files Table */}
              <div className="max-h-72 overflow-y-auto border border-surface-800 rounded-xl bg-surface-950/60 divide-y divide-surface-800 text-xs font-mono">
                {fileTreeLoading ? (
                  <div className="py-12 text-center text-slate-400">
                    <RefreshCw className="w-6 h-6 animate-spin mx-auto text-brand-400 mb-2" />
                    <span>Browsing virtual repository tree...</span>
                  </div>
                ) : fileTreeNodes.length === 0 ? (
                  <div className="py-8 text-center text-slate-500">No files found at this path.</div>
                ) : (
                  fileTreeNodes.map((node) => {
                    const isSelected = selectedFilePaths.includes(node.path);
                    return (
                      <div
                        key={node.path}
                        className={`flex items-center justify-between p-2.5 hover:bg-surface-850/80 transition-colors ${
                          isSelected ? 'bg-brand-500/10' : ''
                        }`}
                      >
                        <div className="flex items-center gap-2.5">
                          <input
                            type="checkbox"
                            checked={isSelected}
                            onChange={(e) => {
                              if (e.target.checked) {
                                setSelectedFilePaths((prev) => [...prev, node.path]);
                              } else {
                                setSelectedFilePaths((prev) => prev.filter((p) => p !== node.path));
                              }
                            }}
                            className="rounded border-surface-700 bg-surface-900 text-brand-500 focus:ring-0 cursor-pointer"
                          />
                          {node.type === 'dir' ? (
                            <Folder className="w-4 h-4 text-amber-400" />
                          ) : (
                            <FileCode className="w-4 h-4 text-blue-400" />
                          )}
                          <span className="text-slate-200 select-all">{node.name}</span>
                        </div>
                        <div className="flex items-center gap-4 text-slate-500 text-[11px]">
                          <span>{node.mode || '-rw-r--r--'}</span>
                          <span className="w-16 text-right">{node.type === 'dir' ? 'DIR' : formatBytes(node.size_bytes)}</span>
                        </div>
                      </div>
                    );
                  })
                )}
              </div>

              {/* Restore destination settings */}
              <div className="bg-surface-950 border border-surface-800 rounded-xl p-4 space-y-3">
                <div className="text-xs font-bold text-white">Restoration Target Directory</div>
                <div className="grid grid-cols-1 sm:grid-cols-2 gap-3 text-xs">
                  <div>
                    <label className="text-[11px] text-slate-400 font-semibold block mb-1">Restore Mode</label>
                    <select
                      value={restoreMode}
                      onChange={(e) => setRestoreMode(e.target.value as any)}
                      className="w-full px-3 py-1.5 rounded-lg bg-surface-900 border border-surface-800 text-white"
                    >
                      <option value="alternate">Alternate Destination (Safe Sandbox)</option>
                      <option value="in_place">In-Place Overwrite (Production)</option>
                    </select>
                  </div>
                  <div>
                    <label className="text-[11px] text-slate-400 font-semibold block mb-1">Destination Path</label>
                    <input
                      type="text"
                      value={restoreTargetDir}
                      onChange={(e) => setRestoreTargetDir(e.target.value)}
                      className="w-full px-3 py-1.5 rounded-lg bg-surface-900 border border-surface-800 text-white font-mono"
                    />
                  </div>
                </div>
              </div>

              {/* Modal Footer */}
              <div className="flex items-center justify-between pt-2">
                <div className="text-xs text-slate-400">
                  <span className="text-white font-bold">{selectedFilePaths.length}</span> items selected for restore
                </div>

                <div className="flex items-center gap-2">
                  <button
                    type="button"
                    onClick={() => setFileTreeModalOpen(false)}
                    className="px-4 py-2 rounded-xl bg-surface-800 hover:bg-surface-750 text-slate-300 text-xs font-semibold cursor-pointer"
                  >
                    Cancel
                  </button>

                  <button
                    type="button"
                    onClick={handleRestoreSelectedFiles}
                    disabled={restoringFiles || selectedFilePaths.length === 0}
                    className="inline-flex items-center gap-2 px-4 py-2 rounded-xl bg-gradient-to-r from-brand-500 to-brand-600 hover:from-brand-600 hover:to-brand-700 text-white text-xs font-semibold shadow-md cursor-pointer disabled:opacity-50"
                  >
                    <RotateCcw className={`w-3.5 h-3.5 ${restoringFiles ? 'animate-spin' : ''}`} />
                    <span>{restoringFiles ? 'Restoring...' : 'Restore Selected Files'}</span>
                  </button>
                </div>
              </div>
            </div>
          </div>
        )}

        {/* ==================================================================== */}
        {/* MODAL: FULL ACCOUNT RESTORE */}
        {/* ==================================================================== */}
        {fullRestoreSnapshot && (
          <div className="fixed inset-0 z-50 bg-black/75 backdrop-blur-xs flex items-center justify-center p-4">
            <div className="bg-surface-900 border border-surface-800 rounded-2xl max-w-lg w-full p-6 space-y-4 shadow-2xl animate-in zoom-in-95">
              <div className="flex items-center justify-between border-b border-surface-800 pb-3">
                <div className="flex items-center gap-2">
                  <RotateCcw className="w-5 h-5 text-brand-400" />
                  <h3 className="text-sm font-bold text-white">Full Account & Workload Restoration</h3>
                </div>
                <button onClick={() => setFullRestoreSnapshot(null)} className="text-slate-400 hover:text-white cursor-pointer">
                  <X className="w-4 h-4" />
                </button>
              </div>

              <div className="p-3.5 rounded-xl bg-amber-500/10 border border-amber-500/20 text-amber-300 text-xs flex items-start gap-2.5">
                <ShieldCheck className="w-5 h-5 text-amber-400 flex-shrink-0" />
                <div>
                  <div className="font-bold">Rollback Safeguard Armed</div>
                  <div>
                    A pre-restore staging checkpoint will be created automatically before any files or databases are unpacked.
                  </div>
                </div>
              </div>

              <div className="space-y-3 text-xs">
                <div>
                  <label className="text-[11px] font-semibold text-slate-300 uppercase tracking-wide block mb-1">
                    Target Unpack Directory
                  </label>
                  <input
                    type="text"
                    value={fullRestoreTargetDir}
                    onChange={(e) => setFullRestoreTargetDir(e.target.value)}
                    className="w-full px-3 py-2 rounded-xl bg-surface-950 border border-surface-800 text-white font-mono"
                  />
                </div>

                <div className="p-3 bg-surface-950 border border-surface-800 rounded-xl space-y-2">
                  <label className="flex items-center gap-2 cursor-pointer">
                    <input
                      type="checkbox"
                      checked={fullRestoreDb}
                      onChange={(e) => setFullRestoreDb(e.target.checked)}
                      className="rounded border-surface-700 bg-surface-900 text-brand-500"
                    />
                    <span className="text-slate-200">Restore Databases (Apply transactional dump)</span>
                  </label>

                  <label className="flex items-center gap-2 cursor-pointer">
                    <input
                      type="checkbox"
                      checked={fullRestoreMail}
                      onChange={(e) => setFullRestoreMail(e.target.checked)}
                      className="rounded border-surface-700 bg-surface-900 text-brand-500"
                    />
                    <span className="text-slate-200">Restore Mailboxes (Restore /var/mail/vhosts)</span>
                  </label>

                  <label className="flex items-center gap-2 cursor-pointer">
                    <input
                      type="checkbox"
                      checked={fullRestoreDryRun}
                      onChange={(e) => setFullRestoreDryRun(e.target.checked)}
                      className="rounded border-surface-700 bg-surface-900 text-brand-500"
                    />
                    <span className="text-amber-300 font-semibold">Dry-Run Simulation (Verify integrity without overwrite)</span>
                  </label>
                </div>
              </div>

              <div className="flex items-center justify-end gap-2 pt-2 border-t border-surface-800">
                <button
                  type="button"
                  onClick={() => setFullRestoreSnapshot(null)}
                  className="px-4 py-2 rounded-xl bg-surface-800 hover:bg-surface-750 text-slate-300 text-xs font-semibold cursor-pointer"
                >
                  Cancel
                </button>

                <button
                  type="button"
                  onClick={handleExecuteFullRestore}
                  disabled={submittingFullRestore}
                  className="inline-flex items-center gap-2 px-4 py-2 rounded-xl bg-brand-500 hover:bg-brand-600 text-white text-xs font-semibold cursor-pointer disabled:opacity-50"
                >
                  <RotateCcw className={`w-3.5 h-3.5 ${submittingFullRestore ? 'animate-spin' : ''}`} />
                  <span>{submittingFullRestore ? 'Restoring Workload...' : 'Confirm Full Restore'}</span>
                </button>
              </div>
            </div>
          </div>
        )}

        {/* ==================================================================== */}
        {/* MODAL: CREATE ENCRYPTED REPOSITORY */}
        {/* ==================================================================== */}
        {newRepoModalOpen && (
          <div className="fixed inset-0 z-50 bg-black/75 backdrop-blur-xs flex items-center justify-center p-4">
            <div className="bg-surface-900 border border-surface-800 rounded-2xl max-w-xl w-full p-6 space-y-4 shadow-2xl animate-in zoom-in-95 max-h-[90vh] overflow-y-auto">
              <div className="flex items-center justify-between border-b border-surface-800 pb-3">
                <div className="flex items-center gap-2">
                  <ShieldCheck className="w-5 h-5 text-brand-400" />
                  <h3 className="text-sm font-bold text-white">Create Encrypted Backup Repository</h3>
                </div>
                <button onClick={() => setNewRepoModalOpen(false)} className="text-slate-400 hover:text-white cursor-pointer">
                  <X className="w-4 h-4" />
                </button>
              </div>

              <form onSubmit={handleCreateRepository} className="space-y-4 text-xs">
                {/* Cloud Provider Preset Buttons */}
                <div>
                  <label className="text-[11px] font-semibold text-slate-400 uppercase tracking-wide block mb-1.5">
                    Storage Provider Preset
                  </label>
                  <div className="grid grid-cols-3 gap-2">
                    {[
                      { id: 'wasabi', label: 'Wasabi Hot Cloud' },
                      { id: 'cloudflare_r2', label: 'Cloudflare R2' },
                      { id: 'backblaze_b2', label: 'Backblaze B2' },
                      { id: 's3', label: 'Amazon S3' },
                      { id: 'local', label: 'Local Directory' },
                      { id: 'nfs', label: 'NFS Network Share' },
                    ].map((p) => (
                      <button
                        key={p.id}
                        type="button"
                        onClick={() => handlePresetSelect(p.id as any)}
                        className={`p-2.5 rounded-xl border text-left cursor-pointer transition-all ${
                          repoDestPreset === p.id
                            ? 'bg-brand-500/10 border-brand-500/30 text-brand-300 font-bold shadow-xs'
                            : 'bg-surface-950 border-surface-800 text-slate-300 hover:bg-surface-850'
                        }`}
                      >
                        <div className="text-[11px]">{p.label}</div>
                      </button>
                    ))}
                  </div>
                </div>

                {/* Name & Engine */}
                <div className="grid grid-cols-2 gap-3">
                  <div>
                    <label className="text-[11px] font-semibold text-slate-300 block mb-1">Repository Name</label>
                    <input
                      type="text"
                      required
                      placeholder="e.g. Primary Wasabi Vault"
                      value={repoName}
                      onChange={(e) => setRepoName(e.target.value)}
                      className="w-full px-3 py-2 rounded-xl bg-surface-950 border border-surface-800 text-white focus:outline-hidden focus:border-brand-500"
                    />
                  </div>

                  <div>
                    <label className="text-[11px] font-semibold text-slate-300 block mb-1">Engine Driver</label>
                    <select
                      value={repoProvider}
                      onChange={(e) => setRepoProvider(e.target.value as any)}
                      className="w-full px-3 py-2 rounded-xl bg-surface-950 border border-surface-800 text-white focus:outline-hidden focus:border-brand-500"
                    >
                      <option value="native">Native Content-Addressed Chunk Engine</option>
                      <option value="restic">Restic Enterprise CLI (Auto-fallback)</option>
                    </select>
                  </div>
                </div>

                {/* S3 Details if cloud */}
                {repoDestPreset !== 'local' && repoDestPreset !== 'nfs' && (
                  <div className="bg-surface-950 p-3.5 rounded-xl border border-surface-800 space-y-3">
                    <div className="text-[11px] font-bold text-white uppercase tracking-wider">Remote S3 Credentials</div>
                    <div className="grid grid-cols-2 gap-2.5">
                      <div>
                        <label className="text-[10px] text-slate-400 block mb-0.5">Bucket Name</label>
                        <input
                          type="text"
                          required
                          placeholder="my-vault-bucket"
                          value={repoS3Bucket}
                          onChange={(e) => {
                            setRepoS3Bucket(e.target.value);
                            setRepoUri(`s3:${repoS3Endpoint}/${e.target.value}`);
                          }}
                          className="w-full px-2.5 py-1.5 rounded-lg bg-surface-900 border border-surface-800 text-white font-mono"
                        />
                      </div>
                      <div>
                        <label className="text-[10px] text-slate-400 block mb-0.5">Region</label>
                        <input
                          type="text"
                          value={repoS3Region}
                          onChange={(e) => setRepoS3Region(e.target.value)}
                          className="w-full px-2.5 py-1.5 rounded-lg bg-surface-900 border border-surface-800 text-white font-mono"
                        />
                      </div>
                    </div>

                    <div className="grid grid-cols-2 gap-2.5">
                      <div>
                        <label className="text-[10px] text-slate-400 block mb-0.5">Access Key ID</label>
                        <input
                          type="text"
                          required
                          value={repoS3AccessKey}
                          onChange={(e) => setRepoS3AccessKey(e.target.value)}
                          className="w-full px-2.5 py-1.5 rounded-lg bg-surface-900 border border-surface-800 text-white font-mono"
                        />
                      </div>
                      <div>
                        <label className="text-[10px] text-slate-400 block mb-0.5">Secret Access Key</label>
                        <input
                          type="password"
                          required
                          value={repoS3SecretKey}
                          onChange={(e) => setRepoS3SecretKey(e.target.value)}
                          className="w-full px-2.5 py-1.5 rounded-lg bg-surface-900 border border-surface-800 text-white font-mono"
                        />
                      </div>
                    </div>
                  </div>
                )}

                {/* URI */}
                <div>
                  <label className="text-[11px] font-semibold text-slate-300 block mb-1">Target Repository URI</label>
                  <input
                    type="text"
                    required
                    value={repoUri}
                    onChange={(e) => setRepoUri(e.target.value)}
                    className="w-full px-3 py-2 rounded-xl bg-surface-950 border border-surface-800 text-white font-mono"
                  />
                </div>

                {/* Encryption Passphrase */}
                <div>
                  <div className="flex items-center justify-between mb-1">
                    <label className="text-[11px] font-semibold text-slate-300">Master Passphrase (AES-256-GCM)</label>
                    <button
                      type="button"
                      onClick={generatePassphrase}
                      className="text-[10px] font-semibold text-brand-400 hover:text-brand-300 cursor-pointer"
                    >
                      Generate Strong Key
                    </button>
                  </div>
                  <input
                    type="text"
                    value={repoPassword}
                    onChange={(e) => setRepoPassword(e.target.value)}
                    className="w-full px-3 py-2 rounded-xl bg-surface-950 border border-surface-800 text-emerald-400 font-mono"
                  />
                </div>

                {/* Deletion protection toggle */}
                <div className="p-3 bg-surface-950 border border-surface-800 rounded-xl flex items-center justify-between">
                  <div className="flex items-center gap-2">
                    <Lock className="w-4 h-4 text-emerald-400" />
                    <div>
                      <div className="font-bold text-white text-xs">Enable Deletion Protection</div>
                      <div className="text-[10px] text-slate-400">
                        Prevents unauthorized or accidental deletion of this backup repository.
                      </div>
                    </div>
                  </div>
                  <input
                    type="checkbox"
                    checked={repoDeletionProtection}
                    onChange={(e) => setRepoDeletionProtection(e.target.checked)}
                    className="rounded border-surface-700 bg-surface-900 text-brand-500 w-4 h-4 cursor-pointer"
                  />
                </div>

                {/* Verification result */}
                {testRepoResult && (
                  <div
                    className={`p-3 rounded-xl border text-xs flex items-center gap-2.5 ${
                      testRepoResult.success
                        ? 'bg-emerald-500/10 border-emerald-500/20 text-emerald-300'
                        : 'bg-rose-500/10 border-rose-500/20 text-rose-300'
                    }`}
                  >
                    {testRepoResult.success ? <CheckCircle2 className="w-4 h-4 text-emerald-400" /> : <AlertTriangle className="w-4 h-4 text-rose-400" />}
                    <span>
                      {testRepoResult.success
                        ? `Destination verified! Auth, Upload, Listing & Deletion drills passed (${testRepoResult.latency_ms}ms).`
                        : `Test failed: ${testRepoResult.error_message}`}
                    </span>
                  </div>
                )}

                {/* Modal Footer */}
                <div className="flex items-center justify-between pt-3 border-t border-surface-800">
                  <button
                    type="button"
                    onClick={handleTestRepoConnection}
                    disabled={testingRepoConn}
                    className="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-xl bg-surface-800 hover:bg-surface-750 text-slate-300 text-xs font-semibold cursor-pointer border border-surface-700 disabled:opacity-50"
                  >
                    <Activity className={`w-3.5 h-3.5 ${testingRepoConn ? 'animate-spin text-amber-400' : ''}`} />
                    <span>{testingRepoConn ? 'Testing...' : 'Test Connection Drill'}</span>
                  </button>

                  <div className="flex items-center gap-2">
                    <button
                      type="button"
                      onClick={() => setNewRepoModalOpen(false)}
                      className="px-4 py-2 rounded-xl bg-surface-800 hover:bg-surface-750 text-slate-300 text-xs font-semibold cursor-pointer"
                    >
                      Cancel
                    </button>

                    <button
                      type="submit"
                      disabled={submittingNewRepo}
                      className="inline-flex items-center gap-2 px-4 py-2 rounded-xl bg-brand-500 hover:bg-brand-600 text-white text-xs font-semibold cursor-pointer disabled:opacity-50 shadow-md"
                    >
                      <Plus className="w-4 h-4" />
                      <span>{submittingNewRepo ? 'Initializing Vault...' : 'Create & Seal Vault'}</span>
                    </button>
                  </div>
                </div>
              </form>
            </div>
          </div>
        )}

        {/* ==================================================================== */}
        {/* MODAL: CREATE INCREMENTAL SNAPSHOT */}
        {/* ==================================================================== */}
        {newSnapModalOpen && (
          <div className="fixed inset-0 z-50 bg-black/75 backdrop-blur-xs flex items-center justify-center p-4">
            <div className="bg-surface-900 border border-surface-800 rounded-2xl max-w-lg w-full p-6 space-y-4 shadow-2xl animate-in zoom-in-95">
              <div className="flex items-center justify-between border-b border-surface-800 pb-3">
                <div className="flex items-center gap-2">
                  <Layers className="w-5 h-5 text-brand-400" />
                  <h3 className="text-sm font-bold text-white">Create On-Demand Incremental Snapshot</h3>
                </div>
                <button onClick={() => setNewSnapModalOpen(false)} className="text-slate-400 hover:text-white cursor-pointer">
                  <X className="w-4 h-4" />
                </button>
              </div>

              <form onSubmit={handleCreateIncrementalSnapshot} className="space-y-3.5 text-xs">
                <div>
                  <label className="text-[11px] font-semibold text-slate-300 block mb-1">Target Repository</label>
                  <select
                    value={snapRepoId}
                    onChange={(e) => setSnapRepoId(e.target.value)}
                    className="w-full px-3 py-2 rounded-xl bg-surface-950 border border-surface-800 text-white"
                  >
                    {repositories.map((r) => (
                      <option key={r.id} value={r.id}>
                        {r.name} ({r.provider} - {r.repo_uri})
                      </option>
                    ))}
                  </select>
                </div>

                <div className="grid grid-cols-2 gap-3">
                  <div>
                    <label className="text-[11px] font-semibold text-slate-300 block mb-1">Workload Scope</label>
                    <select
                      value={snapScope}
                      onChange={(e) => setSnapScope(e.target.value as any)}
                      className="w-full px-3 py-2 rounded-xl bg-surface-950 border border-surface-800 text-white"
                    >
                      <option value="website">Website Files & Config</option>
                      <option value="database">Database (Transactional Snap)</option>
                      <option value="mail">Maildir & IMAP Mailboxes</option>
                      <option value="full_account">Full Account (Everything)</option>
                    </select>
                  </div>

                  <div>
                    <label className="text-[11px] font-semibold text-slate-300 block mb-1">Target Name / Domain</label>
                    <input
                      type="text"
                      placeholder="e.g. example.com"
                      value={snapTargetName}
                      onChange={(e) => setSnapTargetName(e.target.value)}
                      className="w-full px-3 py-2 rounded-xl bg-surface-950 border border-surface-800 text-white"
                    />
                  </div>
                </div>

                {/* Pre-backup database snapshot options */}
                <div className="bg-surface-950 p-3 rounded-xl border border-surface-800 space-y-2">
                  <div className="text-[11px] font-bold text-white">Application-Consistent Database Snapshot</div>
                  <div className="grid grid-cols-2 gap-2">
                    <div>
                      <label className="text-[10px] text-slate-400 block mb-0.5">Database Engine</label>
                      <select
                        value={snapDbType}
                        onChange={(e) => setSnapDbType(e.target.value as any)}
                        className="w-full px-2 py-1.5 rounded-lg bg-surface-900 border border-surface-800 text-white"
                      >
                        <option value="none">None (Files Only)</option>
                        <option value="mysql">MySQL / MariaDB</option>
                        <option value="postgres">PostgreSQL</option>
                      </select>
                    </div>

                    {snapDbType !== 'none' && (
                      <div>
                        <label className="text-[10px] text-slate-400 block mb-0.5">Database Name</label>
                        <input
                          type="text"
                          placeholder="mydb_production"
                          value={snapDbName}
                          onChange={(e) => setSnapDbName(e.target.value)}
                          className="w-full px-2 py-1.5 rounded-lg bg-surface-900 border border-surface-800 text-white"
                        />
                      </div>
                    )}
                  </div>
                </div>

                <div>
                  <label className="text-[11px] font-semibold text-slate-300 block mb-1">Custom Include Paths (Optional)</label>
                  <textarea
                    rows={2}
                    placeholder="/var/www/example.com&#10;/var/mail/vhosts/example.com"
                    value={snapIncludePaths}
                    onChange={(e) => setSnapIncludePaths(e.target.value)}
                    className="w-full px-3 py-1.5 rounded-xl bg-surface-950 border border-surface-800 text-white font-mono text-[11px]"
                  />
                </div>

                <div className="flex items-center justify-end gap-2 pt-2 border-t border-surface-800">
                  <button
                    type="button"
                    onClick={() => setNewSnapModalOpen(false)}
                    className="px-4 py-2 rounded-xl bg-surface-800 hover:bg-surface-750 text-slate-300 text-xs font-semibold cursor-pointer"
                  >
                    Cancel
                  </button>

                  <button
                    type="submit"
                    disabled={submittingNewSnap}
                    className="inline-flex items-center gap-2 px-4 py-2 rounded-xl bg-brand-500 hover:bg-brand-600 text-white text-xs font-semibold cursor-pointer disabled:opacity-50 shadow-md"
                  >
                    <Plus className="w-4 h-4" />
                    <span>{submittingNewSnap ? 'Deduplicating Chunks...' : 'Create Snapshot'}</span>
                  </button>
                </div>
              </form>
            </div>
          </div>
        )}

        {/* ==================================================================== */}
        {/* MODAL: PRUNE & RETENTION POLICY */}
        {/* ==================================================================== */}
        {pruneModalRepo && (
          <div className="fixed inset-0 z-50 bg-black/75 backdrop-blur-xs flex items-center justify-center p-4">
            <div className="bg-surface-900 border border-surface-800 rounded-2xl max-w-lg w-full p-6 space-y-4 shadow-2xl animate-in zoom-in-95">
              <div className="flex items-center justify-between border-b border-surface-800 pb-3">
                <div className="flex items-center gap-2">
                  <Scissors className="w-5 h-5 text-brand-400" />
                  <h3 className="text-sm font-bold text-white">Pruning & Retention Policy - {pruneModalRepo.name}</h3>
                </div>
                <button onClick={() => setPruneModalRepo(null)} className="text-slate-400 hover:text-white cursor-pointer">
                  <X className="w-4 h-4" />
                </button>
              </div>

              <div className="bg-surface-950 p-4 rounded-xl border border-surface-800 space-y-3 text-xs">
                <div className="text-xs font-bold text-white uppercase tracking-wider">Thinning Horizons</div>
                <div className="grid grid-cols-3 gap-2 text-center">
                  <div className="bg-surface-900 p-2.5 rounded-lg border border-surface-800">
                    <span className="text-[10px] text-slate-400 block">Keep Hourly</span>
                    <span className="text-white font-bold">{prunePolicy?.keep_hourly || 24}</span>
                  </div>
                  <div className="bg-surface-900 p-2.5 rounded-lg border border-surface-800">
                    <span className="text-[10px] text-slate-400 block">Keep Daily</span>
                    <span className="text-white font-bold">{prunePolicy?.keep_daily || 7}</span>
                  </div>
                  <div className="bg-surface-900 p-2.5 rounded-lg border border-surface-800">
                    <span className="text-[10px] text-slate-400 block">Keep Weekly</span>
                    <span className="text-white font-bold">{prunePolicy?.keep_weekly || 4}</span>
                  </div>
                </div>

                <div className="p-3 rounded-lg bg-surface-900 border border-surface-800 flex items-center justify-between">
                  <div>
                    <div className="font-semibold text-white">Dry-Run Simulation Mode</div>
                    <div className="text-[10px] text-slate-400">
                      Simulate pruning without deleting any chunks from the vault.
                    </div>
                  </div>
                  <input
                    type="checkbox"
                    checked={pruneDryRun}
                    onChange={(e) => setPruneDryRun(e.target.checked)}
                    className="rounded border-surface-700 bg-surface-950 text-brand-500 w-4 h-4 cursor-pointer"
                  />
                </div>
              </div>

              {pruneResult && (
                <div className="p-3.5 rounded-xl bg-surface-950 border border-surface-800 text-xs font-mono space-y-1">
                  <div className="text-emerald-400 font-bold">
                    {pruneDryRun ? 'Dry Run Simulation Complete' : 'Active Pruning Completed'}
                  </div>
                  <div className="text-slate-300">Snapshots Kept: {pruneResult.snapshots_kept}</div>
                  <div className="text-slate-300">Snapshots Thinning Candidate: {pruneResult.snapshots_removed}</div>
                  <div className="text-slate-300">Space Reclaimable: {formatBytes(pruneResult.bytes_reclaimed)}</div>
                </div>
              )}

              <div className="flex items-center justify-end gap-2 pt-2 border-t border-surface-800">
                <button
                  type="button"
                  onClick={() => setPruneModalRepo(null)}
                  className="px-4 py-2 rounded-xl bg-surface-800 hover:bg-surface-750 text-slate-300 text-xs font-semibold cursor-pointer"
                >
                  Close
                </button>

                <button
                  type="button"
                  onClick={handleExecutePrune}
                  disabled={pruningLoading}
                  className="inline-flex items-center gap-2 px-4 py-2 rounded-xl bg-brand-500 hover:bg-brand-600 text-white text-xs font-semibold cursor-pointer disabled:opacity-50 shadow-md"
                >
                  <Scissors className={`w-3.5 h-3.5 ${pruningLoading ? 'animate-spin' : ''}`} />
                  <span>{pruningLoading ? 'Pruning Vault...' : pruneDryRun ? 'Run Simulation' : 'Execute Prune Now'}</span>
                </button>
              </div>
            </div>
          </div>
        )}
      </div>
    </DashboardShell>
  );
}
