import React from 'react';
import Link from 'next/link';

export const metadata = {
  title: 'Terms of Service — Hostvra Cloud Hosting',
  description: 'Hostvra Cloud Hosting Terms of Service, billing cycles, acceptable use, and service conditions.',
};

export default function TermsPage() {
  return (
    <main className="min-h-screen bg-slate-50 dark:bg-slate-950 text-slate-800 dark:text-slate-200 py-12 px-4">
      <article className="max-w-3xl mx-auto bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 rounded-3xl p-8 space-y-6 shadow-sm">
        <div className="flex items-center justify-between border-b border-slate-100 dark:border-slate-800 pb-4">
          <h1 className="text-2xl font-extrabold text-slate-900 dark:text-white">Terms of Service</h1>
          <Link href="/" className="text-xs font-bold text-emerald-600 hover:underline">&larr; Home</Link>
        </div>
        <p className="text-xs text-slate-500">Effective Date: November 2026 • Hostvra Cloud Infrastructure</p>
        <section className="space-y-3 text-sm leading-relaxed">
          <h2 className="text-base font-bold text-slate-900 dark:text-white">1. Account &amp; Multi-Tenant Isolation</h2>
          <p>Each Hostvra customer receives an isolated hosting environment. Customers are responsible for safeguarding account credentials, 2FA recovery codes, and hosted application code.</p>
          <h2 className="text-base font-bold text-slate-900 dark:text-white">2. Billing, Renewals &amp; Grace Period</h2>
          <p>Renewal invoices are generated 7 days prior to the billing due date. Services remaining unpaid beyond the configured 3-day grace period may be automatically suspended until payment is settled.</p>
          <h2 className="text-base font-bold text-slate-900 dark:text-white">3. Cryptocurrency &amp; Card Payments</h2>
          <p>On-chain cryptocurrency payments settle automatically upon required network confirmations. Customers must send only the selected asset on the selected blockchain network to the invoice deposit address.</p>
          <h2 className="text-base font-bold text-slate-900 dark:text-white">4. Acceptable Use</h2>
          <p>Use of Hostvra infrastructure is subject to our <Link href="/acceptable-use-policy" className="text-emerald-600 underline">Acceptable Use Policy</Link>. Prohibited activities include phishing, malware distribution, DDoS origination, and unauthorized network scanning.</p>
        </section>
      </article>
    </main>
  );
}
