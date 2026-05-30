"use client";

import { Nav } from "@/components/nav";
import { getCurrentUser, register } from "@/lib/api";
import { setAuthSession } from "@/lib/auth";
import { toZhError } from "@/lib/errors";
import { emitTopNotice } from "@/lib/notice";
import { loginWithPassword } from "@/lib/session-bridge";
import Link from "next/link";
import { FormEvent, useEffect, useState } from "react";
import { useRouter } from "next/navigation";

export default function RegisterPage() {
  const router = useRouter();
  const [username, setUsername] = useState("");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);

  function getRedirectTarget() {
    if (typeof window === "undefined") return "/me";
    const query = new URLSearchParams(window.location.search);
    return query.get("redirect") || "/me";
  }

  useEffect(() => {
    void getCurrentUser()
      .then((user) => {
        setAuthSession(user);
        router.replace(getRedirectTarget());
      })
      .catch(() => {});
  }, [router]);

  async function onSubmit(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    setLoading(true);
    setError("");
    try {
      await register({ username, email, password });
      await loginWithPassword(username, password);
      emitTopNotice("注册并登录成功", "success");
      router.replace(getRedirectTarget());
    } catch (err) {
      setError(toZhError(err, "注册失败"));
    } finally {
      setLoading(false);
    }
  }

  return (
    <main className="page">
      <Nav />
      <h1 style={{ marginTop: 0 }}>注册</h1>
      <div className="card">
        <form onSubmit={onSubmit}>
          <input placeholder="用户名" value={username} onChange={(e) => setUsername(e.target.value)} />
          <input placeholder="邮箱" value={email} onChange={(e) => setEmail(e.target.value)} />
          <input
            type="password"
            placeholder="密码（至少8位）"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
          />
          <button type="submit" disabled={loading}>
            {loading ? "注册中..." : "注册并登录"}
          </button>
        </form>
        <p className="tip" style={{ marginTop: 8 }}>
          已有账号？<Link href="/login">去登录</Link>
        </p>
        {error ? <p className="error">{error}</p> : null}
      </div>
    </main>
  );
}
