import { Nav } from "@/components/nav";
import { RankingsPanel } from "@/components/rankings-panel";

export default function RankingsPage() {
  return (
    <main className="page">
      <Nav />
      <div className="reddit-layout">
        <section className="reddit-main section-gap">
          <header className="page-header">
            <h1 className="page-title">排行榜</h1>
          </header>
          <RankingsPanel />
        </section>
        <aside className="reddit-side home-side-align">
          <div className="community-card">
            <h3 style={{ margin: 0 }}>榜单说明</h3>
            <p className="meta" style={{ margin: "8px 0 0" }}>
              榜单数据按热度聚合，支持 24h / 7天 / 30天 / 全部维度切换。
            </p>
          </div>
        </aside>
      </div>
    </main>
  );
}
