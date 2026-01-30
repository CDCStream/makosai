import { createBrowserClient } from '@supabase/ssr';

const supabaseUrl = process.env.NEXT_PUBLIC_SUPABASE_URL!;
const supabaseAnonKey = process.env.NEXT_PUBLIC_SUPABASE_ANON_KEY!;

// Browser client using @supabase/ssr - stores code_verifier in cookies
// This is required for PKCE flow in Next.js SSR environment
export const supabase = createBrowserClient(supabaseUrl, supabaseAnonKey);

// Singleton getter for supabase client
export const getSupabase = () => supabase;
