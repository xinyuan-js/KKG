import Link from "next/link";

export default function OJProfilePage() {
  return (
    <div className="reddit-layout">
      <section className="reddit-main">
        <section className="card oj-side-card section-gap">
          <h1 className="title">个人中心</h1>
          <p className="meta">OJ 与博客资料已统一，请在同一个个人中心管理账号信息。</p>
          <div className="toolbar-row">
            <Link className="tweet-link" href="/me">进入个人中心</Link>
          </div>
        </section>
      </section>
    </div>
  );
}
