"use client";

import { Pager } from "@/components/pager";
import { Avatar } from "@/components/avatar";
import { toZhError } from "@/lib/errors";
import { ojFirstACRank24h, type OJFirstACRankItem } from "@/lib/oj-api";
import Link from "next/link";
import { CSSProperties, useEffect, useMemo, useState } from "react";

export function OJFirstACRankPanel() {
  const pageSize = 8;
  const [page, setPage] = useState(1);
  const [items, setItems] = useState<OJFirstACRankItem[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const pageItems = useMemo(() => items.slice((page - 1) * pageSize, page * pageSize), [items, page]);

  useEffect(() => {
    void load();
  }, []);

  async function load() {
    setLoading(true);
    setError("");
    setPage(1);
    try {
      const data = await ojFirstACRank24h(50);
      setItems(data.records || []);
    } catch (err) {
      setError(toZhError(err, "加载 OJ 排行榜失败"));
      setItems([]);
    } finally {
      setLoading(false);
    }
  }

  return (
    <>
      {loading ? <p className="tip feed-loading-tip">加载中...</p> : null}
      {error ? <p className="error">{error}</p> : null}
      <section className="section-gap ranking-list">
        {!loading && items.length === 0 ? <div className="card">暂无排行数据</div> : null}
        {pageItems.map((item, index) => {
          const absoluteIndex = (page - 1) * pageSize + index;
          const authorName = item.userName || `u/${item.userId}`;
          const userLink = item.blogUserId && item.blogUserId > 0 ? `/users/${item.blogUserId}` : undefined;
          return (
            <article
              key={`oj-rank-${page}-${item.userId}`}
              className="tweet-card ranking-card no-enter"
              style={{ ["--tweet-index" as string]: absoluteIndex } as CSSProperties}
            >
              <div className="tweet-card-content">
                <div className="tweet-head">
                  {userLink ? (
                    <Link href={userLink} className="tweet-author-block tweet-author-link">
                      <Avatar className="tweet-avatar" fallbackClassName="tweet-avatar tweet-avatar-fallback" src={item.userAvatar} name={authorName} />
                      <strong className="tweet-author">{authorName}</strong>
                    </Link>
                  ) : (
                    <div className="tweet-author-block">
                      <Avatar className="tweet-avatar" fallbackClassName="tweet-avatar tweet-avatar-fallback" src={item.userAvatar} name={authorName} />
                      <strong className="tweet-author">{authorName}</strong>
                    </div>
                  )}
                  <div className="ranking-head-right">
                    <span className={`ranking-badge rank-${absoluteIndex + 1 <= 3 ? absoluteIndex + 1 : "other"}`}>TOP {absoluteIndex + 1}</span>
                    <span className="tweet-id">{item.firstAcCount} 题</span>
                  </div>
                </div>
              </div>
            </article>
          );
        })}
      </section>
      {!loading && items.length > pageSize ? (
        <Pager page={page} total={Math.max(1, Math.ceil(items.length / pageSize))} onChange={setPage} />
      ) : null}
    </>
  );
}
