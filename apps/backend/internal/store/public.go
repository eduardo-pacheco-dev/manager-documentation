package store

import (
	"database/sql"
	"errors"
	"strings"
)

// PublicCompanyID resolves the tenant that owns a public short link or demand
// token. Tokens are random and globally unique in practice; the scoped unique
// keys prevent duplicates inside a company.
func (s *Store) PublicCompanyID(table, column, value string) (int64, error) {
	allowed := map[string]string{
		"short_links.code":       "code",
		"demands.upload_token":   "upload_token",
		"demands.download_token": "download_token",
	}
	field, ok := allowed[table+"."+column]
	if !ok {
		return 0, ErrNotFound
	}
	var companyID int64
	err := s.DB.Get(&companyID, `SELECT company_id FROM `+table+` WHERE `+field+` = ? ORDER BY id ASC LIMIT 1`, strings.TrimSpace(value))
	if errors.Is(err, sql.ErrNoRows) {
		return 0, ErrNotFound
	}
	return companyID, err
}
