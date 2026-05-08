"use client";

import Link from "next/link";
import { useSearchParams } from "next/navigation";
import { OJQuestionsPanel } from "@/components/oj-questions-panel";
import { OJFirstACRankPanel } from "@/components/oj-first-ac-rank-panel";

export default function OJHomePage() {
  const searchParams = useSearchParams();
  const tab = searchParams.get("tab") === "questions" ? "questions" : "overview";
  const query = (searchParams.get("q") || "").trim();

  return (
    <div className="reddit-layout">
      <section className="reddit-main section-gap">
        {tab === "questions" ? (
          <OJQuestionsPanel query={query} />
        ) : (
          <>
            <header className="page-header">
              <h1 className="page-title">KKG OJ</h1>
              <p className="tip">题目管理、在线提交、评测记录一体化工作台。</p>
            </header>
            <div className="card oj-quick-grid oj-side-card">
              <Link className="tweet-link" href="/oj?tab=questions">浏览题库</Link>
              <Link className="tweet-link" href="/oj/questions/new">创建题目</Link>
              <Link className="tweet-link" href="/oj/questions/my">我的题目</Link>
              <Link className="tweet-link" href="/me">个人资料</Link>
            </div>
            <section className="section-gap">
              <h2 className="tweet-title" style={{ marginBottom: 8 }}>24h 首次过题排行榜</h2>
              <OJFirstACRankPanel />
            </section>
          </>
        )}
      </section>
      <aside className="reddit-side">
        <div className="community-card oj-side-card">
          <h3 style={{ margin: 0 }}>服务说明</h3>
          <p className="meta" style={{ marginTop: 8 }}>
            OJ 前端通过 <code>/oj-api</code> 转发到 OJ 服务，博客业务与 OJ 服务已拆分。
          </p>
        </div>
      </aside>
    </div>
  );
}
