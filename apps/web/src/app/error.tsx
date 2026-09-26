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
      if (!lastReload || now - parseInt(lastReload, 10) > 6000) {
        sessionStorage.setItem('last_chunk_reload', now.toString());
        const cleanUrl = window.location.pathname.split('?')[0];
        window.location.replace(`${cleanUrl}?_r=${now}`);
      }
    }
  }, [error]);

  const handleRetry = () => {
    if (isChunkError) {
      const cleanUrl = window.location.pathname.split('?')[0];
      window.location.replace(`${cleanUrl}?_r=${Date.now()}`);
    } else {
      reset();
    }
  };

  return (
    <div className="min-h-screen bg-slate-50 text-slate-900 flex flex-col items-center justify-center p-6 text-center font-sans">
      <div className="w-14 h-14 rounded-2xl bg-indigo-50 border border-indigo-200 flex items-center justify-center text-indigo-600 mb-4 shadow-sm">
        <AlertCircle className="w-7 h-7" />
      </div>
      <h2 className="text-xl font-black tracking-tight text-slate-900 mb-2">
        {isChunkError ? 'New Update Available' : 'Something went wrong'}
      </h2>
      <p className="text-sm text-slate-600 max-w-md mb-6 leading-relaxed font-medium">
        {isChunkError
          ? 'A newer version of Hostvra was deployed on the server. Reloading will fetch the latest interface...'
          : error.message || 'An unexpected client error occurred. Please refresh or retry.'}
      </p>
      <div className="flex items-center gap-3">
        <button
          onClick={handleRetry}
          className="inline-flex items-center gap-2 px-5 py-2.5 rounded-xl bg-indigo-600 hover:bg-indigo-700 text-white text-xs font-bold shadow-md shadow-indigo-600/20 transition-all"
        >
          {isChunkError ? <RefreshCw className="w-4 h-4" /> : <RotateCcw className="w-4 h-4" />}
          {isChunkError ? 'Reload Latest Version' : 'Try Again'}
        </button>
        <button
          onClick={() => (window.location.href = '/login')}
          className="px-5 py-2.5 rounded-xl bg-white hover:bg-slate-100 text-slate-700 text-xs font-bold transition-all border border-slate-300 shadow-sm"
        >
          Go to Login
        </button>
      </div>
    </div>
  );
}
