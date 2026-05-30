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

  try {
    await request(BLOG_BASE, "/api/v1/auth/login", {
      method: "POST",
      body: JSON.stringify({ account: ACCOUNT, password: PASSWORD })
    });
  } catch (err) {
    throw new Error(`cannot login as ${ACCOUNT}; set SEED_PASSWORD to the account password or reset it manually. ${err.message}`);
  }
  const me = await request(BLOG_BASE, "/api/v1/auth/me");
  console.log(`logged in as ${me.username}#${me.id}`);
}

function postBody(index, title, topic) {
  return [
    `# ${title}`,
    "",
    `这是一篇用于测试系统展示、搜索和推荐的算法短文。主题是 **${topic}**，内容保持简洁，便于快速阅读和验证页面效果。`,
    "",
    "## 问题场景",
    "",
    "在刷题或业务开发里，我们经常需要把一个看起来复杂的问题拆成输入、状态、转移或统计过程。好的算法设计通常不是从代码开始，而是先确认数据规模和不变量。",
    "",
    "## 解法思路",
    "",
    "1. 明确输入输出。",
    "2. 找到可以复用的中间状态。",
    "3. 用简单样例手动推导。",
    "4. 最后再写代码，并补充边界测试。",
    "",
    "## 复杂度",
    "",
    "- 时间复杂度通常控制在 `O(n)` 或 `O(n log n)`。",
    "- 空间复杂度优先保持在 `O(1)` 或 `O(n)`。",
    "",
    "## 小结",
    "",
    `掌握 ${topic} 的关键，是先把题目抽象成稳定的模型，再用测试用例覆盖边界情况。`
  ].join("\n");
}

const blogTopics = [
  ["从数组求和理解输入输出建模", "数组遍历"],
  ["双指针入门：有序数组中的目标和", "双指针"],
  ["哈希表计数的三个常见用法", "哈希表"],
  ["前缀和为什么能加速区间查询", "前缀和"],
  ["差分数组的直觉与适用边界", "差分数组"],
  ["栈结构与括号匹配问题", "栈"],
  ["队列和广度优先搜索的关系", "队列"],
  ["二分查找的边界写法", "二分查找"],
  ["排序后能简化哪些问题", "排序"],
  ["贪心算法的交换论证直觉", "贪心"],
  ["动态规划的状态定义练习", "动态规划"],
  ["一维 DP：爬楼梯模型", "动态规划"],
  ["字符串扫描与字符统计", "字符串"],
  ["滑动窗口解决连续区间问题", "滑动窗口"],
  ["链表题中的虚拟头节点", "链表"],
  ["树的递归遍历基础", "二叉树"],
  ["图搜索中的 visited 数组", "图论"],
  ["并查集如何维护连通性", "并查集"],
  ["取模运算在计数题中的使用", "数学"],
  ["如何为算法题设计测试用例", "测试用例"]
];

async function createBlogs() {
  let count = 0;
  for (let i = 0; i < blogTopics.length; i++) {
    const [title, topic] = blogTopics[i];
    const post = await request(BLOG_BASE, "/api/v1/posts", {
      method: "POST",
      body: JSON.stringify({
        title,
        slug: `aimin-algo-note-${String(i + 1).padStart(2, "0")}`,
        summary: `算法测试文章：${topic} 的基本思路、复杂度和边界样例。`,
        tags: ["算法", topic, "测试数据"],
        draft_note: "seed content",
        raw_content: postBody(i + 1, title, topic)
      })
    });
    await request(BLOG_BASE, `/api/v1/posts/${post.id}/publish`, { method: "POST" });
    count++;
  }
  console.log(`created and published ${count} blog posts`);
}

const problemFactories = [
  (i) => ({
    title: `A${i} 两数相加`,
    tags: ["入门", "数学"],
    content: "输入两个整数 a 和 b，输出它们的和。",
    answer: "读取两个整数，输出 a+b。",
    cases: [["1 2", "3"], ["-5 8", "3"], ["100 200", "300"]]
  }),
  (i) => ({
    title: `A${i} 求最大值`,
    tags: ["入门", "数组"],
    content: "输入 n 和 n 个整数，输出其中的最大值。",
    answer: "遍历数组，维护当前最大值。",
    cases: [["5\n1 3 2 9 4", "9"], ["3\n-7 -2 -9", "-2"], ["1\n42", "42"]]
  }),
  (i) => ({
    title: `A${i} 统计偶数`,
    tags: ["入门", "计数"],
    content: "输入 n 和 n 个整数，统计其中偶数的个数。",
    answer: "逐个判断 x%2==0 并计数。",
    cases: [["5\n1 2 3 4 5", "2"], ["4\n0 -2 7 8", "3"], ["3\n1 3 5", "0"]]
  }),
  (i) => ({
    title: `A${i} 字符串长度`,
    tags: ["字符串", "入门"],
    content: "输入一个不含空格的字符串，输出它的长度。",
    answer: "读取字符串后输出 len(s)。",
    cases: [["abc", "3"], ["algorithm", "9"], ["x", "1"]]
  }),
  (i) => ({
    title: `A${i} 判断奇偶`,
    tags: ["数学", "条件"],
    content: "输入一个整数，若为偶数输出 Even，否则输出 Odd。",
    answer: "根据 n%2 判断。",
    cases: [["2", "Even"], ["7", "Odd"], ["0", "Even"]]
  }),
  (i) => ({
    title: `A${i} 求 1 到 n 的和`,
    tags: ["数学", "循环"],
    content: "输入正整数 n，输出 1+2+...+n。",
    answer: "可以循环累加，也可以使用 n*(n+1)/2。",
    cases: [["1", "1"], ["10", "55"], ["100", "5050"]]
  }),
  (i) => ({
    title: `A${i} 反转字符串`,
    tags: ["字符串", "双指针"],
    content: "输入一个不含空格的字符串，输出反转后的结果。",
    answer: "从后向前输出字符。",
    cases: [["abc", "cba"], ["level", "level"], ["go", "og"]]
  }),
  (i) => ({
    title: `A${i} 绝对值`,
    tags: ["数学", "条件"],
    content: "输入一个整数，输出它的绝对值。",
    answer: "若 x<0 输出 -x，否则输出 x。",
    cases: [["-3", "3"], ["0", "0"], ["18", "18"]]
  }),
  (i) => ({
    title: `A${i} 统计字符 a`,
    tags: ["字符串", "计数"],
    content: "输入一个小写字符串，输出字符 a 出现的次数。",
    answer: "遍历字符串计数。",
    cases: [["banana", "3"], ["aaaa", "4"], ["test", "0"]]
  }),
  (i) => ({
    title: `A${i} 三数最大值`,
    tags: ["入门", "条件"],
    content: "输入三个整数，输出最大值。",
    answer: "依次比较三个数。",
    cases: [["1 2 3", "3"], ["9 4 7", "9"], ["-1 -5 -3", "-1"]]
  })
];

function questionContent(p, idx) {
  const samples = p.cases.slice(0, 2).map(([input, output], n) => {
    return `样例 ${n + 1}\n输入：\n${input}\n输出：\n${output}`;
  }).join("\n\n");
  return [
    `## 题目描述`,
    p.content,
    "",
    "## 输入格式",
    "按题目描述输入，所有数据规模较小，适合用于系统功能测试。",
    "",
    "## 输出格式",
    "输出题目要求的结果。",
    "",
    "## 样例",
    samples,
    "",
    "## 说明",
    `这是第 ${idx} 道 seed 题，难度为简单。`
  ].join("\n");
}

async function createQuestions() {
  let count = 0;
  for (let i = 1; i <= 100; i++) {
    const p = problemFactories[(i - 1) % problemFactories.length](i);
    await request(OJ_BASE, "/api/question/add", {
      method: "POST",
      body: JSON.stringify({
        title: p.title,
        content: questionContent(p, i),
        tags: JSON.stringify([...p.tags, "seed"]),
        answer: p.answer,
        sampleCase: JSON.stringify(p.cases.slice(0, 2).map(([input, output]) => ({ input, output }))),
        judgeCase: JSON.stringify(p.cases.map(([input, output]) => ({ input, output }))),
        judgeConfig: JSON.stringify({ timeLimit: 1000, memoryLimit: 262144 })
      })
    });
    count++;
  }
  console.log(`created ${count} OJ questions`);
}

await ensureAccount();
await createBlogs();
await createQuestions();
console.log("seed completed");
