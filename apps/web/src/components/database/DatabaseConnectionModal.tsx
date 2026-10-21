'use client';

import React, { useState } from 'react';
import {
  Key,
  X,
  Copy,
  Check,
  Eye,
  EyeOff,
  Server,
  Globe,
  Database,
  FileCode,
  Terminal,
  ShieldCheck,
  Code2
} from 'lucide-react';
import { Database as DatabaseType } from '@/lib/api';

interface DatabaseConnectionModalProps {
  isOpen: boolean;
  onClose: () => void;
  database: DatabaseType;
}

type SnippetTab = 'wordpress' | 'laravel' | 'nodejs' | 'php_pdo' | 'python' | 'cli';

export const DatabaseConnectionModal: React.FC<DatabaseConnectionModalProps> = ({
  isOpen,
  onClose,
  database,
}) => {
  const [activeTab, setActiveTab] = useState<SnippetTab>('wordpress');
  const [copiedKey, setCopiedKey] = useState<string | null>(null);
  const [showPassword, setShowPassword] = useState(false);

  if (!isOpen || !database) return null;

  const dbHost = database.host_allow === 'localhost' || !database.host_allow ? 'localhost' : database.host_allow;
  const dbUser = database.username || database.name;
  const dbPass = database.password || 'HostvraPass2026!';
  const dbName = database.name;
  const dbPort = '3306';

  const copyText = (text: string, key: string) => {
    navigator.clipboard.writeText(text);
    setCopiedKey(key);
    setTimeout(() => setCopiedKey(null), 2000);
  };

  const getSnippet = () => {
    switch (activeTab) {
      case 'wordpress':
        return `// WordPress wp-config.php Database Settings
define( 'DB_NAME', '${dbName}' );
define( 'DB_USER', '${dbUser}' );
define( 'DB_PASSWORD', '${dbPass}' );
define( 'DB_HOST', '${dbHost}' );
define( 'DB_CHARSET', '${database.character_set || 'utf8mb4'}' );
define( 'DB_COLLATE', '${database.collation || ''}' );`;

      case 'laravel':
        return `# Laravel .env Database Configuration
DB_CONNECTION=mysql
DB_HOST=${dbHost === 'localhost' ? '127.0.0.1' : dbHost}
DB_PORT=${dbPort}
DB_DATABASE=${dbName}
DB_USERNAME=${dbUser}
DB_PASSWORD=${dbPass}`;

      case 'nodejs':
        return `# Node.js Prisma / TypeORM / Sequelize .env
DATABASE_URL="mysql://${dbUser}:${dbPass}@${dbHost === 'localhost' ? '127.0.0.1' : dbHost}:${dbPort}/${dbName}?charset=${database.character_set || 'utf8mb4'}"`;

      case 'php_pdo':
        return `<?php
// PHP PDO MySQL Connection
try {
    $dsn = "mysql:host=${dbHost};dbname=${dbName};charset=${database.character_set || 'utf8mb4'}";
    $pdo = new PDO($dsn, "${dbUser}", "${dbPass}", [
        PDO::ATTR_ERRMODE => PDO::ERRMODE_EXCEPTION,
        PDO::ATTR_DEFAULT_FETCH_MODE => PDO::FETCH_ASSOC,
        PDO::ATTR_EMULATE_PREPARES => false,
    ]);
} catch (PDOException $e) {
    die("Database connection failed: " . $e->getMessage());
}`;

      case 'python':
        return `# Python SQLAlchemy / PyMySQL Connection
from sqlalchemy import create_engine

DATABASE_URI = "mysql+pymysql://${dbUser}:${dbPass}@${dbHost === 'localhost' ? '127.0.0.1' : dbHost}:${dbPort}/${dbName}"
engine = create_engine(DATABASE_URI, pool_pre_ping=True)`;

      case 'cli':
        return `# MySQL Terminal CLI Client Connect
mysql -h ${dbHost} -P ${dbPort} -u ${dbUser} -p'${dbPass}' ${dbName}

# Export Dump Command via CLI:
mysqldump -h ${dbHost} -u ${dbUser} -p'${dbPass}' ${dbName} > ${dbName}_backup.sql`;
      default:
        return '';
    }
  };

  return (
    <div className="fixed inset-0 z-50 bg-black/75 backdrop-blur-xs flex items-center justify-center p-3 sm:p-5 animate-fadeIn">
      <div className="w-full max-w-2xl bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-700 rounded-2xl shadow-2xl overflow-hidden font-sans flex flex-col">
        {/* Header */}
        <div className="px-5 py-4 border-b border-slate-200 dark:border-surface-800 bg-slate-50 dark:bg-surface-850 flex items-center justify-between">
          <div className="flex items-center gap-3">
            <div className="w-10 h-10 rounded-xl bg-emerald-500/10 border border-emerald-500/20 flex items-center justify-center text-emerald-600 dark:text-emerald-400">
              <Key className="w-5 h-5" />
            </div>
            <div>
              <h2 className="text-sm font-bold text-slate-900 dark:text-white flex items-center gap-2">
                <span>Database Connection Credentials</span>
                <span className="px-2 py-0.5 rounded-full text-[10px] font-mono bg-emerald-100 dark:bg-emerald-950/40 text-emerald-700 dark:text-emerald-300">
                  {dbName}
                </span>
              </h2>
              <p className="text-xs text-slate-500 dark:text-slate-400">
                Direct credentials and ready-to-use application snippets
              </p>
            </div>
          </div>

          <button
            onClick={onClose}
            className="p-1.5 rounded-lg text-slate-400 hover:text-slate-700 dark:hover:text-white hover:bg-slate-100 dark:hover:bg-surface-800 transition"
          >
            <X className="w-5 h-5" />
          </button>
        </div>

        {/* Credentials Grid */}
        <div className="p-5 border-b border-slate-200 dark:border-surface-800 bg-white dark:bg-surface-900 grid grid-cols-1 sm:grid-cols-2 gap-3 text-xs">
          {/* Host & Port */}
          <div className="p-3 rounded-xl bg-slate-50 dark:bg-surface-950 border border-slate-200 dark:border-surface-800 space-y-1">
            <span className="text-[10px] text-slate-400 uppercase tracking-wider font-semibold block">Host & Port</span>
            <div className="flex items-center justify-between font-mono font-bold text-slate-900 dark:text-white">
              <span>{dbHost}:{dbPort}</span>
              <button
                onClick={() => copyText(`${dbHost}:${dbPort}`, 'host')}
                className="text-slate-400 hover:text-emerald-600"
                title="Copy host and port"
              >
                {copiedKey === 'host' ? <Check className="w-3.5 h-3.5 text-emerald-500" /> : <Copy className="w-3.5 h-3.5" />}
              </button>
            </div>
          </div>

          {/* Database Name */}
          <div className="p-3 rounded-xl bg-slate-50 dark:bg-surface-950 border border-slate-200 dark:border-surface-800 space-y-1">
            <span className="text-[10px] text-slate-400 uppercase tracking-wider font-semibold block">Database Name</span>
            <div className="flex items-center justify-between font-mono font-bold text-slate-900 dark:text-white">
              <span>{dbName}</span>
              <button
                onClick={() => copyText(dbName, 'dbname')}
                className="text-slate-400 hover:text-emerald-600"
                title="Copy database name"
              >
                {copiedKey === 'dbname' ? <Check className="w-3.5 h-3.5 text-emerald-500" /> : <Copy className="w-3.5 h-3.5" />}
              </button>
            </div>
          </div>

          {/* Username */}
          <div className="p-3 rounded-xl bg-slate-50 dark:bg-surface-950 border border-slate-200 dark:border-surface-800 space-y-1">
            <span className="text-[10px] text-slate-400 uppercase tracking-wider font-semibold block">Database User</span>
            <div className="flex items-center justify-between font-mono font-bold text-slate-900 dark:text-white">
              <span>{dbUser}</span>
              <button
                onClick={() => copyText(dbUser, 'user')}
                className="text-slate-400 hover:text-emerald-600"
                title="Copy username"
              >
                {copiedKey === 'user' ? <Check className="w-3.5 h-3.5 text-emerald-500" /> : <Copy className="w-3.5 h-3.5" />}
              </button>
            </div>
          </div>

          {/* Password with Eye Toggle */}
          <div className="p-3 rounded-xl bg-slate-50 dark:bg-surface-950 border border-slate-200 dark:border-surface-800 space-y-1">
            <div className="flex items-center justify-between">
              <span className="text-[10px] text-slate-400 uppercase tracking-wider font-semibold">Password</span>
              <button
                onClick={() => setShowPassword(!showPassword)}
                className="text-[11px] text-emerald-600 hover:underline flex items-center gap-1"
              >
                {showPassword ? <EyeOff className="w-3 h-3" /> : <Eye className="w-3 h-3" />}
                <span>{showPassword ? 'Hide' : 'Reveal'}</span>
              </button>
            </div>
            <div className="flex items-center justify-between font-mono font-bold text-slate-900 dark:text-white">
              <span className="truncate max-w-[180px]">
                {showPassword ? dbPass : '••••••••••••••••'}
              </span>
              <button
                onClick={() => copyText(dbPass, 'pass')}
                className="text-slate-400 hover:text-emerald-600 shrink-0"
                title="Copy password"
              >
                {copiedKey === 'pass' ? <Check className="w-3.5 h-3.5 text-emerald-500" /> : <Copy className="w-3.5 h-3.5" />}
              </button>
            </div>
          </div>
        </div>

        {/* Snippet Tabs */}
        <div className="px-5 pt-3 bg-slate-100 dark:bg-surface-900 border-b border-slate-200 dark:border-surface-800 flex items-center gap-2 overflow-x-auto text-xs">
          {[
            { id: 'wordpress', label: 'WordPress (wp-config)' },
            { id: 'laravel', label: 'Laravel (.env)' },
            { id: 'nodejs', label: 'Node.js (Prisma/URL)' },
            { id: 'php_pdo', label: 'PHP PDO' },
            { id: 'python', label: 'Python (SQLAlchemy)' },
            { id: 'cli', label: 'CLI & mysqldump' },
          ].map((tab) => (
            <button
              key={tab.id}
              onClick={() => setActiveTab(tab.id as SnippetTab)}
              className={`py-2 px-3 border-b-2 font-bold whitespace-nowrap transition cursor-pointer text-xs ${
                activeTab === tab.id
                  ? 'border-emerald-600 text-emerald-600 dark:text-emerald-400 bg-white dark:bg-surface-800 rounded-t-lg'
                  : 'border-transparent text-slate-600 dark:text-slate-400 hover:text-slate-900 dark:hover:text-white'
              }`}
            >
              {tab.label}
            </button>
          ))}
        </div>

        {/* Code Snippet Display */}
        <div className="p-5 bg-slate-950 flex-1 flex flex-col gap-2">
          <div className="flex items-center justify-between text-xs text-slate-400 font-mono">
            <span>Copy-paste directly into your project configuration:</span>
            <button
              onClick={() => copyText(getSnippet(), 'snippet')}
              className="px-3 py-1.5 rounded-lg bg-emerald-600 hover:bg-emerald-700 text-white font-bold flex items-center gap-1.5 transition text-xs shadow-xs"
            >
              {copiedKey === 'snippet' ? <Check className="w-3.5 h-3.5" /> : <Copy className="w-3.5 h-3.5" />}
              <span>{copiedKey === 'snippet' ? 'Copied to Clipboard!' : 'Copy Code Snippet'}</span>
            </button>
          </div>

          <pre className="p-4 rounded-xl bg-slate-900 text-emerald-400 font-mono text-xs overflow-x-auto leading-relaxed border border-slate-800">
            {getSnippet()}
          </pre>
        </div>

        {/* Footer */}
        <div className="px-5 py-3 border-t border-slate-200 dark:border-surface-800 bg-slate-50 dark:bg-surface-850 flex items-center justify-between text-xs">
          <div className="flex items-center gap-1.5 text-slate-500 dark:text-slate-400 text-[11px]">
            <ShieldCheck className="w-3.5 h-3.5 text-emerald-600" />
            <span>Encrypted credentials • Default port 3306</span>
          </div>
          <button
            onClick={onClose}
            className="px-4 py-1.5 rounded-lg bg-white dark:bg-surface-800 border border-slate-200 dark:border-surface-700 font-semibold text-slate-700 dark:text-slate-200 hover:bg-slate-100 dark:hover:bg-surface-700 transition"
          >
            Close
          </button>
        </div>
      </div>
    </div>
  );
};
