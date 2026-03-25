import { NextRequest, NextResponse } from 'next/server';

const SESSION_COOKIE_PATTERN = /^[a-f0-9]{32}$/;

// Server-side route guard: redirects to /login when `kimbap_session` cookie is absent.
// Cookie is set client-side on login, cleared on logout/401.
export function middleware(request: NextRequest) {
  const sessionCookie = request.cookies.get('kimbap_session');

  if (!sessionCookie?.value || !SESSION_COOKIE_PATTERN.test(sessionCookie.value)) {
    const loginUrl = new URL('/login', request.url);
    loginUrl.searchParams.set('redirect', `${request.nextUrl.pathname}${request.nextUrl.search}`);
    return NextResponse.redirect(loginUrl);
  }

  return NextResponse.next();
}

export const config = {
  matcher: ['/dashboard/:path*'],
};
