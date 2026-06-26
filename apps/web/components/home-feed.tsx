"use client";

import { getFeed, type Post } from "@/lib/api";
import { getAccessToken } from "@/lib/auth";
import { toZhError } from "@/lib/errors";
import { Pager } from "@/components/pager";
import { Avatar } from "@/components/avatar";
import Link from "next/link";
import { CSSProperties, useEffect, useState } from "react";

type FeedType = "hot" | "latest" | "recommend";

export function HomeFeed() {
  const pageSize = 8;
  const [feedType, setFeedType] = useState<FeedType>("latest");
  const [page, setPage] = useState(1);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const [posts, setPosts] = useState<Post[]>([]);
  const [total, setTotal] = useState(0);
  const pageCount = Math.max(1, Math.ceil(total / pageSize));

  useEffect(() => {
    void loadFeed(feedType, page);
  }, [feedType, page]);

  function switchFeed(type: FeedType) {
    if (type === feedType) return;
    setPage(1);
    setPosts([]);
    setTotal(0);
    setFeedType(type);
  }

  async function loadFeed(type: FeedType, current: number) {
    setLoading(true);
    setError("");
    try {
      const data = await getFeed(type, getAccessToken() || undefined, { page: current, pageSize });
      setPosts(data.items || []);
      setTotal(data.total || 0);
    } catch (err) {
      setError(toZhError(err, "加载推文流失败"));
      setPosts([]);
      setTotal(0);
    } finally {
      setLoading(false);
    }
  }

  return (
    <section className="feed-stage">
      <div
        className="feed-tabs seg-switch seg-switch-3"
        data-active={feedType === "hot" ? 0 : feedType === "latest" ? 1 : 2}
      >
        <span className="seg-switch-thumb" aria-hidden="true" />
        <button type="button" className={feedType === "hot" ? "" : "ghost"} onClick={() => switchFeed("hot")}>
          热门
        </button>
        <button type="button" className={feedType === "latest" ? "" : "ghost"} onClick={() => switchFeed("latest")}>
          最新
        </button>
        <button
          type="button"
          className={feedType === "recommend" ? "" : "ghost"}
          onClick={() => switchFeed("recommend")}
        >
          推荐
        </button>
      </div>
      {loading ? <p className="tip feed-loading-tip">加载中...</p> : null}
      {error ? <p className="error">{error}</p> : null}
      {!loading && posts.length === 0 ? <div className="card">暂无内容</div> : null}
      <section className="feed">
        {posts.map((post, index) => {
          const cover = extractFirstImage(post.raw_content);
          const authorName = post.author_name || `u/${post.slug.split("-u")[0]}`;
          return (
            <article
              key={post.id}
              className="tweet-card"
              style={{ ["--tweet-index" as string]: index } as CSSProperties}
            >
              <Link href={`/posts/${post.id}`} className="tweet-card-stretch" aria-label={`查看推文 ${post.title}`} />
              <div className="tweet-card-content">
                <div className="tweet-head">
                  <Link href={`/users/${post.author_id}`} className="tweet-author-block tweet-author-link">
                    <Avatar className="tweet-avatar" fallbackClassName="tweet-avatar tweet-avatar-fallback" src={post.author_avatar_url} name={authorName} />
                    <strong className="tweet-author">{authorName}</strong>
                  </Link>
                  <span className="tweet-time">{formatTime(post.publish_at || post.updated_at)}</span>
                </div>
                <h2 className="tweet-title">{post.title}</h2>
                {post.summary?.trim() ? <p className="tweet-body">{post.summary}</p> : null}
                {cover ? (
                  <img
                    className="tweet-cover"
                    src={cover}
                    alt={post.title}
                    loading="lazy"
                    onError={(e) => {
                      e.currentTarget.style.display = "none";
                    }}
                  />
                ) : null}
                <div className="tweet-foot">
                  <div className="tweet-meta-tags">
                    <span className="tweet-tag">#{post.slug}</span>
                    {(post.tags || []).slice(0, 4).map((t) => (
                      <span key={`${post.id}-${t}`} className="tweet-tag soft">
                        {t}
                      </span>
                    ))}
                  </div>
                  <div className="tweet-actions">
                    {post.feed_score ? <span className="tweet-id">Score {post.feed_score.toFixed(2)}</span> : null}
                    {typeof post.comment_count === "number" ? (
                      <span className="tweet-id">评论 {post.comment_count}</span>
                    ) : null}
                  </div>
                </div>
              </div>
            </article>
          );
        })}
      </section>
      {!loading && total > pageSize ? <Pager page={page} total={pageCount} onChange={setPage} /> : null}
    </section>
  );
}

function formatTime(raw?: string) {
  if (!raw) return "刚刚";
  return new Date(raw).toLocaleString();
}

function extractFirstImage(raw?: string) {
  if (!raw) return "";
  const normalized = normalizeBrokenImageSyntax(raw);
  const mdMatch = normalized.match(/!\[[^\]]*]\(\s*([^)]+?)\s*\)/m);
  if (mdMatch?.[1]) {
    return mdMatch[1].trim();
  }
  const htmlMatch = normalized.match(/<img[^>]*src=["']([^"']+)["'][^>]*>/i);
  if (htmlMatch?.[1]) {
    return htmlMatch[1].trim();
  }
  return "";
}

function normalizeBrokenImageSyntax(raw: string) {
  return raw.replace(/!\[([^\]]*)]\s*\n\s*\(\s*([^)]+?)\s*\)/g, "![$1]($2)");
}
