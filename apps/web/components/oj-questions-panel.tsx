"use client";

import Link from "next/link";
import { Pager } from "@/components/pager";
import { ojGetLoginUser, ojListQuestions, ojListQuestionSubmits, type OJQuestionVO } from "@/lib/oj-api";
import { toZhError } from "@/lib/errors";
import { useEffect, useState } from "react";

export function OJQuestionsPanel({ query }: { query: string }) {
  const pageSize = 12;
  const [page, setPage] = useState(1);
  const [items, setItems] = useState<OJQuestionVO[]>([]);
  const [total, setTotal] = useState(0);
  const [error, setError] = useState("");
  const [passedQuestionIDs, setPassedQuestionIDs] = useState<Set<number>>(new Set());

  useEffect(() => {
    setPage(1);
  }, [query]);

  useEffect(() => {
    void loadMyAccepted();
  }, []);

  useEffect(() => {
    void load(query, page);
  }, [query, page]);

  async function load(searchText: string, current: number) {
    try {
      const data = await ojListQuestions({ current, pageSize, searchText });
      setItems(data.records || []);
      setTotal(Number(data.total || 0));
      setError("");
    } catch (err) {
      setError(toZhError(err, "加载题目失败"));
    }
  }

  async function loadMyAccepted() {
    try {
      const me = await ojGetLoginUser();
      if (!me?.id) return;
      const set = new Set<number>();
      let current = 1;
      const pageSize = 20;
      for (;;) {
        const data = await ojListQuestionSubmits({ current, pageSize, userId: me.id });
        for (const s of data.records || []) {
          if (s.status === 2 && Number.isFinite(s.questionId)) {
            set.add(s.questionId);
          }
        }
        const total = Number(data.total || 0);
        if (!data.records?.length || current*pageSize >= total) break;
        current += 1;
      }
      setPassedQuestionIDs(set);
    } catch {
      setPassedQuestionIDs(new Set());
    }
  }

  return (
    <section className="section-gap">
      <header className="page-header">
        <h1 className="page-title">题库</h1>
        {query ? <p className="meta">搜索：{query}</p> : null}
      </header>
      <div className="toolbar-row">
        <Link className="tweet-link" href="/oj/questions/new">创建题目</Link>
        <Link className="tweet-link" href="/oj/questions/my">我的题目</Link>
      </div>
      {error ? <p className="error">{error}</p> : null}
      <section className="feed oj-question-list">
        {items.length === 0 ? <div className="card empty-state">暂无题目</div> : null}
        {items.map((q) => (
          <article key={q.id} className="oj-question-row" style={{ opacity: 1, transform: "none", animation: "none" }}>
            <Link href={`/oj/questions/${q.id}`} className="oj-question-main">
              <div className="oj-question-state">
                <span className={`oj-pass-dot${passedQuestionIDs.has(q.id) ? " pass" : ""}`} />
                <span className="meta">{passedQuestionIDs.has(q.id) ? "已通过" : "未通过"}</span>
              </div>
              <div className="oj-question-body">
                <h3 className="tweet-title oj-question-title">{q.id}. {q.title}</h3>
                <div className="tweet-meta-tags oj-question-tags">{(q.tags || []).map((t) => <span key={`${q.id}-${t}`} className="tweet-tag soft">{t}</span>)}</div>
              </div>
              <div className="oj-question-stats">
                <span className="tweet-id">提交 {q.submitNum}</span>
                <span className="tweet-id">通过 {q.acceptedNum}</span>
                <span className="tweet-id">通过率 {q.submitNum > 0 ? `${Math.round((q.acceptedNum / q.submitNum) * 100)}%` : "0%"}</span>
              </div>
              <div className="oj-question-action">
                <span className="tweet-link">进入</span>
              </div>
            </Link>
            <div className="oj-question-foot">
              <span className="meta">题号 #{q.id}</span>
            </div>
          </article>
        ))}
      </section>
      {total > pageSize ? <Pager page={page} total={Math.ceil(total / pageSize)} onChange={setPage} /> : null}
    </section>
  );
}
