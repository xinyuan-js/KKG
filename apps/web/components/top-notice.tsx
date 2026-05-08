"use client";

import { useEffect, useRef, useState } from "react";

type Notice = { text: string; tone: "success" | "error" };

export function TopNotice() {
  const [notice, setNotice] = useState<Notice | null>(null);
  const timerRef = useRef<number | null>(null);

  useEffect(() => {
    const onNotice = (evt: Event) => {
      const detail = (evt as CustomEvent<Notice>).detail;
      if (!detail?.text) return;
      if (timerRef.current) window.clearTimeout(timerRef.current);
      setNotice({ text: detail.text, tone: detail.tone || "success" });
      timerRef.current = window.setTimeout(() => setNotice(null), 9000);
    };
    window.addEventListener("app:top-notice", onNotice as EventListener);
    return () => window.removeEventListener("app:top-notice", onNotice as EventListener);
  }, []);

  useEffect(() => {
    return () => {
      if (timerRef.current) window.clearTimeout(timerRef.current);
    };
  }, []);

  if (!notice) return null;
  return (
    <div className="top-notice-wrap" role="status" aria-live="polite">
      <div className={`top-notice-card ${notice.tone === "success" ? "success" : "error"}`}>
        <div className="top-notice-head">
          <strong>{notice.tone === "success" ? "操作成功" : "操作失败"}</strong>
          <button type="button" className="top-notice-close" onClick={() => setNotice(null)} aria-label="关闭通知">
            ×
          </button>
        </div>
        <p className="top-notice-text">{notice.text}</p>
      </div>
    </div>
  );
}

