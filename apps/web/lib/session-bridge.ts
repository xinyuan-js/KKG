import { getCurrentUser, login } from "@/lib/api";
import { setAuthSession, clearAccessToken } from "@/lib/auth";

export async function loginWithPassword(account: string, password: string) {
  await login(account, password);
  const user = await getCurrentUser();
  setAuthSession(user);
}

export async function logoutAuthSession() {
  try {
    await fetch(`${process.env.NEXT_PUBLIC_API_BASE || "/blog-api"}/api/v1/auth/logout`, {
      method: "POST",
      credentials: "include"
    });
  } catch {
    // ignore
  }
  clearAccessToken();
}
