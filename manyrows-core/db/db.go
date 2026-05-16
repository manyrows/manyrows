package db

import (
	"context"
	"embed"
	"fmt"
	"strconv"
	"strings"
	"time"

	pgxuuid "github.com/jackc/pgx-gofrs-uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/rs/zerolog/log"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// defaultSchema is what the operator gets without setting MANYROWS_DB_SCHEMA.
// Routing every ManyRows table into a non-public schema keeps the install
// from colliding with whatever else lives in the same database (the
// operator's other apps, ad-hoc tooling, etc.). One ManyRows instance per
// database is still the supported topology — this isn't multi-tenancy.
const defaultSchema = "manyrows"

type DB struct {
	pool        *pgxpool.Pool
	schema      string
	initialized bool
}

type Config struct {
	DatabaseURL string
	// Schema is the Postgres namespace every ManyRows table — and the
	// goose_db_version tracker — lives in. Empty defaults to "manyrows".
	// Override via MANYROWS_DB_SCHEMA when the default name would
	// collide with something already in the database. Validated to
	// identifier chars only before being spliced into DDL.
	Schema            string
	MaxConns          int32
	MaxConnIdleTime   *time.Duration
	MinConns          *int32
	MinIdleConns      *int32
	MaxConnLifetime   *time.Duration
	HealthCheckPeriod *time.Duration

	// StatementTimeout sets Postgres's per-connection `statement_timeout`
	// GUC via SET on every pooled connection. Bounds the wall-clock
	// any single query can spend before the server cancels it — the
	// guardrail against a runaway query pinning a worker forever.
	// Zero (or nil) leaves it unset (server default, usually "off").
	// Override via MANYROWS_DB_STATEMENT_TIMEOUT_SECONDS.
	StatementTimeout *time.Duration

	// ApplicationName is reported to Postgres via `application_name`
	// GUC and shows up in pg_stat_activity / pg_stat_statements.
	// Empty defaults to "manyrows". Operators with multiple installs
	// against one cluster override per-deploy ("manyrows-prod-eu").
	// Override via MANYROWS_DB_APPLICATION_NAME.
	ApplicationName string

	// ConnectTimeout bounds the TCP+TLS handshake on each new
	// connection (pool fill, lazy expansion, replacement after idle
	// expiry). Without it, pgx waits indefinitely on the kernel
	// timeout — bad on platforms where the DB IP can flap during a
	// boot race. Nil leaves pgx's default behaviour intact.
	// Override via MANYROWS_DB_CONNECT_TIMEOUT_SECONDS.
	ConnectTimeout *time.Duration

	// SkipMigrations short-circuits goose on boot. Used by two-step
	// deploys that apply the schema separately from the binary
	// rollout (so the new binary can start without re-racing a
	// migration the previous deploy already ran). Default false —
	// the all-in-one boot is what most operators want.
	// Override via MANYROWS_DB_SKIP_MIGRATIONS=true.
	SkipMigrations bool
}

// defaultApplicationName is what pg_stat_activity sees when the
// operator doesn't override. Pin to the project name so multiple
// pgx binaries against the same cluster don't all show up as "pgx".
const defaultApplicationName = "manyrows"

func New(c Config) (*DB, error) {
	db := &DB{}
	err := db.initPool(c)
	if err != nil {
		return nil, err
	}
	if c.SkipMigrations {
		log.Info().Msg("db: migrations skipped (MANYROWS_DB_SKIP_MIGRATIONS=true)")
		return db, nil
	}
	err = db.runMigrations()
	if err != nil {
		return nil, err
	}
	return db, nil
}

func (d *DB) Pool() *pgxpool.Pool {
	return d.pool
}

// Schema returns the resolved schema name (post-default).
func (d *DB) Schema() string {
	return d.schema
}

// validSchemaName accepts unquoted Postgres identifiers — alphanumeric +
// underscore, must not start with a digit, ≤63 chars (PG's NAMEDATALEN).
// Used to gate the env-supplied schema name before splicing it into DDL.
func validSchemaName(s string) bool {
	if s == "" || len(s) > 63 {
		return false
	}
	for i, r := range s {
		switch {
		case r == '_':
		case r >= 'a' && r <= 'z':
		case r >= 'A' && r <= 'Z':
		case i > 0 && r >= '0' && r <= '9':
		default:
			return false
		}
	}
	return true
}

func (d *DB) initPool(c Config) error {
	if d.initialized {
		return nil
	}

	schema := c.Schema
	if schema == "" {
		schema = defaultSchema
	}
	if !validSchemaName(schema) {
		return fmt.Errorf("invalid db schema name %q (alphanumeric + underscore, must not start with a digit, ≤63 chars)", schema)
	}
	d.schema = schema

	dbConfig, err := pgxpool.ParseConfig(c.DatabaseURL)
	if err != nil {
		log.Err(err).Msg("Unable to parse db pool config")
		return err
	}

	// application_name shows up in pg_stat_activity / pg_stat_statements.
	// Pinning at config time (not via SET in AfterConnect) means it's
	// also reported during the initial handshake, before any query
	// runs. Operators on Postgres dashboards see the install name from
	// connection-open.
	appName := strings.TrimSpace(c.ApplicationName)
	if appName == "" {
		appName = defaultApplicationName
	}
	if dbConfig.ConnConfig.RuntimeParams == nil {
		dbConfig.ConnConfig.RuntimeParams = map[string]string{}
	}
	dbConfig.ConnConfig.RuntimeParams["application_name"] = appName

	if c.ConnectTimeout != nil && *c.ConnectTimeout > 0 {
		dbConfig.ConnConfig.ConnectTimeout = *c.ConnectTimeout
	}

	// Set statement_timeout via the parsed connection's RuntimeParams
	// so it's applied at startup-packet time. Postgres accepts ms
	// suffix, so "30000" / "30s" both work — we send ms to avoid
	// any locale-shenanigans with the duration's String().
	if c.StatementTimeout != nil && *c.StatementTimeout > 0 {
		dbConfig.ConnConfig.RuntimeParams["statement_timeout"] = strconv.FormatInt(c.StatementTimeout.Milliseconds(), 10)
	}

	dbConfig.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
		pgxuuid.Register(conn.TypeMap())
		// Pin search_path on every pooled connection so unqualified
		// table references resolve into our schema. "public" stays in
		// the path so extension functions (gen_random_uuid via
		// pgcrypto on older PG, etc.) keep resolving without explicit
		// qualification.
		if _, err := conn.Exec(ctx, fmt.Sprintf("SET search_path TO %s, public", schema)); err != nil {
			return fmt.Errorf("set search_path: %w", err)
		}
		return nil
	}
	dbConfig.MaxConns = c.MaxConns
	if c.MaxConnIdleTime != nil {
		dbConfig.MaxConnIdleTime = *c.MaxConnIdleTime
	}
	if c.MinConns != nil {
		dbConfig.MinConns = *c.MinConns
	}
	if c.HealthCheckPeriod != nil {
		dbConfig.HealthCheckPeriod = *c.HealthCheckPeriod
	}
	if c.MaxConnLifetime != nil {
		dbConfig.MaxConnLifetime = *c.MaxConnLifetime
	}
	if c.MinIdleConns != nil {
		dbConfig.MinIdleConns = *c.MinIdleConns
	}

	d.pool, err = pgxpool.NewWithConfig(context.Background(), dbConfig)
	if err != nil {
		log.Err(err).Msg("Unable to connect to database")
		return err
	}

	// Schema must exist before any pooled connection runs DDL or
	// goose attempts to create its version table. AfterConnect
	// already pinned search_path; this just creates the namespace
	// the path points at when the DB is empty.
	if _, err := d.pool.Exec(context.Background(), fmt.Sprintf("CREATE SCHEMA IF NOT EXISTS %s", schema)); err != nil {
		log.Err(err).Msg("Unable to create db schema")
		return err
	}

	d.initialized = true
	return nil
}

func (d *DB) Shutdown() {
	if d.pool == nil {
		return
	}
	d.pool.Close()
}

// runMigrations applies any pending goose migrations from the embedded
// migrations/*.sql tree. State is tracked in <schema>.goose_db_version.
// Goose acquires a session-level advisory lock so concurrent app instances
// can't race the same migration.
func (d *DB) runMigrations() error {
	sqlDB := stdlib.OpenDBFromPool(d.pool)
	defer sqlDB.Close()

	goose.SetBaseFS(migrationsFS)
	goose.SetLogger(gooseLogger{})
	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("goose: set dialect: %w", err)
	}

	// Keep goose's version tracker inside our schema too. Otherwise it
	// lands in public.goose_db_version, which defeats the whole point
	// of namespacing — anything inspecting public would see a
	// goose-shaped table from "some app" and have to dig.
	goose.SetTableName(d.schema + ".goose_db_version")

	if err := goose.UpContext(context.Background(), sqlDB, "migrations"); err != nil {
		return fmt.Errorf("goose: up: %w", err)
	}
	return nil
}

// gooseLogger forwards goose's stdlib-style logger calls to zerolog so
// migration output lands in the same structured stream as the rest of
// the app.
type gooseLogger struct{}

func (gooseLogger) Fatalf(format string, v ...interface{}) {
	log.Fatal().Msgf(format, v...)
}

func (gooseLogger) Printf(format string, v ...interface{}) {
	log.Info().Msgf(format, v...)
}
