"use client";

import { ojRegister } from "@/lib/oj-api";
import { toZhError } from "@/lib/errors";
import { emitTopNotice } from "@/lib/notice";
import { register } from "@/lib/api";
import { syncDualLogin } from "@/lib/session-bridge";
import Link from "next/link";
import { FormEvent, useState } from "react";
import { useRouter } from "next/navigation";

export default function OJRegisterPage() {
  const router = useRouter();
  const [userAccount, setUserAccount] = useState("");
  const [userPassword, setUserPassword] = useState("");
  const [checkPassword, setCheckPassword] = useState("");
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);

  async function onSubmit(e: FormEvent) {
    e.preventDefault();
    setError("");
    setLoading(true);
    try {
      await ojRegister(userAccount, userPassword, checkPassword);
      try {
        await register({ username: userAccount, email: `${userAccount}@kkg.local`, password: userPassword });
      } catch {
        // ignore existing-account
      }
      await syncDualLogin(userAccount, userPassword);
      emitTopNotice("注册并登录成功", "success");
      router.push("/oj");
    } catch (err) {
      setError(toZhError(err, "注册失败"));
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="reddit-layout">
      <section className="reddit-main">
        <section className="card oj-auth-card oj-side-card">
          <h1 className="title">OJ 注册</h1>
          <form onSubmit={onSubmit}>
            <input placeholder="账号" value={userAccount} onChange={(e) => setUserAccount(e.target.value)} />
            <input type="password" placeholder="密码" value={userPassword} onChange={(e) => setUserPassword(e.target.value)} />
            <input type="password" placeholder="确认密码" value={checkPassword} onChange={(e) => setCheckPassword(e.target.value)} />
            <button type="submit" disabled={loading}>{loading ? "提交中..." : "注册并登录"}</button>
          </form>
          <p className="tip" style={{ marginTop: 10 }}>已有账号？<Link href="/oj/login">去登录</Link></p>
          {error ? <p className="error">{error}</p> : null}
        </section>
      </section>
      <aside className="reddit-side">
        <div className="community-card oj-side-card">
          <h3 style={{ margin: 0 }}>账号说明</h3>
          <p className="meta" style={{ marginTop: 8 }}>注册后会自动登录并进入 OJ 首页。</p>
        </div>
      </aside>
    </div>
  );
}
