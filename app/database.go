package app

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"log"
	"net"
	"net/url"
	"path"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	// ErrNoFilenameSeparator indicates a migration filename is missing the required '_' separator.
	ErrNoFilenameSeparator = errors.New("no required filename separator '_' found")
	// ErrMultipleSameVersion indicates two or more migration files share the same version number.
	ErrMultipleSameVersion = errors.New("multiple migrations of same version found")
)

// DB wraps the pgxpool.Pool.
type DB struct {
	*pgxpool.Pool
}

// NewDB creates a new PostgreSQL connection pool.
func NewDB(ctx context.Context) (db *DB, err error) {
	c := Config().DB

	conf, err := pgxpool.ParseConfig(
		fmt.Sprintf("postgres://%s:%s@%s/%s", c.User, url.QueryEscape(c.Password), net.JoinHostPort(c.Host, c.Port), c.Name))
	if err != nil {
		return nil, err
	}

	// For now, I just print logs
	// TODO: make proper logging
	conf.ConnConfig.Tracer = NewLoggingQueryTracer()

	pool, err := pgxpool.NewWithConfig(ctx, conf)
	if err != nil {
		err = fmt.Errorf("create db connection pool: %w", err)

		return
	}

	db = &DB{pool}

	_, err = db.Exec(ctx, "SELECT 1")
	if err != nil {
		err = fmt.Errorf("db.exec: %w", err)
	}

	return db, err
}

//go:embed db_migrations/*.sql
var mgFiles embed.FS

const (
	mgTable = "db_migrations"
	mgDir   = "db_migrations"
)

type migration struct {
	Version    int64
	Name       string
	SQL        string
	MigratedAt *time.Time
}

// MigrateDB runs SQL scripts from the './migrations' directory that haven't been committed yet.
// It reads migration files, compares them with the migrations already applied to the database,
// and executes the new migrations in a transaction.
func MigrateDB(ctx context.Context, db *DB) (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("[ERROR] [MIGRATE]: %w", err)
		}
	}()

	allMg := make([]migration, 0)
	completedMg := make([]migration, 0)
	newMg := make([]migration, 0)

	dir, err := mgFiles.ReadDir(mgDir)
	if err != nil {
		return fmt.Errorf("read migrations: %w", err)
	}

	for _, file := range dir {
		mgData, err := mgFiles.ReadFile(path.Join(mgDir, file.Name()))
		if err != nil {
			return fmt.Errorf("read '%s': %w", file.Name(), err)
		}

		name := strings.TrimSuffix(file.Name(), ".sql")

		idx := strings.Index(name, "_")
		if idx < 0 {
			return fmt.Errorf("filename %s: %w", name, ErrNoFilenameSeparator)
		}

		version, err := strconv.ParseInt(name[:idx], 10, 64)
		if err != nil {
			return fmt.Errorf("parse version from migration file: %s: %w", name, err)
		}

		name = name[idx+1:]

		for _, m := range allMg {
			if m.Version == version {
				return fmt.Errorf("version '%d': %w", version, ErrMultipleSameVersion)
			}
		}

		allMg = append(allMg, migration{
			Version:    version,
			Name:       name,
			SQL:        string(mgData),
			MigratedAt: nil,
		})
	}

	if len(allMg) == 0 {
		fmt.Println("no migrations found in " + mgDir)

		return
	}

selectAll:
	err = pgxscan.Select(ctx, db, &completedMg, `SELECT * FROM `+mgTable)

	if err != nil {
		// on clean db run migrations table not exists yet
		// check this by code returned and table name
		// if so, create table and retry
		pgErr, ok := errors.AsType[*pgconn.PgError](err)
		if ok && pgErr.Code == "42P01" && strings.Contains(err.Error(), mgTable) {
			_, err = db.Exec(ctx, `
       CREATE TABLE `+mgTable+` (
          version    BIGINT NOT NULL PRIMARY KEY,
          name       TEXT NOT NULL,
          migrated_at TIMESTAMPTZ NOT NULL
       );`)
			if err != nil {
				return fmt.Errorf("init migrations table: %w", err)
			}

			goto selectAll
		}

		return err
	}

	if err != nil {
		return fmt.Errorf("load completed migrations: %w", err)
	}

iterate:
	for _, m := range allMg {
		for _, c := range completedMg {
			if m.Version == c.Version {
				continue iterate
			}
		}

		newMg = append(newMg, m)
	}

	if len(newMg) == 0 {
		fmt.Println("[MIGRATION] ✅ Nothing to migrate")

		return
	}

	sort.Slice(newMg, func(i, j int) bool {
		return newMg[i].Version < newMg[j].Version
	})

	for _, m := range newMg {
		err = RunInTx(ctx, db, func(ctx context.Context, tx pgx.Tx) error {
			now := time.Now()

			_, err = tx.Exec(ctx, m.SQL)
			if err != nil {
				return fmt.Errorf("❌ %d %s: %w", m.Version, m.Name, err)
			}

			m.MigratedAt = &now

			_, err = tx.Exec(ctx, "INSERT INTO "+mgTable+" (version, name, migrated_at) VALUES ($1, $2, $3)",
				m.Version, m.Name, m.MigratedAt)
			if err != nil {
				return fmt.Errorf("❌ save migration %d %s: %w", m.Version, m.Name, err)
			}

			fmt.Printf("[MIGRATION] %d\n✅ %s\n%s\n", m.Version, m.Name, time.Since(now).String())

			return nil
		})
		if err != nil {
			return err
		}
	}

	return nil
}

// RunInTx executes a function within transaction.
func RunInTx(ctx context.Context, db *DB,
	f func(ctx context.Context, tx pgx.Tx) error) (err error) {
	tx, err := db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}

	err = f(ctx, tx)
	if err != nil {
		err2 := tx.Rollback(ctx)
		if err2 != nil {
			err = fmt.Errorf("%w (rollback transaction: %w)", err, err2)
		}

		return err
	}

	err = tx.Commit(ctx)
	if err != nil {
		err = fmt.Errorf("commit transaction: %w", err)
	}

	return
}

// //////////////////////////////////////////////////////////////////////////////
// //////////////////////////////////////////////////////////////////////////////
// Took bellow code from
// https://gist.github.com/zaydek/91f27cdd35c6240701f81415c3ba7c07
// Leaving it as-is for now.
var (
	replaceTabs                      = regexp.MustCompile(`\t+`)
	replaceSpacesBeforeOpeningParens = regexp.MustCompile(`\s+\(`)
	replaceSpacesAfterOpeningParens  = regexp.MustCompile(`\(\s+`)
	replaceSpacesBeforeClosingParens = regexp.MustCompile(`\s+\)`)
	replaceSpacesAfterClosingParens  = regexp.MustCompile(`\)\s+`)
	replaceSpaces                    = regexp.MustCompile(`\s+`)
)

// interpolateSQL replaces placeholders ($1, $2, ...) in an SQL query
// with their actual values for "pretty logging" purposes.
func interpolateSQL(sql string, args []any) string {
	for i, arg := range args {
		placeholder := fmt.Sprintf("$%d", i+1)
		var value string
		switch v := arg.(type) {
		case string:
			value = "'" + strings.ReplaceAll(v, "'", "''") + "'"
		case []byte:
			value = "'" + string(v) + "'"
		case time.Time:
			value = "'" + v.Format(time.RFC3339) + "'"
		default:
			value = fmt.Sprintf("%v", v)
		}
		sql = strings.Replace(sql, placeholder, value, 1)
	}
	return prettyPrintSQL(sql)
}

// prettyPrintSQL removes empty lines and trims spaces.
func prettyPrintSQL(sql string) string {
	lines := strings.Split(sql, "\n")

	pretty := strings.Join(lines, " ")
	pretty = replaceTabs.ReplaceAllString(pretty, "")
	pretty = replaceSpacesBeforeOpeningParens.ReplaceAllString(pretty, "(")
	pretty = replaceSpacesAfterOpeningParens.ReplaceAllString(pretty, "(")
	pretty = replaceSpacesAfterClosingParens.ReplaceAllString(pretty, ")")
	pretty = replaceSpacesBeforeClosingParens.ReplaceAllString(pretty, ")")

	// Finally, replace multiple spaces with a single space
	pretty = replaceSpaces.ReplaceAllString(pretty, " ")

	return strings.TrimSpace(pretty)
}

type queryTraceKey struct{}
type queryTraceData struct {
	SQL       string
	Args      []any
	StartTime time.Time
}

// LoggingQueryTracer ...
type LoggingQueryTracer struct{}

// NewLoggingQueryTracer creates and returns a new LoggingQueryTracer instance.
func NewLoggingQueryTracer() *LoggingQueryTracer {
	return &LoggingQueryTracer{}
}

// TraceQueryStart is called before a query is sent to the database.
// It records the query SQL and start time into the context.
func (l *LoggingQueryTracer) TraceQueryStart(
	ctx context.Context,
	_ *pgx.Conn,
	data pgx.TraceQueryStartData,
) context.Context {
	traceData := queryTraceData{
		SQL:       data.SQL,
		Args:      data.Args,
		StartTime: time.Now(),
	}

	return context.WithValue(ctx, queryTraceKey{}, traceData)
}

// TraceQueryEnd is called after a query has completed.
// It retrieves the start time and SQL from the context, calculates the duration,
// and logs the complete query information.
func (l *LoggingQueryTracer) TraceQueryEnd(ctx context.Context, _ *pgx.Conn, endData pgx.TraceQueryEndData) {
	data, ok := ctx.Value(queryTraceKey{}).(queryTraceData)
	if !ok {
		log.Println("[ERROR] TraceQueryEnd: could not retrieve trace data from context")
		return
	}

	dur := time.Since(data.StartTime)
	sql := interpolateSQL(data.SQL, data.Args)

	if endData.Err != nil {
		log.Printf("[%s] %s;\n[ERROR] %s", dur, sql, endData.Err)
		return
	}

	log.Printf("[%s] %s;\n", dur, sql)
}
