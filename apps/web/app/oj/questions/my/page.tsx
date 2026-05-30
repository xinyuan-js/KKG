"use client";

import Link from "next/link";
import { isOJNotLoginError, ojMyQuestions, type OJQuestionVO } from "@/lib/oj-api";
import { toZhError } from "@/lib/errors";
import { useRouter } from "next/navigation";
import { useEffect, useState } from "react";

export default function OJMyQuestionsPage() {
  const router = useRouter();
  const [items, setItems] = useState<OJQuestionVO[]>([]);
  const [error, setError] = useState("");

  useEffect(() => {
    void load();
  }, []);

  async function load() {
    try {
      const data = await ojMyQuestions({ current: 1, pageSize: 20 });
      setItems(data.records || []);
      setError("");
    } catch (err) {
      if (isOJNotLoginError(err)) {
        router.replace("/login?redirect=/oj/questions/my");
        return;
      }
      setError(toZhError(err, "加载我的题目失败"));
    }
  }

  return (
    <div className="reddit-layout">
      <section className="reddit-main section-gap">
        <header className="page-header"><h1 className="page-title">我的题目</h1></header>
        {error ? <p className="error">{error}</p> : null}
        {items.length === 0 ? <div className="card empty-state">你还没有创建题目</div> : null}
        {items.map((q) => (
          <article key={q.id} className="card">
            <h3 className="title">{q.title}</h3>
            <p className="meta">提交 {q.submitNum} · 通过 {q.acceptedNum}</p>
            <Link className="tweet-link" href={`/oj/questions/${q.id}`}>进入题目</Link>
          </article>
        ))}
      </section>
      <aside className="reddit-side">
        <div className="community-card">
          <h3 style={{ margin: 0 }}>我的出题</h3>
          <p className="meta" style={{ marginTop: 8 }}>这里仅展示你发布的题目。</p>
        </div>
      </aside>
    </div>
  );
}
