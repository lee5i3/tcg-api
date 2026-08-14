/**
 * Client-side route guard for the admin SPA.
 *
 * Returns the path to redirect to, or null when the current location is
 * allowed: unauthenticated visitors may only see /login, and authenticated
 * admins are bounced from /login back to the dashboard.
 */
export function guardTarget(token: string | null, pathname: string): string | null {
  const onLogin = pathname === "/login" || pathname.startsWith("/login/");
  if (!token && !onLogin) {
    return "/login";
  }
  if (token && onLogin) {
    return "/";
  }
  return null;
}
