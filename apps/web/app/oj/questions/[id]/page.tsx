"use client";

import {
  ojGenerateAgentSolution,
  ojEditQuestion,
  ojGetLoginUser,
  ojGetQuestion,
  ojGetQuestionVO,
  ojListAgentSolutionTasks,
  ojListQuestionSolutions,
  ojListQuestionSubmits,
  ojRunQuestion,
  ojSubmitQuestion,
  type OJAgentTask,
  type OJQuestionSolutionItem,
  type OJQuestionSubmitVO,
  type OJQuestionVO
} from "@/lib/oj-api";
import { toZhError } from "@/lib/errors";
import { emitTopNotice } from "@/lib/notice";
import { ChangeEvent, FormEvent, useEffect, useMemo, useRef, useState } from "react";
import { useParams } from "next/navigation";
import dynamic from "next/dynamic";
import Link from "next/link";

const MonacoEditor = dynamic(() => import("@monaco-editor/react"), { ssr: false });
let monacoInited = false;

export default function OJQuestionDetailPage() {
  const params = useParams<{ id: string }>();
  const id = Number(params?.id);
  const [question, setQuestion] = useState<OJQuestionVO | null>(null);
  const [meUserId, setMeUserId] = useState<number>(0);
  const [submits, setSubmits] = useState<OJQuestionSubmitVO[]>([]);
  const [selectedSubmit, setSelectedSubmit] = useState<OJQuestionSubmitVO | null>(null);
  const [solutions, setSolutions] = useState<OJQuestionSolutionItem[]>([]);
  const [solutionsError, setSolutionsError] = useState("");
  const [language, setLanguage] = useState("go");
  const [code, setCode] = useState("");
  const [canEdit, setCanEdit] = useState(false);
  const [isAdmin, setIsAdmin] = useState(false);
  const [editing, setEditing] = useState(false);
  const [editLoading, setEditLoading] = useState(false);
  const [editTitle, setEditTitle] = useState("");
  const [editContent, setEditContent] = useState("");
  const [editTagsText, setEditTagsText] = useState("");
  const [editAnswer, setEditAnswer] = useState("");
  const [editCases, setEditCases] = useState<Array<{ input: string; output: string }>>([{ input: "", output: "" }]);
  const [editSampleCases, setEditSampleCases] = useState<Array<{ input: string; output: string }>>([{ input: "", output: "" }]);
  const [editCaseFileHint, setEditCaseFileHint] = useState("");
  const [editTimeLimit, setEditTimeLimit] = useState(1000);
  const [editMemoryLimit, setEditMemoryLimit] = useState(128000);
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);
  const [runInput, setRunInput] = useState("");
  const [runLoading, setRunLoading] = useState(false);
  const [runOutput, setRunOutput] = useState("");
  const [runError, setRunError] = useState("");
  const [editorTheme, setEditorTheme] = useState<"vs" | "vs-dark">("vs");
  const [bottomTab, setBottomTab] = useState<"run" | "submit">("submit");
  const [drawerOpen, setDrawerOpen] = useState(false);
  const [detailOpen, setDetailOpen] = useState(false);
  const [autoRefreshSubmits, setAutoRefreshSubmits] = useState(false);
  const [submitCooldownLeft, setSubmitCooldownLeft] = useState(0);
  const [agentOpen, setAgentOpen] = useState(false);
  const [agentLoading, setAgentLoading] = useState(false);
  const [agentError, setAgentError] = useState("");
  const [agentTasks, setAgentTasks] = useState<OJAgentTask[]>([]);
  const submitStatusRef = useRef<Map<number, number>>(new Map());
  const trackedSubmitIDsRef = useRef<Set<number>>(new Set());

  useEffect(() => {
    if (!Number.isFinite(id) || id <= 0) return;
    void loadAll();
  }, [id]);

  useEffect(() => {
    if (!autoRefreshSubmits || !Number.isFinite(id) || id <= 0) return;
    const timer = window.setInterval(() => {
      void loadSubmits();
    }, 2000);
    return () => window.clearInterval(timer);
  }, [autoRefreshSubmits, id]);

  useEffect(() => {
    if (submitCooldownLeft <= 0) return;
    const timer = window.setInterval(() => {
      setSubmitCooldownLeft((v) => (v <= 1 ? 0 : v - 1));
    }, 1000);
    return () => window.clearInterval(timer);
  }, [submitCooldownLeft]);

  useEffect(() => {
    if (typeof document === "undefined") return;
    const sync = () => {
      const t = document.documentElement.getAttribute("data-theme");
      setEditorTheme(t === "dark" ? "vs-dark" : "vs");
    };
    sync();
    const obs = new MutationObserver(sync);
    obs.observe(document.documentElement, { attributes: true, attributeFilter: ["data-theme"] });
    return () => obs.disconnect();
  }, []);

  async function loadAll() {
    try {
      const q = await ojGetQuestionVO(id);
      setQuestion(q);
      await Promise.all([loadSolutions(1, 3), loadSubmits()]);
      try {
        const me = await ojGetLoginUser();
        setMeUserId(me.id || 0);
        const adminFlag = me.userRole === "admin" || me.userRole === "super_admin";
        setIsAdmin(adminFlag);
        setCanEdit(me.id === q.userId || adminFlag);
      } catch {
        setMeUserId(0);
        setIsAdmin(false);
        setCanEdit(false);
      }
      setError("");
    } catch (err) {
      setError(toZhError(err, "加载题目失败"));
    }
  }

  async function loadSolutions(current = 1, pageSize = 3) {
    try {
      const data = await ojListQuestionSolutions({ questionId: id, current, pageSize });
      setSolutions(data.records || []);
      setSolutionsError("");
    } catch (err) {
      setSolutions([]);
      setSolutionsError(toZhError(err, "加载题解列表失败"));
    }
  }

  async function loadSubmits() {
    const s = await ojListQuestionSubmits({ current: 1, pageSize: 20, questionId: id });
    const next = s.records || [];
    for (const it of next) {
      const prev = submitStatusRef.current.get(it.id);
      const now = it.status;
      const isTracked = trackedSubmitIDsRef.current.has(it.id);
      const wasPending = prev === 0 || prev === 1;
      const isFinal = now > 1;
      if (isTracked && wasPending && isFinal && typeof window !== "undefined") {
        window.dispatchEvent(new CustomEvent("oj:submission-final", {
          detail: {
            submitId: it.id,
            status: it.status,
            message: it.judgeInfo?.message || "",
            score: it.judgeInfo?.score ?? 0
          }
        }));
        trackedSubmitIDsRef.current.delete(it.id);
      }
      submitStatusRef.current.set(it.id, it.status);
    }
    setSubmits(next);
    if (next.length > 0 && (!selectedSubmit || !next.some((it) => it.id === selectedSubmit.id))) {
      setSelectedSubmit(next[0]);
    } else if (selectedSubmit) {
      const latestSelected = next.find((it) => it.id === selectedSubmit.id);
      if (latestSelected) setSelectedSubmit(latestSelected);
    }
    const hasPending = next.some((it) => it.status === 0 || it.status === 1);
    setAutoRefreshSubmits(hasPending);
  }

  async function onStartEdit() {
    setError("");
    try {
      const raw = await ojGetQuestion(id);
      setEditTitle(raw.title || "");
      setEditContent(raw.content || "");
      setEditTagsText((parseJSONString<string[]>(raw.tags, []) || []).join(", "));
      setEditAnswer(raw.answer || "");
      const sc = parseJSONString<Array<{ input: string; output: string }>>(raw.sampleCase, []);
      setEditSampleCases(sc.length > 0 ? sc : [{ input: "", output: "" }]);
      const jc = parseJSONString<Array<{ input: string; output: string }>>(raw.judgeCase, []);
      setEditCases(jc.length > 0 ? jc : [{ input: "", output: "" }]);
      const cfg = parseJSONString<{ timeLimit?: number; memoryLimit?: number }>(raw.judgeConfig, {});
      setEditTimeLimit(cfg.timeLimit || 1000);
      setEditMemoryLimit(cfg.memoryLimit || 128000);
      setEditing(true);
    } catch (err) {
      setError(toZhError(err, "加载可编辑题目数据失败"));
    }
  }

  async function onSubmitEdit(e: FormEvent) {
    e.preventDefault();
    setEditLoading(true);
    setError("");
    try {
      await ojEditQuestion({
        id,
        title: editTitle,
        content: editContent,
        tags: parseList(editTagsText),
        sampleCase: editSampleCases.filter((c) => c.input.trim() || c.output.trim()),
        answer: editAnswer,
        judgeCase: editCases.filter((c) => c.input.trim() || c.output.trim()),
        judgeConfig: { timeLimit: editTimeLimit, memoryLimit: editMemoryLimit }
      });
      emitTopNotice("题目已更新", "success");
      setEditing(false);
      await loadAll();
    } catch (err) {
      setError(toZhError(err, "更新题目失败"));
    } finally {
      setEditLoading(false);
    }
  }

  async function openAgentPanel() {
    setAgentOpen(true);
    setAgentError("");
    setAgentLoading(true);
    try {
      const data = await ojListAgentSolutionTasks({ current: 1, pageSize: 10, questionId: id });
      setAgentTasks(data.records || []);
    } catch (err) {
      setAgentTasks([]);
      setAgentError(toZhError(err, "加载 AI 任务失败"));
    } finally {
      setAgentLoading(false);
    }
  }

  async function onGenerateAgentSolution() {
    setAgentError("");
    const hasRunning = agentTasks.some((t) => t.status === "pending" || t.status === "running");
    if (hasRunning) {
      setAgentError("当前已有进行中的任务，请等待完成后再发起");
      return;
    }
    setAgentLoading(true);
    try {
      const ret = await ojGenerateAgentSolution(id);
      emitTopNotice(`AI 题解任务已创建 #${ret.taskId}`, "success");
      const data = await ojListAgentSolutionTasks({ current: 1, pageSize: 10, questionId: id });
      setAgentTasks(data.records || []);
    } catch (err) {
      setAgentError(toZhError(err, "发起 AI 题解任务失败"));
    } finally {
      setAgentLoading(false);
    }
  }

  async function onUploadEditCaseFiles(e: ChangeEvent<HTMLInputElement>) {
    const files = Array.from(e.target.files || []);
    if (files.length === 0) return;
    const ins = new Map<string, File>();
    const outs = new Map<string, File>();
    for (const f of files) {
      const name = f.name.toLowerCase();
      if (name.endsWith(".in")) {
        ins.set(name.slice(0, -3), f);
      } else if (name.endsWith(".out")) {
        outs.set(name.slice(0, -4), f);
      }
    }
    const keys = Array.from(ins.keys()).filter((k) => outs.has(k)).sort();
    if (keys.length === 0) {
      setEditCaseFileHint("未匹配到成对的 .in / .out 文件");
      return;
    }
    const built: Array<{ input: string; output: string }> = [];
    for (const k of keys) {
      const inText = await ins.get(k)!.text();
      const outText = await outs.get(k)!.text();
      built.push({ input: inText, output: outText });
    }
    setEditCases(built);
    setEditCaseFileHint(`已导入 ${built.length} 组测试用例`);
  }

  async function onSubmitCode(e?: FormEvent) {
    e?.preventDefault();
    if (submitCooldownLeft > 0) {
      setError(`提交过于频繁，请 ${submitCooldownLeft} 秒后重试`);
      return;
    }
    setLoading(true);
    setError("");
    try {
      const sid = await ojSubmitQuestion({ questionId: id, language, code });
      setSubmitCooldownLeft(5);
      emitTopNotice(`提交成功，记录 #${sid}`, "success");
      trackedSubmitIDsRef.current.add(sid);
      submitStatusRef.current.set(sid, 0);
      await loadSubmits();
      setAutoRefreshSubmits(true);
      setBottomTab("submit");
      setDrawerOpen(true);
    } catch (err) {
      const msg = toZhError(err, "提交失败");
      if (msg.includes("提交过于频繁") || msg.includes("5 秒")) {
        setSubmitCooldownLeft(5);
      }
      setError(msg);
    } finally {
      setLoading(false);
    }
  }

  async function onRunCode(e: FormEvent) {
    e.preventDefault();
    setRunLoading(true);
    setError("");
    setRunError("");
    setBottomTab("run");
    setDrawerOpen(true);
    try {
      const data = await ojRunQuestion({ questionId: id, language, code, input: runInput });
      setRunOutput(data.output ?? "");
      if (!data.output && data.judgeInfo?.message) {
        setRunError(`运行提示：${data.judgeInfo.message}`);
      }
    } catch (err) {
      const msg = toZhError(err, "运行失败");
      setRunError(msg);
      setRunOutput("");
    } finally {
      setRunLoading(false);
    }
  }

  async function onCopySubmitCode() {
    if (!selectedSubmit?.code) {
      emitTopNotice("当前提交代码不可见，无法复制", "error");
      return;
    }
    try {
      await navigator.clipboard.writeText(selectedSubmit.code);
      emitTopNotice("代码已复制到剪贴板", "success");
    } catch {
      emitTopNotice("复制失败，请检查浏览器权限", "error");
    }
  }

  function onUseSubmitCode() {
    if (!selectedSubmit?.code) {
      emitTopNotice("当前提交代码不可见，无法填入编辑器", "error");
      return;
    }
    setCode(selectedSubmit.code);
    setLanguage(selectedSubmit.language || language);
    setDrawerOpen(true);
    emitTopNotice("已将代码填入编辑器", "success");
  }

  async function onUnbindSolutionPost(postID: number) {
    setError("");
    try {
      // 前端入口已收敛到写作台自动绑定，这里保留移除能力给题主。
      const { ojUnbindQuestionSolution } = await import("@/lib/oj-api");
      await ojUnbindQuestionSolution({ questionId: id, postId: postID });
      await loadSolutions(1, 3);
      emitTopNotice("已移除题解绑定", "success");
    } catch (err) {
      setError(toZhError(err, "移除题解失败"));
    }
  }

  if (!Number.isFinite(id) || id <= 0) return <p className="error">无效题目 ID</p>;
  const hasPendingSubmit = useMemo(() => submits.some((s) => s.status === 0 || s.status === 1), [submits]);

  return (
    <div className={`oj-problem-page${drawerOpen ? " drawer-open" : ""}`}>
      <div className="oj-problem-layout">
        <section className="oj-problem-left section-gap">
        {error ? <p className="error">{error}</p> : null}
        {!question ? <p className="tip">加载中...</p> : (
          <article className="detail-card section-gap">
            <h1 className="detail-title">{question.title}</h1>
            <div className="tweet-meta-tags">{(question.tags || []).map((t) => <span key={`${question.id}-${t}`} className="tweet-tag soft">{t}</span>)}</div>
            <div className="detail-content-wrap"><pre className="detail-content" style={{ whiteSpace: "pre-wrap", margin: 0 }}>{question.content}</pre></div>
            <section className="section-gap">
              <h4 className="title" style={{ fontSize: 16, marginBottom: 0 }}>测试样例</h4>
              {(question.sampleCase || []).length === 0 ? <p className="meta">暂无样例</p> : null}
              {(question.sampleCase || []).map((c, idx) => (
                <div key={idx} className="oj-sample-item">
                  <p className="meta" style={{ margin: 0 }}>样例 {idx + 1}</p>
                  <pre>{c.input || "(空输入)"}</pre>
                  <pre>{c.output || "(空输出)"}</pre>
                </div>
              ))}
            </section>
            <section className="section-gap">
              <div className="toolbar-row">
                <h4 className="title" style={{ fontSize: 16, margin: 0 }}>题解文章</h4>
                <Link
                  className="tweet-link"
                  href={`/write?title=${encodeURIComponent(`${question?.title || "题目"} 题解`)}&tags=${encodeURIComponent(`题解,oj,${(question?.tags || []).join(",")}`)}&summary=${encodeURIComponent(`题目：${question?.title || ""}`)}&questionId=${id}`}
                  target="_blank"
                  rel="noreferrer"
                >
                  写题解
                </Link>
                <Link className="tweet-link" href={`/oj/questions/${id}/solutions`}>查看更多</Link>
              </div>
              {solutions.length === 0 ? <p className="meta">暂无已绑定题解文章</p> : null}
              {solutionsError ? <p className="meta">{solutionsError}</p> : null}
              {solutions.map((it) => (
                <article key={it.id} className="card" style={{ marginBottom: 0 }}>
                  {it.post?.title || it.post?.author_name || it.post?.authorName ? (
                    <div className="toolbar-row" style={{ justifyContent: "space-between", alignItems: "center" }}>
                      <Link
                        href={`/posts/${it.postId}`}
                        target="_blank"
                        rel="noreferrer"
                        style={{ display: "grid", gridTemplateColumns: "32px minmax(0,1fr)", gap: 10, alignItems: "center", minWidth: 0, flex: 1 }}
                      >
                        <img
                          src={it.post?.author_avatar_url || "/default-avatar.png"}
                          alt={it.post?.author_name || it.post?.authorName || "博客作者"}
                          className="nav-avatar"
                          style={{ width: 32, height: 32 }}
                        />
                        <div style={{ minWidth: 0 }}>
                          {it.post?.author_name || it.post?.authorName ? (
                            <p className="meta" style={{ margin: 0 }}>
                              {it.post?.author_name || it.post?.authorName}
                            </p>
                          ) : null}
                          {it.post?.title ? (
                            <p style={{ margin: 0, fontWeight: 600, whiteSpace: "nowrap", overflow: "hidden", textOverflow: "ellipsis" }}>
                              {it.post.title}
                            </p>
                          ) : null}
                        </div>
                      </Link>
                      {canEdit ? (
                        <button type="button" className="ghost" onClick={() => void onUnbindSolutionPost(it.postId)}>移除</button>
                      ) : null}
                    </div>
                  ) : (
                    <div className="toolbar-row" style={{ justifyContent: "space-between", alignItems: "center" }}>
                      <p className="meta" style={{ margin: 0 }}>文章暂不可用</p>
                      {canEdit ? (
                        <button type="button" className="ghost" onClick={() => void onUnbindSolutionPost(it.postId)}>移除</button>
                      ) : null}
                    </div>
                  )}
                  {it.post?.summary ? <p className="meta" style={{ margin: 0 }}>{it.post.summary}</p> : null}
                </article>
              ))}
            </section>
            <p className="meta">提交 {question.submitNum} · 通过 {question.acceptedNum}</p>
            {canEdit ? (
              <div className="toolbar-row">
                <button type="button" className="ghost" onClick={() => void onStartEdit()}>编辑题目</button>
                {isAdmin ? <button type="button" className="ghost" onClick={() => void openAgentPanel()}>AI题解任务</button> : null}
              </div>
            ) : null}
          </article>
        )}

        {editing ? (
          <div className="oj-modal-mask" onClick={() => setEditing(false)}>
          <article className="oj-modal" onClick={(e) => e.stopPropagation()}>
            <h3 className="title">修改题目</h3>
            <form onSubmit={onSubmitEdit}>
              <input placeholder="题目标题" value={editTitle} onChange={(e) => setEditTitle(e.target.value)} />
              <input placeholder="标签（逗号分隔）" value={editTagsText} onChange={(e) => setEditTagsText(e.target.value)} />
              <textarea className="meta-edit-summary" placeholder="题目内容" value={editContent} onChange={(e) => setEditContent(e.target.value)} />
              <textarea placeholder="参考答案（可选）" value={editAnswer} onChange={(e) => setEditAnswer(e.target.value)} />
            <div className="section-gap">
              <div className="toolbar-row">
                <h4 className="title" style={{ fontSize: 16, margin: 0 }}>公开样例</h4>
                <button type="button" className="ghost" onClick={() => setEditSampleCases((prev) => [...prev, { input: "", output: "" }])}>
                  新增样例
                </button>
              </div>
              {editSampleCases.map((c, idx) => (
                <div key={`s-${idx}`} className="card" style={{ marginBottom: 0 }}>
                  <p className="meta" style={{ marginTop: 0 }}>样例 {idx + 1}</p>
                  <textarea
                    placeholder="输入样例"
                    value={c.input}
                    onChange={(e) => setEditSampleCases((prev) => prev.map((it, i) => (i === idx ? { ...it, input: e.target.value } : it)))}
                    style={{ minHeight: 90 }}
                  />
                  <textarea
                    placeholder="输出样例"
                    value={c.output}
                    onChange={(e) => setEditSampleCases((prev) => prev.map((it, i) => (i === idx ? { ...it, output: e.target.value } : it)))}
                    style={{ minHeight: 90 }}
                  />
                  {editSampleCases.length > 1 ? (
                    <button type="button" className="ghost" onClick={() => setEditSampleCases((prev) => prev.filter((_, i) => i !== idx))}>
                      删除样例
                    </button>
                  ) : null}
                </div>
              ))}
            </div>

            <div className="section-gap">
                <div className="toolbar-row">
                  <h4 className="title" style={{ fontSize: 16, margin: 0 }}>隐藏测试用例</h4>
                  <button type="button" className="ghost" onClick={() => setEditCases((prev) => [...prev, { input: "", output: "" }])}>
                    新增用例
                  </button>
                  <label className="tweet-link" style={{ cursor: "pointer" }}>
                    导入 .in/.out
                    <input
                      type="file"
                      accept=".in,.out,text/plain"
                      multiple
                      onChange={onUploadEditCaseFiles}
                      style={{ display: "none" }}
                    />
                  </label>
                </div>
                {editCaseFileHint ? <p className="meta">{editCaseFileHint}</p> : null}
                {editCases.map((c, idx) => (
                  <div key={idx} className="card" style={{ marginBottom: 0 }}>
                    <p className="meta" style={{ marginTop: 0 }}>用例 {idx + 1}</p>
                    <textarea
                      placeholder="输入用例"
                      value={c.input}
                      onChange={(e) => setEditCases((prev) => prev.map((it, i) => (i === idx ? { ...it, input: e.target.value } : it)))}
                      style={{ minHeight: 90 }}
                    />
                    <textarea
                      placeholder="输出用例"
                      value={c.output}
                      onChange={(e) => setEditCases((prev) => prev.map((it, i) => (i === idx ? { ...it, output: e.target.value } : it)))}
                      style={{ minHeight: 90 }}
                    />
                    {editCases.length > 1 ? (
                      <button type="button" className="ghost" onClick={() => setEditCases((prev) => prev.filter((_, i) => i !== idx))}>
                        删除样例
                      </button>
                    ) : null}
                  </div>
                ))}
              </div>
              <div className="oj-grid-2">
                <input type="number" placeholder="时间限制(ms)" value={editTimeLimit} onChange={(e) => setEditTimeLimit(Number(e.target.value || 0))} />
                <input type="number" placeholder="内存限制(kb)" value={editMemoryLimit} onChange={(e) => setEditMemoryLimit(Number(e.target.value || 0))} />
              </div>
              <div className="toolbar-row">
                <button type="submit" disabled={editLoading}>{editLoading ? "保存中..." : "保存修改"}</button>
                <button type="button" className="ghost" onClick={() => setEditing(false)}>取消</button>
              </div>
            </form>
          </article>
          </div>
        ) : null}

        {agentOpen ? (
          <div className="oj-modal-mask" onClick={() => setAgentOpen(false)}>
            <article className="oj-modal" onClick={(e) => e.stopPropagation()}>
              <h3 className="title">AI 题解任务</h3>
              <p className="meta">该操作会创建异步任务，并不是直接生成。若已有进行中任务，会禁止重复发起。</p>
              {agentError ? <p className="error">{agentError}</p> : null}
              <div className="toolbar-row">
                <button type="button" onClick={() => void onGenerateAgentSolution()} disabled={agentLoading}>
                  {agentLoading ? "处理中..." : "确认发起任务"}
                </button>
                <button type="button" className="ghost" onClick={() => setAgentOpen(false)} disabled={agentLoading}>关闭</button>
              </div>
              <div className="section-gap">
                <h4 className="title" style={{ fontSize: 15, margin: 0 }}>本题最近任务</h4>
                {agentTasks.length === 0 ? <p className="meta">暂无任务</p> : null}
                {agentTasks.map((t) => (
                  <div key={t.id} className="card" style={{ marginBottom: 0 }}>
                    <div className="toolbar-row" style={{ justifyContent: "space-between" }}>
                      <p className="meta" style={{ margin: 0 }}>任务 #{t.id}</p>
                      <span className="tweet-tag soft">{t.status}</span>
                    </div>
                    {t.blogPostId ? <p className="meta" style={{ margin: 0 }}>已发布博客ID：{t.blogPostId}</p> : null}
                    {t.lastError ? <p className="meta" style={{ margin: 0 }}>错误：{t.lastError}</p> : null}
                  </div>
                ))}
              </div>
            </article>
          </div>
        ) : null}

        </section>
        <aside className="oj-problem-right section-gap">
        <article className="card oj-side-card" style={{ marginBottom: 0 }}>
          <h3 className="title">代码编辑器</h3>
          <form onSubmit={onSubmitCode} className="section-gap">
            <select className="oj-select" value={language} onChange={(e) => setLanguage(e.target.value)}>
              <option value="go">go</option>
            </select>
            <div className="oj-code-editor-wrap">
              <MonacoEditor
                height="420px"
                language={toMonacoLanguage(language)}
                theme={editorTheme}
                value={code}
                onChange={(v) => setCode(v || "")}
                beforeMount={setupMonaco}
                options={{
                  minimap: { enabled: false },
                  fontSize: 14,
                  tabSize: 2,
                  automaticLayout: true,
                  scrollBeyondLastLine: false,
                  wordWrap: "on",
                  lineNumbers: "on",
                  quickSuggestions: { other: true, comments: false, strings: true },
                  suggestOnTriggerCharacters: true,
                  acceptSuggestionOnEnter: "on",
                  tabCompletion: "on",
                  snippetSuggestions: "top",
                  autoClosingBrackets: "always",
                  autoClosingQuotes: "always",
                  formatOnPaste: true
                }}
              />
            </div>
            <p className="meta" style={{ margin: 0 }}>提交按钮已移到底边栏工作区。</p>
          </form>
        </article>

        </aside>
      </div>

      <section className={`oj-bottom-drawer${drawerOpen ? " open" : ""}`}>
        <button type="button" className="oj-bottom-drawer-handle" onClick={() => setDrawerOpen((v) => !v)}>
          <strong>底边栏工作区</strong>
          <span className="oj-drawer-toggle-icon" aria-hidden="true">{drawerOpen ? "▾" : "▴"}</span>
        </button>
        <div className="oj-bottom-drawer-body">
          <div className="oj-drawer-submit-row">
            <button
              type="button"
              className="oj-cta"
              onClick={() => void onSubmitCode()}
              disabled={loading || !code.trim() || submitCooldownLeft > 0}
            >
              {loading ? "提交中..." : submitCooldownLeft > 0 ? `请等待 ${submitCooldownLeft}s` : "提交评测"}
            </button>
            <span className="meta">提交后自动跳转到提交记录</span>
          </div>
          <div className="feed-tabs seg-switch seg-switch-2 oj-workbench-tabs" data-active={bottomTab === "run" ? 0 : 1}>
            <span className="seg-switch-thumb" aria-hidden="true" />
            <button type="button" className={`seg-trigger${bottomTab === "run" ? " active" : ""}`} onClick={() => setBottomTab("run")}>
              自测运行
            </button>
            <button type="button" className={`seg-trigger${bottomTab === "submit" ? " active" : ""}`} onClick={() => setBottomTab("submit")}>
              提交记录
            </button>
          </div>
          {bottomTab === "run" ? (
            <div className="section-gap">
              <div className="oj-run-grid">
                <form onSubmit={onRunCode} className="oj-run-form">
                  <div className="oj-run-head">
                    <p className="meta" style={{ margin: 0 }}>输入</p>
                    <button
                      type="button"
                      className="text-action"
                      disabled={!question?.sampleCase?.length}
                      onClick={() => setRunInput(question?.sampleCase?.[0]?.input || "")}
                    >
                      载入样例输入
                    </button>
                  </div>
                  <textarea
                    className="meta-edit-summary"
                    placeholder="输入自定义测试数据，例如：1 2"
                    value={runInput}
                    onChange={(e) => setRunInput(e.target.value)}
                    style={{ minHeight: 160 }}
                  />
                  <button type="submit" className="oj-cta" disabled={runLoading || !code.trim()}>
                    {runLoading ? "运行中..." : "运行代码"}
                  </button>
                </form>
                <div className="oj-run-output">
                  <p className="meta" style={{ margin: 0 }}>输出</p>
                  {runError ? <p className="error" style={{ marginTop: 0 }}>{runError}</p> : null}
                  <pre className="detail-content oj-run-output-pre" style={{ whiteSpace: "pre-wrap", margin: 0 }}>
                    {runOutput || "(暂无输出)"}
                  </pre>
                </div>
              </div>
            </div>
          ) : (
            <div className="oj-submit-list">
              {submits.length === 0 ? <p className="meta" style={{ margin: 0 }}>暂无提交记录</p> : null}
              {submits.map((s) => (
                <button
                  type="button"
                  key={s.id}
                  className={`oj-submit-item${selectedSubmit?.id === s.id ? " active" : ""}`}
                  onClick={() => {
                    setSelectedSubmit(s);
                    setDetailOpen(true);
                  }}
                >
                  <span>#{s.id}</span>
                  <span className={`oj-status ${statusClass(s.status, s.judgeInfo?.message || "")}`}>{statusText(s.status, s.judgeInfo?.message || "")}</span>
                  <span>{s.judgeInfo?.score ?? 0} 分</span>
                  <span>{new Date(s.createTime || "").toLocaleTimeString()}</span>
                </button>
              ))}
            </div>
          )}
        </div>
      </section>

      {detailOpen && selectedSubmit ? (
        <div className="oj-modal-mask" onClick={() => setDetailOpen(false)}>
          <div className="oj-modal" onClick={(e) => e.stopPropagation()}>
            <div className="oj-modal-head">
              <h4 className="title" style={{ fontSize: 16, margin: 0 }}>提交详情 #{selectedSubmit.id}</h4>
              <div className="toolbar-row oj-inline-actions" style={{ justifyContent: "flex-end" }}>
                <button type="button" className="text-action" onClick={onUseSubmitCode}>填入编辑器</button>
                <button type="button" className="text-action danger" onClick={() => setDetailOpen(false)}>关闭</button>
              </div>
            </div>
            <div className="oj-detail-grid">
              <div className="oj-detail-kv">
                <span className="meta">提交者</span>
                <strong>{selectedSubmit.userId === meUserId && meUserId > 0 ? "我" : `用户 #${selectedSubmit.userId}`}</strong>
              </div>
              <div className="oj-detail-kv">
                <span className="meta">语言</span>
                <strong>{selectedSubmit.language}</strong>
              </div>
              <div className="oj-detail-kv">
                <span className="meta">状态</span>
                <span className={`oj-status ${statusClass(selectedSubmit.status, selectedSubmit.judgeInfo?.message || "")}`}>
                  {statusText(selectedSubmit.status, selectedSubmit.judgeInfo?.message || "")}
                </span>
              </div>
              <div className="oj-detail-kv">
                <span className="meta">得分</span>
                <strong>{selectedSubmit.judgeInfo?.score ?? 0} 分</strong>
              </div>
              <div className="oj-detail-kv">
                <span className="meta">耗时</span>
                <strong>{selectedSubmit.judgeInfo?.time ?? "-"} ms</strong>
              </div>
              <div className="oj-detail-kv">
                <span className="meta">内存</span>
                <strong>{selectedSubmit.judgeInfo?.memory ?? "-"} KB</strong>
              </div>
              <div className="oj-detail-kv oj-detail-kv-wide">
                <span className="meta">判题信息</span>
                <strong>{selectedSubmit.judgeInfo?.message || "-"}</strong>
              </div>
            </div>
            <section className="oj-code-panel">
              <div className="oj-code-panel-head">
                <strong>提交代码</strong>
                <button type="button" className="icon-action" aria-label="复制代码" onClick={onCopySubmitCode} title="复制代码">
                  <svg viewBox="0 0 24 24" width="15" height="15" aria-hidden="true">
                    <rect x="9" y="9" width="11" height="11" rx="2" fill="none" stroke="currentColor" strokeWidth="1.8" />
                    <rect x="4" y="4" width="11" height="11" rx="2" fill="none" stroke="currentColor" strokeWidth="1.8" />
                  </svg>
                </button>
              </div>
              <pre className="detail-content oj-code-block" style={{ whiteSpace: "pre-wrap", margin: 0, maxHeight: 320, overflow: "auto" }}>
                {selectedSubmit.code || "该提交代码对当前账号不可见"}
              </pre>
            </section>
          </div>
        </div>
      ) : null}
    </div>
  );
}

function parseList(raw: string) {
  return raw.split(/[,，]/).map((x) => x.trim()).filter(Boolean);
}

function parseJSONString<T>(raw: string, fallback: T): T {
  try {
    return (JSON.parse(raw) as T) || fallback;
  } catch {
    return fallback;
  }
}

function statusText(status: number, message: string) {
  if (status === 0) return "Pending";
  if (status === 1) return "Judging";
  if (status === 2) return "AC";
  const m = (message || "").toLowerCase();
  if (m.includes("wrong answer")) return "WA";
  if (m.includes("compile")) return "CE";
  if (m.includes("runtime")) return "RE";
  if (m.includes("time")) return "TLE";
  if (m.includes("memory")) return "MLE";
  return "Failed";
}

function statusClass(status: number, message: string) {
  if (status === 0 || status === 1) return "pending";
  if (status === 2) return "ac";
  const m = (message || "").toLowerCase();
  if (m.includes("wrong answer")) return "wa";
  if (m.includes("compile")) return "ce";
  if (m.includes("runtime")) return "re";
  if (m.includes("time")) return "tle";
  return "fail";
}

function toMonacoLanguage(lang: string) {
  switch (lang) {
    case "cpp":
      return "cpp";
    case "c":
      return "c";
    case "java":
      return "java";
    case "python":
      return "python";
    case "javascript":
      return "javascript";
    default:
      return "go";
  }
}

function snippet(label: string, insertText: string, documentation: string) {
  return { label, insertText, documentation };
}

function registerLang(
  monaco: any,
  language: string,
  snippets: Array<{ label: string; insertText: string; documentation: string }>,
  keywords: string[] = [],
  funcs: string[] = []
) {
  monaco.languages.registerCompletionItemProvider(language, {
    triggerCharacters: [".", "(", "_"],
    provideCompletionItems: () => {
      const snippetSuggestions = snippets.map((s) => ({
        label: s.label,
        kind: monaco.languages.CompletionItemKind.Snippet,
        insertText: s.insertText,
        insertTextRules: monaco.languages.CompletionItemInsertTextRule.InsertAsSnippet,
        documentation: s.documentation
      }));
      const keywordSuggestions = keywords.map((k) => ({
        label: k,
        kind: monaco.languages.CompletionItemKind.Keyword,
        insertText: k,
        documentation: `关键字: ${k}`
      }));
      const functionSuggestions = funcs.map((f) => ({
        label: f,
        kind: monaco.languages.CompletionItemKind.Function,
        insertText: f,
        documentation: `常用函数: ${f}`
      }));
      return { suggestions: [...snippetSuggestions, ...keywordSuggestions, ...functionSuggestions] };
    }
  });
}

function setupMonaco(monaco: any) {
  if (monacoInited) return;
  monacoInited = true;
  registerLang(monaco, "go", [
    snippet("package_main", "package main\n\nimport \"fmt\"\n\nfunc main() {\n\t$0\n}\n", "Go main 模板"),
    snippet("for_loop", "for ${1:i} := 0; ${1:i} < ${2:n}; ${1:i}++ {\n\t$0\n}\n", "for 循环"),
    snippet("if_err", "if err != nil {\n\t$0\n}\n", "if err != nil"),
    snippet("var_int", "var ${1:n} int\n$0", "声明 int 变量"),
    snippet("var_string", "var ${1:s} string\n$0", "声明 string 变量"),
    snippet("short_decl", "${1:x} := ${2:0}\n$0", "短变量声明"),
    snippet("const_decl", "const ${1:name} = ${2:value}\n$0", "常量声明"),
    snippet("map_make", "${1:m} := make(map[${2:string}]${3:int})\n$0", "创建 map"),
    snippet("slice_make", "${1:s} := make([]${2:int}, ${3:0})\n$0", "创建切片"),
    snippet("fmt_printf", "fmt.Printf(\"${1:format}\\n\", ${2:args})\n$0", "格式化输出"),
    snippet("fmt_scan", "fmt.Scan(&${1:a}, &${2:b})\n$0", "标准输入"),
    snippet("switch_expr", "switch ${1:expr} {\ncase ${2:value}:\n\t$0\ndefault:\n}\n", "switch 语句")
  ], [
    "break", "default", "func", "interface", "select",
    "case", "defer", "go", "map", "struct",
    "chan", "else", "goto", "package", "switch",
    "const", "fallthrough", "if", "range", "type",
    "continue", "for", "import", "return", "var",
    "make", "len", "cap", "append", "copy", "delete", "close", "new", "panic", "recover"
  ], [
    "fmt.Printf", "fmt.Println", "fmt.Print", "fmt.Scan", "fmt.Scanf",
    "strings.TrimSpace", "strings.Split", "strings.Contains",
    "sort.Ints", "sort.Strings", "time.Now", "time.Since", "errors.New"
  ]);
  registerLang(monaco, "cpp", [
    snippet("cpp_main", "#include <bits/stdc++.h>\nusing namespace std;\n\nint main() {\n    $0\n    return 0;\n}\n", "C++ main 模板"),
    snippet("for_loop", "for (int ${1:i} = 0; ${1:i} < ${2:n}; ${1:i}++) {\n    $0\n}\n", "for 循环")
  ]);
  registerLang(monaco, "java", [
    snippet("java_main", "import java.util.*;\n\npublic class Main {\n    public static void main(String[] args) {\n        $0\n    }\n}\n", "Java main 模板")
  ]);
  registerLang(monaco, "python", [
    snippet("py_main", "def main():\n    $0\n\nif __name__ == '__main__':\n    main()\n", "Python main 模板"),
    snippet("for_range", "for ${1:i} in range(${2:n}):\n    $0\n", "for range")
  ]);
  registerLang(monaco, "javascript", [
    snippet("js_main", "function main() {\n  $0\n}\n\nmain();\n", "JS main 模板"),
    snippet("for_loop", "for (let ${1:i} = 0; ${1:i} < ${2:n}; ${1:i}++) {\n  $0\n}\n", "for 循环")
  ]);
}
