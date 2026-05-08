"use client";

import { Nav } from "@/components/nav";
import { Pager } from "@/components/pager";
import { searchAll, type SearchPostItem, type SearchUserItem } from "@/lib/api";
import { getAccessToken } from "@/lib/auth";
import { toZhError } from "@/lib/errors";
import Link from "next/link";
import { useSearchParams } from "next/navigation";
import { CSSProperties, useEffect, useState } from "react";

type SearchType = "post" | "user";

export default function SearchPage() {
  const pageSize = 8;
  const searchParams = useSearchParams();
  const type = (searchParams.get("type") === "user" ? "user" : "post") as SearchType;
  const q = (searchParams.get("q") || "").trim();
  const [loading, setLoading] = useState(false);
  const [page, setPage] = useState(1);
  const [error, setError] = useState("");
  const [posts, setPosts] = useState<SearchPostItem[]>([]);
  const [users, setUsers] = useState<SearchUserItem[]>([]);

  useEffect(() => {
    if (!q) {
      setPosts([]);
      setUsers([]);
      setError("");
      return;
    }
    let cancelled = false;
    async function run() {
      setLoading(true);
      setError("");
      setPage(1);
      try {
        const res = await searchAll({
          type,
          q,
          limit: 20,
          token: getAccessToken() || undefined
        });
        if (cancelled) return;
        if (type === "post") {
          setPosts((res.items as SearchPostItem[]) || []);
          setUsers([]);
        } else {
          setUsers((res.items as SearchUserItem[]) || []);
          setPosts([]);
        }
      } catch (err) {
        if (cancelled) return;
        setPosts([]);
        setUsers([]);
        setError(toZhError(err, "搜索失败"));
      } finally {
        if (!cancelled) setLoading(false);
      }
    }
    void run();
    return () => {
      cancelled = true;
    };
  }, [type, q]);

  const count = type === "post" ? posts.length : users.length;
  const pageCount = Math.max(1, Math.ceil(count / pageSize));
  const postPageItems = posts.slice((page - 1) * pageSize, page * pageSize);
  const userPageItems = users.slice((page - 1) * pageSize, page * pageSize);

  return (
    <main className="page">
      <Nav />
      <div className="reddit-layout">
        <section className="reddit-main section-gap">
          <header className="page-header home-page-header">
            <h1 className="page-title">搜索结果</h1>
            <p className="tip">
              {q ? `关键词「${q}」 · ${type === "post" ? "文章" : "用户"} · ${loading ? "加载中..." : `${count} 条结果`}` : "请输入关键词进行搜索"}
            </p>
          </header>
          {error ? <p className="error">{error}</p> : null}
          {!loading && !error && q && count === 0 ? <div className="card">暂无结果</div> : null}
          {type === "post" ? (
            <section className="feed">
              {postPageItems.map((item, idx) => (
                <article key={item.id} className="tweet-card" style={{ ["--tweet-index" as string]: (page - 1) * pageSize + idx } as CSSProperties}>
                  <Link href={`/posts/${item.id}`} className="tweet-card-stretch" aria-label={item.title} />
                  <div className="tweet-card-content">
                    <h3 className="tweet-title">{item.title}</h3>
                    <p className="tweet-body">{item.summary || "（无摘要）"}</p>
                    <div className="tweet-foot">
                      <div className="tweet-meta-tags">
                        {(item.tags || []).slice(0, 5).map((t) => (
                          <span key={`${item.id}-${t}`} className="tweet-tag soft">
                            {t}
                          </span>
                        ))}
                      </div>
                      <span className="tweet-id">Score {Number(item.score || 0).toFixed(2)}</span>
                    </div>
                  </div>
                </article>
              ))}
            </section>
          ) : (
            <section className="feed">
              {userPageItems.map((item, idx) => (
                <Link key={item.id} href={`/users/${item.id}`} className="tweet-card-link">
                  <article className="tweet-card" style={{ ["--tweet-index" as string]: (page - 1) * pageSize + idx } as CSSProperties}>
                    <div className="tweet-card-content">
                      <div className="tweet-head">
                        <strong className="tweet-author">{item.username}</strong>
                        <span className="tweet-id">Score {Number(item.score || 0).toFixed(2)}</span>
                      </div>
                      <p className="tweet-body">{item.email}</p>
                    </div>
                  </article>
                </Link>
              ))}
            </section>
          )}
          {!loading && count > pageSize ? <Pager page={page} total={pageCount} onChange={setPage} /> : null}
        </section>
        <aside className="reddit-side home-side-align">
          <div className="community-card">
            <h3 style={{ margin: 0 }}>搜索提示</h3>
            <p className="meta" style={{ margin: "8px 0 0" }}>
              文章搜索支持标题、摘要和标签。用户搜索会自动排除你自己。
            </p>
          </div>
        </aside>
      </div>
    </main>
  );
}
