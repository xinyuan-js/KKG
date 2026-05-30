"use client";

import { useEffect, useRef, useState } from "react";
import { AUTH_CHANGED_EVENT, getUserProfile } from "@/lib/auth";
import { ojGetLoginUser } from "@/lib/oj-api";
import { emitTopNotice } from "@/lib/notice";

const OJ_EVENT_BASE = process.env.NEXT_PUBLIC_OJ_API_BASE || "/oj-api";

export function OJSubmitNotifier() {
  const [enabled, setEnabled] = useState(false);
  const seenRef = useRef<Set<string>>(new Set());
  const reconnectRef = useRef<number | null>(null);

  useEffect(() => {
    const sync = () => setEnabled(!!getUserProfile());
    sync();
    window.addEventListener(AUTH_CHANGED_EVENT, sync);
    window.addEventListener("storage", sync);
    return () => {
      window.removeEventListener(AUTH_CHANGED_EVENT, sync);
      window.removeEventListener("storage", sync);
    };
  }, []);

  useEffect(() => {
    if (!enabled) return;
    let closed = false;
    let es: EventSource | null = null;

    const connect = async () => {
      try {
        // 先确认 OJ 可基于统一 auth cookie 建立用户上下文，SSE 再复用 cookie。
        await ojGetLoginUser();
      } catch {
        // ignore; below EventSource may still succeed if session already exists
      }
      if (closed) return;
      es = new EventSource(`${OJ_EVENT_BASE}/api/question/submission/events`, { withCredentials: true });
      es.addEventListener("submission", (evt) => {
        try {
          const data = JSON.parse((evt as MessageEvent).data || "{}") as {
            submitId?: number; status?: number; message?: string; score?: number;
          };
          const submitID = Number(data.submitId || 0);
          const status = Number(data.status || 0);
          if (!submitID || status <= 1) return;
          const dedupKey = `${submitID}-${status}`;
          if (seenRef.current.has(dedupKey)) return;
          seenRef.current.add(dedupKey);
          const score = Number(data.score || 0);
          const text = `提交 #${submitID} 判题完成：${statusText(status, data.message || "")}（${score} 分）`;
          emitTopNotice(text, status === 2 ? "success" : "error");
        } catch {
          // ignore malformed payload
        }
      });
      es.onerror = () => {
        if (closed) return;
        es?.close();
        if (reconnectRef.current) window.clearTimeout(reconnectRef.current);
        reconnectRef.current = window.setTimeout(() => {
          void connect();
        }, 2000);
      };
    };

    void connect();
    return () => {
      closed = true;
      if (reconnectRef.current) window.clearTimeout(reconnectRef.current);
      es?.close();
    };
  }, [enabled]);

  useEffect(() => () => {
    if (reconnectRef.current) window.clearTimeout(reconnectRef.current);
  }, []);

  useEffect(() => {
    const onFallbackEvent = (evt: Event) => {
      const detail = (evt as CustomEvent<{ submitId: number; status: number; message?: string; score?: number }>).detail;
      if (!detail) return;
      const submitID = Number(detail.submitId || 0);
      const status = Number(detail.status || 0);
      if (!submitID || status <= 1) return;
      const dedupKey = `${submitID}-${status}`;
      if (seenRef.current.has(dedupKey)) return;
      seenRef.current.add(dedupKey);
      const score = Number(detail.score || 0);
      const text = `提交 #${submitID} 判题完成：${statusText(status, detail.message || "")}（${score} 分）`;
      emitTopNotice(text, status === 2 ? "success" : "error");
    };
    window.addEventListener("oj:submission-final", onFallbackEvent as EventListener);
    return () => window.removeEventListener("oj:submission-final", onFallbackEvent as EventListener);
  }, []);
  return null;
}

function statusText(status: number, message: string) {
  if (status === 2) return "AC";
  const m = (message || "").toLowerCase();
  if (m.includes("wrong answer")) return "WA";
  if (m.includes("compile")) return "CE";
  if (m.includes("runtime")) return "RE";
  if (m.includes("time")) return "TLE";
  if (m.includes("memory")) return "MLE";
  return "Failed";
}
