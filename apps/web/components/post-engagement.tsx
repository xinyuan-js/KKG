"use client";

import { getPostEngagement, togglePostFavorite, togglePostLike, type PostEngagement } from "@/lib/api";
import { getAccessToken } from "@/lib/auth";
import { toZhError } from "@/lib/errors";
import { useEffect, useState } from "react";

export function PostEngagementBar({ postID }: { postID: number }) {
  const [eng, setEng] = useState<PostEngagement | null>(null);
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    void load();
  }, [postID]);

  async function load() {
    try {
      const token = getAccessToken() || undefined;
      const data = await getPostEngagement(postID, token);
      setEng(data);
    } catch (err) {
      setError(toZhError(err, "加载互动数据失败"));
    }
  }

  async function onLike() {
    const token = getAccessToken();
    if (!token) {
      setError("请先登录");
      return;
    }
    setLoading(true);
    setError("");
    try {
      const data = await togglePostLike(postID, token);
      setEng(data);
    } catch (err) {
      setError(toZhError(err, "操作失败"));
    } finally {
      setLoading(false);
    }
  }

  async function onFavorite() {
    const token = getAccessToken();
    if (!token) {
      setError("请先登录");
      return;
    }
    setLoading(true);
    setError("");
    try {
      const data = await togglePostFavorite(postID, token);
      setEng(data);
    } catch (err) {
      setError(toZhError(err, "操作失败"));
    } finally {
      setLoading(false);
    }
  }

  return (
    <section className="card" style={{ marginTop: 12 }}>
      <div style={{ display: "flex", gap: 8, flexWrap: "wrap", alignItems: "center" }}>
        <button type="button" className={eng?.liked ? "" : "ghost"} disabled={loading} onClick={onLike}>
          {eng?.liked ? "已点赞" : "点赞"} {eng?.like_count ?? 0}
        </button>
        <button type="button" className={eng?.favorited ? "" : "ghost"} disabled={loading} onClick={onFavorite}>
          {eng?.favorited ? "已收藏" : "收藏"} {eng?.favorite_count ?? 0}
        </button>
        <span className="meta">评论 {eng?.comment_count ?? 0}</span>
      </div>
      {error ? <p className="error" style={{ marginBottom: 0 }}>{error}</p> : null}
    </section>
  );
}
