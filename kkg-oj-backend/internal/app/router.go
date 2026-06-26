package app

import (
	"yuoj-go-backend/internal/handler"
	"yuoj-go-backend/internal/middleware"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r *gin.Engine, h *handler.Handler) {
	api := r.Group("/api")
	authRequired := middleware.MustLogin(h.UserService(), h.JWTSecret())

	user := api.Group("/user")
	{
		user.POST("/register", h.UserRegister)
		user.POST("/login", h.UserLogin)
		user.GET("/login/wx_open", h.UserWxLogin)
		user.POST("/logout", h.UserLogout)
		user.GET("/get/login", authRequired, h.UserGetLogin)
		user.POST("/add", authRequired, h.UserAdd)
		user.POST("/delete", authRequired, h.UserDelete)
		user.POST("/update", authRequired, h.UserUpdate)
		user.GET("/get", authRequired, h.UserGet)
		user.GET("/get/vo", h.UserGetVO)
		user.POST("/list/page", authRequired, h.UserList)
		user.POST("/list/page/vo", h.UserListVO)
		user.POST("/update/my", authRequired, h.UserUpdateMy)
	}

	question := api.Group("/question")
	{
		question.POST("/add", authRequired, h.QuestionAdd)
		question.POST("/delete", authRequired, h.QuestionDelete)
		question.POST("/restore", authRequired, h.QuestionRestore)
		question.POST("/update", authRequired, h.QuestionUpdate)
		question.GET("/get", authRequired, h.QuestionGet)
		question.GET("/get/vo", h.QuestionGetVO)
		question.POST("/list/page/vo", h.QuestionListVO)
		question.POST("/my/list/page/vo", authRequired, h.QuestionMyListVO)
		question.POST("/list/page", authRequired, h.QuestionList)
		question.POST("/edit", authRequired, h.QuestionEdit)
		question.POST("/run", authRequired, h.QuestionRun)
		question.POST("/question_submit/do", authRequired, h.QuestionSubmitDo)
		question.POST("/question_submit/list/page", authRequired, h.QuestionSubmitList)
		question.GET("/rank/first-ac-24h", h.QuestionFirstACRank24h)
		question.GET("/submission/events", authRequired, h.SubmissionEvents)
		question.POST("/solution/bind", authRequired, h.QuestionSolutionBind)
		question.POST("/solution/unbind", authRequired, h.QuestionSolutionUnbind)
		question.POST("/solution/list/page", h.QuestionSolutionList)
		question.POST("/agent/solution/generate", authRequired, h.AgentGenerateQuestionSolution)
		question.GET("/agent/solution/task", authRequired, h.AgentGetSolutionTask)
		question.POST("/agent/solution/task/list/page", authRequired, h.AgentListSolutionTask)
	}

	// OJ 前台已与博客内容解耦，这里不再对外暴露 post / post_thumb / post_favour 路由。

	file := api.Group("/file")
	file.POST("/upload", authRequired, h.FileUpload)

	api.GET("/", h.WxGet)
	api.POST("/", h.WxPost)
	api.GET("/setMenu", h.WxSetMenu)
}
