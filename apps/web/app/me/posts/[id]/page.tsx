"use client";

import { getMyPostDetail, updatePostMeta } from "@/lib/api";
import { getAccessToken } from "@/lib/auth";
import { toZhError } from "@/lib/errors";
import Link from "next/link";
import { useParams, useRouter } from "next/navigation";
import { FormEvent, KeyboardEvent, useEffect, useMemo, useState } from "react";

export default function PostMetaPage() {
  const router = useRouter();
  const params = useParams<{ id: string }>();

  const [title, setTitle] = useState("");
  const [summary, setSummary] = useState("");
  const [tagsText, setTagsText] = useState("");
  const [initialTitle, setInitialTitle] = useState("");
  const [initialSummary, setInitialSummary] = useState("");
  const [initialTagsText, setInitialTagsText] = useState("");
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");
  const [success, setSuccess] = useState("");

  const dirty = useMemo(
    () => title !== initialTitle || summary !== initialSummary || tagsText !== initialTagsText,
    [title, summary, tagsText, initialTitle, initialSummary, initialTagsText]
  );
  const titleValid = title.trim().length > 0;

  useEffect(() => {
    void init();
  }, [params?.id]);

  async function init() {
    setLoading(true);
    setError("");
    try {
      const token = getAccessToken();
      if (!token) {
        router.replace("/login?redirect=/me/posts");
        return;
      }
      const postID = Number(params?.id);
      if (!Number.isFinite(postID) || postID <= 0) {
        throw new Error("无效文章ID");
      }
      const post = await getMyPostDetail(postID, token);
      setTitle(post.title || "");
      setSummary(post.summary || "");
      setTagsText((post.tags || []).join(", "));
      setInitialTitle(post.title || "");
      setInitialSummary(post.summary || "");
      setInitialTagsText((post.tags || []).join(", "));
    } catch (err) {
      setError(toZhError(err, "加载失败"));
    } finally {
      setLoading(false);
    }
  }

  async function onSubmit(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    setError("");
    setSuccess("");
    setSaving(true);
    try {
      const token = getAccessToken();
      if (!token) {
        router.replace("/login?redirect=/me/posts");
        return;
      }
      const postID = Number(params?.id);
      if (!Number.isFinite(postID) || postID <= 0) {
        throw new Error("无效文章ID");
      }
      await updatePostMeta(token, postID, { title: title.trim(), summary, tags: parseTags(tagsText) });
      setInitialTitle(title);
      setInitialSummary(summary);
      setInitialTagsText(tagsText);
      setSuccess("文章信息已更新（会同步到所有草稿版本）");
    } catch (err) {
      setError(toZhError(err, "保存失败"));
    } finally {
      setSaving(false);
    }
  }

  useEffect(() => {
    function onBeforeUnload(e: BeforeUnloadEvent) {
      if (!dirty || saving) return;
      e.preventDefault();
      e.returnValue = "";
    }
    window.addEventListener("beforeunload", onBeforeUnload);
    return () => window.removeEventListener("beforeunload", onBeforeUnload);
  }, [dirty, saving]);

  function onMetaKeyDown(e: KeyboardEvent<HTMLFormElement>) {
    if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === "s") {
      e.preventDefault();
      if (!saving && dirty && titleValid) {
        void onSubmit(e as unknown as FormEvent<HTMLFormElement>);
      }
    }
  }

  return (
    <>
      <h1 style={{ marginTop: 0 }}>文章信息设置</h1>
      {loading ? <p className="tip">加载中...</p> : null}
      {loading ? null : (
        <div className="card">
          <form onSubmit={onSubmit} onKeyDown={onMetaKeyDown} className="meta-edit-form">
            <label className="meta-edit-label">
              标题
              <input
                className="meta-edit-title"
                placeholder="请输入标题"
                value={title}
                maxLength={255}
                onChange={(e) => setTitle(e.target.value)}
              />
              <span className="tip">标题 {title.length}/255</span>
            </label>
            <label className="meta-edit-label">
              正文摘要
              <textarea
                className="meta-edit-summary"
                placeholder="把它当成贴吧正文摘要去写，支持换行。"
                value={summary}
                maxLength={512}
                onChange={(e) => setSummary(e.target.value)}
              />
              <span className="tip">摘要 {summary.length}/512</span>
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
            <div className="meta-edit-actions">
              <button type="submit" disabled={saving || !dirty || !titleValid}>
                {saving ? "保存中..." : dirty ? "保存文章信息（Ctrl/Cmd+S）" : "暂无改动"}
              </button>
              {dirty ? <span className="tip">有未保存修改</span> : <span className="success">已同步</span>}
            </div>
            {!titleValid ? <p className="error">标题不能为空</p> : null}
          </form>
          <p className="tip">说明：标题和摘要是整篇文章统一字段，不区分草稿版本。</p>
          <div style={{ display: "flex", gap: 8, marginTop: 10, flexWrap: "wrap" }}>
            <Link href="/me/posts" className="ghost action-btn">
              返回我的文章
            </Link>
            <Link href={`/write/${params?.id || ""}`} className="ghost action-btn">
              回到草稿编辑
            </Link>
            {dirty ? (
              <button
                type="button"
                className="ghost action-btn"
                onClick={() => {
                  setTitle(initialTitle);
                  setSummary(initialSummary);
                  setTagsText(initialTagsText);
                  setError("");
                  setSuccess("");
                }}
              >
                撤销本次修改
              </button>
            ) : null}
          </div>
          {error ? <p className="error">{error}</p> : null}
          {success ? <p className="success">{success}</p> : null}
        </div>
      )}
    </>
  );
}

function parseTags(raw: string): string[] {
  const seen = new Set<string>();
  const out: string[] = [];
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
