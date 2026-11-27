package database

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"

	"github.com/eduardo-pacheco-dev/manager-documentation/apps/backend/internal/config"
)

// Tenant opens a pool whose connections are pinned to one company. Every query
// touching a tenant-owned table is rewritten to filter/insert company_id using
// the @dc_company_id user variable initialized when the connection is created.
func Tenant(cfg config.Config, companyID int64) (*sqlx.DB, error) {
	dsn := cfg.DatabaseDSN
	if strings.TrimSpace(dsn) == "" {
		dsn = buildDSN(cfg, cfg.DatabaseName)
	}
	mysqlConfig, err := mysql.ParseDSN(dsn)
	if err != nil {
		return nil, fmt.Errorf("parse tenant DSN: %w", err)
	}
	base, err := mysql.NewConnector(mysqlConfig)
	if err != nil {
		return nil, fmt.Errorf("create tenant connector: %w", err)
	}
	db := sqlx.NewDb(sql.OpenDB(tenantConnector{base: base, companyID: companyID}), "mysql")
	configurePool(db)
	// Tenant queries routinely use SELECT *; sqlx would otherwise reject the
	// extra company_id column when scanning into existing structs.
	return db.Unsafe(), nil
}

type tenantConnector struct {
	base      driver.Connector
	companyID int64
}

func (c tenantConnector) Connect(ctx context.Context) (driver.Conn, error) {
	conn, err := c.base.Connect(ctx)
	if err != nil {
		return nil, err
	}
	stmt, err := conn.Prepare("SET @dc_company_id = ?")
	if err != nil {
		_ = conn.Close()
		return nil, err
	}
	if _, err := stmt.Exec([]driver.Value{c.companyID}); err != nil {
		_ = stmt.Close()
		_ = conn.Close()
		return nil, err
	}
	if err := stmt.Close(); err != nil {
		_ = conn.Close()
		return nil, err
	}
	return &tenantConn{Conn: conn}, nil
}

func (c tenantConnector) Driver() driver.Driver { return c.base.Driver() }

type tenantConn struct {
	driver.Conn
}

func (c *tenantConn) Prepare(query string) (driver.Stmt, error) {
	return c.Conn.Prepare(rewriteTenantSQL(query))
}

func (c *tenantConn) PrepareContext(ctx context.Context, query string) (driver.Stmt, error) {
	preparer, ok := c.Conn.(driver.ConnPrepareContext)
	if !ok {
		return c.Prepare(query)
	}
	return preparer.PrepareContext(ctx, rewriteTenantSQL(query))
}

func (c *tenantConn) Exec(query string, args []driver.Value) (driver.Result, error) {
	execer, ok := c.Conn.(driver.Execer)
	if !ok {
		return nil, driver.ErrSkip
	}
	return execer.Exec(rewriteTenantSQL(query), args)
}

func (c *tenantConn) ExecContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	execer, ok := c.Conn.(driver.ExecerContext)
	if !ok {
		return nil, driver.ErrSkip
	}
	return execer.ExecContext(ctx, rewriteTenantSQL(query), args)
}

func (c *tenantConn) Query(query string, args []driver.Value) (driver.Rows, error) {
	queryer, ok := c.Conn.(driver.Queryer)
	if !ok {
		return nil, driver.ErrSkip
	}
	return queryer.Query(rewriteTenantSQL(query), args)
}

func (c *tenantConn) QueryContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	queryer, ok := c.Conn.(driver.QueryerContext)
	if !ok {
		return nil, driver.ErrSkip
	}
	return queryer.QueryContext(ctx, rewriteTenantSQL(query), args)
}

/* --------------------------------------------------------------------------
 * SQL rewriting
 * ----------------------------------------------------------------------- */

var (
	tenantStatementRE = regexp.MustCompile(`(?is)^\s*(SELECT|INSERT|UPDATE|DELETE)\b`)
	tenantTableRE     = regexp.MustCompile(`(?i)\b(FROM|JOIN|UPDATE|INTO)\s+([a-z_][a-z0-9_]*)`)
	aliasRE           = regexp.MustCompile(`(?i)^\s*(?:AS\s+)?([a-z_][a-z0-9_]*)\b`)
	tenantWhereRE     = regexp.MustCompile(`(?i)\bWHERE\b`)
	tenantValuesRE    = regexp.MustCompile(`(?i)\bVALUES\s*\(`)
	tenantInsertRE    = regexp.MustCompile(`(?is)^\s*(INSERT\s+(?:IGNORE\s+)?INTO\s+)([a-z_][a-z0-9_]*)\s*\(`)
	tenantUpdateRE    = regexp.MustCompile(`(?is)^\s*(UPDATE\s+)([a-z_][a-z0-9_]*)`)
	tenantDeleteRE    = regexp.MustCompile(`(?is)^\s*DELETE\s+FROM\s+([a-z_][a-z0-9_]*)`)
)

var tenantReservedWords = map[string]bool{
	"where": true, "on": true, "using": true, "join": true, "left": true,
	"right": true, "inner": true, "outer": true, "cross": true, "natural": true,
	"set": true, "values": true, "order": true, "group": true, "having": true,
	"limit": true, "offset": true, "union": true, "for": true, "lock": true,
	"as": true, "and": true, "or": true, "when": true, "then": true, "else": true,
	"end": true, "into": true, "from": true, "duplicate": true, "key": true,
	"select": true, "distinct": true, "sql": true, "no": true, "write": true,
}

var tenantTables = map[string]bool{
	"users": true, "sessions": true, "api_tokens": true, "settings": true,
	"storage_settings": true, "notifications": true, "dev_docs": true,
	"collaborators": true, "demands": true, "demand_attachments": true,
	"demand_updates": true, "short_links": true, "short_link_clicks": true,
	"short_link_files": true, "todo_lists": true, "todo_steps": true,
	"todo_tasks": true, "follow_ups": true, "follow_up_assignees": true,
	"follow_up_clients": true, "follow_up_projects": true,
	"follow_up_settings": true, "follow_up_updates": true, "stations": true,
	"station_attachments": true, "station_comments": true, "nokia_sites": true,
	"nokia_roms": true, "nokia_tasks": true, "nokia_site_task": true,
	"nokia_site_agenda_events": true, "nokia_site_attachments": true,
	"nokia_site_comments": true, "nokia_site_rejections": true,
	"radio_links": true, "radio_link_attachments": true, "radio_link_comments": true,
	"projects": true, "project_assignments": true, "project_baseline_tasks": true,
	"project_baselines": true, "project_dependencies": true, "project_holidays": true,
	"project_pipeline_stages": true, "project_resources": true, "project_tasks": true,
	"agenda_calendars": true, "agenda_calendar_shares": true, "agenda_events": true,
	"agenda_event_attendees": true, "agenda_event_exceptions": true,
	"agenda_event_reminders": true, "iepms_users": true, "entity_links": true,
}

// rewriteTenantSQL scopes every tenant-owned table in a statement. SELECTs
// wrap tables in a filtered derived table (safe for joins and subqueries);
// writes get an explicit company_id predicate/value.
func rewriteTenantSQL(query string) string {
	match := tenantStatementRE.FindStringSubmatchIndex(query)
	if match == nil || strings.Contains(query, "@dc_company_id") {
		return query
	}
	kind := strings.ToUpper(query[match[2]:match[3]])
	switch kind {
	case "INSERT":
		return rewriteTenantInsert(query)
	case "UPDATE":
		return rewriteTenantWrite(query, "UPDATE")
	case "DELETE":
		return rewriteTenantWrite(query, "DELETE")
	default:
		return rewriteTenantSelect(query)
	}
}

func rewriteTenantSelect(query string) string {
	indexes := tenantTableRE.FindAllStringSubmatchIndex(query, -1)
	for i := len(indexes) - 1; i >= 0; i-- {
		idx := indexes[i]
		verb := query[idx[2]:idx[3]]
		table := query[idx[4]:idx[5]]
		if !tenantTables[strings.ToLower(table)] {
			continue
		}
		alias, end := tableAlias(query, idx[5], table, "")
		if alias == "" {
			continue
		}
		replacement := fmt.Sprintf("%s (SELECT * FROM %s WHERE company_id = @dc_company_id) AS %s",
			verb, table, alias)
		query = query[:idx[0]] + replacement + query[end:]
	}
	return query
}

func rewriteTenantInsert(query string) string {
	idx := tenantInsertRE.FindStringSubmatchIndex(query)
	if idx == nil || !tenantTables[strings.ToLower(query[idx[4]:idx[5]])] {
		return query
	}
	parenOffset := strings.Index(query[idx[5]:], "(")
	if parenOffset < 0 {
		return query
	}
	openParen := idx[5] + parenOffset
	relativeClose := strings.Index(query[openParen+1:], ")")
	if relativeClose < 0 {
		return query
	}
	columnsEnd := openParen + 1 + relativeClose
	columns := query[openParen+1 : columnsEnd]
	if regexp.MustCompile(`(?i)\bcompany_id\b`).MatchString(columns) {
		return query
	}
	valuesIdx := tenantValuesRE.FindStringIndex(query[columnsEnd+1:])
	if valuesIdx == nil {
		return query
	}
	valuesOpen := columnsEnd + 1 + valuesIdx[1]
	query = query[:valuesOpen] + "@dc_company_id, " + query[valuesOpen:]
	return query[:openParen+1] + "company_id, " + query[openParen+1:]
}

func rewriteTenantWrite(query string, kind string) string {
	var (
		table  string
		alias  string
		search int
	)
	if kind == "UPDATE" {
		idx := tenantUpdateRE.FindStringSubmatchIndex(query)
		if idx == nil {
			return query
		}
		table = query[idx[4]:idx[5]]
		alias, search = tableAlias(query, idx[5], table, "set")
	} else {
		idx := tenantDeleteRE.FindStringSubmatchIndex(query)
		if idx == nil {
			return query
		}
		table = query[idx[2]:idx[3]]
		alias, search = tableAlias(query, idx[3], table, "where")
	}
	if !tenantTables[strings.ToLower(table)] || alias == "" {
		return query
	}
	predicate := alias + ".company_id = @dc_company_id"
	if where := tenantWhereRE.FindStringIndex(query[search:]); where != nil {
		position := search + where[1]
		return query[:position] + " " + predicate + " AND" + query[position:]
	}
	return query + " WHERE " + predicate
}

// tableAlias returns the alias used for a table reference. limitWord stops the
// lookahead for UPDATE/DELETE statements, where no JOIN follows the table.
func tableAlias(query string, from int, fallback, limitWord string) (string, int) {
	rest := query[from:]
	if limitWord != "" {
		if stop := regexp.MustCompile(`(?i)\b` + limitWord + `\b`).FindStringIndex(rest); stop != nil {
			rest = rest[:stop[0]]
		}
	}
	idx := aliasRE.FindStringSubmatchIndex(rest)
	if idx == nil {
		return fallback, from
	}
	alias := rest[idx[2]:idx[3]]
	if tenantReservedWords[strings.ToLower(alias)] {
		return fallback, from
	}
	return alias, from + idx[1]
}

// ParseCompanyID is used by endpoints that identify a tenant in the URL.
func ParseCompanyID(value string) (int64, error) {
	return strconv.ParseInt(strings.TrimSpace(value), 10, 64)
}
