import Link from "next/link";
import { Nav } from "@/components/nav";
import { Avatar } from "@/components/avatar";
import { getPublicUserPage } from "@/lib/api";
import { toZhError } from "@/lib/errors";

type PageProps = {
  params: {
    id: string;
  };
};

export default async function UserPage({ params }: PageProps) {
  const id = Number(params.id);
  if (!Number.isFinite(id) || id <= 0) {
    return (
      <main className="page">
        <Nav />
        <div className="card">
          <h1 className="title">用户不存在</h1>
          <p className="meta">无效的用户 ID</p>
          <Link className="tweet-link" href="/">
            返回首页
          </Link>
        </div>
      </main>
    );
  }

  try {
    const data = await getPublicUserPage(id);
    return (
      <main className="page">
        <Nav />
        <section className="card user-hero">
          <Avatar
            className="user-hero-avatar"
            fallbackClassName="user-hero-avatar user-hero-avatar-fallback"
            src={data.avatar_url}
            name={data.username}
            loading="eager"
          />
          <div>
            <h1 className="detail-title" style={{ margin: 0 }}>
              {data.username}
            </h1>
            <p className="meta" style={{ marginTop: 8 }}>
              用户 ID: {data.id} | 发布推文: {data.published_count} | 注册时间: {formatTime(data.created_at)}
            </p>
          </div>
        </section>

        <h2 style={{ marginTop: 0 }}>TA 的推文</h2>
        {data.posts.length === 0 ? <div className="card">暂无已发布推文</div> : null}
        <section className="feed">
          {data.posts.map((post) => (
            <Link key={post.id} href={`/posts/${post.id}`} className="tweet-card-link">
              <article className="tweet-card" style={{ opacity: 1, transform: "none", animation: "none" }}>
                <div className="tweet-head">
                  <div className="tweet-author-block">
                    <Avatar className="tweet-avatar" fallbackClassName="tweet-avatar tweet-avatar-fallback" src={post.author_avatar_url} name={data.username} />
                    <strong className="tweet-author">{data.username}</strong>
                  </div>
                  <span className="tweet-time">{formatTime(post.publish_at || post.updated_at)}</span>
                </div>
                <h3 className="tweet-title">{post.title}</h3>
                {post.summary?.trim() ? <p className="tweet-body">{post.summary}</p> : null}
              </article>
            </Link>
          ))}
        </section>
      </main>
    );
  } catch (error) {
    const message = toZhError(error, "获取用户信息失败");
    return (
      <main className="page">
        <Nav />
        <div className="card">
          <h1 className="title">用户页暂不可用</h1>
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
  if (!raw) return "未知";
  return new Date(raw).toLocaleString();
}
