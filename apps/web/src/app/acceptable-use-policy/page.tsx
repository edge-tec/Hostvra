import React from 'react';
import Link from 'next/link';

export const metadata = {
  title: 'Acceptable Use Policy — Hostvra',
  description: 'Permitted and prohibited uses of Hostvra cloud hosting, email, and network resources.',
};

export default function AUPPage() {
  return (
    <main className="min-h-screen bg-slate-50 dark:bg-slate-950 text-slate-800 dark:text-slate-200 py-12 px-4">
      <article className="max-w-3xl mx-auto bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 rounded-3xl p-8 space-y-6 shadow-sm">
        <div className="flex items-center justify-between border-b border-slate-100 dark:border-slate-800 pb-4">
          <h1 className="text-2xl font-extrabold text-slate-900 dark:text-white">Acceptable Use Policy (AUP)</h1>
          <Link href="/" className="text-xs font-bold text-emerald-600 hover:underline">&larr; Home</Link>
        </div>
        <section className="space-y-3 text-sm leading-relaxed">
          <h2 className="text-base font-bold text-slate-900 dark:text-white">Prohibited Activities</h2>
          <ul className="list-disc pl-5 space-y-1.5">
            <li>Unsolicited bulk email (spam), phishing sites, or credential harvesting.</li>
            <li>Distribution of malware, ransomware, or command-and-control infrastructure.</li>
            <li>Unauthorized cryptocurrency mining or network denial-of-service (DDoS) attacks.</li>
            <li>Attempting to bypass multi-tenant account isolation or server resource quotas.</li>
          </ul>
        </section>
      </article>
    </main>
  );
}
