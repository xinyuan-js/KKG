"use client";

import { useEffect, useRef, useState } from "react";

export function BackToTop() {
  const [visible, setVisible] = useState(false);
  const [progress, setProgress] = useState(0);
  const [mode, setMode] = useState<"top" | "bottom">("bottom");
  const [maxScroll, setMaxScroll] = useState(0);
  const lastY = useRef(0);

  useEffect(() => {
    const onScroll = () => {
      const y = window.scrollY || 0;
      const max = Math.max(0, document.documentElement.scrollHeight - window.innerHeight);
      const p = max > 0 ? Math.min(100, Math.max(0, Math.round((y / max) * 100))) : 0;
      const delta = y - lastY.current;

      setVisible(max > 180);
      setProgress(p);
      setMaxScroll(max);

      if (y <= 24) {
        setMode("bottom");
      } else if (max - y <= 24) {
        setMode("top");
      } else if (delta > 3) {
        setMode("bottom");
      } else if (delta < -3) {
        setMode("top");
      }
      lastY.current = y;
    };
    onScroll();
    window.addEventListener("resize", onScroll);
    window.addEventListener("scroll", onScroll, { passive: true });
    return () => {
      window.removeEventListener("resize", onScroll);
      window.removeEventListener("scroll", onScroll);
    };
  }, []);

  function onClick() {
    if (mode === "bottom") {
      window.scrollTo({ top: maxScroll, behavior: "smooth" });
      return;
    }
    window.scrollTo({ top: 0, behavior: "smooth" });
  }

  return (
    <button
      type="button"
      aria-label={mode === "bottom" ? "跳到底部" : "回到顶部"}
      className={`back-to-top${visible ? " is-visible" : ""}${mode === "bottom" ? " to-bottom" : " to-top"}`}
      style={{ ["--scroll-progress" as string]: `${progress}%` }}
      onClick={onClick}
    >
      <span className="back-to-top-arrow">{mode === "bottom" ? "↓" : "↑"}</span>
      <span className="back-to-top-progress">{progress}%</span>
    </button>
  );
}
