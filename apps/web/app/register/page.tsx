"use client";

import { Nav } from "@/components/nav";
import { register } from "@/lib/api";
import { getAccessToken } from "@/lib/auth";
import { toZhError } from "@/lib/errors";
import { emitTopNotice } from "@/lib/notice";
import { ojRegister } from "@/lib/oj-api";
import { syncDualLogin } from "@/lib/session-bridge";
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

  useEffect(() => {
    if (getAccessToken()) {
      router.replace("/me");
    }
  }, [router]);

  async function onSubmit(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    setLoading(true);
    setError("");
    try {
      await register({ username, email, password });
      try {
        await ojRegister(username, password, password);
      } catch {
        // ignore existing-account or transient errors, login bridge below handles final state
      }
      await syncDualLogin(username, password);
      emitTopNotice("注册并登录成功", "success");
      router.replace("/me");
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
