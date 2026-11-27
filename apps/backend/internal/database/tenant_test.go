package database

import (
	"strings"
	"testing"
)

func TestRewriteTenantSQL(t *testing.T) {
	tests := []struct {
		name     string
		query    string
		contains []string
		absent   []string
	}{
		{
			name:     "select scopes table",
			query:    `SELECT * FROM users WHERE email = ?`,
			contains: []string{"FROM (SELECT * FROM users WHERE company_id = @dc_company_id) AS users WHERE email = ?"},
		},
		{
			name:  "select scopes joins and subqueries",
			query: "SELECT l.*, (SELECT COUNT(*) FROM todo_tasks t WHERE t.todo_list_id = l.id) AS task_count FROM todo_lists l JOIN users u ON u.id = l.user_id WHERE l.deleted_at IS NULL",
			contains: []string{
				"FROM (SELECT * FROM todo_tasks WHERE company_id = @dc_company_id) AS t",
				"FROM (SELECT * FROM todo_lists WHERE company_id = @dc_company_id) AS l",
				"JOIN (SELECT * FROM users WHERE company_id = @dc_company_id) AS u",
			},
		},
		{
			name:     "insert adds company column and value",
			query:    `INSERT INTO users (name, email) VALUES (?, ?)`,
			contains: []string{"INSERT INTO users (company_id, name, email) VALUES (@dc_company_id, ?, ?)"},
		},
		{
			name:     "api token insert is scoped",
			query:    "INSERT INTO api_tokens (user_id, name, token_hash, created_at) VALUES (?, ?, ?, ?)",
			contains: []string{"INSERT INTO api_tokens (company_id, user_id, name, token_hash, created_at) VALUES (@dc_company_id, ?, ?, ?, ?)"},
		},
		{
			name:     "insert ignore is scoped",
			query:    `INSERT IGNORE INTO notifications (id, user_id) VALUES (?, ?)`,
			contains: []string{"INSERT IGNORE INTO notifications (company_id, id, user_id) VALUES (@dc_company_id, ?, ?)"},
		},
		{
			name:     "update without alias",
			query:    `UPDATE todo_lists SET name = ? WHERE id = ?`,
			contains: []string{"WHERE todo_lists.company_id = @dc_company_id AND id = ?"},
		},
		{
			name:     "delete with alias",
			query:    `DELETE FROM todo_tasks t WHERE t.todo_list_id = ?`,
			contains: []string{"WHERE t.company_id = @dc_company_id AND t.todo_list_id = ?"},
		},
		{
			name:     "non tenant table unchanged",
			query:    `SELECT * FROM companies WHERE slug = ?`,
			contains: []string{"SELECT * FROM companies WHERE slug = ?"},
			absent:   []string{"@dc_company_id"},
		},
		{
			name:     "already scoped query is unchanged",
			query:    `SELECT * FROM users WHERE company_id = @dc_company_id`,
			contains: []string{"SELECT * FROM users WHERE company_id = @dc_company_id"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := rewriteTenantSQL(tt.query)
			for _, expected := range tt.contains {
				if !strings.Contains(got, expected) {
					t.Fatalf("rewriteTenantSQL() = %q, want it to contain %q", got, expected)
				}
			}
			for _, unwanted := range tt.absent {
				if strings.Contains(got, unwanted) {
					t.Fatalf("rewriteTenantSQL() = %q, should not contain %q", got, unwanted)
				}
			}
		})
	}
}
