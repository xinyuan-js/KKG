"use client";

import Link from "next/link";
import { useParams } from "next/navigation";
import { useEffect, useState } from "react";
import { ojListQuestionSolutions, type OJQuestionSolutionItem } from "@/lib/oj-api";
import { toZhError } from "@/lib/errors";

export default function OJQuestionSolutionsPage() {
  const params = useParams<{ id: string }>();
  const questionID = Number(params?.id);
  const [items, setItems] = useState<OJQuestionSolutionItem[]>([]);
  const [error, setError] = useState("");

  useEffect(() => {
    if (!Number.isFinite(questionID) || questionID <= 0) return;
    void loadAll();
  }, [questionID]);

  async function loadAll() {
    try {
      const data = await ojListQuestionSolutions({ questionId: questionID, current: 1, pageSize: 100 });
      setItems(data.records || []);
      setError("");
    } catch (err) {
      setError(toZhError(err, "加载题解文章失败"));
    }
  }

  return (
    <div className="reddit-layout">
      <section className="reddit-main section-gap">
        <header className="page-header">
          <h1 className="page-title">题解文章列表</h1>
        </header>
        <div className="toolbar-row">
          <Link className="tweet-link" href={`/oj/questions/${questionID}`}>返回题目</Link>
          <Link className="tweet-link" href="/">回到博客主页</Link>
        </div>
        {error ? <p className="error">{error}</p> : null}
        {items.length === 0 ? <div className="card empty-state">暂无绑定文章</div> : null}
        {items.map((it) => (
          <article key={it.id} className="tweet-card" style={{ opacity: 1, transform: "none", animation: "none" }}>
            <h3 className="tweet-title">{it.post?.title || `博客文章 #${it.postId}`}</h3>
            {it.post?.summary ? <p className="tweet-body">{it.post.summary}</p> : null}
            <div className="toolbar-row">
              <Link className="tweet-link" href={`/posts/${it.postId}`} target="_blank" rel="noreferrer">查看博客原文</Link>
            </div>
          </article>
        ))}
      </section>
      <aside className="reddit-side">
        <div className="community-card section-gap oj-side-card">
          <h3 style={{ margin: 0 }}>说明</h3>
          <p className="meta" style={{ margin: 0 }}>这里展示该题目关联的完整题解博客列表。</p>
        </div>
      </aside>
    </div>
  );
}
