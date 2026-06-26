package handler

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
	"yuoj-go-backend/internal/common"
	agentintegration "yuoj-go-backend/internal/integration/agent"
	blogintegration "yuoj-go-backend/internal/integration/blog"
	"yuoj-go-backend/internal/model/entity"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func (h *Handler) AgentGenerateQuestionSolution(c *gin.Context) {
	login := h.mustLoginUser(c)
	h.mustAdmin(c)
	if !h.cfg.Agent.Enabled {
		panic(common.NewBizError(common.OperationError, "Agent 功能未启用"))
	}
	var req struct {
		QuestionID int64 `json:"questionId"`
	}
	mustBindJSON(c, &req)
	if req.QuestionID <= 0 {
		panic(common.NewBizError(common.ParamsError, "请求参数错误"))
	}
	var q entity.Question
	mustNoErr(h.db.Where("id=? AND isDelete=0", req.QuestionID).First(&q).Error)
	task := entity.AgentSolutionTask{
		QuestionID:    req.QuestionID,
		TriggerUserID: login.ID,
		Status:        "pending",
		ModelName:     strings.TrimSpace(h.cfg.Agent.Model),
	}
	mustNoErr(h.db.Create(&task).Error)
	go h.runAgentSolutionTask(task.ID)
	c.JSON(http.StatusOK, common.Success(map[string]interface{}{
		"taskId": task.ID,
		"status": task.Status,
	}))
}

func (h *Handler) AgentGetSolutionTask(c *gin.Context) {
	h.mustAdmin(c)
	id := parseIDQuery(c)
	var task entity.AgentSolutionTask
	mustNoErr(h.db.Where("id = ? AND isDelete = 0", id).First(&task).Error)
	c.JSON(http.StatusOK, common.Success(task))
}

func (h *Handler) AgentListSolutionTask(c *gin.Context) {
	h.mustAdmin(c)
	var req struct {
		common.PageRequest
		QuestionID int64  `json:"questionId"`
		Status     string `json:"status"`
	}
	mustBindJSON(c, &req)
	req.Normalize()
	if req.PageSize > 20 {
		panic(common.NewBizError(common.ParamsError, "请求参数错误"))
	}
	query := h.db.Model(&entity.AgentSolutionTask{}).Where("isDelete = 0")
	if req.QuestionID > 0 {
		query = query.Where("questionId = ?", req.QuestionID)
	}
	if strings.TrimSpace(req.Status) != "" {
		query = query.Where("status = ?", strings.TrimSpace(req.Status))
	}
	var total int64
	mustNoErr(query.Count(&total).Error)
	var list []entity.AgentSolutionTask
	mustNoErr(query.Order("id desc").Offset(int((req.Current - 1) * req.PageSize)).Limit(int(req.PageSize)).Find(&list).Error)
	c.JSON(http.StatusOK, common.Success(common.PageResult{
		Records: list,
		Total:   total,
		Current: req.Current,
		Size:    req.PageSize,
	}))
}

func (h *Handler) runAgentSolutionTask(taskID int64) {
	var task entity.AgentSolutionTask
	if err := h.db.Where("id = ? AND isDelete = 0", taskID).First(&task).Error; err != nil {
		return
	}
	_ = h.db.Model(&entity.AgentSolutionTask{}).Where("id = ?", taskID).Updates(map[string]interface{}{
		"status": "running",
	}).Error

	var q entity.Question
	if err := h.db.Where("id = ? AND isDelete = 0", task.QuestionID).First(&q).Error; err != nil {
		h.failAgentTask(taskID, "题目不存在")
		return
	}
	agentUserID, err := h.ensureOJAgentUser()
	if err != nil {
		h.failAgentTask(taskID, "初始化 AI 账号失败: "+err.Error())
		return
	}

	maxRound := h.cfg.Agent.MaxRound
	if maxRound <= 0 {
		maxRound = 3
	}
	type sampleCase struct {
		Input  string `json:"input"`
		Output string `json:"output"`
	}
	var sampleCases []sampleCase
	_ = json.Unmarshal([]byte(q.SampleCase), &sampleCases)

	prompt := h.buildAgentPrompt(q, sampleCases)
	feedback := ""
	var finalMarkdown string
	var finalCode string
	published := false
	for i := 1; i <= maxRound; i++ {
		reply, err := h.callAgentLLM(prompt, feedback)
		if err != nil {
			h.failAgentTask(taskID, "调用模型失败: "+err.Error())
			return
		}
		title, summary, markdown, code := parseAgentReply(reply, q.Title)
		if strings.TrimSpace(code) == "" {
			feedback = "你没有提供可运行的 Go 代码，请输出完整可编译的 Go 代码。"
			continue
		}
		submitID, judgeMessage, judgeScore, err := h.submitAgentCodeAndJudge(agentUserID, q.ID, code)
		if err != nil {
			feedback = "提交评测失败：" + err.Error() + "。请修复后重试。"
			continue
		}
		finalCode = code
		finalMarkdown = markdown
		_ = h.db.Model(&entity.AgentSolutionTask{}).Where("id = ?", taskID).Updates(map[string]interface{}{
			"attempts":       i,
			"answerCode":     code,
			"answerMarkdown": markdown,
			"lastError":      fmt.Sprintf("latest submit id=%d", submitID),
		}).Error
		if strings.EqualFold(judgeMessage, "Accepted") {
			postID, postURL, err := h.publishAgentBlog(q.ID, title, summary, markdown)
			if err != nil {
				h.failAgentTask(taskID, "发布博客失败: "+err.Error())
				return
			}
			_ = h.db.Create(&entity.QuestionSolutionPost{
				QuestionID: q.ID,
				PostID:     postID,
				UserID:     agentUserID,
			}).Error
			_ = h.db.Model(&entity.AgentSolutionTask{}).Where("id = ?", taskID).Updates(map[string]interface{}{
				"status":      "success",
				"blogPostId":  postID,
				"blogPostUrl": postURL,
				"attempts":    i,
				"lastError":   "",
			}).Error
			published = true
			break
		}
		feedback = "判题结果: " + judgeMessage + fmt.Sprintf("，得分=%d。请继续优化代码并输出完整题解与代码。", judgeScore)
	}
	if !published {
		if strings.TrimSpace(finalCode) == "" && strings.TrimSpace(finalMarkdown) == "" {
			h.failAgentTask(taskID, "Agent 未生成有效内容")
			return
		}
		h.failAgentTask(taskID, "达到最大迭代次数，仍未通过评测")
	}
}

func (h *Handler) ensureOJAgentUser() (int64, error) {
	account := strings.TrimSpace(h.cfg.Blog.AgentAccount)
	if account == "" {
		account = "kkg_agent"
	}
	var u entity.User
	err := h.db.Where("userAccount = ? AND isDelete = 0", account).First(&u).Error
	if err == nil {
		if strings.EqualFold(u.UserRole, common.BanRole) {
			_ = h.db.Model(&entity.User{}).Where("id = ?", u.ID).Update("userRole", common.DefaultRole).Error
			u.UserRole = common.DefaultRole
		}
		return u.ID, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return 0, err
	}
	password := strings.TrimSpace(h.cfg.Blog.AgentPassword)
	if len(password) < 8 {
		return 0, errors.New("invalid config: blog.agent_password must be at least 8 characters")
	}
	u = entity.User{
		UserAccount:  account,
		UserPassword: hashAgentPassword(password),
		UserName:     strings.TrimSpace(h.cfg.Blog.AgentDisplayName),
		UserRole:     common.DefaultRole,
	}
	if strings.TrimSpace(u.UserName) == "" {
		u.UserName = "KKG Agent"
	}
	if err := h.db.Create(&u).Error; err != nil {
		return 0, err
	}
	return u.ID, nil
}

func (h *Handler) submitAgentCodeAndJudge(userID, questionID int64, code string) (int64, string, int64, error) {
	qs := entity.QuestionSubmit{
		Language:   "go",
		Code:       code,
		QuestionID: questionID,
		UserID:     userID,
		Status:     0,
		JudgeInfo:  "{}",
	}
	if err := h.db.Create(&qs).Error; err != nil {
		return 0, "", 0, err
	}
	_ = h.db.Model(&entity.Question{}).Where("id=?", questionID).Update("submitNum", gorm.Expr("submitNum + 1")).Error
	h.judgeAsync(qs.ID)
	var out entity.QuestionSubmit
	if err := h.db.Where("id = ? AND isDelete = 0", qs.ID).First(&out).Error; err != nil {
		return qs.ID, "", 0, err
	}
	var info struct {
		Message string `json:"message"`
		Score   int64  `json:"score"`
	}
	_ = json.Unmarshal([]byte(out.JudgeInfo), &info)
	msg := strings.TrimSpace(info.Message)
	if msg == "" {
		if out.Status == 2 {
			msg = "Accepted"
		} else if out.Status == 3 {
			msg = "Wrong Answer"
		} else {
			msg = "Unknown"
		}
	}
	return qs.ID, msg, info.Score, nil
}

func (h *Handler) failAgentTask(taskID int64, msg string) {
	_ = h.db.Model(&entity.AgentSolutionTask{}).Where("id = ?", taskID).Updates(map[string]interface{}{
		"status":    "failed",
		"lastError": msg,
	}).Error
}

func (h *Handler) buildAgentPrompt(q entity.Question, sampleCases interface{}) string {
	sb := strings.Builder{}
	sb.WriteString("你是资深算法工程师，请为以下题目生成高质量题解。")
	sb.WriteString("\n要求：")
	sb.WriteString("\n1) 明确知识点")
	sb.WriteString("\n2) 说明核心思路与复杂度")
	sb.WriteString("\n3) 提供可通过的 Go 代码")
	sb.WriteString("\n4) 使用 Markdown 输出，结构清晰")
	sb.WriteString("\n输出必须是 JSON，字段: title, summary, markdown, code。")
	sb.WriteString("\n题目标题：")
	sb.WriteString(q.Title)
	sb.WriteString("\n题目描述：\n")
	sb.WriteString(q.Content)
	sb.WriteString("\n样例（仅供理解）：\n")
	raw, _ := json.Marshal(sampleCases)
	sb.Write(raw)
	return sb.String()
}

func (h *Handler) callAgentLLM(prompt, feedback string) (string, error) {
	return agentintegration.NewClient(
		h.cfg.Agent.BaseURL,
		h.cfg.Agent.APIKey,
		h.cfg.Agent.Model,
		h.cfg.Agent.Temperature,
		90*time.Second,
	).Chat(prompt, feedback)
}

func parseAgentReply(reply string, questionTitle string) (title, summary, markdown, code string) {
	title = "题解：" + strings.TrimSpace(questionTitle)
	summary = "由 Agent 自动生成的题解"
	markdown = strings.TrimSpace(reply)
	code = ""
	var obj struct {
		Title    string `json:"title"`
		Summary  string `json:"summary"`
		Markdown string `json:"markdown"`
		Code     string `json:"code"`
	}
	if err := json.Unmarshal([]byte(reply), &obj); err == nil {
		if strings.TrimSpace(obj.Title) != "" {
			title = strings.TrimSpace(obj.Title)
		}
		if strings.TrimSpace(obj.Summary) != "" {
			summary = strings.TrimSpace(obj.Summary)
		}
		if strings.TrimSpace(obj.Markdown) != "" {
			markdown = strings.TrimSpace(obj.Markdown)
		}
		if strings.TrimSpace(obj.Code) != "" {
			code = strings.TrimSpace(obj.Code)
		}
	}
	if code == "" {
		start := strings.Index(markdown, "```go")
		if start >= 0 {
			rest := markdown[start+5:]
			end := strings.Index(rest, "```")
			if end > 0 {
				code = strings.TrimSpace(rest[:end])
			}
		}
	}
	return
}

func (h *Handler) publishAgentBlog(questionID int64, title, summary, markdown string) (int64, string, error) {
	return blogintegration.NewClient(h.cfg.Blog.BaseURL, h.cfg.Blog.InternalAuthToken, 30*time.Second).PublishAgentPost(blogintegration.AgentPostRequest{
		QuestionID: questionID,
		Title:      title,
		Summary:    summary,
		Markdown:   markdown,
		Account:    h.cfg.Blog.AgentAccount,
		Password:   h.cfg.Blog.AgentPassword,
		Email:      h.cfg.Blog.AgentEmail,
	})
}

func hashAgentPassword(password string) string {
	sum := md5.Sum([]byte(agentPasswordSalt + password))
	return hex.EncodeToString(sum[:])
}
