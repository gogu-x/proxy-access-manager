package model

import (
	"time"
)

type User struct {
	ID           string    `json:"id" bson:"_id"`
	Username     string    `json:"username" bson:"username"`
	PasswordHash string    `json:"-" bson:"password_hash"`
	Role         string    `json:"role" bson:"role"`
	ProjectIDs   []string  `json:"project_ids" bson:"project_ids"`
	Permissions  []string  `json:"permissions" bson:"permissions"`
	Disabled     bool      `json:"disabled" bson:"disabled"`
	CreatedAt    time.Time `json:"created_at" bson:"created_at"`
}

const (
	RoleAdmin = "admin"
	RoleUser  = "user"

	PermissionDashboardView  = "dashboard.view"
	PermissionProjectsView   = "projects.view"
	PermissionServicesView   = "services.view"
	PermissionServicesManage = "services.manage"
	PermissionHostsView      = "hosts.view"
	PermissionHostsManage    = "hosts.manage"
)

var UserPermissions = []string{
	PermissionDashboardView,
	PermissionProjectsView,
	PermissionServicesView,
	PermissionServicesManage,
	PermissionHostsView,
	PermissionHostsManage,
}

func (u User) HasProject(projectID string) bool {
	if u.Role == RoleAdmin {
		return true
	}
	for _, id := range u.ProjectIDs {
		if id == projectID {
			return true
		}
	}
	return false
}

func (u User) HasPermission(permission string) bool {
	if u.Role == RoleAdmin {
		return true
	}
	for _, item := range u.Permissions {
		if item == permission {
			return true
		}
	}
	if permission == PermissionServicesView {
		return u.HasPermission(PermissionServicesManage)
	}
	if permission == PermissionHostsView {
		return u.HasPermission(PermissionHostsManage)
	}
	return false
}

type RefreshToken struct {
	ID        string    `json:"id" bson:"_id"`
	UserID    string    `json:"user_id" bson:"user_id"`
	TokenHash string    `json:"-" bson:"token_hash"`
	ExpiresAt time.Time `json:"expires_at" bson:"expires_at"`
	Revoked   bool      `json:"revoked" bson:"revoked"`
	CreatedAt time.Time `json:"created_at" bson:"created_at"`
}

type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
	User         User   `json:"user"`
}
