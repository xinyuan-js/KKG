"use client";

import { createPostComment, getPostComments, type Comment } from "@/lib/api";
import { getAccessToken, getUserProfile } from "@/lib/auth";
import { toZhError } from "@/lib/errors";
import Link from "next/link";
import { FormEvent, useEffect, useMemo, useState } from "react";

type Props = {
  postID: number;
};

type GroupedComments = {
  roots: Comment[];
  repliesByRoot: Map<number, Comment[]>;
  byID: Map<number, Comment>;
};

export function PostComments({ postID }: Props) {
  const [comments, setComments] = useState<Comment[]>([]);
  const [loading, setLoading] = useState(true);
  const [submitting, setSubmitting] = useState(false);
  const [content, setContent] = useState("");
  const [error, setError] = useState("");
  const [replyingTo, setReplyingTo] = useState<number>(0);
  const [replyContent, setReplyContent] = useState("");
  const [expandedReplyRoots, setExpandedReplyRoots] = useState<Record<number, boolean>>({});
  const [expandedContent, setExpandedContent] = useState<Record<number, boolean>>({});
  const [authReady, setAuthReady] = useState(false);
  const [token, setToken] = useState("");
  const [username, setUsername] = useState("");

  useEffect(() => {
    const syncAuth = () => {
      const t = getAccessToken() || "";
      const p = getUserProfile();
      setToken(t);
      setUsername(p?.username || "当前用户");
      setAuthReady(true);
    };
    syncAuth();
    window.addEventListener("storage", syncAuth);
    window.addEventListener("focus", syncAuth);
    return () => {
      window.removeEventListener("storage", syncAuth);
      window.removeEventListener("focus", syncAuth);
    };
  }, []);

  useEffect(() => {
    void loadComments();
  }, [postID]);

  async function loadComments() {
    setLoading(true);
    try {
      const data = await getPostComments(postID);
      setComments(data);
    } catch (err) {
      setError(toZhError(err, "加载评论失败"));
    } finally {
      setLoading(false);
    }
  }

  const grouped = useMemo(() => groupComments(comments), [comments]);

  async function onSubmit(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    if (!token) {
      setError("请先登录后再评论");
      return;
    }
    setSubmitting(true);
    setError("");
    try {
      await createPostComment(token, postID, { content });
      setContent("");
      await loadComments();
    } catch (err) {
      setError(toZhError(err, "评论失败"));
    } finally {
      setSubmitting(false);
    }
  }

  async function onReply(parentID: number) {
    if (!token) {
      setError("请先登录后再回复");
      return;
    }
    setSubmitting(true);
    setError("");
    try {
      await createPostComment(token, postID, { content: replyContent, parent_id: parentID });
      setReplyContent("");
      setReplyingTo(0);
      await loadComments();
    } catch (err) {
      setError(toZhError(err, "回复失败"));
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <section className="detail-card comment-section" style={{ marginTop: 14 }}>
      <div className="comment-section-head">
        <h2 className="comment-title">评论区</h2>
        <span className="comment-count">{comments.length} 条讨论</span>
      </div>
      {!authReady ? (
        <p className="tip">登录状态检查中...</p>
      ) : !token ? (
        <p className="tip">
          <Link href="/login">登录</Link> 后可以评论和回复。
        </p>
      ) : (
        <form onSubmit={onSubmit} className="comment-form">
          <textarea
            className="comment-input"
            placeholder={`以 ${username || "当前用户"} 身份发表评论...`}
            value={content}
            maxLength={2000}
            onChange={(e) => setContent(e.target.value)}
          />
          <button type="submit" disabled={submitting || !content.trim()}>
            {submitting ? "提交中..." : "发表评论"}
          </button>
        </form>
      )}

      {error ? <p className="error">{error}</p> : null}
      {loading ? <p className="tip">评论加载中...</p> : null}
      {!loading && grouped.roots.length === 0 ? <p className="tip">还没有评论，来抢首评吧。</p> : null}

      <div className="comment-list">
        {grouped.roots.map((root) => {
          const rootAuthor = getAuthor(root);
          const replies = grouped.repliesByRoot.get(root.id) || [];
          const rootExpanded = !!expandedReplyRoots[root.id];
          const visibleReplies = rootExpanded ? replies : replies.slice(0, 3);
          const rootReplying = replyingTo === root.id;
          return (
            <div className="comment-item" key={root.id}>
              <div className="comment-head">
                {root.author_avatar_url ? (
                  <img className="tweet-avatar" src={root.author_avatar_url} alt={rootAuthor} />
                ) : (
                  <span className="tweet-avatar tweet-avatar-fallback">{rootAuthor.slice(0, 1).toUpperCase()}</span>
                )}
                <strong className="tweet-author">{rootAuthor}</strong>
                <span className="tweet-time">{new Date(root.created_at).toLocaleString()}</span>
              </div>
              <div className="comment-body">
                <CommentText
                  comment={root}
                  expanded={!!expandedContent[root.id]}
                  onToggle={() =>
                    setExpandedContent((prev) => ({
                      ...prev,
                      [root.id]: !prev[root.id]
                    }))
                  }
                />
              </div>
              {token ? (
                <div className="comment-actions">
                  <button
                    type="button"
                    className="ghost comment-reply-btn"
                    onClick={() => setReplyingTo(rootReplying ? 0 : root.id)}
                  >
                    {rootReplying ? "取消回复" : "回复"}
                  </button>
                </div>
              ) : null}
              {rootReplying ? (
                <div className="comment-reply-form">
                  <textarea
                    className="comment-input"
                    placeholder={`回复 @${rootAuthor}...`}
                    value={replyContent}
                    maxLength={2000}
                    onChange={(e) => setReplyContent(e.target.value)}
                  />
                  <button type="button" disabled={submitting || !replyContent.trim()} onClick={() => void onReply(root.id)}>
                    {submitting ? "提交中..." : "提交回复"}
                  </button>
                </div>
              ) : null}

              {replies.length > 0 ? (
                <div className="comment-replies">
                  {visibleReplies.map((reply) => {
                    const author = getAuthor(reply);
                    const target = getReplyTargetName(reply, grouped.byID, rootAuthor);
                    const replying = replyingTo === reply.id;
                    return (
                      <div className="comment-reply-item" key={reply.id}>
                        <div className="comment-head">
                          {reply.author_avatar_url ? (
                            <img className="tweet-avatar" src={reply.author_avatar_url} alt={author} />
                          ) : (
                            <span className="tweet-avatar tweet-avatar-fallback">
                              {author.slice(0, 1).toUpperCase()}
                            </span>
                          )}
                          <strong className="tweet-author">{author}</strong>
                          <span className="tweet-time">{new Date(reply.created_at).toLocaleString()}</span>
                        </div>
                        <div className="comment-body">
                          <CommentText
                            comment={reply}
                            prefix={`@${target} `}
                            expanded={!!expandedContent[reply.id]}
                            onToggle={() =>
                              setExpandedContent((prev) => ({
                                ...prev,
                                [reply.id]: !prev[reply.id]
                              }))
                            }
                          />
                        </div>
                        {token ? (
                          <div className="comment-actions">
                            <button
                              type="button"
                              className="ghost comment-reply-btn"
                              onClick={() => setReplyingTo(replying ? 0 : reply.id)}
                            >
                              {replying ? "取消回复" : "回复"}
                            </button>
                          </div>
                        ) : null}
                        {replying ? (
                          <div className="comment-reply-form">
                            <textarea
                              className="comment-input"
                              placeholder={`回复 @${author}...`}
                              value={replyContent}
                              maxLength={2000}
                              onChange={(e) => setReplyContent(e.target.value)}
                            />
                            <button
                              type="button"
                              disabled={submitting || !replyContent.trim()}
                              onClick={() => void onReply(reply.id)}
                            >
                              {submitting ? "提交中..." : "提交回复"}
                            </button>
                          </div>
                        ) : null}
                      </div>
                    );
                  })}
                  {replies.length > 3 ? (
                    <button
                      type="button"
                      className="ghost comment-more-btn"
                      onClick={() =>
                        setExpandedReplyRoots((prev) => ({
                          ...prev,
                          [root.id]: !prev[root.id]
                        }))
                      }
                    >
                      {rootExpanded ? "收起回复" : `显示更多回复（+${replies.length - visibleReplies.length}）`}
                    </button>
                  ) : null}
                </div>
              ) : null}
            </div>
          );
        })}
      </div>
    </section>
  );
}

function getAuthor(c: Comment): string {
  return c.author_name || `用户${c.user_id}`;
}

function getReplyTargetName(reply: Comment, byID: Map<number, Comment>, fallback: string): string {
  if (!reply.parent_id) return fallback;
  const parent = byID.get(reply.parent_id);
  if (!parent) return fallback;
  return getAuthor(parent);
}

function CommentText(props: {
  comment: Comment;
  prefix?: string;
  expanded: boolean;
  onToggle: () => void;
}) {
  const { comment, prefix = "", expanded, onToggle } = props;
  const body = comment.content;
  const needFold = body.length > 180;
  const shownBody = !needFold || expanded ? body : `${body.slice(0, 180)}...`;
  return (
    <div>
      <p className="comment-content">
        {prefix ? <span className="comment-at">{prefix.trim()}</span> : null}
        {shownBody}
      </p>
      {needFold ? (
        <button type="button" className="ghost comment-more-btn" onClick={onToggle}>
          {expanded ? "收起" : "显示更多"}
        </button>
      ) : null}
    </div>
  );
}

function groupComments(input: Comment[]): GroupedComments {
  const byID = new Map<number, Comment>();
  for (const c of input) {
    byID.set(c.id, c);
  }

  const roots = input.filter((c) => !c.parent_id || !byID.has(c.parent_id));
  const rootIDSet = new Set(roots.map((r) => r.id));

  const repliesByRoot = new Map<number, Comment[]>();
  for (const root of roots) {
    repliesByRoot.set(root.id, []);
  }

  for (const c of input) {
    if (rootIDSet.has(c.id)) {
      continue;
    }
    const rootID = findRootID(c, byID, rootIDSet);
    if (!rootID) {
      continue;
    }
    const arr = repliesByRoot.get(rootID) || [];
    arr.push(c);
    repliesByRoot.set(rootID, arr);
  }

  for (const [k, arr] of repliesByRoot) {
    arr.sort((a, b) => new Date(a.created_at).getTime() - new Date(b.created_at).getTime());
    repliesByRoot.set(k, arr);
  }

  return { roots, repliesByRoot, byID };
}

function findRootID(c: Comment, byID: Map<number, Comment>, rootIDSet: Set<number>): number {
  let cur: Comment | undefined = c;
  const visited = new Set<number>();
  while (cur) {
    if (rootIDSet.has(cur.id)) {
      return cur.id;
    }
    if (!cur.parent_id) {
      return cur.id;
    }
    if (visited.has(cur.id)) {
      return 0;
    }
    visited.add(cur.id);
    cur = byID.get(cur.parent_id);
  }
  return 0;
}
