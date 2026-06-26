const API_BASE_CLIENT = process.env.NEXT_PUBLIC_API_BASE || "/blog-api";
const API_BASE_SERVER = process.env.API_BASE_SERVER || "http://127.0.0.1:8080";

function getAPIBase() {
  return typeof window === "undefined" ? API_BASE_SERVER : API_BASE_CLIENT;
}

async function refreshAuthSession() {
  const resp = await fetch(`${getAPIBase()}/api/v1/auth/refresh`, {
    method: "POST",
    credentials: "include"
  });
  return resp.ok;
}

async function apiFetch(input: RequestInfo | URL, init?: RequestInit) {
  const withCredentials: RequestInit = {
    ...init,
    credentials: "include",
    headers: withoutAuthHeader(init?.headers)
  };
  const resp = await fetch(input, withCredentials);
  const url = String(input);
  if (resp.status !== 401 || url.includes("/api/v1/auth/")) {
    return resp;
  }
  if (!(await refreshAuthSession())) {
    return resp;
  }
  return fetch(input, withCredentials);
}

function withoutAuthHeader(headers?: HeadersInit): HeadersInit | undefined {
  if (!headers) return undefined;
  const next = new Headers(headers);
  next.delete("Authorization");
  next.delete("authorization");
  return next;
}

type Envelope<T> = {
  code: number;
  message: string;
  data: T;
};

export type Post = {
  id: number;
  author_id: number;
  version?: number;
  published_version?: number;
  author_name?: string;
  author_avatar_url?: string;
  title: string;
  slug: string;
  summary: string;
  tags?: string[];
  raw_content?: string;
  status: string;
  publish_at?: string;
  comment_count?: number;
  feed_score?: number;
  updated_at: string;
};

export type FeedPayload = {
  type: "hot" | "latest" | "recommend";
  items: Post[];
  total: number;
  page: number;
  page_size: number;
};

export type RankingPayload = {
  period: "24h" | "7d" | "30d" | "all";
  items: Post[];
};

export type PostEngagement = {
  like_count: number;
  favorite_count: number;
  comment_count: number;
  liked: boolean;
  favorited: boolean;
};

export type LoginPayload = {
  access_token?: string;
  user: {
    id: number;
    username: string;
    email: string;
    avatar_url?: string;
    role: string;
  };
};

export type RegisterPayload = {
  id: number;
  username: string;
  email: string;
  avatar_url?: string;
};

export type UserProfilePayload = {
  id: number;
  username: string;
  email: string;
  avatar_url?: string;
  role: string;
};

export type AdminListPayload<T> = {
  items: T[];
  total: number;
  page: number;
  page_size: number;
};

export type PublicUserPagePayload = {
  id: number;
  username: string;
  avatar_url?: string;
  created_at: string;
  published_count: number;
  posts: Post[];
};

export type PostVersion = {
  id: number;
  post_id: number;
  version: number;
  draft_note?: string;
  title: string;
  summary: string;
  tags?: string[];
  raw_content?: string;
  html_content?: string;
  status: string;
  visibility?: string;
  created_at: string;
  updated_at?: string;
  operator_id: number;
};

export type Comment = {
  id: number;
  post_id: number;
  user_id: number;
  parent_id?: number;
  content: string;
  created_at: string;
  updated_at: string;
  author_name?: string;
  author_avatar_url?: string;
};

export type NotificationItem = {
  id: number;
  receiver_id: number;
  actor_id: number;
  post_id: number;
  comment_id: number;
  parent_comment_id?: number;
  type: string;
  is_read: boolean;
  read_at?: string;
  created_at: string;
  actor_name?: string;
  actor_avatar_url?: string;
  post_title?: string;
  comment_content?: string;
};

export type NotificationListPayload = {
  unread_count: number;
  items: NotificationItem[];
};

export type TweetSearchItem = {
  id: number;
  author_id: number;
  content: string;
  created_at: string;
  score: number;
};

export type TweetSearchPayload = {
  items: TweetSearchItem[];
  from: number;
  size: number;
};

export type SearchUserItem = {
  id: number;
  username: string;
  email: string;
  avatar_url?: string;
  score?: number;
};

export type SearchPostItem = {
  id: number;
  author_id: number;
  title: string;
  summary: string;
  status: string;
  tags?: string[];
  score?: number;
};

export type SearchResponse = {
  type: "post" | "user";
  items: SearchPostItem[] | SearchUserItem[];
};

export type SearchSuggestResponse = {
  type: "post" | "user";
  items: string[];
};

export type AdminUserItem = {
  id: number;
  username: string;
  email: string;
  avatar_url?: string;
  role: "user" | "admin" | "super_admin";
  status: number;
  created_at?: string;
  updated_at?: string;
};

export type AdminAuditItem = {
  id: number;
  actor_id: number;
  actor_role: string;
  actor_name?: string;
  action: string;
  target_type: string;
  target_id: number;
  detail?: string;
  created_at: string;
};

export async function getPublishedPosts(): Promise<Post[]> {
  const resp = await apiFetch(`${getAPIBase()}/api/v1/posts`, { cache: "no-store" });
  if (!resp.ok) {
    throw new Error(`fetch posts failed: ${resp.status}`);
  }
  const json = (await resp.json()) as Envelope<Post[]>;
  return json.data || [];
}

export async function getFeed(
  type: "hot" | "latest" | "recommend",
  token?: string,
  input?: { page?: number; pageSize?: number }
): Promise<FeedPayload> {
  const headers: HeadersInit = {};
  if (token) headers.Authorization = `Bearer ${token}`;
  const page = input?.page || 1;
  const pageSize = input?.pageSize || 8;
  const resp = await apiFetch(`${getAPIBase()}/api/v1/feed?type=${type}&page=${page}&page_size=${pageSize}`, {
    method: "GET",
    headers,
    cache: "no-store"
  });
  const json = (await resp.json()) as Envelope<FeedPayload>;
  if (!resp.ok || json.code !== 0) {
    throw new Error(json.message || "get feed failed");
  }
  return json.data || { type, items: [], total: 0, page, page_size: pageSize };
}

export async function getPostEngagement(postID: number, token?: string): Promise<PostEngagement> {
  const headers: HeadersInit = {};
  if (token) headers.Authorization = `Bearer ${token}`;
  const resp = await apiFetch(`${getAPIBase()}/api/v1/posts/${postID}/engagement`, {
    method: "GET",
    headers,
    cache: "no-store"
  });
  const json = (await resp.json()) as Envelope<PostEngagement>;
  if (!resp.ok || json.code !== 0) {
    throw new Error(json.message || "get engagement failed");
  }
  return json.data;
}

export async function togglePostLike(postID: number, token: string): Promise<PostEngagement> {
  const resp = await apiFetch(`${getAPIBase()}/api/v1/posts/${postID}/like`, {
    method: "POST",
    headers: { Authorization: `Bearer ${token}` }
  });
  const json = (await resp.json()) as Envelope<PostEngagement>;
  if (!resp.ok || json.code !== 0) {
    throw new Error(json.message || "toggle like failed");
  }
  return json.data;
}

export async function togglePostFavorite(postID: number, token: string): Promise<PostEngagement> {
  const resp = await apiFetch(`${getAPIBase()}/api/v1/posts/${postID}/favorite`, {
    method: "POST",
    headers: { Authorization: `Bearer ${token}` }
  });
  const json = (await resp.json()) as Envelope<PostEngagement>;
  if (!resp.ok || json.code !== 0) {
    throw new Error(json.message || "toggle favorite failed");
  }
  return json.data;
}

export async function getPostRankings(limit = 20): Promise<Post[]> {
  const resp = await apiFetch(`${getAPIBase()}/api/v1/rankings/posts?limit=${limit}&period=all`, {
    method: "GET",
    cache: "no-store"
  });
  const json = (await resp.json()) as Envelope<RankingPayload>;
  if (!resp.ok || json.code !== 0) {
    throw new Error(json.message || "get rankings failed");
  }
  return json.data?.items || [];
}

export async function getPostRankingsByPeriod(period: "24h" | "7d" | "30d" | "all", limit = 20): Promise<Post[]> {
  const resp = await apiFetch(`${getAPIBase()}/api/v1/rankings/posts?limit=${limit}&period=${period}`, {
    method: "GET",
    cache: "no-store"
  });
  const json = (await resp.json()) as Envelope<RankingPayload>;
  if (!resp.ok || json.code !== 0) {
    throw new Error(json.message || "get rankings failed");
  }
  return json.data?.items || [];
}

export async function getMyFavorites(token: string): Promise<Post[]> {
  const resp = await apiFetch(`${getAPIBase()}/api/v1/me/favorites?limit=50`, {
    method: "GET",
    headers: {
      Authorization: `Bearer ${token}`
    },
    cache: "no-store"
  });
  const json = (await resp.json()) as Envelope<Post[]>;
  if (!resp.ok || json.code !== 0) {
    throw new Error(json.message || "get my favorites failed");
  }
  return json.data || [];
}

export async function adminListUsers(input: {
  token: string;
  page?: number;
  page_size?: number;
  q?: string;
  role?: string;
  status?: string;
}): Promise<AdminListPayload<AdminUserItem>> {
  const page = input.page || 1;
  const pageSize = input.page_size || 20;
  const q = encodeURIComponent(input.q || "");
  const role = encodeURIComponent(input.role || "");
  const status = encodeURIComponent(input.status || "");
  const resp = await apiFetch(
    `${getAPIBase()}/api/v1/admin/users?page=${page}&page_size=${pageSize}&q=${q}&role=${role}&status=${status}`,
    {
      method: "GET",
      headers: { Authorization: `Bearer ${input.token}` },
      cache: "no-store"
    }
  );
  const json = (await resp.json()) as Envelope<AdminListPayload<AdminUserItem>>;
  if (!resp.ok || json.code !== 0) {
    throw new Error(json.message || "admin list users failed");
  }
  return json.data;
}

export async function adminUpdateUserRole(input: {
  token: string;
  id: number;
  role: "user" | "admin" | "super_admin";
  status: -1 | 0 | 1;
}) {
  const resp = await apiFetch(`${getAPIBase()}/api/v1/admin/users/role`, {
    method: "PUT",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${input.token}`
    },
    body: JSON.stringify(input)
  });
  const json = (await resp.json()) as Envelope<{ updated: boolean }>;
  if (!resp.ok || json.code !== 0) {
    throw new Error(json.message || "admin update user role failed");
  }
  return json.data;
}

export async function adminListPosts(input: {
  token: string;
  page?: number;
  page_size?: number;
  q?: string;
  status?: string;
}): Promise<AdminListPayload<Post>> {
  const page = input.page || 1;
  const pageSize = input.page_size || 20;
  const q = encodeURIComponent(input.q || "");
  const status = encodeURIComponent(input.status || "");
  const resp = await apiFetch(
    `${getAPIBase()}/api/v1/admin/posts?page=${page}&page_size=${pageSize}&q=${q}&status=${status}`,
    {
      method: "GET",
      headers: { Authorization: `Bearer ${input.token}` },
      cache: "no-store"
    }
  );
  const json = (await resp.json()) as Envelope<AdminListPayload<Post>>;
  if (!resp.ok || json.code !== 0) {
    throw new Error(json.message || "admin list posts failed");
  }
  return json.data;
}

export async function adminDeleteUser(input: { token: string; id: number }) {
  const resp = await apiFetch(`${getAPIBase()}/api/v1/admin/users/${input.id}`, {
    method: "DELETE",
    headers: { Authorization: `Bearer ${input.token}` }
  });
  const json = (await resp.json()) as Envelope<{ deleted: boolean }>;
  if (!resp.ok || json.code !== 0) {
    throw new Error(json.message || "admin delete user failed");
  }
  return json.data;
}

export async function adminCreateAudit(input: {
  token: string;
  action: string;
  target_type: string;
  target_id: number;
  detail?: string;
}) {
  const resp = await apiFetch(`${getAPIBase()}/api/v1/admin/audits`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${input.token}`
    },
    body: JSON.stringify(input)
  });
  const json = (await resp.json()) as Envelope<{ created: boolean }>;
  if (!resp.ok || json.code !== 0) {
    throw new Error(json.message || "admin create audit failed");
  }
  return json.data;
}

export async function adminListAudits(input: {
  token: string;
  page?: number;
  page_size?: number;
  action?: string;
}): Promise<AdminListPayload<AdminAuditItem>> {
  const page = input.page || 1;
  const pageSize = input.page_size || 20;
  const action = encodeURIComponent(input.action || "");
  const resp = await apiFetch(
    `${getAPIBase()}/api/v1/admin/audits?page=${page}&page_size=${pageSize}&action=${action}`,
    {
      method: "GET",
      headers: { Authorization: `Bearer ${input.token}` },
      cache: "no-store"
    }
  );
  const json = (await resp.json()) as Envelope<AdminListPayload<AdminAuditItem>>;
  if (!resp.ok || json.code !== 0) {
    throw new Error(json.message || "admin list audits failed");
  }
  return json.data;
}

export async function getPublishedPostDetail(postID: number): Promise<Post> {
  const resp = await apiFetch(`${getAPIBase()}/api/v1/posts/${postID}`, {
    method: "GET",
    cache: "no-store"
  });
  const json = (await resp.json()) as Envelope<Post>;
  if (!resp.ok || json.code !== 0) {
    throw new Error(json.message || "get post detail failed");
  }
  return json.data;
}

export async function getPostComments(postID: number): Promise<Comment[]> {
  const resp = await apiFetch(`${getAPIBase()}/api/v1/posts/${postID}/comments`, {
    method: "GET",
    cache: "no-store"
  });
  const json = (await resp.json()) as Envelope<Comment[]>;
  if (!resp.ok || json.code !== 0) {
    throw new Error(json.message || "get comments failed");
  }
  return json.data || [];
}

export async function createPostComment(
  token: string,
  postID: number,
  input: { content: string; parent_id?: number }
): Promise<Comment> {
  const resp = await apiFetch(`${getAPIBase()}/api/v1/posts/${postID}/comments`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${token}`
    },
    body: JSON.stringify({
      content: input.content,
      parent_id: input.parent_id ?? undefined
    })
  });
  const json = (await resp.json()) as Envelope<Comment>;
  if (!resp.ok || json.code !== 0) {
    throw new Error(json.message || "create comment failed");
  }
  return json.data;
}

export async function getMyNotifications(token: string, limit = 50): Promise<NotificationListPayload> {
  const resp = await apiFetch(`${getAPIBase()}/api/v1/me/notifications?limit=${limit}`, {
    method: "GET",
    headers: {
      Authorization: `Bearer ${token}`
    },
    cache: "no-store"
  });
  const json = (await resp.json()) as Envelope<NotificationListPayload>;
  if (!resp.ok || json.code !== 0) {
    throw new Error(json.message || "get notifications failed");
  }
  return json.data;
}

export async function markMyNotificationRead(token: string, id: number): Promise<void> {
  const resp = await apiFetch(`${getAPIBase()}/api/v1/me/notifications/${id}/read`, {
    method: "POST",
    headers: {
      Authorization: `Bearer ${token}`
    }
  });
  const json = (await resp.json()) as Envelope<{ read: boolean; id: number }>;
  if (!resp.ok || json.code !== 0) {
    throw new Error(json.message || "mark notification read failed");
  }
}

export async function getPublicUserPage(userID: number): Promise<PublicUserPagePayload> {
  const resp = await apiFetch(`${getAPIBase()}/api/v1/users/${userID}`, {
    method: "GET",
    cache: "no-store"
  });
  const json = (await resp.json()) as Envelope<PublicUserPagePayload>;
  if (!resp.ok || json.code !== 0) {
    throw new Error(json.message || "get user page failed");
  }
  return json.data;
}

export async function searchTweets(input: { q: string; from?: number; size?: number }): Promise<TweetSearchPayload> {
  const params = new URLSearchParams();
  params.set("q", input.q);
  params.set("from", String(input.from ?? 0));
  params.set("size", String(input.size ?? 20));
  const resp = await apiFetch(`${getAPIBase()}/api/v1/tweets/search?${params.toString()}`, {
    method: "GET",
    cache: "no-store"
  });
  const json = (await resp.json()) as Envelope<TweetSearchPayload>;
  if (!resp.ok || json.code !== 0) {
    throw new Error(json.message || "search tweets failed");
  }
  return json.data || { items: [], from: input.from ?? 0, size: input.size ?? 20 };
}

export async function searchAll(input: {
  type: "post" | "user";
  q: string;
  limit?: number;
  token?: string;
}): Promise<SearchResponse> {
  const params = new URLSearchParams();
  params.set("type", input.type);
  params.set("q", input.q);
  params.set("limit", String(input.limit ?? 20));
  const headers: HeadersInit = {};
  if (input.token) {
    headers.Authorization = `Bearer ${input.token}`;
  }
  const resp = await apiFetch(`${getAPIBase()}/api/v1/search?${params.toString()}`, {
    method: "GET",
    headers,
    cache: "no-store"
  });
  const json = (await resp.json()) as Envelope<SearchResponse>;
  if (!resp.ok || json.code !== 0) {
    throw new Error(json.message || "search failed");
  }
  return json.data;
}

export async function searchSuggest(input: {
  type: "post" | "user";
  q: string;
  limit?: number;
  token?: string;
}): Promise<SearchSuggestResponse> {
  const params = new URLSearchParams();
  params.set("type", input.type);
  params.set("q", input.q);
  params.set("limit", String(input.limit ?? 5));
  const headers: HeadersInit = {};
  if (input.token) headers.Authorization = `Bearer ${input.token}`;
  const resp = await apiFetch(`${getAPIBase()}/api/v1/search/suggest?${params.toString()}`, {
    method: "GET",
    headers,
    cache: "no-store"
  });
  const json = (await resp.json()) as Envelope<SearchSuggestResponse>;
  if (!resp.ok || json.code !== 0) {
    throw new Error(json.message || "suggest failed");
  }
  return json.data || { type: input.type, items: [] };
}

export async function login(account: string, password: string): Promise<LoginPayload> {
  const resp = await apiFetch(`${getAPIBase()}/api/v1/auth/login`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ account, password })
  });

  const json = (await resp.json()) as Envelope<LoginPayload>;
  if (!resp.ok || json.code !== 0) {
    throw new Error(json.message || "login failed");
  }
  return json.data;
}

export async function getCurrentUser(): Promise<UserProfilePayload> {
  const resp = await apiFetch(`${getAPIBase()}/api/v1/auth/me`, {
    method: "GET",
    cache: "no-store"
  });
  const json = (await resp.json()) as Envelope<UserProfilePayload>;
  if (!resp.ok || json.code !== 0) {
    throw new Error(json.message || "get current user failed");
  }
  return json.data;
}

export async function register(input: {
  username: string;
  email: string;
  password: string;
}): Promise<RegisterPayload> {
  const resp = await apiFetch(`${getAPIBase()}/api/v1/auth/register`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(input)
  });
  const json = (await resp.json()) as Envelope<RegisterPayload>;
  if (!resp.ok || json.code !== 0) {
    throw new Error(json.message || "register failed");
  }
  return json.data;
}

export async function createDraft(input: {
  token: string;
  title: string;
  slug?: string;
  summary: string;
  tags?: string[];
  draft_note?: string;
  raw_content: string;
}): Promise<Post> {
  const resp = await apiFetch(`${getAPIBase()}/api/v1/posts`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${input.token}`
    },
    body: JSON.stringify(input)
  });

  const json = (await resp.json()) as Envelope<Post>;
  if (!resp.ok || json.code !== 0) {
    throw new Error(json.message || "create draft failed");
  }
  return json.data;
}

export async function getMyPosts(token: string): Promise<Post[]> {
  const resp = await apiFetch(`${getAPIBase()}/api/v1/me/posts`, {
    method: "GET",
    headers: {
      Authorization: `Bearer ${token}`
    },
    cache: "no-store"
  });
  const json = (await resp.json()) as Envelope<Post[]>;
  if (!resp.ok || json.code !== 0) {
    throw new Error(json.message || "get my posts failed");
  }
  return json.data || [];
}

export async function getMyProfile(token: string): Promise<UserProfilePayload> {
  const resp = await apiFetch(`${getAPIBase()}/api/v1/me/profile`, {
    method: "GET",
    headers: { Authorization: `Bearer ${token}` },
    cache: "no-store"
  });
  const json = (await resp.json()) as Envelope<UserProfilePayload>;
  if (!resp.ok || json.code !== 0) {
    throw new Error(json.message || "get profile failed");
  }
  return json.data;
}

export async function updateMyProfile(
  token: string,
  input: { username: string; email: string; avatar_url?: string }
): Promise<UserProfilePayload> {
  const resp = await apiFetch(`${getAPIBase()}/api/v1/me/profile`, {
    method: "PUT",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${token}`
    },
    body: JSON.stringify(input)
  });
  const json = (await resp.json()) as Envelope<UserProfilePayload>;
  if (!resp.ok || json.code !== 0) {
    throw new Error(json.message || "update profile failed");
  }
  return json.data;
}

export async function uploadImage(token: string, file: File): Promise<string> {
  const form = new FormData();
  form.append("file", file);
  const resp = await apiFetch(`${getAPIBase()}/api/v1/uploads/image`, {
    method: "POST",
    headers: {
      Authorization: `Bearer ${token}`
    },
    body: form
  });
  const json = (await resp.json()) as Envelope<{ url: string }>;
  if (!resp.ok || json.code !== 0 || !json.data?.url) {
    throw new Error(json.message || "upload image failed");
  }
  return json.data.url;
}

export async function changeMyPassword(
  token: string,
  input: { old_password: string; new_password: string }
): Promise<void> {
  const resp = await apiFetch(`${getAPIBase()}/api/v1/me/password`, {
    method: "PUT",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${token}`
    },
    body: JSON.stringify(input)
  });
  const json = (await resp.json()) as Envelope<{ changed: boolean }>;
  if (!resp.ok || json.code !== 0) {
    throw new Error(json.message || "change password failed");
  }
}

export async function publishPost(token: string, postID: number): Promise<Post> {
  const resp = await apiFetch(`${getAPIBase()}/api/v1/posts/${postID}/publish`, {
    method: "POST",
    headers: {
      Authorization: `Bearer ${token}`
    }
  });
  const json = (await resp.json()) as Envelope<Post>;
  if (!resp.ok || json.code !== 0) {
    throw new Error(json.message || "publish failed");
  }
  return json.data;
}

export async function unpublishPost(token: string, postID: number): Promise<Post> {
  const resp = await apiFetch(`${getAPIBase()}/api/v1/posts/${postID}/unpublish`, {
    method: "POST",
    headers: {
      Authorization: `Bearer ${token}`
    }
  });
  const json = (await resp.json()) as Envelope<Post>;
  if (!resp.ok || json.code !== 0) {
    throw new Error(json.message || "unpublish failed");
  }
  return json.data;
}

export async function createPostDraft(
  token: string,
  postID: number,
  fromVersion?: number,
  draftNote?: string
): Promise<PostVersion> {
  const resp = await apiFetch(`${getAPIBase()}/api/v1/posts/${postID}/drafts`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${token}`
    },
    body: JSON.stringify({ from_version: fromVersion || 0, draft_note: draftNote ?? undefined })
  });
  const json = (await resp.json()) as Envelope<PostVersion>;
  if (!resp.ok || json.code !== 0) {
    throw new Error(json.message || "create post draft failed");
  }
  return json.data;
}

export async function getPostDetail(postID: number, token: string): Promise<Post> {
  const resp = await apiFetch(`${getAPIBase()}/api/v1/posts/${postID}`, {
    method: "GET",
    headers: {
      Authorization: `Bearer ${token}`
    },
    cache: "no-store"
  });
  const json = (await resp.json()) as Envelope<Post>;
  if (!resp.ok || json.code !== 0) {
    throw new Error(json.message || "get post detail failed");
  }
  return json.data;
}

export async function getMyPostDetail(postID: number, token: string): Promise<Post> {
  const resp = await apiFetch(`${getAPIBase()}/api/v1/me/posts/${postID}`, {
    method: "GET",
    headers: {
      Authorization: `Bearer ${token}`
    },
    cache: "no-store"
  });
  const json = (await resp.json()) as Envelope<Post>;
  if (!resp.ok || json.code !== 0) {
    throw new Error(json.message || "get my post detail failed");
  }
  return json.data;
}

export async function updatePostMeta(
  token: string,
  postID: number,
  input: { title: string; summary: string; tags?: string[] }
): Promise<Post> {
  const resp = await apiFetch(`${getAPIBase()}/api/v1/posts/${postID}/meta`, {
    method: "PUT",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${token}`
    },
    body: JSON.stringify(input)
  });
  const json = (await resp.json()) as Envelope<Post>;
  if (!resp.ok || json.code !== 0) {
    throw new Error(json.message || "update post meta failed");
  }
  return json.data;
}

export async function saveDraft(input: {
  token: string;
  postID: number;
  title: string;
  summary: string;
  raw_content: string;
}): Promise<Post> {
  const resp = await apiFetch(`${getAPIBase()}/api/v1/posts/${input.postID}/draft`, {
    method: "PUT",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${input.token}`
    },
    body: JSON.stringify({
      title: input.title,
      summary: input.summary,
      raw_content: input.raw_content
    })
  });
  const json = (await resp.json()) as Envelope<Post>;
  if (!resp.ok || json.code !== 0) {
    throw new Error(json.message || "save draft failed");
  }
  return json.data;
}

export async function getPostDrafts(token: string, postID: number): Promise<PostVersion[]> {
  const resp = await apiFetch(`${getAPIBase()}/api/v1/posts/${postID}/drafts`, {
    method: "GET",
    headers: {
      Authorization: `Bearer ${token}`
    },
    cache: "no-store"
  });
  const json = (await resp.json()) as Envelope<PostVersion[]>;
  if (!resp.ok || json.code !== 0) {
    throw new Error(json.message || "get drafts failed");
  }
  return json.data || [];
}

export async function getPostDraft(token: string, postID: number, version: number): Promise<PostVersion> {
  const resp = await apiFetch(`${getAPIBase()}/api/v1/posts/${postID}/drafts/${version}`, {
    method: "GET",
    headers: {
      Authorization: `Bearer ${token}`
    },
    cache: "no-store"
  });
  const json = (await resp.json()) as Envelope<PostVersion>;
  if (!resp.ok || json.code !== 0) {
    throw new Error(json.message || "get draft failed");
  }
  return json.data;
}

export async function savePostDraft(input: {
  token: string;
  postID: number;
  version: number;
  title: string;
  summary: string;
  draft_note?: string;
  raw_content: string;
}): Promise<PostVersion> {
  const resp = await apiFetch(`${getAPIBase()}/api/v1/posts/${input.postID}/drafts/${input.version}`, {
    method: "PUT",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${input.token}`
    },
    body: JSON.stringify({
      title: input.title,
      summary: input.summary,
      draft_note: input.draft_note ?? undefined,
      raw_content: input.raw_content
    })
  });
  const json = (await resp.json()) as Envelope<PostVersion>;
  if (!resp.ok || json.code !== 0) {
    throw new Error(json.message || "save draft failed");
  }
  return json.data;
}

export async function publishPostDraft(token: string, postID: number, version: number): Promise<Post> {
  const resp = await apiFetch(`${getAPIBase()}/api/v1/posts/${postID}/drafts/${version}/publish`, {
    method: "POST",
    headers: {
      Authorization: `Bearer ${token}`
    }
  });
  const json = (await resp.json()) as Envelope<Post>;
  if (!resp.ok || json.code !== 0) {
    throw new Error(json.message || "publish draft failed");
  }
  return json.data;
}

export async function deletePostDraft(token: string, postID: number, version: number): Promise<void> {
  const resp = await apiFetch(`${getAPIBase()}/api/v1/posts/${postID}/drafts/${version}`, {
    method: "DELETE",
    headers: {
      Authorization: `Bearer ${token}`
    }
  });
  const json = (await resp.json()) as Envelope<{ deleted_version: number }>;
  if (!resp.ok || json.code !== 0) {
    throw new Error(json.message || "delete draft failed");
  }
}

// Legacy exports used by old pages/components.
export const getPostVersions = getPostDrafts;
export const rollbackPostVersion = publishPostDraft;
export const deletePostVersion = deletePostDraft;

export async function deletePost(token: string, postID: number): Promise<void> {
  const resp = await apiFetch(`${getAPIBase()}/api/v1/posts/${postID}`, {
    method: "DELETE",
    headers: {
      Authorization: `Bearer ${token}`
    }
  });
  const json = (await resp.json()) as Envelope<{ deleted_post_id: number }>;
  if (!resp.ok || json.code !== 0) {
    throw new Error(json.message || "delete post failed");
  }
}
