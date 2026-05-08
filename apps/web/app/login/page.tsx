"use client";

import { Nav } from "@/components/nav";
import { getAccessToken } from "@/lib/auth";
import { toZhError } from "@/lib/errors";
import { emitTopNotice } from "@/lib/notice";
import { syncDualLogin } from "@/lib/session-bridge";
import Link from "next/link";
import { FormEvent, useEffect, useState } from "react";
import { useRouter } from "next/navigation";

export default function LoginPage() {
  const router = useRouter();
  const [account, setAccount] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);

  function getRedirectTarget() {
    if (typeof window === "undefined") return "/me";
    const query = new URLSearchParams(window.location.search);
    return query.get("redirect") || "/me";
  }

  useEffect(() => {
    if (getAccessToken()) {
      router.replace(getRedirectTarget());
    }
  }, [router]);

  async function onSubmit(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    setLoading(true);
    setError("");

    try {
      await syncDualLogin(account, password);
      emitTopNotice("登录成功，已建立登录态", "success");
      router.replace(getRedirectTarget());
    } catch (err) {
      setError(toZhError(err, "登录失败"));
    } finally {
      setLoading(false);
    }
  }

  return (
    <main className="page">
      <Nav />
      <h1 style={{ marginTop: 0 }}>登录</h1>
      <div className="card">
        <form onSubmit={onSubmit}>
          <input
            placeholder="用户名或邮箱"
            value={account}
            onChange={(e) => setAccount(e.target.value)}
          />
          <input
            type="password"
            placeholder="密码"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
          />
          <button type="submit" disabled={loading}>
            {loading ? "登录中..." : "登录"}
          </button>
        </form>
        <p className="tip" style={{ marginTop: 8 }}>
          还没有账号？<Link href="/register">去注册</Link>
        </p>
        {error ? <p className="error">{error}</p> : null}
      </div>
    </main>
  );
}
