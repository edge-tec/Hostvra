'use client';

import React, { useState, useMemo } from 'react';
import {
  BookOpen,
  Search,
  Server,
  Globe,
  Mail,
  ShieldCheck,
  Send,
  Key,
  HardDrive,
  Sliders,
  Terminal,
  AlertTriangle,
  CheckCircle2,
  Copy,
  Check,
  ExternalLink,
  ChevronRight,
  Download,
  Lock,
  RefreshCw,
  Inbox,
  Sparkles,
  Layers,
  Activity,
  UserCheck,
  HelpCircle,
  Cpu,
} from 'lucide-react';

export interface EmailGuideProps {
  currentDomain?: string;
  mailHostname?: string;
  serverIPv4?: string;
  imapPort?: number;
  imapsPort?: number;
  pop3Port?: number;
  pop3sPort?: number;
  smtpSubmissionPort?: number;
  smtpsPort?: number;
  storagePath?: string;
  onSelectTab?: (tab: string) => void;
}

interface GuideSection {
  id: number;
  slug: string;
  title: string;
  category: 'Getting Started' | 'Client Setup' | 'DNS & Security' | 'Operations & Monitoring' | 'Troubleshooting' | 'Administration';
  icon: any;
  summary: string;
  content: (props: EmailGuideProps, copyFn: (text: string, key: string) => void, copiedKey: string | null) => React.ReactNode;
}

export function EmailUserGuide(props: EmailGuideProps) {
  const {
    currentDomain = 'example.com',
    mailHostname = `mail.${props.currentDomain || 'example.com'}`,
    serverIPv4 = '127.0.0.1',
    imapPort = 143,
    imapsPort = 993,
    pop3Port = 110,
    pop3sPort = 995,
    smtpSubmissionPort = 587,
    smtpsPort = 465,
    storagePath = '/var/mail/vhosts',
  } = props;

  const [search, setSearch] = useState('');
  const [selectedSectionId, setSelectedSectionId] = useState<number>(1);
  const [copiedKey, setCopiedKey] = useState<string | null>(null);

  const copyToClipboard = (text: string, key: string) => {
    navigator.clipboard.writeText(text);
    setCopiedKey(key);
    setTimeout(() => setCopiedKey(null), 2000);
  };

  const sections: GuideSection[] = useMemo(
    () => [
      {
        id: 1,
        slug: 'overview',
        title: '1. Overview',
        category: 'Getting Started',
        icon: BookOpen,
        summary: 'Architecture of the Hostvra enterprise email server stack.',
        content: (p, copy, cKey) => (
          <div className="space-y-4">
            <p className="text-sm text-slate-700 dark:text-slate-300 leading-relaxed">
              Hostvra features a native, enterprise-grade Linux email server subsystem built upon industry standard open-source
              components. The architecture guarantees zero open-relay exposure, high inbox deliverability, strict RFC compliance,
              and seamless multi-domain virtual mailbox hosting.
            </p>
            <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-3">
              <div className="p-3.5 rounded-xl bg-slate-50 dark:bg-surface-800 border border-slate-200 dark:border-surface-700">
                <div className="flex items-center gap-2 font-bold text-xs text-indigo-600 dark:text-indigo-400">
                  <Server className="w-4 h-4" /> Postfix 3.x MTA
                </div>
                <p className="text-xs text-slate-600 dark:text-slate-400 mt-1">
                  Handles inbound SMTP (25), client submission (587 STARTTLS, 465 SMTPS), virtual alias expansion, and outbound queuing.
                </p>
              </div>
              <div className="p-3.5 rounded-xl bg-slate-50 dark:bg-surface-800 border border-slate-200 dark:border-surface-700">
                <div className="flex items-center gap-2 font-bold text-xs text-blue-600 dark:text-blue-400">
                  <Inbox className="w-4 h-4" /> Dovecot 2.3+ LDA
                </div>
                <p className="text-xs text-slate-600 dark:text-slate-400 mt-1">
                  Delivers to Maildir format under <code className="font-mono text-[11px]">{storagePath}</code>, manages IMAP (993) and POP3 (995) access.
                </p>
              </div>
              <div className="p-3.5 rounded-xl bg-slate-50 dark:bg-surface-800 border border-slate-200 dark:border-surface-700">
                <div className="flex items-center gap-2 font-bold text-xs text-purple-600 dark:text-purple-400">
                  <Sparkles className="w-4 h-4" /> Rspamd Milter
                </div>
                <p className="text-xs text-slate-600 dark:text-slate-400 mt-1">
                  High-speed spam filtering, DKIM signing, ARC validation, DMARC checking, statistical learning, and SURBL URL filtering.
                </p>
              </div>
              <div className="p-3.5 rounded-xl bg-slate-50 dark:bg-surface-800 border border-slate-200 dark:border-surface-700">
                <div className="flex items-center gap-2 font-bold text-xs text-emerald-600 dark:text-emerald-400">
                  <ShieldCheck className="w-4 h-4" /> PostgreSQL Store
                </div>
                <p className="text-xs text-slate-600 dark:text-slate-400 mt-1">
                  Unified database storing virtual domains, mailbox credentials with SHA512-CRYPT hashing, quotas, and audit events.
                </p>
              </div>
            </div>
          </div>
        ),
      },
      {
        id: 2,
        slug: 'add-domain',
        title: '2. Add a Domain',
        category: 'Getting Started',
        icon: Globe,
        summary: 'How to provision a new virtual mail domain on Hostvra.',
        content: (p, copy, cKey) => (
          <div className="space-y-4">
            <p className="text-sm text-slate-700 dark:text-slate-300">
              When adding a domain to Hostvra, the system generates a dedicated 2048-bit RSA DKIM key pair, creates Postfix virtual domain
              mappings, and prepares Dovecot storage directories.
            </p>
            <ol className="list-decimal list-inside space-y-2 text-xs text-slate-600 dark:text-slate-300">
              <li>Click the <strong className="text-slate-900 dark:text-white">Add Domain</strong> button in the top navigation header.</li>
              <li>Enter your domain (e.g. <code className="font-mono font-semibold">{currentDomain}</code>).</li>
              <li>Specify the dedicated Mail Hostname (default: <code className="font-mono font-semibold">{mailHostname}</code>).</li>
              <li>Set the total domain storage quota allocation (e.g. 50 GB).</li>
              <li>Click <strong className="text-slate-900 dark:text-white">Create Domain</strong>. Hostvra automatically triggers Postfix hash synchronization and generates your DNS zone records.</li>
            </ol>
          </div>
        ),
      },
      {
        id: 3,
        slug: 'configure-dns',
        title: '3. Configure DNS',
        category: 'DNS & Security',
        icon: ShieldCheck,
        summary: 'Mandatory DNS records required for RFC-compliant mail exchange.',
        content: (p, copy, cKey) => (
          <div className="space-y-3">
            <p className="text-sm text-slate-700 dark:text-slate-300">
              Every mail domain requires accurate DNS records at your nameserver registrar before receiving or sending messages:
            </p>
            <div className="overflow-x-auto">
              <table className="w-full text-xs text-left border border-slate-200 dark:border-surface-700 rounded-xl overflow-hidden">
                <thead className="bg-slate-100 dark:bg-surface-800 text-slate-700 dark:text-slate-300 font-bold">
                  <tr>
                    <th className="p-2.5">Type</th>
                    <th className="p-2.5">Host / Name</th>
                    <th className="p-2.5">Value / Target</th>
                    <th className="p-2.5">Priority / TTL</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-slate-200 dark:divide-surface-700 font-mono text-[11px]">
                  <tr>
                    <td className="p-2.5 font-bold text-indigo-600">A</td>
                    <td className="p-2.5">{mailHostname}</td>
                    <td className="p-2.5">{serverIPv4}</td>
                    <td className="p-2.5">3600</td>
                  </tr>
                  <tr>
                    <td className="p-2.5 font-bold text-blue-600">MX</td>
                    <td className="p-2.5">{currentDomain}</td>
                    <td className="p-2.5">{mailHostname}.</td>
                    <td className="p-2.5 font-bold">10</td>
                  </tr>
                  <tr>
                    <td className="p-2.5 font-bold text-emerald-600">TXT (SPF)</td>
                    <td className="p-2.5">{currentDomain}</td>
                    <td className="p-2.5">v=spf1 mx a:{mailHostname} ip4:{serverIPv4} ~all</td>
                    <td className="p-2.5">3600</td>
                  </tr>
                  <tr>
                    <td className="p-2.5 font-bold text-purple-600">TXT (DMARC)</td>
                    <td className="p-2.5">_dmarc.{currentDomain}</td>
                    <td className="p-2.5">v=DMARC1; p=quarantine; sp=quarantine; rua=mailto:postmaster@{currentDomain}</td>
                    <td className="p-2.5">3600</td>
                  </tr>
                </tbody>
              </table>
            </div>
          </div>
        ),
      },
      {
        id: 4,
        slug: 'create-mailbox',
        title: '4. Create a Mailbox',
        category: 'Getting Started',
        icon: Mail,
        summary: 'Creating real virtual mailboxes with quota management.',
        content: (p, copy, cKey) => (
          <div className="space-y-3">
            <p className="text-sm text-slate-700 dark:text-slate-300">
              Each mailbox in Hostvra maps directly to Dovecot's Maildir filesystem and Postfix virtual recipient table:
            </p>
            <ul className="list-disc list-inside space-y-2 text-xs text-slate-600 dark:text-slate-300">
              <li>Open the <strong className="text-slate-900 dark:text-white">Create Mailbox</strong> modal from the header.</li>
              <li>Provide the username (e.g. <code className="font-mono">admin</code> or <code className="font-mono">support</code>).</li>
              <li>Assign a robust password with at least 8 characters. The password is hashed using Dovecot-compatible <code className="font-mono text-indigo-500">SHA512-CRYPT</code>.</li>
              <li>Specify the storage quota (default: 5 GB). Dovecot automatically tracks quota utilization.</li>
            </ul>
          </div>
        ),
      },
      {
        id: 5,
        slug: 'login-webmail',
        title: '5. Login to Webmail',
        category: 'Getting Started',
        icon: Inbox,
        summary: 'Accessing email via the integrated webmail suite.',
        content: (p, copy, cKey) => (
          <div className="space-y-3">
            <p className="text-sm text-slate-700 dark:text-slate-300">
              Hostvra embeds a full-featured webmail suite directly in the control panel. Users can read messages, compose rich-text emails,
              manage folders (Inbox, Sent, Drafts, Trash, Junk), and upload attachments.
            </p>
            <div className="p-3 bg-indigo-50 dark:bg-indigo-950/40 border border-indigo-200 dark:border-indigo-800 rounded-xl text-xs text-slate-700 dark:text-slate-300">
              <strong>Quick Access:</strong> Click the <strong className="text-indigo-600 dark:text-indigo-400">Webmail (Inbox)</strong> button in the top header or click <strong className="text-indigo-600 dark:text-indigo-400">Webmail</strong> next to any mailbox row.
            </div>
          </div>
        ),
      },
      {
        id: 6,
        slug: 'configure-outlook',
        title: '6. Configure Outlook',
        category: 'Client Setup',
        icon: Layers,
        summary: 'Step-by-step setup for Microsoft Outlook 2016/2019/365.',
        content: (p, copy, cKey) => (
          <div className="space-y-3 text-xs text-slate-700 dark:text-slate-300">
            <p className="text-sm">Connect Microsoft Outlook using IMAP with SSL/TLS encryption:</p>
            <ol className="list-decimal list-inside space-y-2">
              <li>In Outlook, go to <strong>File &gt; Add Account</strong>.</li>
              <li>Enter your full email address: <code className="font-mono font-semibold">user@{currentDomain}</code>.</li>
              <li>Select <strong>Advanced options &gt; Let me set up my account manually</strong>, then choose <strong>IMAP</strong>.</li>
              <li>
                <strong>Incoming Mail:</strong> Server: <code className="font-mono font-bold">{mailHostname}</code>, Port:{' '}
                <code className="font-mono font-bold">{imapsPort}</code>, Encryption: <strong>SSL/TLS</strong>.
              </li>
              <li>
                <strong>Outgoing Mail:</strong> Server: <code className="font-mono font-bold">{mailHostname}</code>, Port:{' '}
                <code className="font-mono font-bold">{smtpSubmissionPort}</code>, Encryption: <strong>STARTTLS</strong>.
              </li>
              <li>Check <strong>&quot;Require logon using Secure Password Authentication (SPA)&quot;</strong>: <em>Leave Unchecked</em>.</li>
              <li>Click <strong>Next</strong> and enter your mailbox password.</li>
            </ol>
          </div>
        ),
      },
      {
        id: 7,
        slug: 'configure-thunderbird',
        title: '7. Configure Thunderbird',
        category: 'Client Setup',
        icon: Mail,
        summary: 'Mozilla Thunderbird auto-configuration and manual setup.',
        content: (p, copy, cKey) => (
          <div className="space-y-3 text-xs text-slate-700 dark:text-slate-300">
            <p className="text-sm">Thunderbird supports one-click automatic discovery via Hostvra's XML autoconfig endpoint:</p>
            <ol className="list-decimal list-inside space-y-2">
              <li>Open Thunderbird and choose <strong>Account Settings &gt; Add Mail Account</strong>.</li>
              <li>Enter your display name, email address (<code className="font-mono">user@{currentDomain}</code>), and password.</li>
              <li>Click <strong>Continue</strong>. Thunderbird will query <code className="font-mono">autoconfig.{currentDomain}</code>.</li>
              <li>If configuring manually:
                <ul className="list-disc list-inside ml-4 mt-1 space-y-1">
                  <li>Incoming: IMAP, <code className="font-mono">{mailHostname}</code>, Port <code className="font-mono">{imapsPort}</code>, SSL/TLS, Normal Password.</li>
                  <li>Outgoing: SMTP, <code className="font-mono">{mailHostname}</code>, Port <code className="font-mono">{smtpSubmissionPort}</code>, STARTTLS, Normal Password.</li>
                </ul>
              </li>
            </ol>
          </div>
        ),
      },
      {
        id: 8,
        slug: 'configure-apple-mail',
        title: '8. Configure Apple Mail',
        category: 'Client Setup',
        icon: Cpu,
        summary: 'Configuring Apple Mail on macOS.',
        content: (p, copy, cKey) => (
          <div className="space-y-3 text-xs text-slate-700 dark:text-slate-300">
            <p className="text-sm">Set up your Hostvra email on Apple Mail (macOS):</p>
            <ol className="list-decimal list-inside space-y-2">
              <li>Open <strong>Mail &gt; Settings &gt; Accounts &gt; Add Other Mail Account</strong>.</li>
              <li>Enter your Name, Email (<code className="font-mono">user@{currentDomain}</code>), and Password.</li>
              <li>When prompted for server details:
                <div className="grid grid-cols-2 gap-2 mt-1.5 p-2.5 bg-slate-50 dark:bg-surface-800 rounded-lg font-mono text-[11px]">
                  <div>Account Type: <strong>IMAP</strong></div>
                  <div>Incoming Server: <strong>{mailHostname}</strong></div>
                  <div>Outgoing Server: <strong>{mailHostname}</strong></div>
                  <div>Username: <strong>user@{currentDomain}</strong></div>
                </div>
              </li>
              <li>Click <strong>Sign In</strong>. Alternatively, download the <code className="font-mono">.mobileconfig</code> profile from the SMTP tab to install automatically.</li>
            </ol>
          </div>
        ),
      },
      {
        id: 9,
        slug: 'configure-iphone-ipad',
        title: '9. Configure iPhone/iPad',
        category: 'Client Setup',
        icon: Activity,
        summary: 'Adding email accounts to iOS devices.',
        content: (p, copy, cKey) => (
          <div className="space-y-3 text-xs text-slate-700 dark:text-slate-300">
            <p className="text-sm">Setting up Hostvra mail on iPhone and iPad:</p>
            <ol className="list-decimal list-inside space-y-2">
              <li>Open <strong>Settings &gt; Mail &gt; Accounts &gt; Add Account &gt; Other &gt; Add Mail Account</strong>.</li>
              <li>Fill in Name, Email (<code className="font-mono">user@{currentDomain}</code>), Password, and Description.</li>
              <li>Select <strong>IMAP</strong> at the top.</li>
              <li>Under <strong>Incoming Mail Server</strong>: Host Name: <code className="font-mono">{mailHostname}</code>, User Name: <code className="font-mono">user@{currentDomain}</code>, Password: your password.</li>
              <li>Under <strong>Outgoing Mail Server</strong>: Host Name: <code className="font-mono">{mailHostname}</code>, User Name: <code className="font-mono">user@{currentDomain}</code>, Password: your password.</li>
              <li>Tap <strong>Next &gt; Save</strong>. iOS will verify certificates over TLS.</li>
            </ol>
          </div>
        ),
      },
      {
        id: 10,
        slug: 'configure-android',
        title: '10. Configure Android',
        category: 'Client Setup',
        icon: Server,
        summary: 'Android Gmail, Samsung Email, and K-9 Mail configuration.',
        content: (p, copy, cKey) => (
          <div className="space-y-3 text-xs text-slate-700 dark:text-slate-300">
            <p className="text-sm">Configure email on Android devices:</p>
            <ol className="list-decimal list-inside space-y-2">
              <li>Open the Gmail app, tap your profile avatar &gt; <strong>Add another account &gt; Other</strong>.</li>
              <li>Enter your email (<code className="font-mono">user@{currentDomain}</code>) and tap <strong>Manual setup &gt; Personal (IMAP)</strong>.</li>
              <li>Enter your password.</li>
              <li><strong>Incoming server settings:</strong> Server: <code className="font-mono">{mailHostname}</code>, Port: <code className="font-mono">{imapsPort}</code>, Security type: <strong>SSL/TLS</strong>.</li>
              <li><strong>Outgoing server settings:</strong> Require sign-in: <strong>On</strong>, Server: <code className="font-mono">{mailHostname}</code>, Port: <code className="font-mono">{smtpSubmissionPort}</code>, Security type: <strong>STARTTLS</strong>.</li>
            </ol>
          </div>
        ),
      },
      {
        id: 11,
        slug: 'imap-settings',
        title: '11. IMAP Settings',
        category: 'Client Setup',
        icon: Inbox,
        summary: 'IMAP protocol parameters, ports, and folder synchronization.',
        content: (p, copy, cKey) => (
          <div className="space-y-3 text-xs text-slate-700 dark:text-slate-300">
            <p className="text-sm">IMAP keeps all messages and folders synchronized across all user devices:</p>
            <div className="grid grid-cols-1 sm:grid-cols-2 gap-3 font-mono text-[11px]">
              <div className="p-3 bg-slate-50 dark:bg-surface-800 rounded-xl border border-slate-200 dark:border-surface-700">
                <span className="font-bold text-indigo-600 block mb-1">IMAPS (Standard / Recommended)</span>
                <div>Server: <strong>{mailHostname}</strong></div>
                <div>Port: <strong>{imapsPort}</strong></div>
                <div>Security: <strong>SSL/TLS (Implicit TLS)</strong></div>
                <div>Auth: <strong>Normal Password (PLAIN / LOGIN)</strong></div>
              </div>
              <div className="p-3 bg-slate-50 dark:bg-surface-800 rounded-xl border border-slate-200 dark:border-surface-700">
                <span className="font-bold text-slate-600 dark:text-slate-400 block mb-1">IMAP + STARTTLS (Alternative)</span>
                <div>Server: <strong>{mailHostname}</strong></div>
                <div>Port: <strong>{imapPort}</strong></div>
                <div>Security: <strong>STARTTLS (Explicit TLS)</strong></div>
                <div>Auth: <strong>Normal Password</strong></div>
              </div>
            </div>
          </div>
        ),
      },
      {
        id: 12,
        slug: 'pop3-settings',
        title: '12. POP3 Settings',
        category: 'Client Setup',
        icon: Mail,
        summary: 'POP3 parameters for single-device download workflows.',
        content: (p, copy, cKey) => (
          <div className="space-y-3 text-xs text-slate-700 dark:text-slate-300">
            <p className="text-sm">POP3 downloads messages locally to a single client computer:</p>
            <div className="grid grid-cols-1 sm:grid-cols-2 gap-3 font-mono text-[11px]">
              <div className="p-3 bg-slate-50 dark:bg-surface-800 rounded-xl border border-slate-200 dark:border-surface-700">
                <span className="font-bold text-blue-600 block mb-1">POP3S (Secure)</span>
                <div>Server: <strong>{mailHostname}</strong></div>
                <div>Port: <strong>{pop3sPort}</strong></div>
                <div>Security: <strong>SSL/TLS</strong></div>
              </div>
              <div className="p-3 bg-slate-50 dark:bg-surface-800 rounded-xl border border-slate-200 dark:border-surface-700">
                <span className="font-bold text-slate-600 dark:text-slate-400 block mb-1">POP3 + STARTTLS</span>
                <div>Server: <strong>{mailHostname}</strong></div>
                <div>Port: <strong>{pop3Port}</strong></div>
                <div>Security: <strong>STARTTLS</strong></div>
              </div>
            </div>
            <p className="text-amber-600 dark:text-amber-400 text-[11px]">
              Note: IMAP is strongly recommended over POP3 to maintain synchronized Sent and Drafts folders across mobile and desktop.
            </p>
          </div>
        ),
      },
      {
        id: 13,
        slug: 'smtp-settings',
        title: '13. SMTP Settings',
        category: 'Client Setup',
        icon: Send,
        summary: 'Outbound SMTP submission ports, authentication, and cipher suites.',
        content: (p, copy, cKey) => (
          <div className="space-y-3 text-xs text-slate-700 dark:text-slate-300">
            <p className="text-sm">All outbound mail submission requires SASL authentication over an encrypted TLS channel:</p>
            <div className="grid grid-cols-1 sm:grid-cols-2 gap-3 font-mono text-[11px]">
              <div className="p-3 bg-slate-50 dark:bg-surface-800 rounded-xl border border-slate-200 dark:border-surface-700">
                <span className="font-bold text-indigo-600 block mb-1">Port 587 (Standard Submission)</span>
                <div>Host: <strong>{mailHostname}</strong></div>
                <div>Encryption: <strong>STARTTLS</strong></div>
                <div>Auth: <strong>Mandatory (Dovecot SASL)</strong></div>
              </div>
              <div className="p-3 bg-slate-50 dark:bg-surface-800 rounded-xl border border-slate-200 dark:border-surface-700">
                <span className="font-bold text-purple-600 block mb-1">Port 465 (SMTPS / Implicit TLS)</span>
                <div>Host: <strong>{mailHostname}</strong></div>
                <div>Encryption: <strong>SSL/TLS</strong></div>
                <div>Auth: <strong>Mandatory (Dovecot SASL)</strong></div>
              </div>
            </div>
          </div>
        ),
      },
      {
        id: 14,
        slug: 'send-test-email',
        title: '14. Send a Test Email',
        category: 'Operations & Monitoring',
        icon: Terminal,
        summary: 'Testing real RFC 5321 SMTP delivery and inspection.',
        content: (p, copy, cKey) => (
          <div className="space-y-3 text-xs text-slate-700 dark:text-slate-300">
            <p className="text-sm">
              Use the built-in <strong>Send Test Tool</strong> tab to execute an authenticated protocol transaction:
            </p>
            <ol className="list-decimal list-inside space-y-1.5">
              <li>Select the sending mailbox from your configured accounts.</li>
              <li>Provide a recipient email address (such as your personal inbox or mail-tester.com address).</li>
              <li>Click <strong>Transmit Test Email</strong>.</li>
              <li>The diagnostic console displays the real-time SMTP transcript: <code className="font-mono">EHLO</code>, <code className="font-mono">AUTH PLAIN</code>, <code className="font-mono">MAIL FROM</code>, <code className="font-mono">RCPT TO</code>, and Postfix queue ID.</li>
            </ol>
          </div>
        ),
      },
      {
        id: 15,
        slug: 'verify-spf',
        title: '15. Verify SPF',
        category: 'DNS & Security',
        icon: ShieldCheck,
        summary: 'Sender Policy Framework record syntax and validation.',
        content: (p, copy, cKey) => (
          <div className="space-y-3 text-xs text-slate-700 dark:text-slate-300">
            <p className="text-sm">
              SPF authorizes Hostvra's mail server IP address to send email on behalf of <code className="font-mono">{currentDomain}</code>:
            </p>
            <div className="p-3 bg-slate-50 dark:bg-surface-800 rounded-xl font-mono text-xs border border-slate-200 dark:border-surface-700 flex items-center justify-between">
              <span>v=spf1 mx a:{mailHostname} ip4:{serverIPv4} ~all</span>
              <button
                onClick={() => copy(`v=spf1 mx a:${mailHostname} ip4:${serverIPv4} ~all`, 'spf_txt')}
                className="text-slate-400 hover:text-slate-600"
              >
                {cKey === 'spf_txt' ? <Check className="w-4 h-4 text-emerald-500" /> : <Copy className="w-4 h-4" />}
              </button>
            </div>
            <p className="text-slate-500 text-[11px]">
              Rule: Ensure your domain has only ONE SPF TXT record. If you also send email via external providers (e.g. SendGrid), combine them with <code className="font-mono">include:sendgrid.net</code>.
            </p>
          </div>
        ),
      },
      {
        id: 16,
        slug: 'verify-dkim',
        title: '16. Verify DKIM',
        category: 'DNS & Security',
        icon: Key,
        summary: 'DomainKeys Identified Mail cryptographic signing via Rspamd.',
        content: (p, copy, cKey) => (
          <div className="space-y-3 text-xs text-slate-700 dark:text-slate-300">
            <p className="text-sm">
              DKIM digitally signs every outgoing email header using a private RSA key. Destination mail servers verify the signature
              using the public key published in your DNS:
            </p>
            <div className="p-3 bg-slate-50 dark:bg-surface-800 rounded-xl font-mono text-[11px] border border-slate-200 dark:border-surface-700">
              <div className="text-slate-400">DNS Record Name:</div>
              <div className="font-bold text-purple-600 dark:text-purple-400 mb-2">default._domainkey.{currentDomain}</div>
              <div className="text-slate-400">DNS Record Value:</div>
              <div className="break-all font-mono text-slate-700 dark:text-slate-300">v=DKIM1; k=rsa; p=MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEA...</div>
            </div>
          </div>
        ),
      },
      {
        id: 17,
        slug: 'verify-dmarc',
        title: '17. Verify DMARC',
        category: 'DNS & Security',
        icon: Lock,
        summary: 'Domain-based Message Authentication, Reporting, and Conformance.',
        content: (p, copy, cKey) => (
          <div className="space-y-3 text-xs text-slate-700 dark:text-slate-300">
            <p className="text-sm">
              DMARC instructs receiving mail servers (Gmail, Outlook, Yahoo) how to handle messages that fail SPF or DKIM checks:
            </p>
            <div className="p-3 bg-slate-50 dark:bg-surface-800 rounded-xl font-mono text-xs border border-slate-200 dark:border-surface-700">
              <div className="text-slate-400">Record Name: <strong className="text-indigo-600">_dmarc.{currentDomain}</strong></div>
              <div className="mt-1">v=DMARC1; p=quarantine; sp=quarantine; rua=mailto:postmaster@{currentDomain}; pct=100; adkim=r; aspf=r</div>
            </div>
            <p className="text-slate-500 text-[11px]">
              Policies: <code className="font-mono">p=none</code> (monitoring only), <code className="font-mono">p=quarantine</code> (deliver to spam folder on failure), <code className="font-mono">p=reject</code> (drop unauthorized mail completely).
            </p>
          </div>
        ),
      },
      {
        id: 18,
        slug: 'deliverability-health',
        title: '18. Understand Deliverability Health',
        category: 'Operations & Monitoring',
        icon: Activity,
        summary: 'Interpreting health audit scores, deductions, and ratings.',
        content: (p, copy, cKey) => (
          <div className="space-y-3 text-xs text-slate-700 dark:text-slate-300">
            <p className="text-sm">
              The <strong>Deliverability &amp; Health Audit</strong> engine tests your domain and mail server configuration across 10 critical metrics:
            </p>
            <ul className="list-disc list-inside space-y-1">
              <li><strong>Score 90 - 100:</strong> Optimal inbox delivery. SPF, DKIM, DMARC, PTR, and TLS are fully aligned.</li>
              <li><strong>Score 70 - 89:</strong> Action needed. One or more secondary records (such as DMARC rua or FCrDNS) requires attention.</li>
              <li><strong>Score &lt; 70:</strong> Critical delivery warning. Missing MX or SPF records will lead to major email provider rejections.</li>
            </ul>
          </div>
        ),
      },
      {
        id: 19,
        slug: 'spam-protection',
        title: '19. Spam Protection',
        category: 'DNS & Security',
        icon: Sparkles,
        summary: 'Rspamd milter configuration, scoring thresholds, and learning.',
        content: (p, copy, cKey) => (
          <div className="space-y-3 text-xs text-slate-700 dark:text-slate-300">
            <p className="text-sm">Rspamd monitors incoming and outgoing mail via Postfix milters:</p>
            <div className="grid grid-cols-3 gap-2 font-mono text-center text-xs">
              <div className="p-2.5 rounded-lg bg-emerald-50 dark:bg-emerald-950/30 border border-emerald-200 dark:border-emerald-800">
                <div className="font-bold text-emerald-600">Score &lt; 4.0</div>
                <div className="text-[10px] text-slate-500">Ham / Deliver Clean</div>
              </div>
              <div className="p-2.5 rounded-lg bg-amber-50 dark:bg-amber-950/30 border border-amber-200 dark:border-amber-800">
                <div className="font-bold text-amber-600">Score 6.0 - 14.9</div>
                <div className="text-[10px] text-slate-500">Add X-Spam Header</div>
              </div>
              <div className="p-2.5 rounded-lg bg-rose-50 dark:bg-rose-950/30 border border-rose-200 dark:border-rose-800">
                <div className="font-bold text-rose-600">Score &gt;= 15.0</div>
                <div className="text-[10px] text-slate-500">Reject / Drop Message</div>
              </div>
            </div>
          </div>
        ),
      },
      {
        id: 20,
        slug: 'mail-queue',
        title: '20. Mail Queue',
        category: 'Operations & Monitoring',
        icon: Send,
        summary: 'Managing Postfix spools, flushing deferred queues, and purging.',
        content: (p, copy, cKey) => (
          <div className="space-y-3 text-xs text-slate-700 dark:text-slate-300">
            <p className="text-sm">
              The <strong>Mail Queue</strong> tab visualizes Postfix spools in real time via <code className="font-mono">postqueue -p</code>:
            </p>
            <ul className="list-disc list-inside space-y-1">
              <li><strong>Flush Queue:</strong> Executes <code className="font-mono">postqueue -f</code> to attempt immediate delivery of deferred messages.</li>
              <li><strong>Purge Message:</strong> Executes <code className="font-mono">postsuper -d &lt;queue_id&gt;</code> to remove obsolete bounces or stuck emails.</li>
            </ul>
          </div>
        ),
      },
      {
        id: 21,
        slug: 'email-logs',
        title: '21. Email Logs',
        category: 'Operations & Monitoring',
        icon: Sliders,
        summary: 'Viewing real-time delivery logs, bounces, and rejection events.',
        content: (p, copy, cKey) => (
          <div className="space-y-3 text-xs text-slate-700 dark:text-slate-300">
            <p className="text-sm">
              The <strong>Delivery Logs</strong> tab aggregates structured MTA delivery events from Postfix, Dovecot, and Rspamd:
            </p>
            <p>
              Filter logs by <em>Delivered</em>, <em>Bounced</em>, <em>Deferred</em>, or <em>Rejected</em> to diagnose delivery issues with recipient MTAs.
            </p>
          </div>
        ),
      },
      {
        id: 22,
        slug: 'mailbox-quotas',
        title: '22. Mailbox Quotas',
        category: 'Administration',
        icon: HardDrive,
        summary: 'Storage quotas, Dovecot Dict limits, and usage warnings.',
        content: (p, copy, cKey) => (
          <div className="space-y-3 text-xs text-slate-700 dark:text-slate-300">
            <p className="text-sm">
              Each mailbox has an enforced storage limit (e.g. 5 GB or 10 GB). When usage exceeds 90%, Dovecot logs a quota warning.
              If quota reaches 100%, incoming emails are deferred with SMTP code <code className="font-mono">452 4.2.2 Mailbox is full</code> until space is cleared or quota is raised.
            </p>
          </div>
        ),
      },
      {
        id: 23,
        slug: 'password-reset',
        title: '23. Password Reset',
        category: 'Administration',
        icon: Key,
        summary: 'Securely changing mailbox passwords and credential updating.',
        content: (p, copy, cKey) => (
          <div className="space-y-3 text-xs text-slate-700 dark:text-slate-300">
            <p className="text-sm">
              Administrators can reset any mailbox password directly from the <strong>Mailboxes</strong> tab by clicking the key icon.
              The new password takes effect instantly for Webmail, IMAP, and SMTP submission.
            </p>
          </div>
        ),
      },
      {
        id: 24,
        slug: 'mailbox-disable-enable',
        title: '24. Mailbox Disable/Enable',
        category: 'Administration',
        icon: UserCheck,
        summary: 'Suspending mailbox access without deleting stored messages.',
        content: (p, copy, cKey) => (
          <div className="space-y-3 text-xs text-slate-700 dark:text-slate-300">
            <p className="text-sm">
              Click the status toggle or suspend icon next to a mailbox row to suspend an account. When suspended:
            </p>
            <ul className="list-disc list-inside space-y-1">
              <li>Dovecot rejects IMAP/POP3 authentication attempts.</li>
              <li>Postfix rejects SMTP submission attempts.</li>
              <li>Stored email messages remain safe and intact on disk.</li>
            </ul>
          </div>
        ),
      },
      {
        id: 25,
        slug: 'alias',
        title: '25. Alias',
        category: 'Administration',
        icon: Mail,
        summary: 'Virtual email aliases routing to existing accounts.',
        content: (p, copy, cKey) => (
          <div className="space-y-3 text-xs text-slate-700 dark:text-slate-300">
            <p className="text-sm">
              Aliases route incoming messages destined for one address (e.g. <code className="font-mono">sales@{currentDomain}</code>) to an
              existing destination mailbox without requiring additional storage quota.
            </p>
          </div>
        ),
      },
      {
        id: 26,
        slug: 'forwarding',
        title: '26. Forwarding',
        category: 'Administration',
        icon: Send,
        summary: 'Forwarding messages to external or internal recipients.',
        content: (p, copy, cKey) => (
          <div className="space-y-3 text-xs text-slate-700 dark:text-slate-300">
            <p className="text-sm">
              Forwarders transmit incoming mail to an external email address (such as Gmail). When forwarding externally, Hostvra utilizes
              SRS (Sender Rewriting Scheme) to preserve SPF alignment and prevent external spam classification.
            </p>
          </div>
        ),
      },
      {
        id: 27,
        slug: 'troubleshooting',
        title: '27. Troubleshooting',
        category: 'Troubleshooting',
        icon: AlertTriangle,
        summary: 'System diagnosis commands and health verification.',
        content: (p, copy, cKey) => (
          <div className="space-y-3 text-xs text-slate-700 dark:text-slate-300">
            <p className="text-sm">Check mail subsystem daemon status on your Hostvra server:</p>
            <div className="p-3 bg-slate-900 text-slate-200 rounded-xl font-mono text-xs space-y-1">
              <div>systemctl status postfix</div>
              <div>systemctl status dovecot</div>
              <div>systemctl status rspamd</div>
              <div>ss -tulpn | grep -E &apos;:(25|587|465|993|995|110|143)&apos;</div>
            </div>
          </div>
        ),
      },
      {
        id: 28,
        slug: 'dns-troubleshooting',
        title: '28. DNS Troubleshooting',
        category: 'Troubleshooting',
        icon: Globe,
        summary: 'Resolving NXDOMAIN, TTL delays, and record mismatches.',
        content: (p, copy, cKey) => (
          <div className="space-y-3 text-xs text-slate-700 dark:text-slate-300">
            <p className="text-sm">Verify your DNS records using standard command-line tools:</p>
            <div className="p-3 bg-slate-900 text-slate-200 rounded-xl font-mono text-xs space-y-1">
              <div>dig +short MX {currentDomain}</div>
              <div>dig +short TXT {currentDomain}</div>
              <div>dig +short TXT default._domainkey.{currentDomain}</div>
              <div>dig +short TXT _dmarc.{currentDomain}</div>
            </div>
          </div>
        ),
      },
      {
        id: 29,
        slug: 'smtp-troubleshooting',
        title: '29. SMTP Troubleshooting',
        category: 'Troubleshooting',
        icon: Send,
        summary: 'Fixing authentication failures, 554 relay denied, and submission errors.',
        content: (p, copy, cKey) => (
          <div className="space-y-3 text-xs text-slate-700 dark:text-slate-300">
            <p className="text-sm">Common SMTP submission errors and remedies:</p>
            <ul className="list-disc list-inside space-y-1.5">
              <li><strong>535 Authentication failed:</strong> Ensure your mail client username is your <em>full email address</em> (<code className="font-mono">user@{currentDomain}</code>), not just the username.</li>
              <li><strong>554 5.7.1 Relay access denied:</strong> You must authenticate before sending. In Outlook/Thunderbird, check &quot;My outgoing server requires authentication&quot;.</li>
            </ul>
          </div>
        ),
      },
      {
        id: 30,
        slug: 'imap-troubleshooting',
        title: '30. IMAP Troubleshooting',
        category: 'Troubleshooting',
        icon: Inbox,
        summary: 'Fixing folder synchronization issues and connection timeouts.',
        content: (p, copy, cKey) => (
          <div className="space-y-3 text-xs text-slate-700 dark:text-slate-300">
            <p className="text-sm">
              If IMAP fails to connect on port 993, check that your server firewall permits incoming TCP traffic on port 993 and that your TLS certificate covers <code className="font-mono">{mailHostname}</code>.
            </p>
          </div>
        ),
      },
      {
        id: 31,
        slug: 'tls-ssl-troubleshooting',
        title: '31. TLS/SSL Troubleshooting',
        category: 'Troubleshooting',
        icon: ShieldCheck,
        summary: 'Let&apos;s Encrypt certificate renewal and host validation.',
        content: (p, copy, cKey) => (
          <div className="space-y-3 text-xs text-slate-700 dark:text-slate-300">
            <p className="text-sm">
              Postfix and Dovecot use TLS certificates generated via Let&apos;s Encrypt. Verify certificate expiration and hostnames via:
            </p>
            <div className="p-3 bg-slate-900 text-slate-200 rounded-xl font-mono text-xs">
              openssl s_client -connect {mailHostname}:993 -servername {mailHostname}
            </div>
          </div>
        ),
      },
      {
        id: 32,
        slug: 'bounce-deferred-troubleshooting',
        title: '32. Bounce/Deferred Mail Troubleshooting',
        category: 'Troubleshooting',
        icon: AlertTriangle,
        summary: 'Analyzing Non-Delivery Reports (NDR) and DSN status codes.',
        content: (p, copy, cKey) => (
          <div className="space-y-3 text-xs text-slate-700 dark:text-slate-300">
            <p className="text-sm">Understanding delivery failure codes:</p>
            <ul className="list-disc list-inside space-y-1">
              <li><strong>5.1.1 User unknown:</strong> Recipient mailbox does not exist at destination domain.</li>
              <li><strong>5.7.1 Service unavailable:</strong> Client host blocked using Spamhaus or missing reverse DNS (PTR).</li>
              <li><strong>4.4.1 Connection timed out:</strong> Destination MX server unreachable or port 25 blocked by your VPS provider.</li>
            </ul>
          </div>
        ),
      },
      {
        id: 33,
        slug: 'security-best-practices',
        title: '33. Security Best Practices',
        category: 'DNS & Security',
        icon: Lock,
        summary: 'Enforcing rate limits, brute-force protection, and Fail2ban.',
        content: (p, copy, cKey) => (
          <div className="space-y-3 text-xs text-slate-700 dark:text-slate-300">
            <p className="text-sm">Hostvra hardens mail servers with the following safeguards:</p>
            <ul className="list-disc list-inside space-y-1">
              <li>Open relay strictly forbidden: external senders cannot relay through port 25 or 587 without SASL authentication.</li>
              <li>Automated brute-force banning via Fail2ban integration on repeated authentication failures.</li>
              <li>Hourly outbound rate-limiting per mailbox and per domain to isolate compromised user credentials.</li>
            </ul>
          </div>
        ),
      },
      {
        id: 34,
        slug: 'backup-and-recovery',
        title: '34. Backup and Recovery',
        category: 'Administration',
        icon: HardDrive,
        summary: 'Backing up Maildir storage, database metadata, and keys.',
        content: (p, copy, cKey) => (
          <div className="space-y-3 text-xs text-slate-700 dark:text-slate-300">
            <p className="text-sm">Hostvra&apos;s backup system archives:</p>
            <ul className="list-disc list-inside space-y-1">
              <li>Virtual mailbox storage under <code className="font-mono">{storagePath}</code>.</li>
              <li>PostgreSQL database tables containing domain records, mailboxes, aliases, and hashed credentials.</li>
              <li>DKIM private keys stored in the secure key repository.</li>
            </ul>
          </div>
        ),
      },
      {
        id: 35,
        slug: 'administrator-guide',
        title: '35. Administrator Guide',
        category: 'Administration',
        icon: Server,
        summary: 'Cluster operations, Postfix tune parameters, and server nodes.',
        content: (p, copy, cKey) => (
          <div className="space-y-3 text-xs text-slate-700 dark:text-slate-300">
            <p className="text-sm">
              Use the <strong>Mail Servers</strong> tab to manage multi-node mail clusters, inspect service daemons,
              monitor storage consumption, and execute preflight checks before spinning up new infrastructure nodes.
            </p>
          </div>
        ),
      },
    ],
    [currentDomain, mailHostname, serverIPv4, imapPort, imapsPort, pop3Port, pop3sPort, smtpSubmissionPort, smtpsPort, storagePath]
  );

  const filteredSections = useMemo(() => {
    if (!search.trim()) return sections;
    const q = search.toLowerCase();
    return sections.filter(
      (s) => s.title.toLowerCase().includes(q) || s.summary.toLowerCase().includes(q) || s.category.toLowerCase().includes(q)
    );
  }, [sections, search]);

  const activeSection = sections.find((s) => s.id === selectedSectionId) || sections[0];

  return (
    <div className="space-y-6">
      {/* Header Banner */}
      <div className="p-6 rounded-2xl bg-gradient-to-r from-indigo-50/50 dark:from-indigo-950/40 via-white dark:via-surface-900 to-white dark:to-surface-900 border border-indigo-200 dark:border-indigo-500/20 flex flex-col md:flex-row items-start md:items-center justify-between gap-4 shadow-xs">
        <div>
          <div className="flex items-center gap-2">
            <span className="px-2.5 py-1 rounded-full text-xs font-semibold bg-indigo-100 dark:bg-indigo-500/20 text-indigo-800 dark:text-indigo-300 border border-indigo-200 dark:border-indigo-500/30">
              Enterprise Knowledge Base
            </span>
            <span className="text-xs text-slate-500">35 Comprehensive Topics</span>
          </div>
          <h2 className="text-xl font-bold text-slate-900 dark:text-white mt-2">
            Email Hosting User &amp; Administration Guide
          </h2>
          <p className="text-xs text-slate-600 dark:text-slate-300 mt-1 max-w-2xl">
            Live technical guide populated dynamically from active server settings: Hostname{' '}
            <code className="font-mono font-bold text-indigo-600 dark:text-indigo-400">{mailHostname}</code>, Storage{' '}
            <code className="font-mono text-slate-600 dark:text-slate-400">{storagePath}</code>, Ports{' '}
            <code className="font-mono text-indigo-600 dark:text-indigo-400">{imapsPort}</code> /{' '}
            <code className="font-mono text-indigo-600 dark:text-indigo-400">{smtpSubmissionPort}</code>.
          </p>
        </div>

        {/* Search */}
        <div className="w-full md:w-72 relative">
          <Search className="w-4 h-4 text-slate-400 absolute left-3.5 top-1/2 -translate-y-1/2" />
          <input
            type="text"
            placeholder="Search 35 topics..."
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            className="w-full pl-9 pr-3.5 py-2 bg-white dark:bg-surface-950 border border-slate-200 dark:border-surface-800 rounded-xl text-xs text-slate-900 dark:text-white placeholder:text-slate-400 focus:outline-none focus:ring-2 focus:ring-indigo-500/20"
          />
        </div>
      </div>

      {/* Main Grid: Sidebar Navigator + Topic Reader */}
      <div className="grid grid-cols-1 lg:grid-cols-12 gap-6 items-start">
        {/* Topic List (4 columns) */}
        <div className="lg:col-span-4 bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-800 rounded-2xl p-3 shadow-xs max-h-[780px] overflow-y-auto space-y-1">
          <div className="text-[11px] font-bold text-slate-400 uppercase tracking-wider px-3 py-1.5">
            Topics ({filteredSections.length} of 35)
          </div>
          {filteredSections.map((s) => {
            const isSelected = s.id === activeSection.id;
            const Icon = s.icon;
            return (
              <button
                key={s.id}
                onClick={() => setSelectedSectionId(s.id)}
                className={`w-full flex items-center justify-between text-left p-2.5 rounded-xl text-xs transition-all ${
                  isSelected
                    ? 'bg-indigo-50 dark:bg-indigo-950/40 text-indigo-700 dark:text-indigo-300 font-bold border border-indigo-200 dark:border-indigo-800/60 shadow-xs'
                    : 'text-slate-700 dark:text-slate-300 hover:bg-slate-50 dark:hover:bg-surface-800 font-medium'
                }`}
              >
                <div className="flex items-center gap-2.5 truncate">
                  <Icon className={`w-4 h-4 flex-shrink-0 ${isSelected ? 'text-indigo-600 dark:text-indigo-400' : 'text-slate-400'}`} />
                  <span className="truncate">{s.title}</span>
                </div>
                <ChevronRight className={`w-3.5 h-3.5 flex-shrink-0 ${isSelected ? 'text-indigo-600 dark:text-indigo-400' : 'text-slate-400'}`} />
              </button>
            );
          })}
        </div>

        {/* Topic Content Viewer (8 columns) */}
        <div className="lg:col-span-8 bg-white dark:bg-surface-900 border border-slate-200 dark:border-surface-800 rounded-2xl p-6 shadow-xs space-y-6 min-h-[500px]">
          <div className="border-b border-slate-200 dark:border-surface-800 pb-4">
            <div className="flex items-center gap-2 text-xs font-semibold text-indigo-600 dark:text-indigo-400 uppercase tracking-wider">
              <span>{activeSection.category}</span>
              <span>&bull;</span>
              <span>Section {activeSection.id}</span>
            </div>
            <h3 className="text-xl font-bold text-slate-900 dark:text-white mt-1">{activeSection.title}</h3>
            <p className="text-xs text-slate-500 mt-1">{activeSection.summary}</p>
          </div>

          {/* Dynamic Content */}
          <div className="pt-2">
            {activeSection.content(props, copyToClipboard, copiedKey)}
          </div>

          {/* Prev / Next Navigation Buttons */}
          <div className="border-t border-slate-200 dark:border-surface-800 pt-4 flex items-center justify-between">
            <button
              disabled={activeSection.id <= 1}
              onClick={() => setSelectedSectionId((prev) => Math.max(1, prev - 1))}
              className="px-3.5 py-1.5 rounded-xl border border-slate-200 dark:border-surface-700 text-xs font-semibold text-slate-700 dark:text-slate-200 hover:bg-slate-50 dark:hover:bg-surface-800 disabled:opacity-40 transition"
            >
              &larr; Previous Topic
            </button>
            <span className="text-xs text-slate-400">
              {activeSection.id} of 35
            </span>
            <button
              disabled={activeSection.id >= 35}
              onClick={() => setSelectedSectionId((prev) => Math.min(35, prev + 1))}
              className="px-3.5 py-1.5 rounded-xl bg-indigo-600 hover:bg-indigo-500 text-white text-xs font-semibold shadow-xs disabled:opacity-40 transition"
            >
              Next Topic &rarr;
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}
