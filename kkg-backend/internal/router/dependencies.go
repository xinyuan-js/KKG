package router

import (
	"kkg-backend/internal/bootstrap"
	"kkg-backend/internal/handler"
	ojhandler "kkg-backend/internal/oj/handler"
	"kkg-backend/internal/repository"
	"kkg-backend/internal/service"
)

type dependencies struct {
	authHandler       *handler.AuthHandler
	postHandler       *handler.PostHandler
	commentHandler    *handler.CommentHandler
	notification      *handler.NotificationHandler
	userHandler       *handler.UserHandler
	uploadHandler     *handler.UploadHandler
	tweetHandler      *handler.TweetHandler
	searchHandler     *handler.SearchHandler
	adminAuditHandler *handler.AdminAuditHandler
	healthHandler     *handler.HealthHandler
	internalHandler   *handler.InternalHandler
	ojHandler         *ojhandler.Handler
}

func newDependencies(app *bootstrap.App) dependencies {
	userRepo := repository.NewUserRepository(app.DB)
	postRepo := repository.NewPostRepository(app.DB)
	commentRepo := repository.NewCommentRepository(app.DB)
	notificationRepo := repository.NewNotificationRepository(app.DB)
	tweetRepo := repository.NewTweetRepository(app.DB)
	adminAuditRepo := repository.NewAdminAuditRepository(app.DB)

	authService := service.NewAuthService(userRepo, app.Sessions)
	postService := service.NewPostService(postRepo, app.Redis, app.ES, app.Config.ESPostIndex)
	commentService := service.NewCommentService(commentRepo, postRepo, notificationRepo)
	notificationService := service.NewNotificationService(notificationRepo)
	userService := service.NewUserService(userRepo, postRepo)
	tweetService := service.NewTweetService(tweetRepo, app.ES, app.Config.ElasticsearchIndex)
	searchService := service.NewSearchService(app.ES, app.Config.ESPostIndex, app.Config.ESUserIndex)
	adminAuditService := service.NewAdminAuditService(adminAuditRepo)

	return dependencies{
		authHandler:       handler.NewAuthHandler(authService),
		postHandler:       handler.NewPostHandler(postService, adminAuditService),
		commentHandler:    handler.NewCommentHandler(commentService),
		notification:      handler.NewNotificationHandler(notificationService),
		userHandler:       handler.NewUserHandler(userService, adminAuditService),
		uploadHandler:     handler.NewUploadHandler(app.Storage),
		tweetHandler:      handler.NewTweetHandler(tweetService),
		searchHandler:     handler.NewSearchHandler(searchService, userService),
		adminAuditHandler: handler.NewAdminAuditHandler(adminAuditService),
		healthHandler:     handler.NewHealthHandler(app),
		internalHandler:   handler.NewInternalHandler(userRepo, postService),
		ojHandler:         newOJHandler(app),
	}
}
