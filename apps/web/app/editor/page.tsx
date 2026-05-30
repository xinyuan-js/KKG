"use client";

import { Nav } from "@/components/nav";
import { MarkdownEditor } from "@/components/markdown-editor";
import { createDraft, getCurrentUser, publishPost } from "@/lib/api";
import { getAccessToken } from "@/lib/auth";
import { toZhError } from "@/lib/errors";
import { ojBindQuestionSolution } from "@/lib/oj-api";
import { FormEvent, useEffect, useState } from "react";
import { usePathname, useRouter, useSearchParams } from "next/navigation";

export default function EditorPage() {
  const router = useRouter();
  const pathname = usePathname();
  const searchParams = useSearchParams();
  const [title, setTitle] = useState("");
  const [summary, setSummary] = useState("");
  const [tagsText, setTagsText] = useState("");
  const [draftNote, setDraftNote] = useState("");
  const [questionIdText, setQuestionIdText] = useState("");
  const [rawContent, setRawContent] = useState("");
  const [useMdEditor, setUseMdEditor] = useState(false);
  const [sideCollapsed, setSideCollapsed] = useState(false);
  const [error, setError] = useState("");
  const [success, setSuccess] = useState("");
  const [loading, setLoading] = useState(false);
  const [publishing, setPublishing] = useState(false);
  const [checking, setChecking] = useState(true);

  useEffect(() => {
    setSideCollapsed(useMdEditor);
  }, [useMdEditor]);

  useEffect(() => {
    void getCurrentUser()
      .then(() => setChecking(false))
      .catch(() => router.replace(`/login?redirect=${encodeURIComponent(pathname || "/write")}`));
  }, [router, pathname]);

  useEffect(() => {
    const presetTitle = (searchParams.get("title") || "").trim();
    const presetTags = (searchParams.get("tags") || "").trim();
    const presetSummary = (searchParams.get("summary") || "").trim();
    const presetQuestionId = (searchParams.get("questionId") || "").trim();
    if (presetTitle && !title) setTitle(presetTitle);
    if (presetTags && !tagsText) setTagsText(presetTags);
    if (presetSummary && !summary) setSummary(presetSummary);
    if (presetQuestionId && !questionIdText) setQuestionIdText(presetQuestionId);
  }, [searchParams, title, tagsText, summary, questionIdText]);

  async function submitCreate(directPublish: boolean) {
    setError("");
    setSuccess("");
    setLoading(!directPublish);
    setPublishing(directPublish);

    try {
      const token = getAccessToken();
      if (!token) {
        throw new Error("请先登录");
      }
      if (hasInlineBase64Image(rawContent)) {
        throw new Error("检测到图片仍是 base64，请等待上传完成后再保存");
      }

      const post = await createDraft({
        token,
        title,
        summary,
        tags: buildTags(tagsText, questionIdText),
        draft_note: draftNote,
        raw_content: rawContent
      });
      const qid = Number(questionIdText);
      const shouldBindQuestion = Number.isFinite(qid) && qid > 0;
      let bindWarn = "";
      if (shouldBindQuestion) {
        try {
          await ojBindQuestionSolution({ questionId: qid, postId: post.id });
        } catch (e) {
          bindWarn = `；题号 ${qid} 绑定失败：${toZhError(e, "请稍后在题目页重试绑定")}`;
        }
      }
      if (directPublish) {
        await publishPost(token, post.id);
        setSuccess(`已创建并直接发布${bindWarn}`);
        router.push(`/posts/${post.id}`);
      } else {
        setSuccess(`博客创建成功，正在进入草稿列表${bindWarn}`);
        router.push(`/write/${post.id}`);
      }
    } catch (err) {
      setError(toZhError(err, "创建失败"));
    } finally {
      setLoading(false);
      setPublishing(false);
    }
  }

  async function onSubmit(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    await submitCreate(false);
  }

  async function onDirectPublish() {
    await submitCreate(true);
  }

  return (
    <main className="page">
      <Nav />
      <header className="page-header">
        <h1 className="page-title">写作台</h1>
        <p className="tip">在这里完成草稿编写、发布设置与直发操作。</p>
      </header>
      {checking ? <p className="tip">校验登录状态中...</p> : null}
      {checking ? null : (
        <form className={`writer-shell${sideCollapsed ? " side-collapsed" : ""}`} onSubmit={onSubmit}>
          <section className="card writer-main">
            <label className="meta-edit-label">
              标题
              <input placeholder="输入标题" value={title} onChange={(e) => setTitle(e.target.value)} />
            </label>
            <label className="meta-edit-label">
              摘要
              <textarea
                className="meta-edit-summary writer-summary"
                placeholder="给读者一个清晰摘要（支持换行）"
                value={summary}
                onChange={(e) => setSummary(e.target.value)}
                maxLength={512}
              />
            </label>
            <label className="meta-edit-label">
              标签
              <input
                placeholder="多个标签用逗号分隔，例如：golang, 后端, es"
                value={tagsText}
                onChange={(e) => setTagsText(e.target.value)}
              />
              <span className="tip">最多 10 个标签，每个不超过 32 字符</span>
            </label>
            <div className="writer-editor-head">
              <strong>正文内容</strong>
              <div className="toolbar-row">
                <button type="button" className={useMdEditor ? "" : "ghost"} onClick={() => setUseMdEditor((v) => !v)}>
                  {useMdEditor ? "关闭 MD 编辑器" : "开启 MD 编辑器"}
                </button>
                <span className="meta">默认关闭，可按需启用</span>
              </div>
            </div>
            {useMdEditor ? (
              <MarkdownEditor value={rawContent} onChange={setRawContent} placeholder="Markdown 内容" />
            ) : (
              <textarea
                className="meta-edit-summary writer-content"
                placeholder="Markdown 内容（普通输入模式）"
                value={rawContent}
                onChange={(e) => setRawContent(e.target.value)}
              />
            )}
          </section>

          <aside className={`card writer-side${sideCollapsed ? " is-collapsed" : ""}`}>
            {sideCollapsed ? (
              <button
                type="button"
                className="writer-collapsed-trigger"
                onClick={() => setSideCollapsed(false)}
                aria-label="展开发布面板"
              >
                <span className="writer-collapsed-icon">⟪</span>
                <span>发布</span>
              </button>
            ) : (
              <>
            <div className="writer-side-head">
              <strong>发布面板</strong>
              <button type="button" className="ghost" onClick={() => setSideCollapsed(true)}>
                收起
              </button>
            </div>
            <label className="meta-edit-label">
              草稿注释
              <input
                placeholder="仅作者可见，用于记录版本说明"
                value={draftNote}
                onChange={(e) => setDraftNote(e.target.value)}
              />
            </label>
            <label className="meta-edit-label">
              关联题号（可选）
              <input
                type="number"
                placeholder="例如 1001，发布时自动绑定题解"
                value={questionIdText}
                onChange={(e) => setQuestionIdText(e.target.value)}
              />
            </label>
            <div className="writer-actions">
              <button type="submit" disabled={loading || publishing}>
                {loading ? "提交中..." : "创建草稿"}
              </button>
              <button type="button" onClick={onDirectPublish} disabled={loading || publishing}>
                {publishing ? "发布中..." : "创建并直接发布"}
              </button>
            </div>
            <p className="tip">提示：创建草稿后会进入该文章草稿列表。</p>
              </>
            )}
          </aside>

          {(error || success) ? (
            <section className="card writer-status">
              {error ? <p className="error">{error}</p> : null}
              {success ? <p className="success">{success}</p> : null}
            </section>
          ) : null}
        </form>
      )}
    </main>
  );
}

function hasInlineBase64Image(content: string): boolean {
  return /data:image\/[a-zA-Z0-9.+-]+;base64,/i.test(content);
}

function buildTags(raw: string, questionIdText: string): string[] {
  const seen = new Set<string>();
  const out: string[] = [];
  const qid = Number(questionIdText);
  if (Number.isFinite(qid) && qid > 0) {
    const qtag = `q${qid}`;
    seen.add(qtag);
    out.push(qtag);
  }
  raw
    .split(/[,，]/)
    .map((x) => x.trim())
    .filter(Boolean)
    .forEach((t) => {
      const tag = t.toLowerCase().slice(0, 32);
      if (!tag || seen.has(tag)) return;
      seen.add(tag);
      if (out.length < 10) out.push(tag);
    });
  return out;
}
