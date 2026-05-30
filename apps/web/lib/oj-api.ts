const OJ_API_BASE =
  (typeof window === "undefined"
    ? process.env.OJ_API_BASE_SERVER || process.env.NEXT_PUBLIC_OJ_API_BASE
    : process.env.NEXT_PUBLIC_OJ_API_BASE) || "/oj-api";
const AUTH_API_BASE =
  (typeof window === "undefined" ? process.env.API_BASE_SERVER : process.env.NEXT_PUBLIC_API_BASE) || "/blog-api";

type Envelope<T> = {
  code: number;
  message: string;
  data: T;
};

export class OJApiError extends Error {
  code: number;
  constructor(code: number, message: string) {
    super(message);
    this.code = code;
  }
}

export function isOJNotLoginError(err: unknown) {
  return err instanceof OJApiError && err.code === 40100;
}

export type OJPageResult<T> = {
  records: T[];
  total: number;
  current: number;
  size: number;
};

export type OJUserVO = {
  id: number;
  userName: string;
  userAvatar?: string;
  userProfile?: string;
  userRole: string;
  createTime?: string;
  updateTime?: string;
};

export type OJQuestionVO = {
  id: number;
  title: string;
  content: string;
  tags: string[];
  sampleCase?: Array<{ input: string; output: string }>;
  submitNum: number;
  acceptedNum: number;
  judgeConfig: { timeLimit?: number; memoryLimit?: number };
  thumbNum: number;
  favourNum: number;
  userId: number;
  createTime?: string;
  updateTime?: string;
};

export type OJQuestionEntity = {
  id: number;
  title: string;
  content: string;
  tags: string;
  answer: string;
  sampleCase: string;
  judgeCase: string;
  judgeConfig: string;
  userId: number;
};

export type OJQuestionSubmitVO = {
  id: number;
  language: string;
  code?: string;
  judgeInfo?: { message?: string; time?: number; memory?: number; score?: number };
  status: number;
  questionId: number;
  userId: number;
  createTime?: string;
  updateTime?: string;
};

export type OJQuestionSolutionItem = {
  id: number;
  questionId: number;
  postId: number;
  userId: number;
  createTime?: string;
  post?: {
    id?: number;
    title?: string;
    summary?: string;
    author_id?: number;
    author_name?: string;
    authorName?: string;
    author_avatar_url?: string;
    updated_at?: string;
  };
};

export type OJAgentTask = {
  id: number;
  questionId: number;
  triggerUserId: number;
  status: "pending" | "running" | "success" | "failed" | string;
  attempts: number;
  blogPostId: number;
  blogPostUrl?: string;
  modelName?: string;
  lastError?: string;
  createTime?: string;
  updateTime?: string;
};

export type OJFirstACRankItem = {
  userId: number;
  blogUserId?: number;
  userName: string;
  userAvatar?: string;
  firstAcCount: number;
  rank: number;
};

async function ojFetch<T>(path: string, init?: RequestInit): Promise<T> {
  const doFetch = () => fetch(`${OJ_API_BASE}${path}`, {
    ...init,
    credentials: "include",
    headers: {
      ...(init?.body ? { "Content-Type": "application/json" } : {}),
      ...(init?.headers || {})
    }
  });

  let resp = await doFetch();
  let json = (await resp.json()) as Envelope<T>;
  if (resp.status === 401 || json.code === 40100) {
    const refreshed = await fetch(`${AUTH_API_BASE}/api/v1/auth/refresh`, {
      method: "POST",
      credentials: "include"
    });
    if (refreshed.ok) {
      resp = await doFetch();
      json = (await resp.json()) as Envelope<T>;
    }
  }
  if (!resp.ok || json.code !== 0) {
    throw new OJApiError(json.code ?? resp.status, json.message || `request failed: ${resp.status}`);
  }
  return json.data;
}

export async function ojGetLoginUser() {
  return ojFetch<OJUserVO>("/api/user/get/login", { method: "GET" });
}

export async function ojLogout() {
  return ojFetch<boolean>("/api/user/logout", { method: "POST", body: "{}" });
}

export async function ojUpdateMyUser(input: Partial<{ userName: string; userAvatar: string; userProfile: string }>) {
  return ojFetch<boolean>("/api/user/update/my", {
    method: "POST",
    body: JSON.stringify(input)
  });
}

export async function ojUploadAvatar(file: File) {
  const form = new FormData();
  form.append("biz", "user_avatar");
  form.append("file", file);
  const doUpload = () => fetch(`${OJ_API_BASE}/api/file/upload`, {
    method: "POST",
    credentials: "include",
    body: form
  });
  let resp = await doUpload();
  let json = (await resp.json()) as Envelope<string>;
  if (resp.status === 401 || json.code === 40100) {
    const refreshed = await fetch(`${AUTH_API_BASE}/api/v1/auth/refresh`, {
      method: "POST",
      credentials: "include"
    });
    if (refreshed.ok) {
      resp = await doUpload();
      json = (await resp.json()) as Envelope<string>;
    }
  }
  if (!resp.ok || json.code !== 0) {
    throw new Error(json.message || "upload failed");
  }
  return json.data;
}

export async function ojListQuestions(input: {
  current?: number;
  pageSize?: number;
  userId?: number;
  title?: string;
  searchText?: string;
  tags?: string[];
}) {
  return ojFetch<OJPageResult<OJQuestionVO>>("/api/question/list/page/vo", {
    method: "POST",
    body: JSON.stringify({ current: 1, pageSize: 10, ...input })
  });
}

export async function ojMyQuestions(input?: { current?: number; pageSize?: number }) {
  return ojFetch<OJPageResult<OJQuestionVO>>("/api/question/my/list/page/vo", {
    method: "POST",
    body: JSON.stringify({ current: 1, pageSize: 10, ...(input || {}) })
  });
}

export async function ojGetQuestionVO(id: number) {
  return ojFetch<OJQuestionVO>(`/api/question/get/vo?id=${id}`, { method: "GET" });
}

export async function ojGetQuestion(id: number) {
  return ojFetch<OJQuestionEntity>(`/api/question/get?id=${id}`, { method: "GET" });
}

export async function ojAddQuestion(input: {
  title: string;
  content: string;
  tags: string[];
  sampleCase: Array<{ input: string; output: string }>;
  answer: string;
  judgeCase: Array<{ input: string; output: string }>;
  judgeConfig: { timeLimit: number; memoryLimit: number };
}) {
  return ojFetch<number>("/api/question/add", {
    method: "POST",
    body: JSON.stringify({
      ...input,
      tags: JSON.stringify(input.tags),
      sampleCase: JSON.stringify(input.sampleCase),
      judgeCase: JSON.stringify(input.judgeCase),
      judgeConfig: JSON.stringify(input.judgeConfig)
    })
  });
}

export async function ojEditQuestion(input: {
  id: number;
  title: string;
  content: string;
  tags: string[];
  sampleCase: Array<{ input: string; output: string }>;
  answer: string;
  judgeCase: Array<{ input: string; output: string }>;
  judgeConfig: { timeLimit: number; memoryLimit: number };
}) {
  return ojFetch<boolean>("/api/question/edit", {
    method: "POST",
    body: JSON.stringify({
      ...input,
      tags: JSON.stringify(input.tags),
      sampleCase: JSON.stringify(input.sampleCase),
      judgeCase: JSON.stringify(input.judgeCase),
      judgeConfig: JSON.stringify(input.judgeConfig)
    })
  });
}

export async function ojSubmitQuestion(input: { questionId: number; language: string; code: string }) {
  return ojFetch<number>("/api/question/question_submit/do", {
    method: "POST",
    body: JSON.stringify(input)
  });
}

export async function ojRunQuestion(input: {
  questionId: number;
  language: string;
  code: string;
  input: string;
}) {
  return ojFetch<{ output: string; judgeInfo?: { message?: string; time?: number; memory?: number; score?: number } }>("/api/question/run", {
    method: "POST",
    body: JSON.stringify(input)
  });
}

export async function ojListQuestionSubmits(input: {
  current?: number;
  pageSize?: number;
  questionId?: number;
  userId?: number;
}) {
  return ojFetch<OJPageResult<OJQuestionSubmitVO>>("/api/question/question_submit/list/page", {
    method: "POST",
    body: JSON.stringify({ current: 1, pageSize: 10, ...input })
  });
}

export async function ojAdminListUsers(input?: {
  current?: number;
  pageSize?: number;
}) {
  return ojFetch<OJPageResult<OJUserVO>>("/api/user/list/page", {
    method: "POST",
    body: JSON.stringify({ current: 1, pageSize: 20, ...(input || {}) })
  });
}

export async function ojAdminUpdateUser(input: {
  id: number;
  userRole?: string;
  userName?: string;
  userAvatar?: string;
  userProfile?: string;
}) {
  return ojFetch<boolean>("/api/user/update", {
    method: "POST",
    body: JSON.stringify(input)
  });
}

export async function ojAdminDeleteUser(id: number) {
  return ojFetch<boolean>("/api/user/delete", {
    method: "POST",
    body: JSON.stringify({ id })
  });
}

export async function ojDeleteQuestion(id: number) {
  return ojFetch<boolean>("/api/question/delete", {
    method: "POST",
    body: JSON.stringify({ id })
  });
}

export async function ojBindQuestionSolution(input: { questionId: number; postId: number }) {
  return ojFetch<number>("/api/question/solution/bind", {
    method: "POST",
    body: JSON.stringify(input)
  });
}

export async function ojUnbindQuestionSolution(input: { questionId: number; postId: number }) {
  return ojFetch<boolean>("/api/question/solution/unbind", {
    method: "POST",
    body: JSON.stringify(input)
  });
}

export async function ojListQuestionSolutions(input: {
  questionId: number;
  current?: number;
  pageSize?: number;
}) {
  return ojFetch<OJPageResult<OJQuestionSolutionItem>>("/api/question/solution/list/page", {
    method: "POST",
    body: JSON.stringify({ current: 1, pageSize: 10, ...input })
  });
}

export async function ojGenerateAgentSolution(questionId: number) {
  return ojFetch<{ taskId: number; status: string }>("/api/question/agent/solution/generate", {
    method: "POST",
    body: JSON.stringify({ questionId })
  });
}

export async function ojGetAgentSolutionTask(id: number) {
  return ojFetch<OJAgentTask>(`/api/question/agent/solution/task?id=${id}`, { method: "GET" });
}

export async function ojListAgentSolutionTasks(input: {
  current?: number;
  pageSize?: number;
  questionId?: number;
  status?: string;
}) {
  return ojFetch<OJPageResult<OJAgentTask>>("/api/question/agent/solution/task/list/page", {
    method: "POST",
    body: JSON.stringify({ current: 1, pageSize: 10, ...input })
  });
}

export async function ojFirstACRank24h(limit = 20) {
  return ojFetch<{ windowHours: number; records: OJFirstACRankItem[] }>(
    `/api/question/rank/first-ac-24h?limit=${limit}`,
    { method: "GET" }
  );
}
