"use client";

import { ojAddQuestion } from "@/lib/oj-api";
import { toZhError } from "@/lib/errors";
import { ChangeEvent, FormEvent, useState } from "react";
import { useRouter } from "next/navigation";

export default function OJNewQuestionPage() {
  const router = useRouter();
  const [title, setTitle] = useState("");
  const [content, setContent] = useState("");
  const [tagsText, setTagsText] = useState("");
  const [answer, setAnswer] = useState("");
  const [sampleCases, setSampleCases] = useState<Array<{ input: string; output: string }>>([{ input: "", output: "" }]);
  const [judgeCases, setJudgeCases] = useState<Array<{ input: string; output: string }>>([{ input: "", output: "" }]);
  const [caseFileHint, setCaseFileHint] = useState("");
  const [timeLimit, setTimeLimit] = useState(1000);
  const [memoryLimit, setMemoryLimit] = useState(128000);
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);

  async function onSubmit(e: FormEvent) {
    e.preventDefault();
    setError("");
    setLoading(true);
    try {
      const id = await ojAddQuestion({
        title,
        content,
        tags: parseList(tagsText),
        sampleCase: sampleCases.filter((c) => c.input.trim() || c.output.trim()),
        answer,
        judgeCase: judgeCases.filter((c) => c.input.trim() || c.output.trim()),
        judgeConfig: { timeLimit, memoryLimit }
      });
      router.push(`/oj/questions/${id}`);
    } catch (err) {
      setError(toZhError(err, "创建题目失败"));
    } finally {
      setLoading(false);
    }
  }

  async function onUploadCaseFiles(e: ChangeEvent<HTMLInputElement>) {
    const files = Array.from(e.target.files || []);
    if (files.length === 0) return;
    const ins = new Map<string, File>();
    const outs = new Map<string, File>();
    for (const f of files) {
      const name = f.name.toLowerCase();
      if (name.endsWith(".in")) {
        ins.set(name.slice(0, -3), f);
      } else if (name.endsWith(".out")) {
        outs.set(name.slice(0, -4), f);
      }
    }
    const keys = Array.from(ins.keys()).filter((k) => outs.has(k)).sort();
    if (keys.length === 0) {
      setCaseFileHint("未匹配到成对的 .in / .out 文件");
      return;
    }
    const built: Array<{ input: string; output: string }> = [];
    for (const k of keys) {
      const inText = await ins.get(k)!.text();
      const outText = await outs.get(k)!.text();
      built.push({ input: inText, output: outText });
    }
    setJudgeCases(built);
    setCaseFileHint(`已导入 ${built.length} 组测试用例`);
  }

  return (
    <div className="reddit-layout">
      <section className="reddit-main">
        <section className="card">
          <h1 className="title">创建题目</h1>
          <form onSubmit={onSubmit}>
            <input placeholder="题目标题" value={title} onChange={(e) => setTitle(e.target.value)} />
            <input placeholder="标签（逗号分隔）" value={tagsText} onChange={(e) => setTagsText(e.target.value)} />
            <textarea className="meta-edit-summary" placeholder="题目内容" value={content} onChange={(e) => setContent(e.target.value)} />
            <textarea placeholder="参考答案（可选）" value={answer} onChange={(e) => setAnswer(e.target.value)} />
            <div className="section-gap">
              <div className="toolbar-row">
                <h3 className="title" style={{ fontSize: 16, margin: 0 }}>公开样例</h3>
                <button
                  type="button"
                  className="ghost"
                  onClick={() => setSampleCases((prev) => [...prev, { input: "", output: "" }])}
                >
                  新增样例
                </button>
              </div>
              {sampleCases.map((c, idx) => (
                <div key={idx} className="card" style={{ marginBottom: 0 }}>
                  <p className="meta" style={{ marginTop: 0 }}>样例 {idx + 1}</p>
                  <textarea
                    placeholder="输入样例"
                    value={c.input}
                    onChange={(e) => setSampleCases((prev) => prev.map((it, i) => (i === idx ? { ...it, input: e.target.value } : it)))}
                    style={{ minHeight: 90 }}
                  />
                  <textarea
                    placeholder="输出样例"
                    value={c.output}
                    onChange={(e) => setSampleCases((prev) => prev.map((it, i) => (i === idx ? { ...it, output: e.target.value } : it)))}
                    style={{ minHeight: 90 }}
                  />
                  {sampleCases.length > 1 ? (
                    <div className="toolbar-row">
                      <button type="button" className="ghost" onClick={() => setSampleCases((prev) => prev.filter((_, i) => i !== idx))}>删除样例</button>
                    </div>
                  ) : null}
                </div>
              ))}
            </div>

            <div className="section-gap">
              <div className="toolbar-row">
                <h3 className="title" style={{ fontSize: 16, margin: 0 }}>隐藏测试用例</h3>
                <label className="tweet-link" style={{ cursor: "pointer" }}>
                  导入 .in/.out
                  <input
                    type="file"
                    accept=".in,.out,text/plain"
                    multiple
                    onChange={onUploadCaseFiles}
                    style={{ display: "none" }}
                  />
                </label>
                <button
                  type="button"
                  className="ghost"
                  onClick={() => setJudgeCases((prev) => [...prev, { input: "", output: "" }])}
                >
                  新增用例
                </button>
              </div>
              {caseFileHint ? <p className="meta">{caseFileHint}</p> : null}
              {judgeCases.map((c, idx) => (
                <div key={idx} className="card" style={{ marginBottom: 0 }}>
                  <p className="meta" style={{ marginTop: 0 }}>用例 {idx + 1}</p>
                  <textarea
                    placeholder="输入样例"
                    value={c.input}
                    onChange={(e) => setJudgeCases((prev) => prev.map((it, i) => (i === idx ? { ...it, input: e.target.value } : it)))}
                    style={{ minHeight: 90 }}
                  />
                  <textarea
                    placeholder="输出样例"
                    value={c.output}
                    onChange={(e) => setJudgeCases((prev) => prev.map((it, i) => (i === idx ? { ...it, output: e.target.value } : it)))}
                    style={{ minHeight: 90 }}
                  />
                  {judgeCases.length > 1 ? (
                    <div className="toolbar-row">
                      <button type="button" className="ghost" onClick={() => setJudgeCases((prev) => prev.filter((_, i) => i !== idx))}>删除用例</button>
                    </div>
                  ) : null}
                </div>
              ))}
            </div>
            <div className="oj-grid-2">
              <label className="meta-edit-label">
                <span className="meta">时间限制(ms)</span>
                <input type="number" value={timeLimit} onChange={(e) => setTimeLimit(Number(e.target.value || 0))} />
              </label>
              <label className="meta-edit-label">
                <span className="meta">内存限制(kb)</span>
                <input type="number" value={memoryLimit} onChange={(e) => setMemoryLimit(Number(e.target.value || 0))} />
              </label>
            </div>
            <button type="submit" disabled={loading}>{loading ? "提交中..." : "创建题目"}</button>
          </form>
          {error ? <p className="error">{error}</p> : null}
        </section>
      </section>
      <aside className="reddit-side">
        <div className="community-card">
          <h3 style={{ margin: 0 }}>出题规范</h3>
          <p className="meta" style={{ marginTop: 8 }}>建议至少填写 1 组样例，限制参数保持合理范围。</p>
        </div>
      </aside>
    </div>
  );
}

function parseList(raw: string) {
  return raw.split(/[,，]/).map((x) => x.trim()).filter(Boolean);
}
