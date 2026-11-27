package models

import (
	"strings"
	"time"
)

// User is the persisted account record. PasswordHash must never be serialized
// directly to clients.
type User struct {
	ID                 int64              `db:"id" json:"id"`
	Name               string             `db:"name" json:"name"`
	Surname            *string            `db:"surname" json:"surname"`
	Email              string             `db:"email" json:"email"`
	PasswordHash       string             `db:"password_hash" json:"-"`
	IsAdmin            bool               `db:"is_admin" json:"is_admin"`
	IsActive           bool               `db:"is_active" json:"is_active"`
	Theme              string             `db:"theme" json:"theme"`
	EnabledModules     NullableStringList `db:"enabled_modules" json:"-"`
	ModuleOrder        NullableStringList `db:"module_order" json:"-"`
	ModuleSections     NullableStringMap  `db:"module_sections" json:"-"`
	DashboardHidden    NullableStringList `db:"dashboard_hidden_widgets" json:"-"`
	DashboardOrder     NullableStringList `db:"dashboard_widget_order" json:"-"`
	DashboardSpans     NullableIntMap     `db:"dashboard_widget_spans" json:"-"`
	EmailNotifications bool               `db:"email_notifications" json:"email_notifications"`
	CreatedAt          time.Time          `db:"created_at" json:"created_at"`
	UpdatedAt          time.Time          `db:"updated_at" json:"updated_at"`
}

func (u User) FullName() string {
	if u.Surname == nil {
		return strings.TrimSpace(u.Name)
	}
	return strings.TrimSpace(u.Name + " " + *u.Surname)
}

func (u User) FirstName() string {
	parts := strings.Fields(u.Name)
	if len(parts) == 0 {
		return u.Name
	}
	return parts[0]
}

// PublicUser is the shape returned by the API. It intentionally omits secrets
// and internal preference fields.
type PublicUser struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Surname   *string   `json:"surname,omitempty"`
	FullName  string    `json:"full_name"`
	Email     string    `json:"email"`
	IsAdmin   bool      `json:"is_admin"`
	IsActive  bool      `json:"is_active"`
	Theme     string    `json:"theme"`
	CreatedAt time.Time `json:"created_at"`
}

func (u User) Public() PublicUser {
	return PublicUser{
		ID:        u.ID,
		Name:      u.Name,
		Surname:   u.Surname,
		FullName:  u.FullName(),
		Email:     u.Email,
		IsAdmin:   u.IsAdmin,
		IsActive:  u.IsActive,
		Theme:     u.Theme,
		CreatedAt: u.CreatedAt,
	}
}
