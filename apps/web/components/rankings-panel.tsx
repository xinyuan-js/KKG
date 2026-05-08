"use client";

import { getPostRankingsByPeriod, type Post } from "@/lib/api";
import { toZhError } from "@/lib/errors";
import { Pager } from "@/components/pager";
import Link from "next/link";
import { CSSProperties, useEffect, useMemo, useState } from "react";

type Period = "24h" | "7d" | "30d" | "all";

export function RankingsPanel() {
  const pageSize = 8;
  const [period, setPeriod] = useState<Period>("all");
  const [page, setPage] = useState(1);
  const [posts, setPosts] = useState<Post[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const [renderRound, setRenderRound] = useState(0);
  const pageCount = Math.max(1, Math.ceil(posts.length / pageSize));
  const pageItems = useMemo(() => posts.slice((page - 1) * pageSize, page * pageSize), [posts, page]);

  useEffect(() => {
    void load(period);
  }, [period]);

  async function load(p: Period) {
    setLoading(true);
    setError("");
    setPage(1);
    setRenderRound((v) => v + 1);
    try {
      const data = await getPostRankingsByPeriod(p, 30);
      setPosts(data);
    } catch (err) {
      setError(toZhError(err, "加载排行榜失败"));
      setPosts([]);
    } finally {
      setLoading(false);
    }
  }

  return (
    <>
      <div className="feed-tabs seg-switch seg-switch-4" data-active={period === "24h" ? 0 : period === "7d" ? 1 : period === "30d" ? 2 : 3}>
        <span className="seg-switch-thumb" aria-hidden="true" />
        <button type="button" className={period === "24h" ? "" : "ghost"} onClick={() => setPeriod("24h")}>
          24h
        </button>
        <button type="button" className={period === "7d" ? "" : "ghost"} onClick={() => setPeriod("7d")}>
          7天
        </button>
        <button type="button" className={period === "30d" ? "" : "ghost"} onClick={() => setPeriod("30d")}>
          30天
        </button>
        <button type="button" className={period === "all" ? "" : "ghost"} onClick={() => setPeriod("all")}>
          全部
        </button>
      </div>
      {loading ? <p className="tip feed-loading-tip">加载中...</p> : null}
      {error ? <p className="error">{error}</p> : null}
      <section className="section-gap ranking-list">
        {!loading && posts.length === 0 ? <div className="card">暂无排行数据</div> : null}
        {pageItems.map((post, index) => {
          const absoluteIndex = (page - 1) * pageSize + index;
          const authorName = post.author_name || `u/${post.author_id}`;
          return (
            <article
              key={`${period}-${renderRound}-${page}-${post.id}`}
              className="tweet-card ranking-card no-enter"
              style={{ ["--tweet-index" as string]: absoluteIndex } as CSSProperties}
            >
              <Link href={`/posts/${post.id}`} className="tweet-card-stretch" aria-label={`查看文章 ${post.title}`} />
              <div className="tweet-card-content">
                <div className="tweet-head">
                  <Link href={`/users/${post.author_id}`} className="tweet-author-block tweet-author-link">
                    {post.author_avatar_url ? (
                      <img className="tweet-avatar" src={post.author_avatar_url} alt={authorName} />
                    ) : (
                      <span className="tweet-avatar tweet-avatar-fallback">{authorName.slice(0, 1).toUpperCase()}</span>
                    )}
                    <strong className="tweet-author">{authorName}</strong>
                  </Link>
                  <div className="ranking-head-right">
                    <span className={`ranking-badge rank-${absoluteIndex + 1 <= 3 ? absoluteIndex + 1 : "other"}`}>TOP {absoluteIndex + 1}</span>
                    {post.feed_score ? <span className="tweet-id">Score {post.feed_score.toFixed(2)}</span> : null}
                  </div>
                </div>
                <Link href={`/posts/${post.id}`}>
                  <h3 className="tweet-title" style={{ marginBottom: 6 }}>{post.title}</h3>
                </Link>
                {post.summary?.trim() ? <p className="tweet-body">{post.summary}</p> : null}
                <div className="tweet-foot">
                  <div className="tweet-meta-tags">
                    {(post.tags || []).slice(0, 5).map((t) => (
                      <span key={`${post.id}-${t}`} className="tweet-tag soft">
                        {t}
                      </span>
                    ))}
                  </div>
                </div>
              </div>
            </article>
          );
        })}
      </section>
      {!loading && posts.length > pageSize ? <Pager page={page} total={pageCount} onChange={setPage} /> : null}
    </>
  );
}
