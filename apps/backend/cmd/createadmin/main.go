package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"golang.org/x/crypto/bcrypt"

	"github.com/eduardo-pacheco-dev/manager-documentation/apps/backend/internal/config"
	"github.com/eduardo-pacheco-dev/manager-documentation/apps/backend/internal/database"
	"github.com/eduardo-pacheco-dev/manager-documentation/apps/backend/internal/models"
	"github.com/eduardo-pacheco-dev/manager-documentation/apps/backend/internal/store"
)

// createadmin creates a new administrator or promotes an existing account.
//
//	go run ./cmd/createadmin -email admin@example.com -password secret -name Administrador
func main() {
	name := flag.String("name", "", "nome do administrador")
	email := flag.String("email", "", "e-mail do administrador")
	password := flag.String("password", "", "senha (obrigatória ao criar uma conta)")
	company := flag.String("company", "padrao", "identificador da empresa")
	superAdmin := flag.Bool("super", false, "promover a super-administrador global")
	flag.Parse()

	normalizedEmail := strings.ToLower(strings.TrimSpace(*email))
	if normalizedEmail == "" || !strings.Contains(normalizedEmail, "@") {
		fmt.Fprintln(os.Stderr, "informe um e-mail válido com -email")
		os.Exit(1)
	}

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}
	db, err := database.Open(cfg)
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	defer db.Close()
	repo := store.New(db)

	tenant, err := repo.FindCompanyByIdentifier(*company)
	if err != nil {
		log.Fatalf("empresa %q não encontrada", *company)
	}
	existing, err := repo.FindUserByCompanyEmail(tenant.ID, normalizedEmail)
	if err == nil {
		existing.IsAdmin = true
		existing.IsSuperAdmin = *superAdmin
		existing.IsActive = true
		if trimmed := strings.TrimSpace(*name); trimmed != "" {
			existing.Name = trimmed
		}
		if *password != "" {
			hash, hashErr := bcrypt.GenerateFromPassword([]byte(*password), bcrypt.DefaultCost)
			if hashErr != nil {
				log.Fatalf("hash: %v", hashErr)
			}
			existing.PasswordHash = string(hash)
		}
		if updateErr := repo.UpdateUser(&existing); updateErr != nil {
			log.Fatalf("update user: %v", updateErr)
		}
		fmt.Printf("usuário %s promovido a administrador e ativado\n", normalizedEmail)
		return
	}

	if *password == "" || len([]rune(*password)) < 8 {
		fmt.Fprintln(os.Stderr, "informe uma senha com pelo menos 8 caracteres com -password")
		os.Exit(1)
	}
	userName := strings.TrimSpace(*name)
	if userName == "" {
		userName = "Administrador"
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(*password), bcrypt.DefaultCost)
	if err != nil {
		log.Fatalf("hash: %v", err)
	}
	user := models.User{
		CompanyID:          tenant.ID,
		Name:               userName,
		Email:              normalizedEmail,
		PasswordHash:       string(hash),
		IsAdmin:            true,
		IsActive:           true,
		Theme:              "system",
		EmailNotifications: true,
	}
	if err := repo.CreateUser(&user); err != nil {
		log.Fatalf("create user: %v", err)
	}
	fmt.Printf("administrador %s criado (id %d)\n", normalizedEmail, user.ID)
}
