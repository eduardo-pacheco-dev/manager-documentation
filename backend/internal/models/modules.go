package models

import "sort"

const (
	SectionSidebar = "sidebar"
	SectionModules = "modules"
)

// Module describes a navigable area of the application.
type Module struct {
	Key            string `json:"key"`
	Label          string `json:"label"`
	Icon           string `json:"icon"`
	Path           string `json:"path"`
	AdminOnly      bool   `json:"admin_only"`
	Primary        bool   `json:"primary"`
	Toggleable     bool   `json:"toggleable"`
	SystemLink     bool   `json:"system_link"`
	DefaultSection string `json:"default_section"`
}

// CoreModules is the plugin-free navigation catalog.
var CoreModules = []Module{
	{
		Key:            "dashboard",
		Label:          "Início",
		Icon:           "home",
		Path:           "/dashboard",
		Primary:        true,
		Toggleable:     false,
		DefaultSection: SectionSidebar,
	},
	{
		Key:            "users",
		Label:          "Usuários",
		Icon:           "users",
		Path:           "/users",
		AdminOnly:      true,
		Toggleable:     false,
		SystemLink:     true,
		DefaultSection: SectionSidebar,
	},
	{
		Key:            "dev-docs",
		Label:          "Documentação Dev",
		Icon:           "document",
		Path:           "/dev-docs",
		AdminOnly:      true,
		Toggleable:     false,
		SystemLink:     true,
		DefaultSection: SectionSidebar,
	},
	{
		Key:            "agenda",
		Label:          "Agenda",
		Icon:           "calendar",
		Path:           "/agenda",
		Toggleable:     true,
		DefaultSection: SectionModules,
	},
	{
		Key:            "tasks",
		Label:          "Tarefas",
		Icon:           "check-circle",
		Path:           "/tasks",
		Toggleable:     true,
		DefaultSection: SectionModules,
	},
	{
		Key:            "follow-ups",
		Label:          "Follow-ups",
		Icon:           "clipboard",
		Path:           "/follow-ups",
		Toggleable:     true,
		DefaultSection: SectionModules,
	},
	{
		Key:            "stations",
		Label:          "Estações",
		Icon:           "signal",
		Path:           "/stations",
		Toggleable:     true,
		DefaultSection: SectionModules,
	},
	{
		Key:            "nokia",
		Label:          "Projetos Nokia",
		Icon:           "signal",
		Path:           "/nokia",
		Toggleable:     true,
		DefaultSection: SectionModules,
	},
	{
		Key:            "links",
		Label:          "Links curtos",
		Icon:           "link",
		Path:           "/links",
		Toggleable:     true,
		DefaultSection: SectionModules,
	},
	{
		Key:            "demands",
		Label:          "Demandas",
		Icon:           "task",
		Path:           "/demands",
		Toggleable:     true,
		DefaultSection: SectionModules,
	},
	{
		Key:            "projects",
		Label:          "Projetos",
		Icon:           "chart",
		Path:           "/projects",
		Toggleable:     true,
		DefaultSection: SectionModules,
	},
}

func AvailableModules(user User) []Module {
	result := make([]Module, 0, len(CoreModules))
	for _, module := range CoreModules {
		if module.AdminOnly && !user.IsAdmin {
			continue
		}
		result = append(result, module)
	}
	return result
}

func EnabledModules(user User) []Module {
	available := AvailableModules(user)
	enabled := make([]Module, 0, len(available))
	for _, module := range available {
		if module.Key == "dashboard" || module.SystemLink {
			enabled = append(enabled, module)
			continue
		}
		if !user.EnabledModules.Valid {
			enabled = append(enabled, module)
			continue
		}
		if contains(user.EnabledModules.Values, module.Key) {
			enabled = append(enabled, module)
		}
	}
	return SortModules(user, enabled)
}

func ToggleableModules(user User) []Module {
	result := make([]Module, 0, len(CoreModules))
	for _, module := range AvailableModules(user) {
		if module.Toggleable {
			result = append(result, module)
		}
	}
	return SortModules(user, result)
}

func SortModules(user User, modules []Module) []Module {
	if !user.ModuleOrder.Valid || len(user.ModuleOrder.Values) == 0 {
		return modules
	}
	positions := make(map[string]int, len(user.ModuleOrder.Values))
	for index, key := range user.ModuleOrder.Values {
		positions[key] = index
	}
	sort.SliceStable(modules, func(i, j int) bool {
		left, okLeft := positions[modules[i].Key]
		right, okRight := positions[modules[j].Key]
		if !okLeft {
			left = 1 << 30
		}
		if !okRight {
			right = 1 << 30
		}
		return left < right
	})
	return modules
}

func ModuleSection(user User, module Module) string {
	section, ok := user.ModuleSections.OrEmpty()[module.Key]
	if ok && (section == SectionSidebar || section == SectionModules) {
		return section
	}
	return module.DefaultSection
}

func ModuleByKey(key string, user User) (Module, bool) {
	for _, module := range AvailableModules(user) {
		if module.Key == key {
			return module, true
		}
	}
	return Module{}, false
}

func contains(values []string, wanted string) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}
