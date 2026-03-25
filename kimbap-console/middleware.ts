import { NextRequest, NextResponse } from 'next/server';

// Server-side route guard: redirects to /login when `kimbap_session` cookie is absent.
// Cookie is set client-side on login, cleared on logout/401.
export function middleware(request: NextRequest) {
  const sessionCookie = request.cookies.get('kimbap_session');

  if (!sessionCookie?.value) {
    const loginUrl = new URL('/login', request.url);
    loginUrl.searchParams.set('redirect', request.nextUrl.pathname);
    return NextResponse.redirect(loginUrl);
  }

  return NextResponse.next();
}

export const config = {
  matcher: ['/dashboard/:path*'],
};
