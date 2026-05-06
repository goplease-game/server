// Package test ...
package test

import (
	"context"
	"fmt"
	"log"
	"strings"
	"testing"

	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/jackc/pgx/v5"
	"github.com/logrusorgru/aurora"
	"github.com/ognev-dev/goplease/app"
	"github.com/ognev-dev/goplease/app/ds"
	"github.com/ognev-dev/goplease/app/service"
	"github.com/ognev-dev/goplease/email"
	"github.com/ognev-dev/goplease/test/factory"
	"github.com/ognev-dev/goplease/tracing"
	otelTrace "go.opentelemetry.io/otel/trace"
)

// App holds common dependencies for tests among different test namespaces.
type App struct {
	Conf    *app.ConfigT
	Tracer  otelTrace.Tracer
	DB      *app.DB
	Service *service.Service
	Factory *factory.Factory
}

// NewApp ...
func NewApp() *App {
	ctx := context.Background()
	tracer, err := tracing.New(ctx)
	if err != nil {
		panic("[TEST] New app: " + err.Error())
	}

	db, err := app.NewDB(ctx)
	if err != nil {
		panic("[TEST] New app: " + err.Error())
	}

	err = app.MigrateDB(ctx, db)
	if err != nil {
		log.Fatal(err)
	}

	return &App{
		Conf:    app.Config(),
		Tracer:  tracer,
		DB:      db,
		Service: service.New(db, tracer),
		Factory: factory.New(db),
	}
}

// Shutdown ...
// Initiator of NewApp must care to call Shutdown().
func (a *App) Shutdown() {
	a.DB.Close()
}

// Data is a type alias representing a map of database columns and their expected values,
// used for assertions against the database.
type Data map[string]any

// notNull is an unexported type used as a marker value within the Data map
// to check if a database column contains a non-NULL value.
type notNull bool

func (nn notNull) String() string {
	return "NOT NULL"
}

// NotNull is the exported constant marker value to assert that a database column
// must not be NULL. Use it as a value in a Data map: Data{"column_name": test.NotNull}.
var NotNull notNull

// CheckErr is a test helper function that fails the test immediately if the provided error is not nil.
func CheckErr(t *testing.T, err error) {
	t.Helper()

	if err != nil {
		t.Fatal(err)
	}
}

// LoadEmailVars retrieves the template variables from the most recent email sent to the given recipient
// via the TestSender email driver.
func LoadEmailVars(t *testing.T, to string) map[string]any {
	t.Helper()

	c, err := email.LoadTestEmail(to)
	if err != nil {
		t.Error(err)
	}

	return c.Variables()
}

func dbIdent(i string) string {
	return pgx.Identifier{i}.Sanitize()
}

func countDatabaseRows(t *testing.T, db *app.DB, table string, data Data) int {
	t.Helper()

	args := make([]any, 0)
	wheres := make([]string, 0)

	argIndex := 1

	for col, val := range data {
		col = dbIdent(col)
		if val == nil {
			wheres = append(wheres, col+" IS NULL")

			continue
		}

		var whereExpr string

		switch val.(type) {
		case bool:
			whereExpr = fmt.Sprintf(`%s IS %v`, col, val)
		case notNull:
			whereExpr = col + " IS NOT NULL"
		default:
			whereExpr = fmt.Sprintf(`%s = $%d`, col, argIndex)

			args = append(args, val)
			argIndex++
		}

		wheres = append(wheres, whereExpr)
	}

	query := fmt.Sprintf("SELECT COUNT(1) AS COUNT FROM %s WHERE %s", dbIdent(table), strings.Join(wheres, " AND "))

	var count int

	err := pgxscan.Get(context.Background(), db, &count, query, args...)
	if err != nil {
		t.Fatal(aurora.Bold(aurora.Red(fmt.Sprintf("CountDatabaseRows: %s", err))).String())
	}

	return count
}

// AssertInDB asserts that at least one row exists in the given table that matches the criteria in 'data'.
func AssertInDB(t *testing.T, db *app.DB, table string, data Data) {
	t.Helper()

	count := countDatabaseRows(t, db, table, data)
	if count == 0 {
		println(aurora.Bold(aurora.Red("❌ Table '" + table + "' missing row with data:")).String())

		for k, v := range data {
			println("\t" + k + "=" + aurora.Blue(fmt.Sprintf("%+v", v)).String())
		}

		t.FailNow()
	}
}

// AssertNotInDB asserts that no rows exist in the given table that match the criteria in 'data'.
func AssertNotInDB(t *testing.T, db *app.DB, table string, data Data) {
	t.Helper()

	count := countDatabaseRows(t, db, table, data)
	if count != 0 {
		t.Fail()
		println(aurora.Bold(aurora.Red("❌ Table '" + table + "' has row with data:")).String())

		for k, v := range data {
			println("\t" + k + ": " + fmt.Sprintf("%+v", v))
		}
	}
}

// AssertDeleted asserts that row with "deleted_at" is not null.
func AssertDeleted(t *testing.T, db *app.DB, table string, id ds.ID) {
	t.Helper()

	AssertInDB(t, db, table, Data{
		"id":         id,
		"deleted_at": NotNull,
	})
}
