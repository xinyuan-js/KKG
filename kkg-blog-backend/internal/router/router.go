package router

import (
	"awesomeProject/internal/bootstrap"
	"awesomeProject/internal/handler"
	"awesomeProject/internal/middleware"
	"awesomeProject/internal/repository"
	"awesomeProject/internal/service"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func New(app *bootstrap.App) *gin.Engine {
	r := gin.Default()
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://127.0.0.1:3001", "http://localhost:3001"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
	}))

	healthHandler := handler.NewHealthHandler(app)
	userRepo := repository.NewUserRepository(app.DB)
	postRepo := repository.NewPostRepository(app.DB)
	commentRepo := repository.NewCommentRepository(app.DB)
	notificationRepo := repository.NewNotificationRepository(app.DB)
	tweetRepo := repository.NewTweetRepository(app.DB)
	adminAuditRepo := repository.NewAdminAuditRepository(app.DB)
	authService := service.NewAuthService(userRepo, app.Config.JWTSecretKey)
	postService := service.NewPostService(postRepo, app.Redis, app.ES, app.Config.ESPostIndex)
	commentService := service.NewCommentService(commentRepo, postRepo, notificationRepo)
	notificationService := service.NewNotificationService(notificationRepo)
	userService := service.NewUserService(userRepo, postRepo)
	tweetService := service.NewTweetService(tweetRepo, app.ES, app.Config.ElasticsearchIndex)
	searchService := service.NewSearchService(app.ES, app.Config.ESPostIndex, app.Config.ESUserIndex)
	adminAuditService := service.NewAdminAuditService(adminAuditRepo)
	authHandler := handler.NewAuthHandler(authService)
	postHandler := handler.NewPostHandler(postService, adminAuditService)
	commentHandler := handler.NewCommentHandler(commentService)
	notificationHandler := handler.NewNotificationHandler(notificationService)
	userHandler := handler.NewUserHandler(userService, adminAuditService)
	uploadHandler := handler.NewUploadHandler(app.Storage)
	tweetHandler := handler.NewTweetHandler(tweetService)
	searchHandler := handler.NewSearchHandler(searchService, userService)
	adminAuditHandler := handler.NewAdminAuditHandler(adminAuditService)

	r.GET("/health", healthHandler.Ping)

	v1 := r.Group("/api/v1")
	{
		authGroup := v1.Group("/auth")
		{
			authGroup.POST("/register", authHandler.Register)
			authGroup.POST("/login", authHandler.Login)
		}

		v1.GET("/posts", postHandler.ListPublished)
		v1.GET("/feed", middleware.OptionalJWT(app.Config.JWTSecretKey), postHandler.ListFeed)
		v1.GET("/rankings/posts", postHandler.Ranking)
		v1.GET("/posts/:id", postHandler.Get)
		v1.GET("/posts/:id/engagement", middleware.OptionalJWT(app.Config.JWTSecretKey), postHandler.GetEngagement)
		v1.GET("/posts/:id/comments", commentHandler.ListByPost)
		v1.GET("/users/:id", userHandler.GetPublicPage)
		v1.GET("/tweets/search", tweetHandler.Search)
		v1.GET("/search", middleware.OptionalJWT(app.Config.JWTSecretKey), searchHandler.Search)
		v1.GET("/search/suggest", middleware.OptionalJWT(app.Config.JWTSecretKey), searchHandler.Suggest)

		authed := v1.Group("")
		authed.Use(middleware.JWT(app.Config.JWTSecretKey))
		authed.Use(middleware.RequireActiveUser(app.DB))
		{
			authed.GET("/me/profile", authHandler.GetProfile)
			authed.PUT("/me/profile", authHandler.UpdateProfile)
			authed.PUT("/me/password", authHandler.ChangePassword)
			authed.GET("/me/notifications", notificationHandler.ListMine)
			authed.POST("/me/notifications/:id/read", notificationHandler.MarkRead)
			authed.GET("/me/posts", postHandler.ListMine)
			authed.GET("/admin/posts", postHandler.AdminList)
			authed.GET("/admin/users", userHandler.AdminList)
			authed.PUT("/admin/users/role", userHandler.AdminUpdateRole)
			authed.DELETE("/admin/users/:id", userHandler.AdminDelete)
			authed.GET("/admin/audits", adminAuditHandler.List)
			authed.POST("/admin/audits", adminAuditHandler.Create)
			authed.GET("/me/favorites", postHandler.ListMyFavorites)
			authed.GET("/me/posts/:id", postHandler.GetMine)
			authed.POST("/posts", postHandler.CreateDraft)
			authed.PUT("/posts/:id/meta", postHandler.UpdateMeta)
			authed.DELETE("/posts/:id", postHandler.DeletePost)
			authed.GET("/posts/:id/drafts", postHandler.ListDrafts)
			authed.POST("/posts/:id/drafts", postHandler.CreateDraftCopy)
			authed.GET("/posts/:id/drafts/:version", postHandler.GetDraft)
			authed.PUT("/posts/:id/drafts/:version", postHandler.SaveDraftByVersion)
			authed.DELETE("/posts/:id/drafts/:version", postHandler.DeleteDraft)
			authed.POST("/posts/:id/drafts/:version/publish", postHandler.PublishDraft)
			authed.GET("/posts/:id/versions", postHandler.ListVersions)
			authed.DELETE("/posts/:id/versions/:version", postHandler.DeleteVersion)
			authed.PUT("/posts/:id/draft", postHandler.SaveDraft)
			authed.POST("/posts/:id/publish", postHandler.Publish)
			authed.POST("/posts/:id/unpublish", postHandler.Unpublish)
			authed.POST("/posts/:id/rollback/:version", postHandler.Rollback)
			authed.POST("/posts/:id/comments", commentHandler.CreateByPost)
			authed.POST("/posts/:id/like", postHandler.ToggleLike)
			authed.POST("/posts/:id/favorite", postHandler.ToggleFavorite)
			authed.POST("/uploads/image", uploadHandler.UploadImage)
			authed.POST("/tweets", tweetHandler.Create)
		}
	}

	return r
}
