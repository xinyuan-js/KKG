"use client";

import { HomeFeed } from "@/components/home-feed";
import { Nav } from "@/components/nav";
import { RankingsPanel } from "@/components/rankings-panel";
import Link from "next/link";
import { useSearchParams } from "next/navigation";

export default function HomePage() {
  const searchParams = useSearchParams();
  const tab = searchParams.get("tab") === "rankings" ? "rankings" : "home";

  return (
    <main className="page">
      <Nav />
      <div className="reddit-layout">
        <section className="reddit-main">
          <header className="page-header home-page-header">
            <h1 className="page-title">{tab === "rankings" ? "排行榜" : "首页"}</h1>
          </header>
          {tab === "rankings" ? (
            <RankingsPanel />
          ) : (
            <HomeFeed />
          )}
        </section>
        <aside className="reddit-side home-side-align">
          <div className="community-card">
            <h3 style={{ margin: 0 }}>社区动态</h3>
            <p className="meta" style={{ margin: "8px 0 0" }}>
              使用写作、评论、点赞、收藏持续完善你的内容流。右侧面板固定展示，便于快速导航与查看节奏。
            </p>
          </div>
          <div className="community-card">
            <h3 style={{ margin: 0 }}>快速入口</h3>
            <div className="section-gap" style={{ marginTop: 8 }}>
              <Link className="tweet-link" href="/write">
                发布内容
              </Link>
              <Link className="tweet-link" href="/?tab=rankings">
                查看排行榜
              </Link>
              <Link className="tweet-link" href="/me">
                个人中心
              </Link>
              <Link className="tweet-link" href="/oj">
                OJ 控制台
              </Link>
            </div>
          </div>
        </aside>
      </div>
    </main>
  );
}
