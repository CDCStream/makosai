'use client';

import { useEffect, useState, useRef } from 'react';
import { useRouter } from 'next/navigation';
import { supabase } from '@/lib/supabase';

export default function CallbackHandlePage() {
  const router = useRouter();
  const [error, setError] = useState<string | null>(null);
  const [processing, setProcessing] = useState(true);
  const hasRun = useRef(false);

  useEffect(() => {
    // Prevent double execution in strict mode
    if (hasRun.current) return;
    hasRun.current = true;

    const handleCallback = async () => {
      try {
        const queryParams = new URLSearchParams(window.location.search);

        // Check for error in URL
        const urlError = queryParams.get('error');
        if (urlError) {
          const errorDescription = queryParams.get('error_description');
          setError(decodeURIComponent(errorDescription || urlError));
          setProcessing(false);
          setTimeout(() => router.push('/login'), 3000);
          return;
        }

        // PKCE flow: exchange auth code for session
        // The code_verifier is stored in localStorage by the Supabase client
        const code = queryParams.get('code');

        if (code) {
          console.log('Exchanging code for session...');
          const { data, error: exchangeError } = await supabase.auth.exchangeCodeForSession(code);

          if (exchangeError) {
            console.error('Code exchange error:', exchangeError);
            setError(exchangeError.message);
            setProcessing(false);
            setTimeout(() => router.push('/login'), 3000);
            return;
          }

          if (data.session) {
            console.log('Session obtained successfully');
            router.push('/');
            return;
          }
        }

        // Fallback: check if session already exists
        const { data: { session }, error: sessionError } = await supabase.auth.getSession();

        if (sessionError) {
          console.error('Session error:', sessionError);
          setError(sessionError.message);
          setProcessing(false);
          setTimeout(() => router.push('/login'), 3000);
          return;
        }

        if (session) {
          router.push('/');
          return;
        }

        // No code and no session - something went wrong
        setError('No authentication code received. Please try again.');
        setProcessing(false);
        setTimeout(() => router.push('/login'), 3000);

      } catch (err) {
        console.error('Callback error:', err);
        setError('An unexpected error occurred');
        setProcessing(false);
        setTimeout(() => router.push('/login'), 3000);
      }
    };

    handleCallback();
  }, [router]);

  return (
    <div className="min-h-screen flex items-center justify-center bg-gradient-to-br from-teal-50 to-cyan-50">
      <div className="text-center">
        {error ? (
          <div className="bg-red-50 border border-red-200 rounded-xl p-6 max-w-md mx-4">
            <p className="text-red-600 font-medium">{error}</p>
            <p className="text-gray-500 text-sm mt-2">Redirecting to login...</p>
          </div>
        ) : (
          <>
            <div className="w-16 h-16 border-4 border-teal-500 border-t-transparent rounded-full animate-spin mx-auto mb-4"></div>
            <p className="text-gray-600 font-medium">Completing sign in...</p>
          </>
        )}
      </div>
    </div>
  );
}
