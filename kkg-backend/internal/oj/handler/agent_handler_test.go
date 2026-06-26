package handler

import (
	"strings"
	"testing"
)

func TestParseAgentReplyPrefersCodeFieldFromFencedJSON(t *testing.T) {
	reply := "```json\n" + `{
  "title": "题解：最小跨度",
  "summary": "动态规划",
  "markdown": "说明里的示例代码不应被提交。\n` + "```go\\npackage main\\nfunc broken(){ _ = dpPrev }\\n```" + `",
  "code": "package main\n\nfunc main() {\n\tprintln(\"accepted\")\n}"
}` + "\n```"

	title, summary, markdown, code := parseAgentReply(reply, "原题")

	if title != "题解：最小跨度" {
		t.Fatalf("title = %q", title)
	}
	if summary != "动态规划" {
		t.Fatalf("summary = %q", summary)
	}
	if !strings.Contains(markdown, "示例代码") {
		t.Fatalf("markdown was not parsed from JSON field: %q", markdown)
	}
	if strings.Contains(code, "dpPrev") {
		t.Fatalf("code was extracted from markdown block instead of code field: %q", code)
	}
	if !strings.Contains(code, `println("accepted")`) {
		t.Fatalf("unexpected code: %q", code)
	}
}

func TestExtractAgentJSONFromTextWrapper(t *testing.T) {
	raw := `模型输出如下：
{
  "title": "A",
  "summary": "B",
  "markdown": "C",
  "code": "D"
}
请查收。`

	_, _, _, code := parseAgentReply(raw, "原题")
	if code != "D" {
		t.Fatalf("code = %q", code)
	}
}
