package store

import (
	"database/sql"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/eduardo-pacheco-dev/manager-documentation/apps/backend/internal/models"
)

func (s *Store) FindCompanyByID(id int64) (models.Company, error) {
	var company models.Company
	err := s.DB.Get(&company, `SELECT * FROM companies WHERE id = ?`, id)
	if errors.Is(err, sql.ErrNoRows) {
		return company, ErrNotFound
	}
	return company, err
}

func (s *Store) FindCompanyBySlug(slug string) (models.Company, error) {
	var company models.Company
	err := s.DB.Get(&company, `SELECT * FROM companies WHERE slug = ?`, strings.ToLower(strings.TrimSpace(slug)))
	if errors.Is(err, sql.ErrNoRows) {
		return company, ErrNotFound
	}
	return company, err
}

// FindCompanyByIdentifier accepts either the company ID or its slug. The login
// form normally sends the slug, while integrations may prefer the numeric ID.
func (s *Store) FindCompanyByIdentifier(value string) (models.Company, error) {
	identifier := strings.ToLower(strings.TrimSpace(value))
	if identifier == "" {
		return models.Company{}, ErrNotFound
	}
	if id, err := strconv.ParseInt(identifier, 10, 64); err == nil {
		return s.FindCompanyByID(id)
	}
	return s.FindCompanyBySlug(identifier)
}

func (s *Store) ListCompanies() ([]models.Company, error) {
	companies := []models.Company{}
	err := s.DB.Select(&companies, `SELECT * FROM companies ORDER BY id`)
	return companies, err
}

func (s *Store) CreateCompany(company *models.Company) error {
	tx, err := s.DB.Beginx()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if err := createCompany(tx, company); err != nil {
		return err
	}
	return tx.Commit()
}

// CreateCompanyWithAdmin provisions a company and its first administrator in
// one transaction so a failed user insert never leaves an orphan tenant.
func (s *Store) CreateCompanyWithAdmin(company *models.Company, admin *models.User) error {
	tx, err := s.DB.Beginx()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if err := createCompany(tx, company); err != nil {
		return err
	}
	if admin != nil {
		admin.CompanyID = company.ID
		if err := createUser(tx, admin); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func createCompany(exec sqlExecer, company *models.Company) error {
	now := time.Now().UTC()
	company.CreatedAt = now
	company.UpdatedAt = now
	company.Slug = strings.ToLower(strings.TrimSpace(company.Slug))
	company.Name = strings.TrimSpace(company.Name)
	result, err := exec.Exec(`
		INSERT INTO companies (name, slug, is_active, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?)`,
		company.Name, company.Slug, company.IsActive, company.CreatedAt, company.UpdatedAt)
	if err != nil {
		return err
	}
	company.ID, err = result.LastInsertId()
	if err != nil {
		return err
	}
	if _, err := exec.Exec(`INSERT INTO storage_settings (id, company_id, updated_at) VALUES (1, ?, ?)`, company.ID, now); err != nil {
		return err
	}
	if _, err := exec.Exec(`INSERT INTO follow_up_settings (id, company_id, updated_at, summary_times, summary_days) VALUES (1, ?, ?, '08:00,17:00', '0,1,2,3,4,5,6')`, company.ID, now); err != nil {
		return err
	}
	return nil
}

func (s *Store) UpdateCompany(company *models.Company) error {
	company.UpdatedAt = time.Now().UTC()
	company.Slug = strings.ToLower(strings.TrimSpace(company.Slug))
	company.Name = strings.TrimSpace(company.Name)
	_, err := s.DB.Exec(`
		UPDATE companies SET name = ?, slug = ?, is_active = ?, updated_at = ? WHERE id = ?`,
		company.Name, company.Slug, company.IsActive, company.UpdatedAt, company.ID)
	return err
}
