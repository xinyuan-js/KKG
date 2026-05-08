"use client";

import Link from "next/link";
import { getMyNotifications, getMyProfile, searchSuggest } from "@/lib/api";
import {
  AUTH_CHANGED_EVENT,
  clearAccessToken,
  getAccessToken,
  getUserProfile,
  setUserProfile
} from "@/lib/auth";
import { syncDualLogout } from "@/lib/session-bridge";
import { emitTopNotice } from "@/lib/notice";
import { swipeNavigate } from "@/lib/view-transition";
import { KeyboardEvent, useEffect, useMemo, useRef, useState } from "react";
import { usePathname, useRouter, useSearchParams } from "next/navigation";

export function Nav() {
  const router = useRouter();
  const pathname = usePathname();
  const searchParams = useSearchParams();
  const [loggedIn, setLoggedIn] = useState(false);
  const [username, setUsername] = useState("");
  const [avatarURL, setAvatarURL] = useState("");
  const [unreadCount, setUnreadCount] = useState(0);
  const [searchType, setSearchType] = useState<"post" | "user">("post");
  const [query, setQuery] = useState("");
  const [suggestions, setSuggestions] = useState<string[]>([]);
  const [activeIndex, setActiveIndex] = useState(-1);
  const [searchOpen, setSearchOpen] = useState(false);
  const [searching, setSearching] = useState(false);
  const searchWrapRef = useRef<HTMLDivElement | null>(null);
  const [theme, setTheme] = useState<"light" | "dark">("light");
  const [themeReady, setThemeReady] = useState(false);

  const canSearch = useMemo(() => query.trim().length > 0, [query]);

  function syncFromLocal() {
    const token = getAccessToken();
    const profile = getUserProfile();
    setLoggedIn(!!token);
    setUsername(profile?.username || "");
    setAvatarURL(profile?.avatar_url || "");
    if (!token) {
      setUnreadCount(0);
    }
  }

  async function validateToken() {
    const token = getAccessToken();
    if (!token) {
      setUnreadCount(0);
      return;
    }
    try {
      const [p, n] = await Promise.all([getMyProfile(token), getMyNotifications(token, 20)]);
      setUserProfile(p);
      setUnreadCount(n.unread_count || 0);
    } catch {
      // token失效时统一清理，避免“看起来已登录但接口401”。
      clearAccessToken();
      setUnreadCount(0);
    }
  }

  useEffect(() => {
    const t = document.documentElement.getAttribute("data-theme");
    setTheme(t === "dark" ? "dark" : "light");
    setThemeReady(true);
  }, []);

  useEffect(() => {
    syncFromLocal();
    void validateToken();

    const onStorage = (e: StorageEvent) => {
      if (!e.key || e.key === "access_token" || e.key === "user_profile") {
        syncFromLocal();
      }
    };
    const onAuthChanged = () => syncFromLocal();
    const onFocus = () => {
      syncFromLocal();
      void validateToken();
    };

    window.addEventListener("storage", onStorage);
    window.addEventListener(AUTH_CHANGED_EVENT, onAuthChanged);
    window.addEventListener("focus", onFocus);
    document.addEventListener("visibilitychange", onFocus);
    return () => {
      window.removeEventListener("storage", onStorage);
      window.removeEventListener(AUTH_CHANGED_EVENT, onAuthChanged);
      window.removeEventListener("focus", onFocus);
      document.removeEventListener("visibilitychange", onFocus);
    };
  }, []);

  useEffect(() => {
    syncFromLocal();
  }, [pathname]);

  useEffect(() => {
    setSearchOpen(false);
    setSuggestions([]);
    setActiveIndex(-1);
  }, [pathname]);

  function toggleTheme() {
    const next: "light" | "dark" = theme === "light" ? "dark" : "light";
    setTheme(next);
    document.documentElement.setAttribute("data-theme", next);
    window.localStorage.setItem("theme_mode", next);
    document.cookie = `theme_mode=${next}; path=/; max-age=31536000; samesite=lax`;
  }

  useEffect(() => {
    const kw = query.trim();
    if (!kw) {
      setSuggestions([]);
      setActiveIndex(-1);
      return;
    }
    const timer = window.setTimeout(async () => {
      try {
        const data = await searchSuggest({
          type: searchType,
          q: kw,
          limit: 5,
          token: getAccessToken() || undefined
        });
        setSuggestions(data.items || []);
      } catch {
        setSuggestions([]);
      }
      setActiveIndex(-1);
    }, 220);
    return () => window.clearTimeout(timer);
  }, [query, searchType]);

  useEffect(() => {
    function onDocMouseDown(e: MouseEvent) {
      const target = e.target as Node | null;
      if (!searchWrapRef.current || !target) return;
      if (!searchWrapRef.current.contains(target)) {
        setSearchOpen(false);
        setActiveIndex(-1);
      }
    }
    document.addEventListener("mousedown", onDocMouseDown);
    return () => document.removeEventListener("mousedown", onDocMouseDown);
  }, []);

  async function onLogout() {
    await syncDualLogout();
    setLoggedIn(false);
    setUsername("");
    setAvatarURL("");
    emitTopNotice("已退出登录", "success");
    router.push("/login");
  }

  function runSearch(text: string) {
    const kw = text.trim();
    if (!kw) return;
    setSearching(true);
    setSearchOpen(false);
    setSuggestions([]);
    setActiveIndex(-1);
    router.push(`/search?type=${searchType}&q=${encodeURIComponent(kw)}`);
    setTimeout(() => setSearching(false), 120);
  }

  function onInputKeyDown(e: KeyboardEvent<HTMLInputElement>) {
    if (!suggestions.length) {
      if (e.key === "Enter") {
        e.preventDefault();
        void runSearch(query);
      }
      return;
    }
    if (e.key === "ArrowDown") {
      e.preventDefault();
      setActiveIndex((prev) => (prev + 1) % suggestions.length);
      return;
    }
    if (e.key === "ArrowUp") {
      e.preventDefault();
      setActiveIndex((prev) => (prev <= 0 ? suggestions.length - 1 : prev - 1));
      return;
    }
    if (e.key === "Escape") {
      setSearchOpen(false);
      setSuggestions([]);
      setActiveIndex(-1);
      return;
    }
    if (e.key === "Enter") {
      e.preventDefault();
      if (activeIndex >= 0 && activeIndex < suggestions.length) {
        const value = suggestions[activeIndex];
        setQuery(value);
        setSuggestions([]);
        setActiveIndex(-1);
        void runSearch(value);
      } else {
        void runSearch(query);
      }
    }
  }

  const mainTab = pathname === "/rankings" || (pathname === "/" && searchParams.get("tab") === "rankings") ? "rankings" : "home";

  return (
    <div className="nav">
      <Link href="/" className="brand">
        KKG
      </Link>
      <div className="nav-search-wrap" ref={searchWrapRef}>
        <div className="nav-search-type seg-switch seg-switch-2" data-active={searchType === "post" ? 0 : 1}>
          <span className="seg-switch-thumb" aria-hidden="true" />
          <button type="button" className={searchType === "post" ? "" : "ghost"} onClick={() => setSearchType("post")}>
            文章
          </button>
          <button type="button" className={searchType === "user" ? "" : "ghost"} onClick={() => setSearchType("user")}>
            用户
          </button>
        </div>
        <input
          className="nav-search-input"
          value={query}
          onFocus={() => setSearchOpen(true)}
          onChange={(e) => setQuery(e.target.value)}
          onKeyDown={onInputKeyDown}
          placeholder={searchType === "post" ? "搜索文章标题或摘要" : "搜索用户"}
        />
        <button type="button" className="ghost nav-search-btn" disabled={!canSearch || searching} onClick={() => runSearch(query)}>
          {searching ? "..." : "搜索"}
        </button>
        {searchOpen && suggestions.length > 0 ? (
          <div className="nav-search-panel">
            <div className="nav-search-suggest">
              {suggestions.map((text, idx) => (
                <button
                  key={`${text}-${idx}`}
                  type="button"
                  className={`nav-search-suggest-item${activeIndex === idx ? " active" : ""}`}
                  onClick={() => {
                    setQuery(text);
                    setSuggestions([]);
                    setActiveIndex(-1);
                    runSearch(text);
                  }}
                >
                  {text}
                </button>
              ))}
            </div>
          </div>
        ) : null}
      </div>
      <div className="links">
        <div className="nav-main-switch seg-switch seg-switch-2" data-active={mainTab === "rankings" ? 1 : 0}>
          <span className="seg-switch-thumb" aria-hidden="true" />
          <Link href="/?tab=home" className={`nav-main-link${mainTab === "home" ? " active" : ""}`}>
            首页
          </Link>
          <Link href="/?tab=rankings" prefetch={false} className={`nav-main-link${mainTab === "rankings" ? " active" : ""}`}>
            排行榜
          </Link>
        </div>
        <button type="button" className={`theme-toggle ${theme === "dark" ? "is-dark" : ""}`} onClick={toggleTheme}>
          <span className="theme-toggle-track">
            <span className="theme-toggle-thumb">{themeReady ? (theme === "dark" ? "🌙" : "☀️") : "•"}</span>
          </span>
        </button>
        {loggedIn ? (
          <div className="nav-user-menu">
            <Link href="/me" prefetch={false} className="nav-user-link">
              {avatarURL ? (
                <img className="nav-avatar" src={avatarURL} alt={username || "avatar"} />
              ) : (
                <span className="nav-avatar nav-avatar-fallback">{(username || "U").slice(0, 1).toUpperCase()}</span>
              )}
              <span>{username || "个人中心"}</span>
            </Link>
            <div className="nav-user-dropdown">
              <Link href="/write" prefetch={false}>
                写文章
              </Link>
              <Link href="/me" prefetch={false}>
                个人资料
              </Link>
              <Link href="/me/posts" prefetch={false}>
                我的文章
              </Link>
              <Link href="/me/favorites" prefetch={false}>
                我的收藏
              </Link>
              <Link href="/me/inbox" prefetch={false} className="nav-inbox-link">
                @我
                {unreadCount > 0 ? <span className="nav-badge">{unreadCount > 99 ? "99+" : unreadCount}</span> : null}
              </Link>
              {["admin", "super_admin"].includes(getUserProfile()?.role || "") ? (
                <Link href="/me/admin" prefetch={false}>
                  管理中心
                </Link>
              ) : null}
              <button type="button" className="nav-logout-btn" onClick={onLogout}>
                退出登录
              </button>
            </div>
          </div>
        ) : (
          <Link href="/login">登录</Link>
        )}
        <button
          type="button"
          className="nav-transfer nav-transfer-right"
          aria-label="切换到 OJ"
          onClick={() => swipeNavigate(router, "/oj", "to-oj")}
        >
          <span className="nav-transfer-text">OJ</span>
          <span className="nav-transfer-arrow" aria-hidden="true">→</span>
        </button>
      </div>
    </div>
  );
}
