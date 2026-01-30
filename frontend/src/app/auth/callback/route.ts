import { NextResponse } from 'next/server';
import type { NextRequest } from 'next/server';

export async function GET(request: NextRequest) {
  const requestUrl = new URL(request.url);
  const code = requestUrl.searchParams.get('code');
  const error = requestUrl.searchParams.get('error');
  const error_description = requestUrl.searchParams.get('error_description');

  // If there's an error, redirect to login with error message
  if (error) {
    const loginUrl = new URL('/login', requestUrl.origin);
    loginUrl.searchParams.set('error', error_description || error);
    return NextResponse.redirect(loginUrl);
  }

  // Redirect to a client-side page that will handle the code exchange
  // The browser has the PKCE code_verifier cookie, so it can complete the exchange
  if (code) {
    const callbackUrl = new URL('/auth/callback/handle', requestUrl.origin);
    callbackUrl.searchParams.set('code', code);
    return NextResponse.redirect(callbackUrl);
  }

  // No code, redirect to home
  return NextResponse.redirect(new URL('/', requestUrl.origin));
}


