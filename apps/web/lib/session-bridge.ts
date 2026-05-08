import { login } from "@/lib/api";
import { setAuthSession, clearAccessToken } from "@/lib/auth";
import { ojLogin, ojLogout } from "@/lib/oj-api";

export async function syncBlogLogin(account: string, password: string) {
  const payload = await login(account, password);
  setAuthSession(payload.access_token, payload.user);
}

export async function syncOJLogin(account: string, password: string) {
  await ojLogin(account, password);
}

export async function syncDualLogin(account: string, password: string) {
  let blogErr: unknown = null;
  let ojErr: unknown = null;
  await Promise.all([
    syncBlogLogin(account, password).catch((e) => {
      blogErr = e;
    }),
    syncOJLogin(account, password).catch((e) => {
      ojErr = e;
    })
  ]);
  if (blogErr && ojErr) {
    throw blogErr;
  }
}

export async function syncDualLogout() {
  clearAccessToken();
  try {
    await ojLogout();
  } catch {
    // ignore
  }
}
