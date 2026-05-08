"use client";

import { EmptyState } from "@/components/empty-state";
import { getMyNotifications, markMyNotificationRead } from "@/lib/api";
import { getAccessToken } from "@/lib/auth";
import { toZhError } from "@/lib/errors";
import Link from "next/link";
import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";

type InboxItem = {
  id: number;
  post_id: number;
  is_read: boolean;
  created_at: string;
  actor_name?: string;
  actor_avatar_url?: string;
  post_title?: string;
  comment_content?: string;
};

export default function InboxPage() {
  const router = useRouter();
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [unreadCount, setUnreadCount] = useState(0);
  const [items, setItems] = useState<InboxItem[]>([]);

  useEffect(() => {
    void load();
  }, []);

  async function load() {
    setLoading(true);
    setError("");
    try {
      const token = getAccessToken();
      if (!token) {
        router.replace("/login?redirect=/me/inbox");
        return;
      }
      const data = await getMyNotifications(token, 100);
      setUnreadCount(data.unread_count || 0);
      setItems((data.items || []) as InboxItem[]);
    } catch (err) {
      setError(toZhError(err, "加载消息失败"));
    } finally {
      setLoading(false);
    }
  }

  async function onRead(id: number) {
    const token = getAccessToken();
    if (!token) return;
    try {
      await markMyNotificationRead(token, id);
      setItems((prev) => prev.map((x) => (x.id === id ? { ...x, is_read: true } : x)));
      setUnreadCount((prev) => Math.max(0, prev - 1));
    } catch {
      // ignore
    }
  }

  return (
    <>
      <header className="page-header">
        <h1 className="page-title">@我的信息箱</h1>
      </header>
      <p className="tip">未读 {unreadCount} 条</p>
      {loading ? <p className="tip">加载中...</p> : null}
      {error ? <p className="error">{error}</p> : null}
      {!loading && items.length === 0 ? <EmptyState title="暂无新消息" desc="当有人回复你时会出现在这里。" /> : null}
      <section className="feed">
        {items.map((n) => {
          const actor = n.actor_name || "某位用户";
          return (
            <div className={`card inbox-item ${n.is_read ? "is-read" : ""}`} key={n.id}>
              <div className="inbox-head">
                <div className="tweet-author-block">
                  {n.actor_avatar_url ? (
                    <img className="tweet-avatar" src={n.actor_avatar_url} alt={actor} />
                  ) : (
                    <span className="tweet-avatar tweet-avatar-fallback">{actor.slice(0, 1).toUpperCase()}</span>
                  )}
                  <strong className="tweet-author">{actor}</strong>
                  {!n.is_read ? <span className="nav-badge">新</span> : null}
                </div>
                <span className="tweet-time">{new Date(n.created_at).toLocaleString()}</span>
              </div>
              <p className="comment-content">
                回复了你：{n.comment_content || "（无内容）"}
              </p>
              <div className="toolbar-row">
                <Link className="tweet-link" href={`/posts/${n.post_id}`} onClick={() => void onRead(n.id)}>
                  查看推文
                </Link>
                {!n.is_read ? (
                  <button type="button" className="ghost" onClick={() => void onRead(n.id)}>
                    标记已读
                  </button>
                ) : null}
              </div>
            </div>
          );
        })}
      </section>
    </>
  );
}
