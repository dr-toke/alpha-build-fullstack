package store

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib" // registers "pgx" database/sql driver, for goose only
	"github.com/pressly/goose/v3"

	dbsql "database/sql"
)

// TestMain starts ONE throwaway Postgres 16 container for the whole
// internal/store test suite (11 files, dozens of tests) rather than one per
// test function — internal/db/migrations/migrations_test.go proved the
// per-test-container pattern works (M0), but paying that ~2s startup cost
// eleven-plus times over would make this package's test suite slow enough
// that people stop running it locally. Individual tests get isolation by
// using unique, randomly-suffixed identifiers (handles, slugs, names) rather
// than a fresh database per test — see testAccount/testListing/etc. helpers
// in each _test.go file.
var testStore *Store

func TestMain(m *testing.M) {
	if _, err := exec.LookPath("docker"); err != nil {
		fmt.Println("docker not available, skipping internal/store tests")
		os.Exit(0)
	}

	name := fmt.Sprintf("drtoke-storetest-%d", time.Now().UnixNano())
	run := exec.Command("docker", "run", "-d", "--rm", "--name", name,
		"-e", "POSTGRES_PASSWORD=test",
		"-e", "POSTGRES_USER=drtoke",
		"-e", "POSTGRES_DB=drtoke_test",
		"-p", "127.0.0.1::5432",
		"postgres:16",
	)
	if out, err := run.CombinedOutput(); err != nil {
		fmt.Printf("docker run: %v\n%s\n", err, out)
		os.Exit(1)
	}
	cleanup := func() { _ = exec.Command("docker", "rm", "-f", name).Run() }

	portOut, err := exec.Command("docker", "port", name, "5432/tcp").CombinedOutput()
	if err != nil {
		cleanup()
		fmt.Printf("docker port: %v\n%s\n", err, portOut)
		os.Exit(1)
	}
	addr := strings.TrimSpace(strings.Split(string(portOut), "\n")[0])
	port := addr[strings.LastIndex(addr, ":")+1:]
	dsn := fmt.Sprintf("postgres://drtoke:test@127.0.0.1:%s/drtoke_test?sslmode=disable", port)

	if err := waitAndMigrate(dsn); err != nil {
		cleanup()
		fmt.Println(err)
		os.Exit(1)
	}

	ctx := context.Background()
	s, err := New(ctx, dsn)
	if err != nil {
		cleanup()
		fmt.Printf("store.New: %v\n", err)
		os.Exit(1)
	}
	testStore = s

	code := m.Run()

	s.Close()
	cleanup()
	os.Exit(code)
}

func waitAndMigrate(dsn string) error {
	var db *dbsql.DB
	var err error
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		db, err = dbsql.Open("pgx", dsn)
		if err == nil {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			pingErr := db.PingContext(ctx)
			cancel()
			if pingErr == nil {
				break
			}
			_ = db.Close()
		}
		time.Sleep(500 * time.Millisecond)
	}
	if db == nil {
		return fmt.Errorf("postgres did not become ready: %w", err)
	}
	defer db.Close()

	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("goose.SetDialect: %w", err)
	}
	if err := goose.Up(db, "../db/migrations"); err != nil {
		return fmt.Errorf("goose.Up: %w", err)
	}
	return nil
}
