"use client";

import { Pager } from "@/components/pager";
import {
  adminCreateAudit,
  adminListAudits,
  adminListPosts,
  adminListUsers,
  adminUpdateUserRole,
  type AdminAuditItem,
  deletePost,
  type AdminUserItem,
  type Post
} from "@/lib/api";
import { getAccessToken, getUserProfile } from "@/lib/auth";
import { toZhError } from "@/lib/errors";
import {
  ojAdminListUsers,
  ojAdminUpdateUser,
  ojDeleteQuestion,
  ojListQuestionSubmits,
  ojListQuestions,
  type OJQuestionSubmitVO,
  type OJQuestionVO,
  type OJUserVO
} from "@/lib/oj-api";
import Link from "next/link";
import { useEffect, useMemo, useState } from "react";

type Tab = "users" | "posts" | "questions" | "submits" | "audits";

export default function AdminPage() {
  const [me, setMe] = useState<ReturnType<typeof getUserProfile>>(null);
  const [hydrated, setHydrated] = useState(false);
  const role = me?.role || "user";
  const isAdmin = role === "admin" || role === "super_admin";

  const [tab, setTab] = useState<Tab>("users");
  const [busy, setBusy] = useState(false);
  const [err, setErr] = useState("");
  const [ok, setOk] = useState("");

  const [users, setUsers] = useState<AdminUserItem[]>([]);
  const [usersPage, setUsersPage] = useState(1);
  const [usersTotal, setUsersTotal] = useState(0);
  const usersPageSize = 10;
  const [usersQ, setUsersQ] = useState("");
  const [usersRole, setUsersRole] = useState("");
  const [usersStatus, setUsersStatus] = useState("");

  const [posts, setPosts] = useState<Post[]>([]);
  const [postsPage, setPostsPage] = useState(1);
  const [postsTotal, setPostsTotal] = useState(0);
  const postsPageSize = 10;
  const [postsQ, setPostsQ] = useState("");
  const [postsStatus, setPostsStatus] = useState("");

  const [questions, setQuestions] = useState<OJQuestionVO[]>([]);
  const [qPage, setQPage] = useState(1);
  const [qTotal, setQTotal] = useState(0);

  const [submits, setSubmits] = useState<OJQuestionSubmitVO[]>([]);
  const [sPage, setSPage] = useState(1);
  const [sTotal, setSTotal] = useState(0);

  const [ojUsers, setOJUsers] = useState<OJUserVO[]>([]);
  const [audits, setAudits] = useState<AdminAuditItem[]>([]);
  const [aPage, setAPage] = useState(1);
  const [aTotal, setATotal] = useState(0);
  const [aAction, setAAction] = useState("");

  const usersPageCount = useMemo(() => Math.max(1, Math.ceil(usersTotal / usersPageSize)), [usersTotal]);
  const postsPageCount = useMemo(() => Math.max(1, Math.ceil(postsTotal / postsPageSize)), [postsTotal]);
  const qPageCount = useMemo(() => Math.max(1, Math.ceil(qTotal / 10)), [qTotal]);
  const sPageCount = useMemo(() => Math.max(1, Math.ceil(sTotal / 10)), [sTotal]);
  const aPageCount = useMemo(() => Math.max(1, Math.ceil(aTotal / 10)), [aTotal]);

  useEffect(() => {
    setMe(getUserProfile());
    setHydrated(true);
  }, []);

  useEffect(() => {
    if (!hydrated || !isAdmin) return;
    void reload();
  }, [hydrated, isAdmin, tab, usersPage, postsPage, qPage, sPage, aPage]); // eslint-disable-line react-hooks/exhaustive-deps

  async function writeAudit(action: string, targetType: string, targetID: number, detail: string) {
    try {
      const token = getAccessToken();
      if (!token) return;
      await adminCreateAudit({
        token,
        action,
        target_type: targetType,
        target_id: targetID,
        detail
      });
    } catch {
      // 管理动作不因日志失败而失败
    }
  }

  async function reload() {
    if (!isAdmin) return;
    setBusy(true);
    setErr("");
    try {
      const token = getAccessToken();
      if (!token) throw new Error("请先登录");
      if (tab === "users") {
        const [blogUsers, ojList] = await Promise.all([
          adminListUsers({
            token,
            page: usersPage,
            page_size: usersPageSize,
            q: usersQ,
            role: usersRole,
            status: usersStatus
          }),
          ojAdminListUsers({ current: 1, pageSize: 200 })
        ]);
        setUsers(blogUsers.items || []);
        setUsersTotal(blogUsers.total || 0);
        setOJUsers(ojList.records || []);
      } else if (tab === "posts") {
        const data = await adminListPosts({
          token,
          page: postsPage,
          page_size: postsPageSize,
          q: postsQ,
          status: postsStatus
        });
        setPosts(data.items || []);
        setPostsTotal(data.total || 0);
      } else if (tab === "questions") {
        const data = await ojListQuestions({ current: qPage, pageSize: 10 });
        setQuestions(data.records || []);
        setQTotal(data.total || 0);
      } else if (tab === "submits") {
        const data = await ojListQuestionSubmits({ current: sPage, pageSize: 10 });
        setSubmits(data.records || []);
        setSTotal(data.total || 0);
      } else {
        const data = await adminListAudits({ token, page: aPage, page_size: 10, action: aAction });
        setAudits(data.items || []);
        setATotal(data.total || 0);
      }
    } catch (e) {
      setErr(toZhError(e, "加载管理数据失败"));
    } finally {
      setBusy(false);
    }
  }

  if (!hydrated) {
    return (
      <section className="card">
        <h1 className="page-title">管理中心</h1>
        <p className="tip">加载中...</p>
      </section>
    );
  }

  if (!isAdmin) {
    return (
      <section className="card">
        <h1 className="page-title">管理中心</h1>
        <p className="error">当前账号无管理权限</p>
      </section>
    );
  }

  return (
    <section className="section-gap">
      <header className="page-header">
        <h1 className="page-title">管理中心</h1>
        <p className="tip">统一管理用户、推文、题目、提交记录与管理日志</p>
      </header>

      <div className="feed-tabs seg-switch seg-switch-5" data-active={tab === "users" ? 0 : tab === "posts" ? 1 : tab === "questions" ? 2 : tab === "submits" ? 3 : 4}>
        <span className="seg-switch-thumb" aria-hidden="true" />
        <button type="button" className={tab === "users" ? "" : "ghost"} onClick={() => setTab("users")}>用户</button>
        <button type="button" className={tab === "posts" ? "" : "ghost"} onClick={() => setTab("posts")}>推文</button>
        <button type="button" className={tab === "questions" ? "" : "ghost"} onClick={() => setTab("questions")}>题目</button>
        <button type="button" className={tab === "submits" ? "" : "ghost"} onClick={() => setTab("submits")}>提交</button>
        <button type="button" className={tab === "audits" ? "" : "ghost"} onClick={() => setTab("audits")}>日志</button>
      </div>

      {busy ? <p className="tip">加载中...</p> : null}
      {err ? <p className="error">{err}</p> : null}
      {ok ? <p className="success">{ok}</p> : null}

      {tab === "users" ? (
        <section className="card section-gap">
          <div className="toolbar-row">
            <input placeholder="用户名/邮箱搜索" value={usersQ} onChange={(e) => setUsersQ(e.target.value)} />
            <select value={usersRole} onChange={(e) => setUsersRole(e.target.value)}>
              <option value="">全部角色</option>
              <option value="user">user</option>
              <option value="admin">admin</option>
              <option value="super_admin">super_admin</option>
            </select>
            <select value={usersStatus} onChange={(e) => setUsersStatus(e.target.value)}>
              <option value="">全部状态</option>
              <option value="active">正常</option>
              <option value="disabled">已禁用</option>
              <option value="deleted">已删除</option>
            </select>
            <button type="button" onClick={() => void reload()}>查询</button>
          </div>
          {users.map((u) => {
            const ojUser = ojUsers.find((x) => x.userName === u.username);
            const isSelf = me?.id === u.id;
            return (
              <article key={u.id} className="tweet-card">
                <div className="tweet-card-content tweet-search-content">
                  <div className="tweet-head">
                    <strong>{u.username}</strong>
                    <span className="meta">{u.email}</span>
                  </div>
                  <div className="toolbar-row">
                    <span className="tweet-tag">Blog: {u.role}</span>
                    <span className="tweet-tag soft">OJ: {ojUser?.userRole || "未知"}</span>
                    <span className="tweet-tag soft">状态: {u.status === 1 ? "正常" : u.status === 0 ? "已禁用" : "已删除"}</span>
                  </div>
                  <div className="toolbar-row">
                    <select defaultValue={u.role} disabled={isSelf} onChange={async (e) => {
                      try {
                        const token = getAccessToken();
                        if (!token) throw new Error("请先登录");
                        const nextRole = e.target.value as "user" | "admin" | "super_admin";
                        await adminUpdateUserRole({ token, id: u.id, role: nextRole, status: u.status as -1 | 0 | 1 });
                        if (ojUser) await ojAdminUpdateUser({ id: ojUser.id, userRole: nextRole });
                        await writeAudit("user_role_update", "user", u.id, `${u.username}: ${u.role} -> ${nextRole}`);
                        setOk(`已更新 ${u.username} 角色为 ${nextRole}`);
                        void reload();
                      } catch (e2) {
                        setErr(toZhError(e2, "更新角色失败"));
                      }
                    }}>
                      <option value="user">user</option>
                      <option value="admin">admin</option>
                      <option value="super_admin">super_admin</option>
                    </select>
                    <button type="button" className="ghost" disabled={isSelf || u.status === -1} onClick={async () => {
                      try {
                        const token = getAccessToken();
                        if (!token) throw new Error("请先登录");
                        const ns = u.status === 0 ? 1 : 0;
                        await adminUpdateUserRole({ token, id: u.id, role: u.role, status: ns as -1 | 0 | 1 });
                        await writeAudit("user_disable_update", "user", u.id, `${u.username}: ${u.status} -> ${ns}`);
                        setOk(`已${ns === 1 ? "恢复启用" : "禁用"} ${u.username}`);
                        void reload();
                      } catch (e2) {
                        setErr(toZhError(e2, "更新禁用状态失败"));
                      }
                    }}>{u.status === 0 ? "启用" : "禁用"}</button>
                    <button type="button" className="ghost" disabled={isSelf} onClick={async () => {
                      try {
                        const token = getAccessToken();
                        if (!token) throw new Error("请先登录");
                        const ns = u.status === -1 ? 1 : -1;
                        await adminUpdateUserRole({ token, id: u.id, role: u.role, status: ns as -1 | 0 | 1 });
                        await writeAudit("user_delete_soft_update", "user", u.id, `${u.username}: ${u.status} -> ${ns}`);
                        setOk(`已${ns === 1 ? "恢复" : "删除"} ${u.username}`);
                        void reload();
                      } catch (e2) {
                        setErr(toZhError(e2, "更新删除状态失败"));
                      }
                    }}>{u.status === -1 ? "恢复" : "删除"}</button>
                  </div>
                </div>
              </article>
            );
          })}
          <Pager page={usersPage} total={usersPageCount} onChange={setUsersPage} />
        </section>
      ) : null}

      {tab === "posts" ? (
        <section className="card section-gap">
          <div className="toolbar-row">
            <input placeholder="标题/摘要搜索" value={postsQ} onChange={(e) => setPostsQ(e.target.value)} />
            <select value={postsStatus} onChange={(e) => setPostsStatus(e.target.value)}>
              <option value="">全部状态</option>
              <option value="published">published</option>
              <option value="draft">draft</option>
            </select>
            <button type="button" onClick={() => void reload()}>查询</button>
          </div>
          {posts.map((p) => (
            <article key={p.id} className="tweet-card">
                <div className="tweet-card-content tweet-search-content">
                <div className="tweet-head">
                  <Link href={`/posts/${p.id}`} className="tweet-link">{p.title}</Link>
                  <span className="meta">{p.author_name || `u/${p.author_id}`}</span>
                </div>
                <div className="toolbar-row">
                  <span className="tweet-tag">{p.status}</span>
                  <span className="tweet-tag soft">{p.slug}</span>
                  <button type="button" className="ghost" onClick={async () => {
                    if (!window.confirm(`确认隐藏推文 #${p.id} 吗？`)) return;
                    try {
                      const token = getAccessToken();
                      if (!token) throw new Error("请先登录");
                      await deletePost(token, p.id);
                      await writeAudit("post_hide", "post", p.id, `hide ${p.title}`);
                      setOk(`已隐藏推文 #${p.id}`);
                      void reload();
                    } catch (e) {
                      setErr(toZhError(e, "隐藏推文失败"));
                    }
                  }}>隐藏</button>
                </div>
              </div>
            </article>
          ))}
          <Pager page={postsPage} total={postsPageCount} onChange={setPostsPage} />
        </section>
      ) : null}

      {tab === "questions" ? (
        <section className="card section-gap">
          {questions.map((q) => (
            <article key={q.id} className="tweet-card">
                <div className="tweet-card-content tweet-search-content">
                <div className="tweet-head">
                  <Link href={`/oj/questions/${q.id}`} className="tweet-link">{q.title}</Link>
                  <span className="meta">通过 {q.acceptedNum}/{q.submitNum}</span>
                </div>
                <div className="toolbar-row">
                  <span className="tweet-tag soft">{(q.tags || []).join(", ") || "无标签"}</span>
                  <button type="button" className="ghost" onClick={async () => {
                    if (!window.confirm(`确认隐藏题目 #${q.id} 吗？`)) return;
                    try {
                      await ojDeleteQuestion(q.id);
                      await writeAudit("question_hide", "question", q.id, `hide ${q.title}`);
                      setOk(`已隐藏题目 #${q.id}`);
                      void reload();
                    } catch (e) {
                      setErr(toZhError(e, "隐藏题目失败"));
                    }
                  }}>隐藏</button>
                </div>
              </div>
            </article>
          ))}
          <Pager page={qPage} total={qPageCount} onChange={setQPage} />
        </section>
      ) : null}

      {tab === "submits" ? (
        <section className="card section-gap">
          {submits.map((s) => (
            <article key={s.id} className="tweet-card">
                <div className="tweet-card-content tweet-search-content">
                <div className="tweet-head">
                  <strong>提交 #{s.id}</strong>
                  <span className="meta">题目 #{s.questionId} | 用户 #{s.userId}</span>
                </div>
                <div className="toolbar-row">
                  <span className="tweet-tag">状态 {s.status}</span>
                  <span className="tweet-tag soft">{s.language}</span>
                  <span className="tweet-tag soft">{s.judgeInfo?.message || "处理中"}</span>
                  <Link href={`/oj/questions/${s.questionId}`} className="tweet-link">查看题目</Link>
                </div>
              </div>
            </article>
          ))}
          <Pager page={sPage} total={sPageCount} onChange={setSPage} />
        </section>
      ) : null}

      {tab === "audits" ? (
        <section className="card section-gap">
          <div className="toolbar-row">
            <input placeholder="按 action 过滤，如 user_delete" value={aAction} onChange={(e) => setAAction(e.target.value)} />
            <button type="button" onClick={() => void reload()}>查询</button>
          </div>
          {audits.map((a) => (
            <article key={a.id} className="tweet-card">
                <div className="tweet-card-content tweet-search-content">
                <div className="tweet-head">
                  <strong>{a.action}</strong>
                  <span className="meta">{a.actor_name || `#${a.actor_id}`} · {new Date(a.created_at).toLocaleString()}</span>
                </div>
                <div className="toolbar-row">
                  <span className="tweet-tag">{a.actor_role}</span>
                  <span className="tweet-tag soft">{a.target_type} #{a.target_id}</span>
                  {a.detail ? <span className="tweet-tag soft">{a.detail}</span> : null}
                </div>
              </div>
            </article>
          ))}
          <Pager page={aPage} total={aPageCount} onChange={setAPage} />
        </section>
      ) : null}
    </section>
  );
}
