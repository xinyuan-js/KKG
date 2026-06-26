const BLOG_BASE = process.env.SEED_BLOG_BASE || "http://127.0.0.1/blog-api";
const OJ_BASE = process.env.SEED_OJ_BASE || "http://127.0.0.1/oj-api";
const ACCOUNT = process.env.SEED_ACCOUNT || "aimin";
const EMAIL = process.env.SEED_EMAIL || "aimin@kkg.local";
const PASSWORD = process.env.SEED_PASSWORD || "12345678";

let cookie = "";

function updateCookie(resp) {
  const list = typeof resp.headers.getSetCookie === "function"
    ? resp.headers.getSetCookie()
    : [resp.headers.get("set-cookie")].filter(Boolean);
  const parts = [];
  for (const item of list) {
    const pair = item.split(";")[0];
    if (pair) parts.push(pair);
  }
  if (parts.length) {
    const jar = new Map(cookie.split(";").map((p) => p.trim()).filter(Boolean).map((p) => {
      const i = p.indexOf("=");
      return [p.slice(0, i), p.slice(i + 1)];
    }));
    for (const part of parts) {
      const i = part.indexOf("=");
      jar.set(part.slice(0, i), part.slice(i + 1));
    }
    cookie = [...jar.entries()].map(([k, v]) => `${k}=${v}`).join("; ");
  }
}

async function request(base, path, options = {}) {
  const headers = {
    ...(options.body && !(options.body instanceof FormData) ? { "Content-Type": "application/json" } : {}),
    ...(cookie ? { Cookie: cookie } : {}),
    ...(options.headers || {})
  };
  const resp = await fetch(`${base}${path}`, { ...options, headers });
  updateCookie(resp);
  const text = await resp.text();
  let json = null;
  try {
    json = text ? JSON.parse(text) : null;
  } catch {
    json = { raw: text };
  }
  if (!resp.ok || (json && typeof json.code === "number" && json.code !== 0)) {
    const msg = json?.message || text || `${resp.status}`;
    throw new Error(`${options.method || "GET"} ${path} failed: ${msg}`);
  }
  return json?.data;
}

async function ensureAccount() {
  try {
    await request(BLOG_BASE, "/api/v1/auth/register", {
      method: "POST",
      body: JSON.stringify({ username: ACCOUNT, email: EMAIL, password: PASSWORD })
    });
    console.log(`registered ${ACCOUNT}`);
  } catch (err) {
    if (!String(err.message).includes("already exists")) {
      throw err;
    }
    console.log(`${ACCOUNT} already exists`);
  }

  await request(BLOG_BASE, "/api/v1/auth/login", {
    method: "POST",
    body: JSON.stringify({ account: ACCOUNT, password: PASSWORD })
  }).catch((err) => {
    throw new Error(`cannot login as ${ACCOUNT}; set SEED_PASSWORD to the account password or reset it manually. ${err.message}`);
  });
  const me = await request(BLOG_BASE, "/api/v1/auth/me");
  console.log(`logged in as ${me.username}#${me.id}`);
}

const topics = [
  ["数组", "连续内存、下标访问、原地修改、扫描不变量"],
  ["链表", "指针重连、虚拟头节点、快慢指针、局部反转"],
  ["栈", "后进先出、单调结构、表达式求值、状态回滚"],
  ["队列", "先进先出、循环数组、BFS 分层、任务调度"],
  ["哈希表", "键值映射、频次统计、去重、冲突下的期望复杂度"],
  ["字符串", "字符编码、模式匹配、滚动哈希、自动机思想"],
  ["前缀和", "区间聚合、差量转化、二维扩展、可逆运算"],
  ["差分数组", "批量区间更新、事件扫描、离线还原、边界处理"],
  ["双指针", "有序性、左右边界、窗口收缩、相向扫描"],
  ["滑动窗口", "动态区间、计数约束、最值维护、可行性判定"],
  ["二分查找", "答案单调性、边界模板、可行性函数、浮点精度"],
  ["排序", "稳定性、比较器、离散化、排序后贪心"],
  ["堆", "优先级维护、TopK、延迟删除、多路归并"],
  ["并查集", "连通块、路径压缩、按秩合并、关系维护"],
  ["树", "递归定义、遍历序列、子树信息、最近公共祖先"],
  ["二叉搜索树", "中序有序、插入删除、秩查询、平衡化动机"],
  ["堆和优先队列", "最小堆、最大堆、调度模拟、实时中位数"],
  ["图", "邻接表、BFS/DFS、拓扑序、最短路建模"],
  ["有向无环图", "入度、拓扑排序、最长路径、依赖分析"],
  ["最短路", "Dijkstra、松弛、边权约束、路径恢复"],
  ["线段树", "区间合并、懒标记、动态维护、可结合信息"],
  ["树状数组", "lowbit、前缀查询、单点修改、逆序对"],
  ["字典树", "前缀共享、词频统计、异或路径、字符串集合"],
  ["KMP", "失配函数、前后缀、线性匹配、周期判断"],
  ["动态规划表", "状态定义、转移顺序、滚动数组、路径计数"]
];

function blogBody(index, topic, focus) {
  return [
    `# ${topic} 数据结构笔记 ${index}: ${focus}`,
    "",
    `本文面向后续 RAG 检索测试，围绕 **${topic}** 的 ${focus} 展开，强调定义、适用场景、复杂度和常见题型。`,
    "",
    "## 核心模型",
    "",
    `${topic} 的关键不是记住代码模板，而是明确它维护了什么信息、每次操作如何保持不变量，以及查询结果为什么可信。`,
    "",
    "## 适用场景",
    "",
    "- 输入规模较大，朴素枚举会超时。",
    "- 操作之间存在可复用的中间状态。",
    "- 题目可以抽象为更新、查询、合并、匹配或连通性判断。",
    "",
    "## 设计步骤",
    "",
    "1. 先写出暴力解，确定正确性基线。",
    "2. 找到重复计算的位置，用结构化状态替换重复扫描。",
    "3. 明确边界：空结构、单元素、重复值、负数、极端顺序。",
    "4. 为样例和隐藏数据分别准备覆盖用例。",
    "",
    "## 复杂度与风险",
    "",
    `使用 ${topic} 通常是为了把复杂度从平方级降到线性、对数级或接近线性。实现风险集中在边界下标、初始化值、重复元素和状态同步。`,
    "",
    "## 关联题型",
    "",
    `本知识点适合配套 3 到 5 道题：一道基础建模题、一道边界强化题、一道复合数据结构题，以及一道偏工程输入输出的综合题。`
  ].join("\n");
}

async function createBlogs() {
  let count = 0;
  for (let i = 0; i < topics.length; i++) {
    const [topic] = topics[i];
    for (let j = 0; j < 4; j++) {
      const focus = ["模型与不变量", "典型操作与复杂度", "边界测试设计", "与其他结构的组合"][j];
      const index = i * 4 + j + 1;
      const post = await request(BLOG_BASE, "/api/v1/posts", {
        method: "POST",
        body: JSON.stringify({
          title: `${topic}专题：${focus}`,
          slug: `aimin-ds-rag-${String(index).padStart(3, "0")}`,
          summary: `${topic} 的 ${focus}，用于博客检索、推荐和题库 RAG 知识铺垫。`,
          tags: ["数据结构", topic, "RAG", "题库"],
          draft_note: "aimin ds rag seed",
          raw_content: blogBody(index, topic, focus)
        })
      });
      await request(BLOG_BASE, `/api/v1/posts/${post.id}/publish`, { method: "POST" });
      count++;
    }
  }
  console.log(`created and published ${count} blog posts`);
}

const sum = (arr) => arr.reduce((a, b) => a + b, 0);
const lines = (s) => s.trim().split(/\n+/).map((x) => x.trim()).filter(Boolean);
const ints = (s) => s.match(/-?\d+/g)?.map(Number) || [];

function solveRangeSum(input) {
  const a = ints(input);
  const n = a[0], q = a[1], arr = a.slice(2, 2 + n);
  let p = 2 + n;
  const out = [];
  for (let i = 0; i < q; i++) {
    const l = a[p++] - 1, r = a[p++] - 1;
    out.push(String(sum(arr.slice(l, r + 1))));
  }
  return out.join("\n");
}

function solveWindowDistinct(input) {
  const a = ints(input);
  const n = a[0], k = a[1], arr = a.slice(2, 2 + n);
  const freq = new Map();
  let ans = 0, l = 0;
  for (let r = 0; r < n; r++) {
    freq.set(arr[r], (freq.get(arr[r]) || 0) + 1);
    while (freq.size > k) {
      const v = arr[l++];
      const next = freq.get(v) - 1;
      if (next === 0) freq.delete(v); else freq.set(v, next);
    }
    ans = Math.max(ans, r - l + 1);
  }
  return String(ans);
}

function solveNextGreater(input) {
  const a = ints(input);
  const n = a[0], arr = a.slice(1, 1 + n);
  const ans = Array(n).fill(-1), st = [];
  for (let i = 0; i < n; i++) {
    while (st.length && arr[i] > arr[st.at(-1)]) ans[st.pop()] = arr[i];
    st.push(i);
  }
  return ans.join(" ");
}

function solveMinQueue(input) {
  const rows = lines(input);
  const q = Number(rows[0]);
  const arr = [], out = [];
  let head = 0;
  for (let i = 1; i <= q; i++) {
    const [op, x] = rows[i].split(/\s+/);
    if (op === "push") arr.push(Number(x));
    if (op === "pop" && head < arr.length) head++;
    if (op === "min") out.push(String(Math.min(...arr.slice(head))));
  }
  return out.join("\n");
}

function solveDSU(input) {
  const a = ints(input);
  const n = a[0], m = a[1];
  const parent = Array.from({ length: n + 1 }, (_, i) => i);
  const find = (x) => parent[x] === x ? x : parent[x] = find(parent[x]);
  const out = [];
  let p = 2;
  for (let i = 0; i < m; i++) {
    const t = a[p++], x = a[p++], y = a[p++];
    if (t === 1) parent[find(x)] = find(y);
    else out.push(find(x) === find(y) ? "Yes" : "No");
  }
  return out.join("\n");
}

function solveTopo(input) {
  const a = ints(input);
  const n = a[0], m = a[1];
  const g = Array.from({ length: n + 1 }, () => []);
  const indeg = Array(n + 1).fill(0);
  let p = 2;
  for (let i = 0; i < m; i++) {
    const u = a[p++], v = a[p++];
    g[u].push(v);
    indeg[v]++;
  }
  const q = [];
  for (let i = 1; i <= n; i++) if (indeg[i] === 0) q.push(i);
  let head = 0, seen = 0;
  while (head < q.length) {
    const u = q[head++]; seen++;
    for (const v of g[u]) if (--indeg[v] === 0) q.push(v);
  }
  return seen === n ? "DAG" : "Cycle";
}

function solveDijkstra(input) {
  const a = ints(input);
  const n = a[0], m = a[1], s = a[2];
  const g = Array.from({ length: n + 1 }, () => []);
  let p = 3;
  for (let i = 0; i < m; i++) {
    const u = a[p++], v = a[p++], w = a[p++];
    g[u].push([v, w]);
    g[v].push([u, w]);
  }
  const dist = Array(n + 1).fill(Infinity);
  const used = Array(n + 1).fill(false);
  dist[s] = 0;
  for (let it = 0; it < n; it++) {
    let u = -1;
    for (let i = 1; i <= n; i++) if (!used[i] && (u < 0 || dist[i] < dist[u])) u = i;
    if (u < 0 || dist[u] === Infinity) break;
    used[u] = true;
    for (const [v, w] of g[u]) dist[v] = Math.min(dist[v], dist[u] + w);
  }
  return dist.slice(1).map((d) => d === Infinity ? -1 : d).join(" ");
}

function solveTriePrefix(input) {
  const rows = lines(input);
  const n = Number(rows[0]), words = rows.slice(1, 1 + n);
  const q = Number(rows[1 + n]);
  const prefixes = rows.slice(2 + n, 2 + n + q);
  return prefixes.map((p) => words.filter((w) => w.startsWith(p)).length).join("\n");
}

function solveKMP(input) {
  const [text, pattern] = lines(input);
  let cnt = 0, pos = text.indexOf(pattern);
  while (pos !== -1) {
    cnt++;
    pos = text.indexOf(pattern, pos + 1);
  }
  return String(cnt);
}

function solveLIS(input) {
  const a = ints(input);
  const n = a[0], arr = a.slice(1, 1 + n);
  const dp = [];
  for (const x of arr) {
    let l = 0, r = dp.length;
    while (l < r) {
      const mid = (l + r) >> 1;
      if (dp[mid] < x) l = mid + 1; else r = mid;
    }
    dp[l] = x;
  }
  return String(dp.length);
}

function solveKnapsack(input) {
  const a = ints(input);
  const n = a[0], cap = a[1];
  const dp = Array(cap + 1).fill(0);
  let p = 2;
  for (let i = 0; i < n; i++) {
    const w = a[p++], v = a[p++];
    for (let c = cap; c >= w; c--) dp[c] = Math.max(dp[c], dp[c - w] + v);
  }
  return String(dp[cap]);
}

function solveTreeDepth(input) {
  const a = ints(input);
  const n = a[0];
  const g = Array.from({ length: n + 1 }, () => []);
  let p = 1;
  for (let i = 0; i < n - 1; i++) {
    const u = a[p++], v = a[p++];
    g[u].push(v); g[v].push(u);
  }
  let ans = 0;
  const dfs = (u, fa, d) => {
    ans = Math.max(ans, d);
    for (const v of g[u]) if (v !== fa) dfs(v, u, d + 1);
  };
  dfs(1, 0, 0);
  return String(ans);
}

function solveBSTRank(input) {
  const a = ints(input);
  const n = a[0], arr = a.slice(1, 1 + n).sort((x, y) => x - y);
  const q = a[1 + n], queries = a.slice(2 + n, 2 + n + q);
  return queries.map((x) => String(arr.filter((v) => v < x).length + 1)).join("\n");
}

function solveBITInversions(input) {
  const a = ints(input);
  const n = a[0], arr = a.slice(1, 1 + n);
  let ans = 0;
  for (let i = 0; i < n; i++) for (let j = i + 1; j < n; j++) if (arr[i] > arr[j]) ans++;
  return String(ans);
}

function solveSegmentMax(input) {
  const a = ints(input);
  const n = a[0], q = a[1], arr = a.slice(2, 2 + n);
  let p = 2 + n;
  const out = [];
  for (let i = 0; i < q; i++) {
    const op = a[p++], x = a[p++], y = a[p++];
    if (op === 1) arr[x - 1] = y;
    else out.push(String(Math.max(...arr.slice(x - 1, y))));
  }
  return out.join("\n");
}

function solveExpression(input) {
  const tokens = lines(input)[0].split(/\s+/);
  const st = [];
  for (const t of tokens) {
    if (/^-?\d+$/.test(t)) st.push(Number(t));
    else {
      const b = st.pop(), a = st.pop();
      if (t === "+") st.push(a + b);
      if (t === "-") st.push(a - b);
      if (t === "*") st.push(a * b);
    }
  }
  return String(st[0]);
}

function solveHeapTopK(input) {
  const a = ints(input);
  const n = a[0], k = a[1], arr = a.slice(2, 2 + n).sort((x, y) => y - x);
  return arr.slice(0, k).join(" ");
}

function solveMatrixSum(input) {
  const a = ints(input);
  const n = a[0], m = a[1], q = a[2];
  const mat = Array.from({ length: n }, (_, i) => a.slice(3 + i * m, 3 + (i + 1) * m));
  let p = 3 + n * m;
  const out = [];
  for (let i = 0; i < q; i++) {
    const x1 = a[p++] - 1, y1 = a[p++] - 1, x2 = a[p++] - 1, y2 = a[p++] - 1;
    let s = 0;
    for (let x = x1; x <= x2; x++) for (let y = y1; y <= y2; y++) s += mat[x][y];
    out.push(String(s));
  }
  return out.join("\n");
}

function solveDiff(input) {
  const a = ints(input);
  const n = a[0], m = a[1], arr = Array(n).fill(0);
  let p = 2;
  for (let i = 0; i < m; i++) {
    const l = a[p++] - 1, r = a[p++] - 1, v = a[p++];
    for (let j = l; j <= r; j++) arr[j] += v;
  }
  return arr.join(" ");
}

function solveBinaryAnswer(input) {
  const a = ints(input);
  const n = a[0], k = a[1], arr = a.slice(2, 2 + n).sort((x, y) => x - y);
  let best = Infinity;
  for (let i = 0; i + k - 1 < n; i++) best = Math.min(best, arr[i + k - 1] - arr[i]);
  return String(best);
}

const templates = [
  ["区间求和查询", "给定数组和多次区间询问，输出每个闭区间的元素和。", "前缀和", solveRangeSum, ["5 3\n1 2 3 4 5\n1 3\n2 5\n4 4", "6 2\n-1 5 0 2 -3 4\n1 6\n3 5"]],
  ["至多 K 种数字的最长窗口", "给定序列，求至多包含 K 种不同数字的最长连续子数组长度。", "滑动窗口", solveWindowDistinct, ["8 2\n1 2 1 3 4 3 3 2", "7 3\n1 2 3 1 4 2 2"]],
  ["下一个更大元素", "对每个位置，输出它右侧第一个更大的元素，不存在则输出 -1。", "单调栈", solveNextGreater, ["6\n2 1 2 4 3 5", "5\n5 4 3 2 1"]],
  ["支持最小值的队列", "依次执行 push、pop、min 操作，输出每次 min 的结果。", "队列", solveMinQueue, ["8\npush 3\npush 1\nmin\npop\nmin\npush 2\npop\nmin", "7\npush 5\npush -2\npush 4\nmin\npop\nmin\npop"]],
  ["动态连通性", "维护无向图连通关系，操作 1 合并两点，操作 2 查询两点是否连通。", "并查集", solveDSU, ["5 6\n1 1 2\n1 3 4\n2 1 3\n1 2 3\n2 1 4\n2 4 5", "4 5\n2 1 2\n1 1 2\n1 2 3\n2 1 3\n2 3 4"]],
  ["课程依赖是否成环", "给定有向依赖边，判断图是 DAG 还是存在环。", "拓扑排序", solveTopo, ["4 3\n1 2\n1 3\n3 4", "3 3\n1 2\n2 3\n3 1"]],
  ["带权图单源最短路", "给定无向正权图和起点，输出到每个点的最短距离，不可达输出 -1。", "最短路", solveDijkstra, ["5 6 1\n1 2 2\n1 3 5\n2 3 1\n2 4 2\n3 5 5\n4 5 1", "4 2 2\n1 2 7\n3 4 1"]],
  ["前缀词频查询", "给定单词集合和多个前缀，输出每个前缀匹配的单词数量。", "字典树", solveTriePrefix, ["5\napple\napp\nape\nbat\nbatch\n4\nap\napp\nba\ncat", "4\ncode\ncoder\ncoding\ncope\n3\nco\ncod\ncode"]],
  ["模式串出现次数", "给定文本串和模式串，统计模式串在文本中出现的次数，重叠出现也计数。", "KMP", solveKMP, ["abababa\naba", "aaaaa\naa"]],
  ["最长上升子序列长度", "给定整数序列，输出严格上升子序列的最大长度。", "动态规划", solveLIS, ["8\n10 9 2 5 3 7 101 18", "6\n1 1 1 1 1 1"]],
  ["01 背包最大价值", "每个物品最多选一次，在容量限制内求最大总价值。", "动态规划", solveKnapsack, ["4 7\n3 4\n4 5\n2 3\n5 8", "3 5\n2 6\n3 10\n4 12"]],
  ["树的最大深度", "给定一棵以 1 为根的无向树，输出根到最深节点的边数。", "树", solveTreeDepth, ["6\n1 2\n1 3\n2 4\n2 5\n5 6", "4\n1 2\n2 3\n3 4"]],
  ["二叉搜索树排名查询", "给定互不相同的键值集合，查询每个 x 在 BST 中序序列里的排名位置。", "二叉搜索树", solveBSTRank, ["5\n8 3 10 1 6\n4\n1\n5\n8\n11", "4\n4 2 9 7\n3\n3\n7\n10"]],
  ["逆序对数量", "输出数组中满足 i<j 且 a[i]>a[j] 的数对数量。", "树状数组", solveBITInversions, ["5\n5 3 2 4 1", "6\n1 3 2 3 1 2"]],
  ["单点修改区间最大值", "维护数组，操作 1 p x 表示修改 a[p]=x，操作 2 l r 查询区间最大值。", "线段树", solveSegmentMax, ["5 5\n1 5 2 4 3\n2 1 5\n1 3 9\n2 2 4\n1 5 0\n2 4 5", "4 4\n-1 -5 7 2\n2 1 2\n1 2 8\n2 1 3\n2 3 4"]],
  ["逆波兰表达式求值", "给定用空格分隔的逆波兰表达式，运算符包含 +、-、*，输出计算结果。", "栈", solveExpression, ["2 3 4 * +", "5 1 2 + 4 * + 3 -"]],
  ["最大的 K 个数", "给定数组和 K，按从大到小输出最大的 K 个数。", "堆", solveHeapTopK, ["7 3\n5 1 9 3 7 9 2", "5 2\n-1 -5 0 8 8"]],
  ["二维矩阵区域和", "给定矩阵和多个子矩形查询，输出每个子矩形元素和。", "二维前缀和", solveMatrixSum, ["3 3 3\n1 2 3\n4 5 6\n7 8 9\n1 1 2 2\n2 2 3 3\n1 2 3 2", "2 4 2\n1 -1 2 0\n3 4 -2 5\n1 1 2 4\n1 3 2 4"]],
  ["区间加法还原数组", "初始数组全为 0，执行多次区间加法后输出最终数组。", "差分数组", solveDiff, ["6 3\n1 3 2\n2 5 1\n4 6 -1", "5 4\n1 5 3\n2 2 -2\n3 4 5\n5 5 -1"]],
  ["选择 K 个点的最小跨度", "给定若干坐标，选择 K 个点，使最大坐标与最小坐标之差最小。", "排序二分", solveBinaryAnswer, ["7 3\n10 1 7 3 2 8 12", "5 4\n4 9 1 20 7"]]
];

function buildProblems() {
  const problems = [];
  for (let i = 0; i < topics.length; i++) {
    const [topic, note] = topics[i];
    for (let j = 0; j < 4; j++) {
      const base = templates[(i * 4 + j) % templates.length];
      const [name, desc, coreTag, solver, caseInputs] = base;
      const id = i * 4 + j + 1;
      const judgeCases = caseInputs.map((input) => ({ input, output: solver(input) }));
      const extraInput = caseInputs[j % caseInputs.length];
      const sampleCase = judgeCases.slice(0, 2);
      problems.push({
        title: `DS${String(id).padStart(3, "0")} ${topic}：${name}`,
        content: questionContent(id, topic, note, name, desc, sampleCase),
        tags: [topic, coreTag, "中等", "RAG题库"],
        answer: answerText(topic, coreTag, name),
        sampleCase,
        judgeCase: [
          ...judgeCases,
          { input: extraInput, output: solver(extraInput) }
        ],
        judgeConfig: { timeLimit: 1000, memoryLimit: 262144 }
      });
    }
  }
  return problems;
}

function questionContent(id, topic, note, name, desc, sampleCase) {
  const samples = sampleCase.map((c, i) => [
    `### 样例 ${i + 1}`,
    "输入：",
    "```",
    c.input,
    "```",
    "输出：",
    "```",
    c.output,
    "```"
  ].join("\n")).join("\n\n");
  return [
    "## 题目描述",
    `${desc} 本题归入 ${topic} 专题，考察 ${note}。`,
    "",
    "## 输入格式",
    "第一行通常给出规模参数，后续行给出数组、边、字符串或操作序列。具体含义见题目描述。",
    "",
    "## 输出格式",
    "对每个查询或整道题输出对应结果；多行答案按查询顺序输出。",
    "",
    "## 数据范围",
    "测试数据覆盖负数、重复值、孤立点、边界窗口、空查询结果和多次更新。平台样例规模较小，真实解法应按专题知识点设计。",
    "",
    samples,
    "",
    "## 提示",
    `第 ${id} 题用于 RAG 题库铺垫，不是纯入门语法题。建议先写清状态定义，再处理下标和边界。`
  ].join("\n");
}

function answerText(topic, coreTag, name) {
  return [
    `核心知识点：${topic} / ${coreTag}。`,
    `参考解法：先把「${name}」抽象成可维护状态，再根据操作类型进行更新或查询。`,
    "实现时需要注意输入可能包含重复值、负数、不可达节点或多次修改；输出前统一去除多余空格。"
  ].join("\n");
}

function validateProblems(problems) {
  if (problems.length !== 100) throw new Error(`expected 100 problems, got ${problems.length}`);
  for (const p of problems) {
    const payloads = [p.content, p.answer, JSON.stringify(p.sampleCase), JSON.stringify(p.judgeCase), JSON.stringify(p.judgeConfig)];
    if (p.title.length > 80) throw new Error(`title too long: ${p.title}`);
    if (p.content.length > 8192) throw new Error(`content too long: ${p.title}`);
    if (p.answer.length > 8192) throw new Error(`answer too long: ${p.title}`);
    if (payloads.some((x) => !x || x.includes("undefined"))) throw new Error(`bad payload: ${p.title}`);
  }
}

async function createQuestions() {
  const problems = buildProblems();
  validateProblems(problems);
  let count = 0;
  for (const p of problems) {
    await request(OJ_BASE, "/api/question/add", {
      method: "POST",
      body: JSON.stringify({
        title: p.title,
        content: p.content,
        tags: JSON.stringify(p.tags),
        answer: p.answer,
        sampleCase: JSON.stringify(p.sampleCase),
        judgeCase: JSON.stringify(p.judgeCase),
        judgeConfig: JSON.stringify(p.judgeConfig)
      })
    });
    count++;
  }
  console.log(`created ${count} OJ questions`);
}

if (process.env.SEED_DRY_RUN === "1") {
  const problems = buildProblems();
  validateProblems(problems);
  console.log(`dry run ok: ${topics.length * 4} blog posts, ${problems.length} OJ questions`);
} else {
  await ensureAccount();
  await createBlogs();
  await createQuestions();
  console.log("seed completed");
}
