"use client";

import { toZhError } from "@/lib/errors";
import { emitTopNotice } from "@/lib/notice";
import { syncDualLogin } from "@/lib/session-bridge";
import Link from "next/link";
import { FormEvent, useState } from "react";
import { useRouter } from "next/navigation";

export default function OJLoginPage() {
  const router = useRouter();
  const [userAccount, setUserAccount] = useState("");
  const [userPassword, setUserPassword] = useState("");
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);

  async function onSubmit(e: FormEvent) {
    e.preventDefault();
    setError("");
    setLoading(true);
    try {
      await syncDualLogin(userAccount, userPassword);
      emitTopNotice("登录成功", "success");
      router.push("/oj");
    } catch (err) {
      setError(toZhError(err, "登录失败"));
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="reddit-layout">
      <section className="reddit-main">
        <section className="card oj-auth-card oj-side-card">
          <h1 className="title">OJ 登录</h1>
          <form onSubmit={onSubmit}>
            <input placeholder="账号" value={userAccount} onChange={(e) => setUserAccount(e.target.value)} />
            <input type="password" placeholder="密码" value={userPassword} onChange={(e) => setUserPassword(e.target.value)} />
            <button type="submit" disabled={loading}>{loading ? "登录中..." : "登录"}</button>
          </form>
          <p className="tip" style={{ marginTop: 10 }}>没有账号？<Link href="/oj/register">去注册</Link></p>
          {error ? <p className="error">{error}</p> : null}
        </section>
      </section>
      <aside className="reddit-side">
        <div className="community-card oj-side-card">
          <h3 style={{ margin: 0 }}>欢迎使用</h3>
          <p className="meta" style={{ marginTop: 8 }}>登录后可以提交代码、管理题目、查看评测状态。</p>
        </div>
      </aside>
    </div>
  );
}
