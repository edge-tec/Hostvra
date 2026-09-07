'use client';

import React, { useEffect, useState } from 'react';
import { AlertCircle, RotateCcw, RefreshCw } from 'lucide-react';

export default function Error({
  error,
  reset,
}: {
  error: Error & { digest?: string };
  reset: () => void;
}) {
  const [isChunkError, setIsChunkError] = useState(false);

  useEffect(() => {
    console.error('Captured application error:', error);
    const msg = (error?.message || '').toLowerCase();
    const chunkFailed =
      msg.includes('loading chunk') ||
      msg.includes('chunkloaderror') ||
      msg.includes('failed to fetch dynamically imported module') ||
      msg.includes('load chunk failed');

    if (chunkFailed) {
      setIsChunkError(true);
      const lastReload = sessionStorage.getItem('last_chunk_reload');
      const now = Date.now();
      if (!lastReload || now - parseInt(lastReload, 10) > 8000) {
        sessionStorage.setItem('last_chunk_reload', now.toString());
        window.location.reload();
      }
    }
  }, [error]);

  const handleRetry = () => {
    if (isChunkError) {
      window.location.reload();
    } else {
      reset();
    }
  };

  return (
    <div className="min-h-screen bg-[#090d16] text-white flex flex-col items-center justify-center p-6 text-center">
      <div className="w-14 h-14 rounded-2xl bg-rose-500/10 border border-rose-500/20 flex items-center justify-center text-rose-400 mb-4 shadow-xl">
        <AlertCircle className="w-7 h-7" />
      </div>
      <h2 className="text-xl font-bold tracking-tight text-white mb-2">
        {isChunkError ? 'New Update Available' : 'Something went wrong'}
      </h2>
      <p className="text-sm text-slate-400 max-w-md mb-6 leading-relaxed">
        {isChunkError
          ? 'A newer version of Hostvra was deployed on the server. Reloading the page will load the latest interface...'
          : error.message || 'An unexpected client error occurred. Please refresh or retry.'}
      </p>
      <div className="flex items-center gap-3">
        <button
          onClick={handleRetry}
          className="inline-flex items-center gap-2 px-4 py-2 rounded-xl bg-indigo-600 hover:bg-indigo-500 text-white text-xs font-semibold shadow-lg transition-all"
        >
          {isChunkError ? <RefreshCw className="w-4 h-4" /> : <RotateCcw className="w-4 h-4" />}
          {isChunkError ? 'Reload Latest Version' : 'Try Again'}
        </button>
        <button
          onClick={() => (window.location.href = '/login')}
          className="px-4 py-2 rounded-xl bg-slate-800 hover:bg-slate-700 text-slate-300 text-xs font-semibold transition-all border border-slate-700/60"
        >
          Go to Login
        </button>
      </div>
    </div>
  );
}
