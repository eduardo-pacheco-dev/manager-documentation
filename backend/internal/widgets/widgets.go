package widgets

import (
	"fmt"
	"time"

	"github.com/eduardo-pacheco-dev/manager-documentation/backend/internal/models"
	"github.com/eduardo-pacheco-dev/manager-documentation/backend/internal/modules"
	"github.com/eduardo-pacheco-dev/manager-documentation/backend/internal/store"
)

type Widget struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Icon        string `json:"icon"`
	Order       int    `json:"-"`
	DefaultSpan int    `json:"default_span"`
	AdminOnly   bool   `json:"admin_only"`
	Data        any    `json:"data,omitempty"`
}

type UserCountData struct {
	Total int `json:"total"`
}

type RecentUser struct {
	ID        int64     `json:"id"`
	FullName  string    `json:"full_name"`
	Email     string    `json:"email"`
	IsAdmin   bool      `json:"is_admin"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
}

type RecentUsersData struct {
	Users []RecentUser `json:"users"`
}

var registry = []Widget{
	{
		ID:          "core.users",
		Title:       "Usuários",
		Description: "Total de contas cadastradas no sistema.",
		Icon:        "users",
		Order:       10,
		DefaultSpan: 1,
		AdminOnly:   true,
	},
	{
		ID:          "core.users.recent",
		Title:       "Usuários recentes",
		Description: "As últimas contas criadas na aplicação.",
		Icon:        "clock",
		Order:       40,
		DefaultSpan: 3,
		AdminOnly:   true,
	},
	{
		ID:          "stations.sites",
		Title:       "Sites",
		Description: "Estações agrupadas por site.",
		Icon:        "signal",
		Order:       35,
		DefaultSpan: 2,
	},
	{
		ID:          "follow-up.overdue",
		Title:       "Meus follow-ups",
		Description: "Follow-ups atribuídos a você.",
		Icon:        "clipboard",
		Order:       45,
		DefaultSpan: 2,
	},
	{
		ID:          "follow-up.charts",
		Title:       "Painel de follow-ups",
		Description: "Distribuição por status, prioridade e totais.",
		Icon:        "chart",
		Order:       46,
		DefaultSpan: 3,
	},
	{
		ID:          "nokia.charts",
		Title:       "Projetos Nokia",
		Description: "Aceite, status das FAMs e pendências dos sites.",
		Icon:        "signal",
		Order:       50,
		DefaultSpan: 3,
	},
	{
		ID:          "example.status",
		Title:       "Widget de exemplo",
		Description: "Status do módulo Example convertido.",
		Icon:        "cube",
		Order:       55,
		DefaultSpan: 1,
	},
}

// Available returns widgets visible to the user, sorted by registry order and
// hydrated with their data.
func Available(s *store.Store, m *modules.Store, user models.User) ([]Widget, error) {
	result := make([]Widget, 0, len(registry))
	for _, widget := range registry {
		if widget.AdminOnly && !user.IsAdmin {
			continue
		}
		switch widget.ID {
		case "core.users":
			total, err := s.UserCount()
			if err != nil {
				return nil, err
			}
			widget.Data = UserCountData{Total: total}
		case "core.users.recent":
			users, err := s.RecentUsers(5)
			if err != nil {
				return nil, err
			}
			recent := make([]RecentUser, 0, len(users))
			for _, item := range users {
				recent = append(recent, RecentUser{
					ID:        item.ID,
					FullName:  item.FullName(),
					Email:     item.Email,
					IsAdmin:   item.IsAdmin,
					IsActive:  item.IsActive,
					CreatedAt: item.CreatedAt,
				})
			}
			widget.Data = RecentUsersData{Users: recent}
		case "stations.sites":
			sites, total, err := m.StationSites("", 6)
			if err != nil {
				return nil, err
			}
			widget.Data = map[string]any{"sites": sites, "total": total}
		case "follow-up.overdue":
			settings, _ := m.FollowUpSettings()
			data, err := m.FollowUpOverdueWidget(user.Email, settings.WidgetLimit)
			if err != nil {
				return nil, err
			}
			widget.Data = data
		case "follow-up.charts":
			data, err := m.FollowUpChartsWidget()
			if err != nil {
				return nil, err
			}
			widget.Data = data
		case "nokia.charts":
			data, err := m.NokiaMetrics()
			if err != nil {
				return nil, err
			}
			widget.Data = data
		case "example.status":
			widget.Data = map[string]any{"message": "Módulo Example ativo.", "version": "1.0.0"}
		}
		result = append(result, widget)
	}
	return result, nil
}

func Find(id string, available []Widget) (Widget, bool) {
	for _, widget := range available {
		if widget.ID == id {
			return widget, true
		}
	}
	return Widget{}, false
}

// NormalizeOrder keeps known IDs in the requested order and appends any new
// widgets at the end, preserving registry order.
func NormalizeOrder(available []Widget, order []string) []string {
	valid := make(map[string]bool, len(available))
	registryOrder := make([]string, 0, len(available))
	for _, widget := range available {
		valid[widget.ID] = true
		registryOrder = append(registryOrder, widget.ID)
	}

	result := make([]string, 0, len(available))
	seen := map[string]bool{}
	for _, id := range order {
		if valid[id] && !seen[id] {
			result = append(result, id)
			seen[id] = true
		}
	}
	for _, id := range registryOrder {
		if !seen[id] {
			result = append(result, id)
		}
	}
	return result
}

func ApplyOrder(available []Widget, order []string) []Widget {
	positions := map[string]int{}
	for index, id := range NormalizeOrder(available, order) {
		positions[id] = index
	}
	result := make([]Widget, len(available))
	copy(result, available)
	for i := 0; i < len(result); i++ {
		for j := i + 1; j < len(result); j++ {
			if positions[result[j].ID] < positions[result[i].ID] {
				result[i], result[j] = result[j], result[i]
			}
		}
	}
	return result
}

func Greeting(user models.User) string {
	hour := time.Now().Hour()
	greeting := "Boa noite"
	switch {
	case hour < 12:
		greeting = "Bom dia"
	case hour < 18:
		greeting = "Boa tarde"
	}
	return fmt.Sprintf("%s, %s!", greeting, user.FirstName())
}
