package seed

import (
	"embed"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/eduardo-pacheco-dev/manager-documentation/backend/internal/config"
	"github.com/eduardo-pacheco-dev/manager-documentation/backend/internal/models"
	"github.com/eduardo-pacheco-dev/manager-documentation/backend/internal/store"
)

//go:embed content/*.md
var contentFS embed.FS

// Run creates the initial administrator, a demo account and the starter wiki.
// It is idempotent and safe to call on every boot.
func Run(s *store.Store, cfg config.Config) error {
	admin, err := ensureUser(s, models.User{
		Name:               "Administrador",
		Email:              strings.ToLower(cfg.SeedAdminEmail),
		IsAdmin:            true,
		IsActive:           true,
		Theme:              "system",
		EmailNotifications: true,
	}, cfg.SeedAdminPass)
	if err != nil {
		return fmt.Errorf("seed admin: %w", err)
	}

	if !cfg.SeedDemoData {
		return nil
	}

	if _, err := ensureUser(s, models.User{
		Name:               "Usuário",
		Surname:            strPtr("Demo"),
		Email:              "demo@example.com",
		IsAdmin:            false,
		IsActive:           true,
		Theme:              "system",
		EmailNotifications: true,
	}, "password"); err != nil {
		return fmt.Errorf("seed demo user: %w", err)
	}

	if err := ensureDocs(s, admin.ID); err != nil {
		return fmt.Errorf("seed docs: %w", err)
	}

	items, _, err := s.LatestNotifications(admin.ID, 1)
	if err == nil && len(items) == 0 {
		if _, settingErr := s.GetSetting("seed.welcome_notification"); settingErr != nil {
			_ = s.CreateNotification(&models.Notification{
				ID:        uuid.NewString(),
				UserID:    admin.ID,
				Title:     "Bem-vindo ao Documentation Center",
				Body:      "A nova central de documentação está pronta. Explore o dashboard e a wiki de desenvolvimento.",
				Level:     "info",
				CreatedAt: time.Now().UTC(),
			})
			_ = s.SetSetting("seed.welcome_notification", "1")
		}
	}
	return nil
}

func ensureUser(s *store.Store, user models.User, password string) (models.User, error) {
	existing, err := s.FindUserByEmail(user.Email)
	if err == nil {
		return existing, nil
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return models.User{}, err
	}
	user.PasswordHash = string(hash)
	if err := s.CreateUser(&user); err != nil {
		return models.User{}, err
	}
	log.Printf("seed: created user %s", user.Email)
	return user, nil
}

type docMeta struct {
	File     string
	Title    string
	Category string
	Summary  string
}

var starterDocs = []docMeta{
	{"content/01-visao-geral.md", "Visão geral do sistema", "getting-started", "O que é o Documentation Center, stack utilizada e por onde começar."},
	{"content/02-configuracao-ambiente.md", "Configuração do ambiente", "getting-started", "Requisitos, instalação e execução do backend e frontend em desenvolvimento."},
	{"content/03-arquitetura.md", "Arquitetura da aplicação", "architecture", "Como backend e frontend se comunicam, camadas do servidor e fluxo de uma requisição."},
	{"content/04-autenticacao.md", "Autenticação e autorização", "architecture", "Sessões de navegador, tokens de API, CSRF, contas inativas e permissões administrativas."},
	{"content/05-banco-de-dados.md", "Banco de dados e migrações", "database", "Tabelas, migrações incrementais e backup do MySQL."},
	{"content/06-api.md", "API de autenticação", "api", "Endpoints de cadastro, login, logout, usuário atual e upload de imagens."},
	{"content/07-frontend.md", "Frontend e Carbon Design System", "frontend", "Organização da SPA, componentes Carbon, temas claro/escuro e responsividade."},
	{"content/08-deploy.md", "Deploy", "deployment", "Build do frontend, execução do binário Go, Docker e variáveis de ambiente."},
	{"content/09-seguranca.md", "Segurança", "conventions", "Boas práticas aplicadas: senhas, sessões, CSRF, uploads, Markdown e permissões."},
	{"content/10-testes.md", "Testes e qualidade", "testing", "Como executar testes do backend, verificar build do frontend e validar a API."},
	{"content/11-projetos.md", "Projetos", "modules", "EAP, dependências, calendário, recursos, custos, linha de base e caminho crítico."},
	{"content/12-colaboradores.md", "Colaboradores", "modules", "Cadastro central de pessoas, custos e capacidade usados como recursos nos projetos."},
}

func ensureDocs(s *store.Store, authorID int64) error {
	for _, meta := range starterDocs {
		raw, err := contentFS.ReadFile(meta.File)
		if err != nil {
			return err
		}
		slug := slugify(meta.Title)
		if existing, err := s.FindDocBySlug(slug); err == nil && existing.ID > 0 {
			continue
		}
		summary := meta.Summary
		doc := models.DevDoc{
			UserID:      authorID,
			Title:       meta.Title,
			Slug:        slug,
			Category:    meta.Category,
			Summary:     &summary,
			Content:     strings.TrimSpace(string(raw)),
			IsPublished: true,
		}
		if err := s.CreateDoc(&doc); err != nil {
			return err
		}
	}
	return nil
}

func slugify(value string) string {
	replacer := strings.NewReplacer(
		"á", "a", "à", "a", "â", "a", "ã", "a", "ä", "a",
		"é", "e", "è", "e", "ê", "e", "ë", "e",
		"í", "i", "ì", "i", "î", "i", "ï", "i",
		"ó", "o", "ò", "o", "ô", "o", "õ", "o", "ö", "o",
		"ú", "u", "ù", "u", "û", "u", "ü", "u",
		"ç", "c", "ñ", "n",
		"Á", "a", "À", "a", "Â", "a", "Ã", "a", "Ä", "a",
		"É", "e", "È", "e", "Ê", "e", "Ë", "e",
		"Í", "i", "Ì", "i", "Î", "i", "Ï", "i",
		"Ó", "o", "Ò", "o", "Ô", "o", "Õ", "o", "Ö", "o",
		"Ú", "u", "Ù", "u", "Û", "u", "Ü", "u",
		"Ç", "c", "Ñ", "n",
	)
	value = strings.ToLower(replacer.Replace(value))
	var builder strings.Builder
	previousDash := false
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			builder.WriteRune(r)
			previousDash = false
		default:
			if !previousDash && builder.Len() > 0 {
				builder.WriteByte('-')
				previousDash = true
			}
		}
	}
	result := strings.Trim(builder.String(), "-")
	if result == "" {
		return "documento"
	}
	return result
}

func strPtr(value string) *string {
	return &value
}
