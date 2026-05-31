"use client";

import Link from "next/link";
import { usePathname, useRouter, useSearchParams } from "next/navigation";
import { FormEvent, useEffect, useState } from "react";
import { ojGetLoginUser, type OJUserVO } from "@/lib/oj-api";
import { getUserProfile } from "@/lib/auth";
import { logoutAuthSession } from "@/lib/session-bridge";
import { emitTopNotice } from "@/lib/notice";
import { swipeNavigate } from "@/lib/view-transition";
import { Avatar } from "@/components/avatar";

export function OJNav() {
  const pathname = usePathname();
  const searchParams = useSearchParams();
  const router = useRouter();
  const [me, setMe] = useState<OJUserVO | null>(null);
  const [questionQuery, setQuestionQuery] = useState("");
  const [theme, setTheme] = useState<"light" | "dark">("light");
  const [themeReady, setThemeReady] = useState(false);

  useEffect(() => {
    const t = document.documentElement.getAttribute("data-theme");
    setTheme(t === "dark" ? "dark" : "light");
    setThemeReady(true);
  }, []);

  useEffect(() => {
    const profile = getUserProfile();
    void ojGetLoginUser()
      .then((user) => {
        setMe({
          ...user,
          userAvatar: user.userAvatar || profile?.avatar_url || "",
          userName: user.userName || profile?.username || ""
        });
      })
      .catch(() => setMe(null));
  }, [pathname]);

  function toggleTheme() {
    const next: "light" | "dark" = theme === "light" ? "dark" : "light";
    setTheme(next);
    if (typeof document !== "undefined") {
      document.documentElement.setAttribute("data-theme", next);
    }
    if (typeof window !== "undefined") {
      window.localStorage.setItem("theme_mode", next);
    }
    if (typeof document !== "undefined") {
      document.cookie = `theme_mode=${next}; path=/; max-age=31536000; samesite=lax`;
    }
  }

  async function onLogout() {
    try {
      await logoutAuthSession();
      emitTopNotice("已退出登录", "success");
    } finally {
      setMe(null);
      router.push("/login?redirect=/oj");
    }
  }
  function onQuestionSearch(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    const q = questionQuery.trim();
    router.push(q ? `/oj?tab=questions&q=${encodeURIComponent(q)}` : "/oj?tab=questions");
  }

  const links = [
    ["/oj?tab=overview", "总览"],
    ["/oj?tab=questions", "题目"]
  ] as const;
  const isActive = (href: string) => {
    if (pathname.startsWith("/oj/questions")) {
      if (href.startsWith("/oj?tab=questions")) return true;
      if (href.startsWith("/oj?tab=overview")) return false;
    }
    if (href.startsWith("/oj?tab=overview")) return pathname === "/oj" && (searchParams.get("tab") || "overview") !== "questions";
    if (href.startsWith("/oj?tab=questions")) return pathname === "/oj" && searchParams.get("tab") === "questions";
    if (href === "/me") return pathname === "/me" || pathname.startsWith("/me/");
    return pathname === href || pathname.startsWith(`${href}/`);
  };

  return (
    <div className="nav">
      <div className="brand-wrap">
        <button
          type="button"
          className="nav-transfer nav-transfer-left"
          aria-label="切换到博客"
          onClick={() => swipeNavigate(router, "/", "to-blog")}
        >
          <span className="nav-transfer-arrow" aria-hidden="true">←</span>
          <span className="nav-transfer-text">BLOG</span>
        </button>
        <Link href="/oj" className="brand">KKG OJ</Link>
      </div>
      <div className="nav-search-wrap">
        <div
          className="nav-main-switch seg-switch seg-switch-2"
          data-active={isActive("/oj?tab=questions") ? 1 : 0}
          style={{ margin: 0 }}
        >
          <span className="seg-switch-thumb" aria-hidden="true" />
          {links.map(([href, label]) => (
            <Link key={href} href={href} className={`nav-main-link${isActive(href) ? " active" : ""}`}>
              {label}
            </Link>
          ))}
        </div>
        <form className="toolbar-row" onSubmit={onQuestionSearch}>
          <input
            className="nav-search-input"
            placeholder="搜索题目"
            value={questionQuery}
            onChange={(e) => setQuestionQuery(e.target.value)}
          />
          <button type="submit" className="nav-search-btn">搜索</button>
        </form>
      </div>
      <div className="links">
        <button type="button" className={`theme-toggle ${theme === "dark" ? "is-dark" : ""}`} onClick={toggleTheme} aria-label="切换主题">
          <span className="theme-toggle-track">
            <span className="theme-toggle-thumb">{themeReady ? (theme === "dark" ? "🌙" : "☀️") : "•"}</span>
          </span>
        </button>
        {me ? (
          <div className="nav-user-menu">
            <Link href="/me" prefetch={false} className="nav-user-link">
              <Avatar className="nav-avatar" fallbackClassName="nav-avatar nav-avatar-fallback" src={me.userAvatar} name={me.userName} />
              <span>{me.userName || "个人中心"}</span>
            </Link>
            <div className="nav-user-dropdown">
              <Link href="/oj" prefetch={false}>OJ 总览</Link>
              <Link href="/oj/questions" prefetch={false}>题库</Link>
              <Link href="/write" prefetch={false}>写文章</Link>
              <Link href="/me" prefetch={false}>个人中心</Link>
              <Link href="/" prefetch={false}>返回博客</Link>
              <button type="button" className="nav-logout-btn" onClick={onLogout}>退出登录</button>
            </div>
          </div>
        ) : (
          <>
            <Link className="tweet-link" href="/">返回博客</Link>
            <Link className="tweet-link" href="/login?redirect=/oj">登录</Link>
            <Link className="tweet-link" href="/register?redirect=/oj">注册</Link>
          </>
        )}
      </div>
    </div>
  );
}
