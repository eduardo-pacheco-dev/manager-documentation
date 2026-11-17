package modules

import (
	"database/sql"
	"errors"
	"strings"
	"time"
)

/* --------------------------------------------------------------------------
 * iEPMS projects
 * ----------------------------------------------------------------------- */

// IepmsProject is a project available to be associated with iEPMS users. The
// catalogue is managed from the settings screen and used by the user detail
// checklist.
type IepmsProject struct {
	ID        int64     `db:"id" json:"id"`
	Name      string    `db:"name" json:"name"`
	Users     int       `db:"users" json:"users"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}

// IepmsUserProject associates an iEPMS user with a project. ExpiresAt is
// returned as an ISO date (YYYY-MM-DD) or null when the project does not
// expire.
type IepmsUserProject struct {
	ProjectID int64     `db:"project_id" json:"project_id"`
	Name      string    `db:"name" json:"name"`
	ExpiresAt *string   `db:"expires_at" json:"expires_at"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}

type IepmsUserProjectInput struct {
	ProjectID int64
	ExpiresAt *string
}

const iepmsProjectSelect = `
	SELECT p.id,
	       p.name,
	       (SELECT COUNT(*) FROM iepms_user_projects up WHERE up.project_id = p.id) AS users,
	       p.created_at,
	       p.updated_at
	FROM iepms_projects p`

func (s *Store) ListIepmsProjects() ([]IepmsProject, error) {
	projects := []IepmsProject{}
	err := s.DB.Select(&projects, iepmsProjectSelect+` ORDER BY p.name`)
	return projects, err
}

func (s *Store) FindIepmsProject(id int64) (IepmsProject, error) {
	var project IepmsProject
	err := s.DB.Get(&project, iepmsProjectSelect+` WHERE p.id = ?`, id)
	if errors.Is(err, sql.ErrNoRows) {
		return project, ErrNotFound
	}
	return project, err
}

func (s *Store) CreateIepmsProject(name string) (IepmsProject, error) {
	timestamp := now()
	result, err := s.DB.Exec(
		`INSERT INTO iepms_projects (name, created_at, updated_at) VALUES (?, ?, ?)`,
		strings.TrimSpace(name), timestamp, timestamp,
	)
	if err != nil {
		return IepmsProject{}, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return IepmsProject{}, err
	}
	return s.FindIepmsProject(id)
}

func (s *Store) UpdateIepmsProject(id int64, name string) (IepmsProject, error) {
	if _, err := s.FindIepmsProject(id); err != nil {
		return IepmsProject{}, err
	}
	if _, err := s.DB.Exec(
		`UPDATE iepms_projects SET name = ?, updated_at = ? WHERE id = ?`,
		strings.TrimSpace(name), now(), id,
	); err != nil {
		return IepmsProject{}, err
	}
	return s.FindIepmsProject(id)
}

func (s *Store) DeleteIepmsProject(id int64) error {
	result, err := s.DB.Exec(`DELETE FROM iepms_projects WHERE id = ?`, id)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) ListIepmsUserProjects(userID int64) ([]IepmsUserProject, error) {
	projects := []IepmsUserProject{}
	err := s.DB.Select(&projects, `
		SELECT p.id AS project_id,
		       p.name,
		       DATE_FORMAT(up.expires_at, '%Y-%m-%d') AS expires_at,
		       up.created_at,
		       up.updated_at
		FROM iepms_user_projects up
		JOIN iepms_projects p ON p.id = up.project_id
		WHERE up.iepms_user_id = ?
		ORDER BY p.name`, userID)
	return projects, err
}

// ReplaceIepmsUserProjects replaces the full checklist of a user in a single
// transaction, so removed projects are dropped and new ones are inserted.
func (s *Store) ReplaceIepmsUserProjects(userID int64, items []IepmsUserProjectInput) error {
	tx, err := s.DB.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`DELETE FROM iepms_user_projects WHERE iepms_user_id = ?`, userID); err != nil {
		return err
	}

	timestamp := now()
	for _, item := range items {
		var expires any
		if item.ExpiresAt != nil && strings.TrimSpace(*item.ExpiresAt) != "" {
			expires = strings.TrimSpace(*item.ExpiresAt)
		}
		if _, err := tx.Exec(
			`INSERT INTO iepms_user_projects (iepms_user_id, project_id, expires_at, created_at, updated_at)
			 VALUES (?, ?, ?, ?, ?)`,
			userID, item.ProjectID, expires, timestamp, timestamp,
		); err != nil {
			return err
		}
	}

	return tx.Commit()
}
