import { OJQuestionsPanel } from "@/components/oj-questions-panel";

export default async function OJQuestionsPage({
  searchParams
}: {
  searchParams: Promise<{ q?: string }>;
}) {
  const sp = await searchParams;
  const q = (sp?.q || "").trim();
  return (
    <div className="reddit-layout">
      <section className="reddit-main">
        <OJQuestionsPanel query={q} />
      </section>
      <aside className="reddit-side">
        <div className="community-card section-gap oj-side-card">
          <h3 style={{ margin: 0 }}>题库说明</h3>
          <p className="meta" style={{ margin: 0 }}>展示公开题目，支持在线提交与评测结果回看。</p>
        </div>
      </aside>
    </div>
  );
}
