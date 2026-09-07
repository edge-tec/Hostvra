'use client';

import React, { useEffect } from 'react';
import { AlertCircle, RotateCcw, RefreshCw } from 'lucide-react';

export default function GlobalError({
  error,
  reset,
}: {
  error: Error & { digest?: string };
  reset: () => void;
}) {
  useEffect(() => {
    const msg = (error?.message || '').toLowerCase();
    if (
      msg.includes('loading chunk') ||
      msg.includes('chunkloaderror') ||
      msg.includes('failed to fetch dynamically imported module')
    ) {
      const lastReload = sessionStorage.getItem('last_chunk_reload');
      const now = Date.now();
      if (!lastReload || now - parseInt(lastReload, 10) > 8000) {
        sessionStorage.setItem('last_chunk_reload', now.toString());
        window.location.reload();
      }
    }
  }, [error]);

  return (
    <html lang="en">
      <body className="min-h-screen bg-[#090d16] text-white flex flex-col items-center justify-center p-6 text-center font-sans">
        <div className="w-14 h-14 rounded-2xl bg-rose-500/10 border border-rose-500/20 flex items-center justify-center text-rose-400 mb-4 shadow-xl">
          <AlertCircle className="w-7 h-7" />
        </div>
        <h2 className="text-xl font-bold tracking-tight text-white mb-2">
          Application Update
        </h2>
        <p className="text-sm text-slate-400 max-w-md mb-6 leading-relaxed">
          The application assets were updated on the server. Please reload to fetch the latest version.
        </p>
        <button
          onClick={() => window.location.reload()}
          className="inline-flex items-center gap-2 px-5 py-2.5 rounded-xl bg-indigo-600 hover:bg-indigo-500 text-white text-xs font-semibold shadow-lg transition-all"
        >
          <RefreshCw className="w-4 h-4" />
          Reload Application
        </button>
      </body>
    </html>
  );
}
