"use client";

import { EmptyState } from "@/components/empty-state";
import { getMyFavorites, type Post } from "@/lib/api";
import { getAccessToken } from "@/lib/auth";
import { toZhError } from "@/lib/errors";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { useEffect, useState } from "react";

export default function MyFavoritesPage() {
  const router = useRouter();
  const [posts, setPosts] = useState<Post[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");

  useEffect(() => {
    void load();
  }, []);

  async function load() {
    setLoading(true);
    setError("");
    try {
      const token = getAccessToken();
      if (!token) {
        router.replace("/login?redirect=/me/favorites");
        return;
      }
      const data = await getMyFavorites(token);
      setPosts(data);
    } catch (err) {
      setError(toZhError(err, "加载收藏失败"));
    } finally {
      setLoading(false);
    }
  }

  return (
    <>
      <header className="page-header">
        <h1 className="page-title">我的收藏</h1>
      </header>
      {loading ? <p className="tip">加载中...</p> : null}
      {error ? <p className="error">{error}</p> : null}
      {!loading && posts.length === 0 ? <EmptyState title="你还没有收藏内容" desc="在文章详情页点“收藏”后会展示在这里。" /> : null}
      <section className="section-gap">
        {posts.map((post) => {
          const authorName = post.author_name || `u/${post.author_id}`;
          return (
            <article className="card" key={post.id}>
              <Link href={`/posts/${post.id}`}>
                <h3 className="title">{post.title}</h3>
              </Link>
              <p className="meta">作者 {authorName}</p>
              {(post.tags || []).length > 0 ? (
                <div className="tweet-meta-tags">
                  {(post.tags || []).slice(0, 5).map((t) => (
                    <span key={`${post.id}-${t}`} className="tweet-tag soft">
                      {t}
                    </span>
                  ))}
                </div>
              ) : null}
              {post.summary ? <p>{post.summary}</p> : null}
            </article>
          );
        })}
      </section>
    </>
  );
}
