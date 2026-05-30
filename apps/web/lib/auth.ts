export const ACCESS_TOKEN_KEY = "access_token";
export const USER_PROFILE_KEY = "user_profile";
export const AUTH_CHANGED_EVENT = "auth:changed";

export type UserProfile = {
  id: number;
  username: string;
  email: string;
  avatar_url?: string;
  role?: string;
};

export function setAccessToken(token: string) {
  if (typeof window === "undefined") return;
  if (token) {
    localStorage.setItem(ACCESS_TOKEN_KEY, "cookie");
  }
}

export function setAuthSession(profile: UserProfile, token = "cookie") {
  if (typeof window === "undefined") return;
  setAccessToken(token);
  setUserProfile(profile);
  emitAuthChanged();
}

export function getAccessToken(): string {
  if (typeof window === "undefined") return "";
  return localStorage.getItem(ACCESS_TOKEN_KEY) || (localStorage.getItem(USER_PROFILE_KEY) ? "cookie" : "");
}

export function getUserProfile(): UserProfile | null {
  if (typeof window === "undefined") return null;
  const raw = localStorage.getItem(USER_PROFILE_KEY);
  if (!raw) return null;
  try {
    return JSON.parse(raw) as UserProfile;
  } catch {
    return null;
  }
}

export function clearAccessToken() {
  if (typeof window === "undefined") return;
  localStorage.removeItem(ACCESS_TOKEN_KEY);
  localStorage.removeItem(USER_PROFILE_KEY);
  emitAuthChanged();
}

export function setUserProfile(profile: UserProfile) {
  if (typeof window === "undefined") return;
  localStorage.setItem(USER_PROFILE_KEY, JSON.stringify(profile));
  emitAuthChanged();
}

export function emitAuthChanged() {
  if (typeof window === "undefined") return;
  window.dispatchEvent(new Event(AUTH_CHANGED_EVENT));
}
