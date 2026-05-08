export function EmptyState({ title, desc }: { title: string; desc?: string }) {
  return (
    <div className="card empty-state">
      <h3 className="title">{title}</h3>
      {desc ? <p className="meta">{desc}</p> : null}
    </div>
  );
}
