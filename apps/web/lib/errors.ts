const ZH_RE = /[\u4e00-\u9fa5]/;

const MESSAGE_MAP: Array<[RegExp, string]> = [
  [/fetch posts failed/i, "加载文章列表失败，请稍后重试"],
  [/get feed failed/i, "加载内容流失败，请稍后重试"],
  [/get engagement failed/i, "加载互动数据失败"],
  [/toggle like failed/i, "点赞操作失败，请稍后重试"],
  [/toggle favorite failed/i, "收藏操作失败，请稍后重试"],
  [/get rankings failed/i, "加载排行榜失败，请稍后重试"],
  [/get my favorites failed/i, "加载收藏列表失败，请稍后重试"],
  [/get post detail failed/i, "加载文章详情失败"],
  [/get comments failed/i, "加载评论失败"],
  [/create comment failed/i, "评论发布失败，请稍后重试"],
  [/get notifications failed/i, "加载通知失败"],
  [/mark notification read failed/i, "通知状态更新失败"],
  [/get user page failed/i, "加载用户主页失败"],
  [/search tweets failed/i, "搜索失败，请稍后重试"],
  [/search failed/i, "搜索失败，请稍后重试"],
  [/suggest failed/i, "推荐词加载失败，请稍后重试"],
  [/login failed/i, "登录失败，请检查账号或密码"],
  [/register failed/i, "注册失败，请稍后重试"],
  [/create draft failed/i, "创建草稿失败"],
  [/get my posts failed/i, "加载我的文章失败"],
  [/get profile failed/i, "获取个人信息失败"],
  [/update profile failed/i, "更新个人信息失败"],
  [/upload image failed/i, "图片上传失败，请稍后重试"],
  [/change password failed/i, "修改密码失败，请稍后重试"],
  [/publish failed/i, "发布失败，请稍后重试"],
  [/unpublish failed/i, "撤回发布失败，请稍后重试"],
  [/create post draft failed/i, "创建文章草稿失败"],
  [/update post meta failed/i, "保存文章设置失败"],
  [/save draft failed/i, "保存草稿失败，请稍后重试"],
  [/get drafts failed/i, "加载草稿列表失败"],
  [/get draft failed/i, "加载草稿失败"],
  [/publish draft failed/i, "发布草稿失败"],
  [/delete draft failed/i, "删除草稿失败"],
  [/delete post failed/i, "删除文章失败"],
  [/query is required/i, "请输入搜索关键词"],
  [/invalid post id/i, "文章参数无效"],
  [/invalid version/i, "版本参数无效"],
  [/post not found/i, "文章不存在或已删除"],
  [/version not found/i, "草稿版本不存在"],
  [/forbidden/i, "没有访问权限"],
  [/unauthorized/i, "登录状态已失效，请重新登录"],
  [/invalid user context/i, "用户状态异常，请重新登录"],
  [/slug already exists for current user/i, "链接标识重复，请换一个标题后重试"],
  [/networkerror/i, "网络异常，请检查连接后重试"]
];

export function toZhError(err: unknown, fallback = "操作失败，请稍后重试"): string {
  const raw = err instanceof Error ? err.message : "";
  const msg = String(raw || "").trim();
  if (!msg) return fallback;
  if (ZH_RE.test(msg)) return msg;

  for (const [rule, text] of MESSAGE_MAP) {
    if (rule.test(msg)) return text;
  }

  const lower = msg.toLowerCase();
  if (lower.includes("failed: 401")) return "登录状态已失效，请重新登录";
  if (lower.includes("failed: 403")) return "没有访问权限";
  if (lower.includes("failed: 404")) return "请求的内容不存在";
  if (lower.includes("failed: 429")) return "请求过于频繁，请稍后重试";
  if (lower.includes("failed: 5")) return "服务暂时不可用，请稍后再试";
  return fallback;
}
