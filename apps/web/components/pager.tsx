"use client";

type Props = {
  page: number;
  total: number;
  onChange: (page: number) => void;
};

export function Pager({ page, total, onChange }: Props) {
  if (total <= 1) return null;

  const items = buildPages(page, total);
  return (
    <nav className="pager" aria-label="分页">
      <button type="button" className="ghost" disabled={page <= 1} onClick={() => onChange(page - 1)}>
        上一页
      </button>
      {items.map((it, idx) =>
        it === "..." ? (
          <span key={`dots-${idx}`} className="pager-dots">
            ...
          </span>
        ) : (
          <button
            key={it}
            type="button"
            className={it === page ? "pager-page active" : "pager-page ghost"}
            onClick={() => onChange(it)}
          >
            {it}
          </button>
        )
      )}
      <button type="button" className="ghost" disabled={page >= total} onClick={() => onChange(page + 1)}>
        下一页
      </button>
    </nav>
  );
}

function buildPages(page: number, total: number): Array<number | "..."> {
  if (total <= 7) return Array.from({ length: total }, (_, i) => i + 1);
  if (page <= 4) return [1, 2, 3, 4, 5, "...", total];
  if (page >= total - 3) return [1, "...", total - 4, total - 3, total - 2, total - 1, total];
  return [1, "...", page - 1, page, page + 1, "...", total];
}

