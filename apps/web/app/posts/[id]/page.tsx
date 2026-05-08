import Link from "next/link";
import { Nav } from "@/components/nav";
import { getPublishedPostDetail } from "@/lib/api";
import { PostComments } from "@/components/post-comments";
import { PostEngagementBar } from "@/components/post-engagement";
import { toZhError } from "@/lib/errors";
import ReactMarkdown from "react-markdown";
import { defaultUrlTransform } from "react-markdown";
import remarkGfm from "remark-gfm";
import rehypeRaw from "rehype-raw";
import rehypeSanitize, { defaultSchema } from "rehype-sanitize";

type PageProps = {
  params: {
    id: string;
  };
};

export default async function PostDetailPage({ params }: PageProps) {
  const id = Number(params.id);
  if (!Number.isFinite(id) || id <= 0) {
    return (
      <main className="page">
        <Nav />
        <div className="card">
          <h1 className="title">推文不存在</h1>
          <p className="meta">无效的文章 ID</p>
          <Link className="tweet-link" href="/">
            返回首页
          </Link>
        </div>
      </main>
    );
  }

  try {
    const post = await getPublishedPostDetail(id);
    const content = normalizeBrokenImageSyntax(post.raw_content || "");
    const cover = extractFirstImage(content);
    const authorName = post.author_name || `u/${post.slug.split("-u")[0]}`;
    const questionId = extractQuestionId(post.tags || [], post.summary || "");

    return (
      <main className="page">
        <Nav />
        <section className="detail-layout">
          <article className="detail-card">
            <div className="detail-head">
              <span className="detail-slug">#{post.slug}</span>
              <span className="detail-time">{formatTime(post.publish_at || post.updated_at)}</span>
            </div>
            <h1 className="detail-title">{post.title}</h1>
            <div className="detail-meta-row">
              <Link href={`/users/${post.author_id}`} className="detail-author">
                {post.author_avatar_url ? (
                  <img className="tweet-avatar" src={post.author_avatar_url} alt={authorName} />
                ) : (
                  <span className="tweet-avatar tweet-avatar-fallback">{authorName.slice(0, 1).toUpperCase()}</span>
                )}
                <span className="tweet-author">{authorName}</span>
              </Link>
            </div>
            {(post.tags || []).length > 0 ? (
              <div className="tweet-meta-tags">
                {(post.tags || []).map((t) => (
                  <span key={`${post.id}-${t}`} className="tweet-tag soft">
                    {t}
                  </span>
                ))}
              </div>
            ) : null}
            {post.summary?.trim() ? <p className="detail-lead">{post.summary}</p> : null}
            {cover ? <img className="detail-cover" src={cover} alt={post.title} loading="lazy" /> : null}
            <div className="detail-content-wrap">
              <div className="detail-content markdown-body">
                <ReactMarkdown
                  remarkPlugins={[remarkGfm]}
                  urlTransform={(url, key) => {
                    if (key === "src" && /^data:image\/[a-zA-Z0-9.+-]+;base64,/i.test(url)) {
                      return url;
                    }
                    return defaultUrlTransform(url);
                  }}
                  rehypePlugins={[
                    rehypeRaw,
                    [
                      rehypeSanitize,
                      {
                        ...defaultSchema,
                        protocols: {
                          ...(defaultSchema.protocols || {}),
                          src: ["http", "https", "data"]
                        },
                        attributes: {
                          ...(defaultSchema.attributes || {}),
                          img: [
                            ...(((defaultSchema.attributes || {}).img as string[]) || []),
                            "src",
                            "alt",
                            "title",
                            "width",
                            "height"
                          ]
                        }
                      }
                    ]
                  ]}
                >
                  {content || "（暂无正文）"}
                </ReactMarkdown>
              </div>
            </div>
            <div className="detail-foot">
              <Link className="tweet-link" href="/">
                返回推文流
              </Link>
            </div>
          </article>
          <aside className="detail-side">
            <div className="card detail-side-card">
              <h3 className="detail-side-title">文章信息</h3>
              <div className="detail-side-row">
                <span className="meta">作者</span>
                <Link href={`/users/${post.author_id}`} className="detail-side-value">
                  {authorName}
                </Link>
              </div>
              <div className="detail-side-row">
                <span className="meta">发布时间</span>
                <span className="detail-side-value">{formatTime(post.publish_at || post.updated_at)}</span>
              </div>
              {questionId ? (
                <div className="detail-side-row">
                  <span className="meta">关联题目</span>
                  <Link href={`/oj/questions/${questionId}`} className="detail-side-value">
                    #{questionId}
                  </Link>
                </div>
              ) : null}
              {(post.tags || []).length > 0 ? (
                <div className="detail-side-tags">
                  {(post.tags || []).map((t) => (
                    <span key={`side-${post.id}-${t}`} className="tweet-tag soft">
                      {t}
                    </span>
                  ))}
                </div>
              ) : null}
              <Link className="tweet-link detail-side-btn" href="/">
                返回首页
              </Link>
            </div>
            {questionId ? (
              <Link className="question-jump-mini" href={`/oj/questions/${questionId}`}>
                <span className="question-jump-mini-label">题目跳转</span>
                <strong className="question-jump-mini-title">#{questionId}</strong>
                <span className="question-jump-mini-arrow" aria-hidden="true">↗</span>
              </Link>
            ) : null}
          </aside>
        </section>
        <PostEngagementBar postID={post.id} />
        <PostComments postID={post.id} />
      </main>
    );
  } catch (error) {
    const message = toZhError(error, "获取详情失败");
    return (
      <main className="page">
        <Nav />
        <div className="card">
          <h1 className="title">推文暂不可用</h1>
          <p className="error">{message}</p>
          <Link className="tweet-link" href="/">
            返回首页
          </Link>
        </div>
      </main>
    );
  }
}

function formatTime(raw?: string) {
  if (!raw) return "刚刚";
  return new Date(raw).toLocaleString();
}

function normalizeBrokenImageSyntax(raw: string) {
  return raw.replace(/!\[([^\]]*)]\s*\r?\n\s*\(\s*([^)]+?)\s*\)/g, "![$1]($2)");
}

function extractFirstImage(raw?: string) {
  if (!raw) return "";
  const mdMatch = raw.match(/!\[[^\]]*]\(\s*([^)]+?)\s*\)/m);
  if (mdMatch?.[1]) return mdMatch[1].trim();
  const htmlMatch = raw.match(/<img[^>]*src=["']([^"']+)["'][^>]*>/i);
  if (htmlMatch?.[1]) return htmlMatch[1].trim();
  return "";
}

function extractQuestionId(tags: string[], summary: string) {
  for (const tag of tags) {
    const trimmed = (tag || "").trim();
    const m = trimmed.match(/^(?:q|question|题号)[\s:：#_-]*(\d+)$/i);
    if (m?.[1]) return Number(m[1]);
  }
  const s = (summary || "").trim();
  const m = s.match(/(?:题号|question|q)[\s:：#_-]*(\d+)/i);
  if (m?.[1]) return Number(m[1]);
  return 0;
}
