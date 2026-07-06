package router

import (
	"github.com/gin-gonic/gin"
	"github.com/quocdev03/user-access-management/internal/constant"
	"github.com/quocdev03/user-access-management/internal/handler"
	"github.com/quocdev03/user-access-management/internal/middleware"
	"github.com/quocdev03/user-access-management/internal/repository"
)

func setupAdminRoutes(rg *gin.RouterGroup, adminHandler *handler.AdminHandler, authMw gin.HandlerFunc, roleRepo *repository.RoleRepository) {
	admin := rg.Group("/admin")
	admin.Use(authMw)
	admin.Use(middleware.RequireRole(constant.RoleAdmin, constant.RoleModerator))

	users := admin.Group("/users")
	{
		users.GET("", middleware.PermissionMiddleware(roleRepo, "users.read"), adminHandler.ListUsers)
		users.GET("/:id", middleware.PermissionMiddleware(roleRepo, "users.read"), adminHandler.GetUserDetail)

		users.PUT("/:id", middleware.PermissionMiddleware(roleRepo, "users.update"), adminHandler.UpdateUser)

		users.PATCH("/:id/status", middleware.PermissionMiddleware(roleRepo, "users.lock"), adminHandler.ChangeUserStatus)
		users.POST("/:id/password/reset", middleware.PermissionMiddleware(roleRepo, "users.reset_password"), adminHandler.ResetPassword)

		users.POST("/:id/notify", middleware.PermissionMiddleware(roleRepo, "users.notify"), adminHandler.NotifyUser)
	}

	roles := admin.Group("/roles")
	{
		roles.GET("", middleware.PermissionMiddleware(roleRepo, "roles.read"), adminHandler.ListRoles)
		roles.POST("", middleware.PermissionMiddleware(roleRepo, "roles.create"), adminHandler.CreateRole)
		roles.PUT("/:id", middleware.PermissionMiddleware(roleRepo, "roles.update"), adminHandler.UpdateRole)
		roles.DELETE("/:id", middleware.PermissionMiddleware(roleRepo, "roles.delete"), adminHandler.DeleteRole)
		roles.PUT("/:id/permissions", middleware.PermissionMiddleware(roleRepo, "permissions.assign"), adminHandler.AssignPermissions)
	}

	admin.POST("/users/:id/roles", middleware.PermissionMiddleware(roleRepo, "roles.assign"), adminHandler.AssignUserRole)
	admin.DELETE("/users/:id/roles/:roleId", middleware.PermissionMiddleware(roleRepo, "roles.assign"), adminHandler.RemoveUserRole)

	auditLogs := admin.Group("/audit-logs")
	{
		auditLogs.GET("/export", middleware.PermissionMiddleware(roleRepo, "audit_logs.export"), adminHandler.Export)
	}

}
