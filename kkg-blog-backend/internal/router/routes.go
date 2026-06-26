package router

import (
	"awesomeProject/internal/bootstrap"
	"awesomeProject/internal/middleware"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r *gin.Engine, app *bootstrap.App, deps dependencies) {
	r.GET("/health", deps.healthHandler.Ping)

	v1 := r.Group("/api/v1")
	{
		registerInternalRoutes(v1, app, deps)
		registerAuthRoutes(v1, app, deps)
		registerPublicRoutes(v1, app, deps)
		registerAuthedRoutes(v1, app, deps)
	}
}

func registerInternalRoutes(v1 *gin.RouterGroup, app *bootstrap.App, deps dependencies) {
	internal := v1.Group("/internal")
	internal.Use(middleware.InternalAuth(app.Config.InternalAuthToken))
	{
		internal.POST("/agent/posts", deps.internalHandler.PublishAgentPost)
	}
}

func registerAuthRoutes(v1 *gin.RouterGroup, app *bootstrap.App, deps dependencies) {
	authGroup := v1.Group("/auth")
	{
		authGroup.POST("/register", deps.authHandler.Register)
		authGroup.POST("/login", deps.authHandler.Login)
		authGroup.POST("/refresh", deps.authHandler.Refresh)
		authGroup.POST("/logout", deps.authHandler.Logout)
		authGroup.GET("/me", middleware.JWT(app.Config.JWTSecretKey), middleware.RequireActiveUser(app.DB), deps.authHandler.GetProfile)
	}
}

func registerPublicRoutes(v1 *gin.RouterGroup, app *bootstrap.App, deps dependencies) {
	v1.GET("/posts", deps.postHandler.ListPublished)
	v1.GET("/feed", middleware.OptionalJWT(app.Config.JWTSecretKey), deps.postHandler.ListFeed)
	v1.GET("/rankings/posts", deps.postHandler.Ranking)
	v1.GET("/posts/:id", deps.postHandler.Get)
	v1.GET("/posts/:id/engagement", middleware.OptionalJWT(app.Config.JWTSecretKey), deps.postHandler.GetEngagement)
	v1.GET("/posts/:id/comments", deps.commentHandler.ListByPost)
	v1.GET("/users/:id", deps.userHandler.GetPublicPage)
	v1.GET("/tweets/search", deps.tweetHandler.Search)
	v1.GET("/search", middleware.OptionalJWT(app.Config.JWTSecretKey), deps.searchHandler.Search)
	v1.GET("/search/suggest", middleware.OptionalJWT(app.Config.JWTSecretKey), deps.searchHandler.Suggest)
}

func registerAuthedRoutes(v1 *gin.RouterGroup, app *bootstrap.App, deps dependencies) {
	authed := v1.Group("")
	authed.Use(middleware.JWT(app.Config.JWTSecretKey))
	authed.Use(middleware.RequireActiveUser(app.DB))
	{
		authed.GET("/me/profile", deps.authHandler.GetProfile)
		authed.PUT("/me/profile", deps.authHandler.UpdateProfile)
		authed.PUT("/me/password", deps.authHandler.ChangePassword)
		authed.GET("/me/notifications", deps.notification.ListMine)
		authed.POST("/me/notifications/:id/read", deps.notification.MarkRead)
		authed.GET("/me/posts", deps.postHandler.ListMine)
		authed.GET("/admin/posts", deps.postHandler.AdminList)
		authed.GET("/admin/users", deps.userHandler.AdminList)
		authed.PUT("/admin/users/role", deps.userHandler.AdminUpdateRole)
		authed.DELETE("/admin/users/:id", deps.userHandler.AdminDelete)
		authed.GET("/admin/audits", deps.adminAuditHandler.List)
		authed.POST("/admin/audits", deps.adminAuditHandler.Create)
		authed.GET("/me/favorites", deps.postHandler.ListMyFavorites)
		authed.GET("/me/posts/:id", deps.postHandler.GetMine)
		authed.POST("/posts", deps.postHandler.CreateDraft)
		authed.PUT("/posts/:id/meta", deps.postHandler.UpdateMeta)
		authed.DELETE("/posts/:id", deps.postHandler.DeletePost)
		authed.GET("/posts/:id/drafts", deps.postHandler.ListDrafts)
		authed.POST("/posts/:id/drafts", deps.postHandler.CreateDraftCopy)
		authed.GET("/posts/:id/drafts/:version", deps.postHandler.GetDraft)
		authed.PUT("/posts/:id/drafts/:version", deps.postHandler.SaveDraftByVersion)
		authed.DELETE("/posts/:id/drafts/:version", deps.postHandler.DeleteDraft)
		authed.POST("/posts/:id/drafts/:version/publish", deps.postHandler.PublishDraft)
		authed.GET("/posts/:id/versions", deps.postHandler.ListVersions)
		authed.DELETE("/posts/:id/versions/:version", deps.postHandler.DeleteVersion)
		authed.PUT("/posts/:id/draft", deps.postHandler.SaveDraft)
		authed.POST("/posts/:id/publish", deps.postHandler.Publish)
		authed.POST("/posts/:id/unpublish", deps.postHandler.Unpublish)
		authed.POST("/posts/:id/rollback/:version", deps.postHandler.Rollback)
		authed.POST("/posts/:id/comments", deps.commentHandler.CreateByPost)
		authed.POST("/posts/:id/like", deps.postHandler.ToggleLike)
		authed.POST("/posts/:id/favorite", deps.postHandler.ToggleFavorite)
		authed.POST("/uploads/image", deps.uploadHandler.UploadImage)
		authed.POST("/tweets", deps.tweetHandler.Create)
	}
}
