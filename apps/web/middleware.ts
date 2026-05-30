import { NextRequest, NextResponse } from "next/server";

const ACCESS_TOKEN_KEY = "access_token";
const REFRESH_TOKEN_KEY = "refresh_token";

export function middleware(request: NextRequest) {
  const token = request.cookies.get(ACCESS_TOKEN_KEY)?.value || request.cookies.get(REFRESH_TOKEN_KEY)?.value;
  const { pathname } = request.nextUrl;

  if ((pathname.startsWith("/editor") || pathname.startsWith("/write") || pathname.startsWith("/me")) && !token) {
    const loginUrl = new URL("/login", request.url);
    loginUrl.searchParams.set("redirect", pathname);
    return NextResponse.redirect(loginUrl);
  }

  return NextResponse.next();
}

export const config = {
  matcher: ["/editor/:path*", "/write/:path*", "/me/:path*"]
};
