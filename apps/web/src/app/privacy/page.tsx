import React from 'react';
import Link from 'next/link';

export const metadata = {
  title: 'Privacy Policy — Hostvra Cloud Hosting',
  description: 'How Hostvra collects, secures, and protects customer account, billing, and hosting data.',
};

export default function PrivacyPage() {
  return (
    <main className="min-h-screen bg-slate-50 dark:bg-slate-950 text-slate-800 dark:text-slate-200 py-6 sm:py-12 px-3 sm:px-4">
      <article className="max-w-3xl mx-auto bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 rounded-2xl sm:rounded-3xl p-5 sm:p-8 space-y-5 sm:space-y-6 shadow-sm">
        <div className="flex flex-wrap items-center justify-between gap-2 border-b border-slate-100 dark:border-slate-800 pb-4">
          <h1 className="text-xl sm:text-2xl font-extrabold text-slate-900 dark:text-white">Privacy Policy</h1>
          <Link href="/" className="text-xs font-bold text-emerald-600 hover:underline">&larr; Home</Link>
        </div>
        <p className="text-xs text-slate-500">Effective Date: November 2026 • Hostvra Cloud Infrastructure</p>
        <section className="space-y-3 text-sm leading-relaxed">
          <h2 className="text-base font-bold text-slate-900 dark:text-white">1. Data We Collect</h2>
          <p>We collect account registration details (name, email), domain WHOIS/registrar contact information where required by ICANN, billing invoices, and server security audit logs.</p>
          <h2 className="text-base font-bold text-slate-900 dark:text-white">2. Payment &amp; Security Controls</h2>
          <p>Card payments are processed by PCI-compliant processors (e.g., Stripe). Hostvra never stores raw credit card numbers. Cryptocurrency invoices record public on-chain deposit addresses and transaction hashes for ledger reconciliation.</p>
          <h2 className="text-base font-bold text-slate-900 dark:text-white">3. Strict Tenant Isolation</h2>
          <p>Customer websites, databases, files, and email accounts are isolated per user account and are never shared with other tenants.</p>
        </section>
      </article>
    </main>
  );
}
