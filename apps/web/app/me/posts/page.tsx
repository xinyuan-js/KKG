"use client";

import { EmptyState } from "@/components/empty-state";
import { deletePost, getMyPosts, type Post, unpublishPost } from "@/lib/api";
import { getAccessToken } from "@/lib/auth";
import { toZhError } from "@/lib/errors";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { useEffect, useState } from "react";

export default function MyPostsPage() {
  const router = useRouter();
  const [posts, setPosts] = useState<Post[]>([]);
  const [statusFilter, setStatusFilter] = useState<"all" | "draft" | "published">("all");
  const [keyword, setKeyword] = useState("");
  const [loadingPosts, setLoadingPosts] = useState(false);
  const [error, setError] = useState("");
  const [success, setSuccess] = useState("");

  useEffect(() => {
    void loadMyPosts();
  }, []);

  async function loadMyPosts() {
    setLoadingPosts(true);
    setError("");
    setSuccess("");
    try {
      const token = getAccessToken();
      if (!token) {
        router.replace("/login?redirect=/me/posts");
        return;
      }
      const data = await getMyPosts(token);
      setPosts(data);
    } catch (err) {
      setError(toZhError(err, "加载文章失败"));
    } finally {
      setLoadingPosts(false);
    }
  }

  async function onDeletePost(postID: number) {
    setError("");
    setSuccess("");
    if (!window.confirm("确认删除这篇文章吗？删除后将不再出现在我的文章中。")) {
      return;
    }
    try {
      const token = getAccessToken();
      if (!token) {
        router.replace("/login?redirect=/me/posts");
        return;
      }
      await deletePost(token, postID);
      await loadMyPosts();
      setSuccess(`文章 ${postID} 已删除`);
    } catch (err) {
      setError(toZhError(err, "删除失败"));
    }
  }

  async function onUnpublish(postID: number) {
    setError("");
    setSuccess("");
    if (!window.confirm("确认撤回这篇文章的已发布版本吗？撤回后首页将不再展示。")) {
      return;
    }
    try {
      const token = getAccessToken();
      if (!token) {
        router.replace("/login?redirect=/me/posts");
        return;
      }
      await unpublishPost(token, postID);
      await loadMyPosts();
      setSuccess(`文章 ${postID} 已撤回发布`);
    } catch (err) {
      setError(toZhError(err, "撤回失败"));
    }
  }

  const filtered = filteredPosts(posts, statusFilter, keyword);

  return (
    <>
      <header className="page-header">
        <h1 className="page-title">我的文章</h1>
      </header>
      <div className="card section-gap">
        <p className="meta" style={{ margin: 0 }}>
          全部 {posts.length} | 已发布 {posts.filter((p) => p.status === "published").length} | 草稿{" "}
          {posts.filter((p) => p.status !== "published").length}
        </p>
        <div className="toolbar-row">
          <input
            className="toolbar-grow"
            placeholder="按标题或 slug 搜索"
            value={keyword}
            onChange={(e) => setKeyword(e.target.value)}
          />
          <button type="button" className={statusFilter === "all" ? "" : "ghost"} onClick={() => setStatusFilter("all")}>
            全部
          </button>
          <button
            type="button"
            className={statusFilter === "draft" ? "" : "ghost"}
            onClick={() => setStatusFilter("draft")}
          >
            草稿
          </button>
          <button
            type="button"
            className={statusFilter === "published" ? "" : "ghost"}
            onClick={() => setStatusFilter("published")}
          >
            已发布
          </button>
          <Link href="/me" className="ghost action-btn">
            返回个人信息
          </Link>
        </div>
      </div>

      {loadingPosts ? <p className="tip">加载中...</p> : null}
      {error ? <p className="error">{error}</p> : null}
      {success ? <p className="success">{success}</p> : null}

      {!loadingPosts && filtered.length === 0 ? (
        <EmptyState title="没有匹配的文章" desc="试试切换筛选条件，或者先去写作页创建草稿。" />
      ) : null}
      <section className="section-gap">
        {filtered.map((post) => (
          <div className="card" key={post.id}>
            <h3 className="title">{post.title}</h3>
            <p className="meta">
              状态：{post.status} | slug: {post.slug}
            </p>
            <p>{post.summary || "（无摘要）"}</p>
            <div className="post-actions">
              <Link className="ghost action-btn" href={`/write/${post.id}`}>
                编辑
              </Link>
              <Link className="ghost action-btn" href={`/me/posts/${post.id}`}>
                文章信息
              </Link>
              {post.status === "published" ? (
                <button type="button" className="ghost action-btn" onClick={() => onUnpublish(post.id)}>
                  撤回发布
                </button>
              ) : null}
              <button type="button" className="ghost action-btn" onClick={() => onDeletePost(post.id)}>
                删除
              </button>
            </div>
          </div>
        ))}
      </section>
    </>
  );
}

function filteredPosts(posts: Post[], status: "all" | "draft" | "published", keyword: string) {
  const kw = keyword.trim().toLowerCase();
  return posts.filter((p) => {
    if (status === "published" && p.status !== "published") return false;
    if (status === "draft" && p.status === "published") return false;
    if (!kw) return true;
    return (p.title || "").toLowerCase().includes(kw) || (p.slug || "").toLowerCase().includes(kw);
  });
}
