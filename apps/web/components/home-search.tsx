"use client";

import { searchAll, searchSuggest, type SearchPostItem, type SearchUserItem } from "@/lib/api";
import { getAccessToken } from "@/lib/auth";
import { toZhError } from "@/lib/errors";
import Link from "next/link";
import { FormEvent, KeyboardEvent, useEffect, useMemo, useState } from "react";

type SearchType = "post" | "user";

export function HomeSearch() {
  const [searchType, setSearchType] = useState<SearchType>("post");
  const [query, setQuery] = useState("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const [postResult, setPostResult] = useState<SearchPostItem[]>([]);
  const [userResult, setUserResult] = useState<SearchUserItem[]>([]);
  const [searched, setSearched] = useState(false);
  const [suggestions, setSuggestions] = useState<string[]>([]);
  const [activeIndex, setActiveIndex] = useState(-1);

  const canSearch = useMemo(() => query.trim().length > 0, [query]);

  useEffect(() => {
    const kw = query.trim();
    if (!kw) {
      setSuggestions([]);
      setActiveIndex(-1);
      return;
    }
    const timer = window.setTimeout(async () => {
      try {
        const data = await searchSuggest({
          type: searchType,
          q: kw,
          limit: 5,
          token: getAccessToken() || undefined
        });
        setSuggestions(data.items || []);
        setActiveIndex(-1);
      } catch {
        setSuggestions([]);
        setActiveIndex(-1);
      }
    }, 220);
    return () => window.clearTimeout(timer);
  }, [query, searchType]);

  async function onSubmit(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    if (!canSearch) return;
    setSuggestions([]);
    setActiveIndex(-1);
    await triggerSearch(query.trim());
  }

  function onInputKeyDown(e: KeyboardEvent<HTMLInputElement>) {
    if (!suggestions.length) return;
    if (e.key === "ArrowDown") {
      e.preventDefault();
      setActiveIndex((prev) => (prev + 1) % suggestions.length);
      return;
    }
    if (e.key === "ArrowUp") {
      e.preventDefault();
      setActiveIndex((prev) => (prev <= 0 ? suggestions.length - 1 : prev - 1));
      return;
    }
    if (e.key === "Escape") {
      setSuggestions([]);
      setActiveIndex(-1);
      return;
    }
    if (e.key === "Enter" && activeIndex >= 0 && activeIndex < suggestions.length) {
      e.preventDefault();
      const value = suggestions[activeIndex];
      setQuery(value);
      setSuggestions([]);
      setActiveIndex(-1);
      void triggerSearch(value);
    }
  }

  async function triggerSearch(kw: string) {
    setLoading(true);
    setError("");
    try {
      const data = await searchAll({
        type: searchType,
        q: kw.trim(),
        limit: 5,
        token: getAccessToken() || undefined
      });
      if (searchType === "post") {
        setPostResult((data.items as SearchPostItem[]) || []);
      } else {
        setUserResult((data.items as SearchUserItem[]) || []);
      }
      setSearched(true);
    } catch (err) {
      setError(toZhError(err, "搜索失败"));
      setPostResult([]);
      setUserResult([]);
      setSearched(true);
    } finally {
      setLoading(false);
    }
  }

  const count = searchType === "post" ? postResult.length : userResult.length;

  function clearSearch() {
    setQuery("");
    setSuggestions([]);
    setActiveIndex(-1);
    setPostResult([]);
    setUserResult([]);
    setSearched(false);
    setError("");
  }

  return (
    <section className="card home-search-card">
      <form className="home-search-form" onSubmit={onSubmit}>
        <div className="home-search-type seg-switch seg-switch-2" data-active={searchType === "post" ? 0 : 1}>
          <span className="seg-switch-thumb" aria-hidden="true" />
          <button type="button" className={searchType === "post" ? "" : "ghost"} onClick={() => setSearchType("post")}>
            文章
          </button>
          <button type="button" className={searchType === "user" ? "" : "ghost"} onClick={() => setSearchType("user")}>
            用户
          </button>
        </div>
        <input
          className="home-search-input"
          placeholder={searchType === "post" ? "搜索全站文章标题和摘要" : "搜索用户（自动排除自己）"}
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          onKeyDown={onInputKeyDown}
        />
        <button type="button" className="ghost" onClick={clearSearch} disabled={!query && !searched}>
          清空
        </button>
        <button type="submit" disabled={!canSearch || loading}>
          {loading ? "搜索中..." : "搜索"}
        </button>
      </form>
      {suggestions.length > 0 ? (
        <div className="home-search-suggest">
          {suggestions.map((text, idx) => (
            <button
              key={`${text}-${idx}`}
              type="button"
              className={`home-search-suggest-item${activeIndex === idx ? " active" : ""}`}
              onMouseDown={() => {
                setQuery(text);
                setSuggestions([]);
                setActiveIndex(-1);
                void triggerSearch(text);
              }}
            >
              {text}
            </button>
          ))}
        </div>
      ) : null}

      {error ? <p className="error" style={{ margin: "8px 0 0" }}>{error}</p> : null}
      {searched ? <p className="tip" style={{ margin: "8px 0 0" }}>找到 {count} 条结果</p> : null}

      {searchType === "post" && postResult.length > 0 ? (
        <div className="home-search-results">
          {postResult.map((post) => (
            <Link key={post.id} href={`/posts/${post.id}`} className="home-search-item">
              <strong>{post.title}</strong>
              <span className="meta">{post.summary || "（无摘要）"}</span>
              <span className="meta">score {Number(post.score || 0).toFixed(2)}</span>
            </Link>
          ))}
        </div>
      ) : null}

      {searchType === "user" && userResult.length > 0 ? (
        <div className="home-search-results">
          {userResult.map((user) => (
            <Link key={user.id} href={`/users/${user.id}`} className="home-search-item">
              <strong>{user.username}</strong>
              <span className="meta">{user.email}</span>
              <span className="meta">score {Number(user.score || 0).toFixed(2)}</span>
            </Link>
          ))}
        </div>
      ) : null}
    </section>
  );
}
