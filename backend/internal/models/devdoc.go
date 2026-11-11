package models

import "time"

type DevDocCategory struct {
	Key   string `json:"key"`
	Label string `json:"label"`
}

var DevDocCategories = []DevDocCategory{
	{Key: "getting-started", Label: "Primeiros passos"},
	{Key: "architecture", Label: "Arquitetura"},
	{Key: "database", Label: "Banco de dados"},
	{Key: "api", Label: "API"},
	{Key: "frontend", Label: "Frontend"},
	{Key: "testing", Label: "Testes"},
	{Key: "deployment", Label: "Deploy"},
	{Key: "conventions", Label: "Convenções"},
}

func ValidCategory(key string) bool {
	for _, category := range DevDocCategories {
		if category.Key == key {
			return true
		}
	}
	return false
}

func CategoryLabel(key string) string {
	for _, category := range DevDocCategories {
		if category.Key == key {
			return category.Label
		}
	}
	return key
}

type DevDoc struct {
	ID          int64     `db:"id" json:"id"`
	UserID      int64     `db:"user_id" json:"user_id"`
	Title       string    `db:"title" json:"title"`
	Slug        string    `db:"slug" json:"slug"`
	Category    string    `db:"category" json:"category"`
	Summary     *string   `db:"summary" json:"summary"`
	Content     string    `db:"content" json:"content"`
	IsPublished bool      `db:"is_published" json:"is_published"`
	CreatedAt   time.Time `db:"created_at" json:"created_at"`
	UpdatedAt   time.Time `db:"updated_at" json:"updated_at"`
	AuthorName  *string   `db:"author_name" json:"author_name,omitempty"`
	AuthorID    *int64    `db:"author_id" json:"author_id,omitempty"`
}

type DevDocSummary struct {
	ID          int64     `db:"id" json:"id"`
	Title       string    `db:"title" json:"title"`
	Slug        string    `db:"slug" json:"slug"`
	Category    string    `db:"category" json:"category"`
	Summary     *string   `db:"summary" json:"summary"`
	IsPublished bool      `db:"is_published" json:"is_published"`
	UpdatedAt   time.Time `db:"updated_at" json:"updated_at"`
	AuthorName  *string   `db:"author_name" json:"author_name"`
}
