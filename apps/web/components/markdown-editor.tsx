"use client";

import { useEffect, useRef } from "react";
import { getAccessToken } from "@/lib/auth";
import { toZhError } from "@/lib/errors";

type MarkdownEditorProps = {
  value: string;
  onChange: (next: string) => void;
  placeholder?: string;
};

type ToastEditorInstance = {
  getMarkdown: () => string;
  setMarkdown: (markdown: string, cursorToEnd?: boolean) => void;
  destroy: () => void;
};

export function MarkdownEditor({ value, onChange, placeholder }: MarkdownEditorProps) {
  const hostRef = useRef<HTMLDivElement | null>(null);
  const editorRef = useRef<ToastEditorInstance | null>(null);
  const onChangeRef = useRef(onChange);
  const syncingRef = useRef(false);

  useEffect(() => {
    onChangeRef.current = onChange;
  }, [onChange]);

  useEffect(() => {
    let cancelled = false;

    const mount = async () => {
      if (!hostRef.current) return;
      const mod = await import("@toast-ui/editor");
      if (cancelled || !hostRef.current) return;
      const editor = new mod.Editor({
        el: hostRef.current,
        initialValue: value || "",
        initialEditType: "markdown",
        previewStyle: "vertical",
        height: "560px",
        autofocus: false,
        usageStatistics: false,
        placeholder: placeholder || "请输入 Markdown 内容",
        hooks: {
          addImageBlobHook: (blob: Blob, callback: (url: string, text?: string) => void) => {
            void (async () => {
              try {
                const url = await uploadImageBlob(blob);
                callback(url, "image");
                // 强制把最新内容同步回上层，避免“界面变了但保存值没变”的情况。
                if (editorRef.current) {
                  onChangeRef.current(editorRef.current.getMarkdown());
                }
              } catch (err) {
                const msg = toZhError(err, "上传失败");
                window.alert(msg);
              }
            })();
            return false;
          }
        },
        events: {
          change: () => {
            if (syncingRef.current || !editorRef.current) return;
            onChangeRef.current(editorRef.current.getMarkdown());
          }
        }
      }) as unknown as ToastEditorInstance;
      editorRef.current = editor;
    };

    void mount();
    return () => {
      cancelled = true;
      if (editorRef.current) {
        editorRef.current.destroy();
        editorRef.current = null;
      }
    };
  }, []);

  useEffect(() => {
    const ins = editorRef.current;
    if (!ins) return;
    const current = ins.getMarkdown();
    if (current === value) return;
    syncingRef.current = true;
    ins.setMarkdown(value || "", false);
    syncingRef.current = false;
  }, [value]);

  return (
    <div>
      <div className="md-helper-bar">
        <button type="button" className="ghost md-helper-btn" onClick={() => insertImageTemplate("default")}>
          插入图片
        </button>
        <button type="button" className="ghost md-helper-btn" onClick={() => insertImageTemplate("center")}>
          居中图片
        </button>
        <button type="button" className="ghost md-helper-btn" onClick={() => insertImageTemplate("right")}>
          右对齐图片
        </button>
        <button type="button" className="ghost md-helper-btn" onClick={() => insertImageTemplate("size")}>
          固定宽度
        </button>
      </div>
      <div className="md-advanced-wrap">
        <div ref={hostRef} />
      </div>
    </div>
  );

  function insertImageTemplate(kind: "default" | "center" | "right" | "size") {
    const ins = editorRef.current as unknown as {
      replaceSelection?: (text: string) => void;
      getMarkdown: () => string;
      setMarkdown: (markdown: string, cursorToEnd?: boolean) => void;
    } | null;
    if (!ins) return;

    const snippet = (() => {
      if (kind === "default") return `\n![图片说明](https://your-image-url)\n`;
      if (kind === "center") {
        return `\n<p align="center">\n  <img src="https://your-image-url" alt="图片说明" width="520" />\n</p>\n`;
      }
      if (kind === "right") {
        return `\n<p align="right">\n  <img src="https://your-image-url" alt="图片说明" width="320" />\n</p>\n`;
      }
      return `\n<img src="https://your-image-url" alt="图片说明" width="420" />\n`;
    })();

    if (typeof ins.replaceSelection === "function") {
      ins.replaceSelection(snippet);
      onChangeRef.current(ins.getMarkdown());
      return;
    }

    const next = `${ins.getMarkdown()}\n${snippet}`;
    ins.setMarkdown(next, false);
    onChangeRef.current(next);
  }
}

async function uploadImageBlob(blob: Blob): Promise<string> {
  const token = getAccessToken();
  if (!token) {
    throw new Error("请先登录后再上传图片");
  }
  const file = blob instanceof File ? blob : new File([blob], `image-${Date.now()}.png`, { type: blob.type || "image/png" });
  const form = new FormData();
  form.append("file", file, file.name);

  const API_BASE = process.env.NEXT_PUBLIC_API_BASE || "http://127.0.0.1:8080";
  const resp = await fetch(`${API_BASE}/api/v1/uploads/image`, {
    method: "POST",
    headers: {
      Authorization: `Bearer ${token}`
    },
    body: form
  });
  const json = (await resp.json()) as { code: number; message: string; data?: { url?: string } };
  if (!resp.ok || json.code !== 0 || !json.data?.url) {
    throw new Error(toZhError(json.message || "", "上传图片失败"));
  }
  return json.data.url;
}
