export type TopNoticeTone = "success" | "error";

export function emitTopNotice(text: string, tone: TopNoticeTone = "success") {
  if (typeof window === "undefined" || !text) return;
  window.dispatchEvent(new CustomEvent("app:top-notice", { detail: { text, tone } }));
}

