"use client";

import { Nav } from "@/components/nav";
import { MarkdownEditor } from "@/components/markdown-editor";
import {
  createPostDraft,
  deletePostDraft,
  getPostDraft,
  getPostDrafts,
  getMyPostDetail,
  publishPostDraft,
  savePostDraft,
  unpublishPost,
  type Post,
  type PostVersion
} from "@/lib/api";
import { getAccessToken } from "@/lib/auth";
import { toZhError } from "@/lib/errors";
import Link from "next/link";
import { useParams, usePathname, useRouter } from "next/navigation";
import { FormEvent, useEffect, useState } from "react";

export default function WriteByIDPage() {
  const router = useRouter();
  const pathname = usePathname();
  const params = useParams<{ id: string }>();

  const [post, setPost] = useState<Post | null>(null);
  const [drafts, setDrafts] = useState<PostVersion[]>([]);
  const [selectedVersion, setSelectedVersion] = useState(0);

  const [draftNote, setDraftNote] = useState("");
  const [rawContent, setRawContent] = useState("");
  const [useMdEditor, setUseMdEditor] = useState(false);

  const [checking, setChecking] = useState(true);
  const [loading, setLoading] = useState(false);
  const [loadingDraft, setLoadingDraft] = useState(false);
  const [error, setError] = useState("");
  const [success, setSuccess] = useState("");

  useEffect(() => {
    const token = getAccessToken();
    if (!token) {
      router.replace(`/login?redirect=${encodeURIComponent(pathname || "/write")}`);
      return;
    }
    setChecking(false);
    void initialize(token);
  }, [router, pathname, params?.id]);

  async function initialize(token: string) {
    setError("");
    try {
      const postID = Number(params?.id);
      if (!Number.isFinite(postID) || postID <= 0) {
        throw new Error("无效文章ID");
      }
      const detail = await getMyPostDetail(postID, token);
      setPost(detail);
      const data = await getPostDrafts(token, postID);
      setDrafts(data);
      let nextSelected = detail.published_version || 0;
      if (!nextSelected && data.length > 0) {
        nextSelected = data[0].version;
      }
      if (nextSelected > 0) {
        await loadDraft(token, postID, nextSelected);
      }
    } catch (err) {
      setError(toZhError(err, "加载失败"));
    }
  }

  async function loadDraft(token: string, postID: number, version: number) {
    setLoadingDraft(true);
    try {
      const draft = await getPostDraft(token, postID, version);
      setSelectedVersion(version);
      setDraftNote(draft.draft_note || "");
      setRawContent(draft.raw_content || "");
      setError("");
    } catch (err) {
      setError(toZhError(err, "加载草稿失败"));
    } finally {
      setLoadingDraft(false);
    }
  }

  async function onCreateDraft() {
    setError("");
    setSuccess("");
    try {
      const token = getAccessToken();
      if (!token) {
        router.replace(`/login?redirect=${encodeURIComponent(pathname || "/write")}`);
        return;
      }
      const postID = Number(params?.id);
      if (!Number.isFinite(postID) || postID <= 0) {
        throw new Error("无效文章ID");
      }
      const draft = await createPostDraft(token, postID, selectedVersion || undefined, "");
      const data = await getPostDrafts(token, postID);
      setDrafts(data);
      await loadDraft(token, postID, draft.version);
      setSuccess(`已新建草稿 v${draft.version}`);
    } catch (err) {
      setError(toZhError(err, "新建草稿失败"));
    }
  }

  async function onSubmit(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    setError("");
    setSuccess("");
    setLoading(true);

    try {
      const token = getAccessToken();
      if (!token) {
        throw new Error("未找到 access_token，请先登录");
      }
      const postID = Number(params?.id);
      if (!Number.isFinite(postID) || postID <= 0) {
        throw new Error("无效文章ID");
      }
      if (selectedVersion <= 0) {
        throw new Error("请先选择一个草稿");
      }
      if (hasInlineBase64Image(rawContent)) {
        throw new Error("检测到图片仍是 base64，请等待上传完成后再保存");
      }

      const draft = await saveCurrentVersion(token, postID, selectedVersion);
      setDraftNote(draft.draft_note || "");
      setRawContent(draft.raw_content || "");
      setSuccess(`草稿 v${selectedVersion} 已保存`);

      const data = await getPostDrafts(token, postID);
      setDrafts(data);
    } catch (err) {
      setError(toZhError(err, "保存失败"));
    } finally {
      setLoading(false);
    }
  }

  async function saveCurrentVersion(token: string, postID: number, version: number) {
    return savePostDraft({
      token,
      postID,
      version,
      title: post?.title || "",
      summary: post?.summary || "",
      draft_note: draftNote,
      raw_content: rawContent
    });
  }

  async function onSaveAndPublish() {
    setError("");
    setSuccess("");
    setLoading(true);
    try {
      const token = getAccessToken();
      if (!token) {
        router.replace(`/login?redirect=${encodeURIComponent(pathname || "/write")}`);
        return;
      }
      const postID = Number(params?.id);
      if (!Number.isFinite(postID) || postID <= 0) {
        throw new Error("无效文章ID");
      }
      if (selectedVersion <= 0) {
        throw new Error("请先选择一个草稿");
      }
      if (hasInlineBase64Image(rawContent)) {
        throw new Error("检测到图片仍是 base64，请等待上传完成后再发布");
      }

      const draft = await saveCurrentVersion(token, postID, selectedVersion);
      setDraftNote(draft.draft_note || "");
      setRawContent(draft.raw_content || "");

      const nextPost = await publishPostDraft(token, postID, selectedVersion);
      setPost(nextPost);
      const data = await getPostDrafts(token, postID);
      setDrafts(data);
      setSuccess(`已保存并发布 v${selectedVersion}`);
    } catch (err) {
      setError(toZhError(err, "发布失败"));
    } finally {
      setLoading(false);
    }
  }

  async function onPublishDraft(version: number) {
    setError("");
    setSuccess("");
    if (!window.confirm(`确认将草稿 v${version} 设为对外正式版本吗？`)) {
      return;
    }
    try {
      const token = getAccessToken();
      if (!token) {
        router.replace(`/login?redirect=${encodeURIComponent(pathname || "/write")}`);
        return;
      }
      const postID = Number(params?.id);
      if (!Number.isFinite(postID) || postID <= 0) {
        throw new Error("无效文章ID");
      }
      const nextPost = await publishPostDraft(token, postID, version);
      setPost(nextPost);
      const data = await getPostDrafts(token, postID);
      setDrafts(data);
      setSuccess(`已发布草稿 v${version}`);
    } catch (err) {
      setError(toZhError(err, "发布失败"));
    }
  }

  async function onDeleteDraft(version: number) {
    setError("");
    setSuccess("");
    if (!window.confirm(`确认删除草稿 v${version} 吗？删除后不可恢复。`)) {
      return;
    }
    try {
      const token = getAccessToken();
      if (!token) {
        router.replace(`/login?redirect=${encodeURIComponent(pathname || "/write")}`);
        return;
      }
      const postID = Number(params?.id);
      if (!Number.isFinite(postID) || postID <= 0) {
        throw new Error("无效文章ID");
      }
      await deletePostDraft(token, postID, version);
      const data = await getPostDrafts(token, postID);
      setDrafts(data);
      if (selectedVersion === version && data.length > 0) {
        await loadDraft(token, postID, data[0].version);
      }
      setSuccess(`草稿 v${version} 已删除`);
    } catch (err) {
      setError(toZhError(err, "删除草稿失败"));
    }
  }

  async function onUnpublish() {
    setError("");
    setSuccess("");
    if (!window.confirm("确认撤回当前已发布版本吗？撤回后首页将不再展示该博客。")) {
      return;
    }
    try {
      const token = getAccessToken();
      if (!token) {
        router.replace(`/login?redirect=${encodeURIComponent(pathname || "/write")}`);
        return;
      }
      const postID = Number(params?.id);
      if (!Number.isFinite(postID) || postID <= 0) {
        throw new Error("无效文章ID");
      }
      const nextPost = await unpublishPost(token, postID);
      setPost(nextPost);
      const data = await getPostDrafts(token, postID);
      setDrafts(data);
      setSuccess("已撤回发布");
    } catch (err) {
      setError(toZhError(err, "撤回发布失败"));
    }
  }

  return (
    <main className="page">
      <Nav />
      <h1 style={{ marginTop: 0 }}>博客草稿列表</h1>
      {checking ? <p className="tip">校验登录状态中...</p> : null}
      {checking ? null : (
        <>
          <p className="tip">
            博客 ID: {params?.id} | 当前公开草稿:{" "}
            {post?.published_version ? `v${post.published_version}` : "未发布"}
            {selectedVersion ? ` | 当前编辑草稿: v${selectedVersion}` : ""}
          </p>

          <div className="card" style={{ display: "flex", gap: 8, flexWrap: "wrap" }}>
            <button type="button" onClick={onCreateDraft}>
              新建草稿副本
            </button>
            <Link href={`/me/posts/${params?.id || ""}`} className="ghost action-btn">
              文章信息
            </Link>
            {post?.published_version ? (
              <button type="button" className="ghost" onClick={onUnpublish}>
                撤回已发布版本
              </button>
            ) : null}
          </div>

          <h2>草稿列表</h2>
          {drafts.length === 0 ? <div className="card">暂无草稿</div> : null}
          {sortDrafts(drafts).map((d) => {
            const isPublished = post?.published_version === d.version;
            const isSelected = selectedVersion === d.version;
            return (
              <div className="card" key={d.id}>
                <p className="meta">
                  v{d.version} | 状态: {isPublished ? "published" : "draft"} | 最后修改:{" "}
                  {new Date(d.updated_at || d.created_at).toLocaleString()}
                </p>
                <p>{d.title}</p>
                <p className="tip">注释：{d.draft_note || "（无）"}</p>
                <div style={{ display: "flex", gap: 8, flexWrap: "wrap" }}>
                  <button
                    type="button"
                    className={isSelected ? "" : "ghost"}
                    onClick={async () => {
                      const token = getAccessToken();
                      if (!token) {
                        return;
                      }
                      const postID = Number(params?.id);
                      if (!Number.isFinite(postID) || postID <= 0) {
                        return;
                      }
                      await loadDraft(token, postID, d.version);
                    }}
                  >
                    {isSelected ? "当前编辑中" : "进入编辑"}
                  </button>
                  <button type="button" onClick={() => onPublishDraft(d.version)} disabled={isPublished}>
                    发布该草稿
                  </button>
                  <button type="button" className="ghost" onClick={() => onDeleteDraft(d.version)} disabled={isPublished}>
                    删除草稿
                  </button>
                </div>
              </div>
            );
          })}

          <h2>编辑器</h2>
          <div className="card">
            {loadingDraft ? <p className="tip">正在加载草稿内容...</p> : null}
            <form onSubmit={onSubmit}>
              <input
                placeholder="草稿注释（仅作者可见）"
                value={draftNote}
                onChange={(e) => setDraftNote(e.target.value)}
              />
              <div className="toolbar-row">
                <button type="button" className={useMdEditor ? "" : "ghost"} onClick={() => setUseMdEditor((v) => !v)}>
                  {useMdEditor ? "关闭 MD 编辑器" : "开启 MD 编辑器"}
                </button>
                <span className="meta">默认关闭，可按需启用高级编辑</span>
              </div>
              {useMdEditor ? (
                <MarkdownEditor value={rawContent} onChange={setRawContent} placeholder="Markdown 内容" />
              ) : (
                <textarea
                  className="meta-edit-summary"
                  placeholder="Markdown 内容（普通输入模式）"
                  value={rawContent}
                  onChange={(e) => setRawContent(e.target.value)}
                />
              )}
              <div style={{ display: "flex", gap: 8, flexWrap: "wrap" }}>
                <button type="submit" disabled={loading || loadingDraft || selectedVersion <= 0}>
                  {loading ? "处理中..." : "保存当前草稿"}
                </button>
                <button
                  type="button"
                  onClick={onSaveAndPublish}
                  disabled={loading || loadingDraft || selectedVersion <= 0}
                >
                  {loading ? "处理中..." : "保存并直接发布"}
                </button>
              </div>
            </form>
            {error ? <p className="error">{error}</p> : null}
            {success ? <p className="success">{success}</p> : null}
          </div>
        </>
      )}
    </main>
  );
}

function sortDrafts(input: PostVersion[]) {
  return [...input].sort((a, b) => {
    if (a.status === "published" && b.status !== "published") {
      return -1;
    }
    if (a.status !== "published" && b.status === "published") {
      return 1;
    }
    const at = new Date(a.updated_at || a.created_at).getTime();
    const bt = new Date(b.updated_at || b.created_at).getTime();
    return bt - at;
  });
}

function hasInlineBase64Image(content: string): boolean {
  return /data:image\/[a-zA-Z0-9.+-]+;base64,/i.test(content);
}
