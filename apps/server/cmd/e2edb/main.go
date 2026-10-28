// e2edb creates and removes an isolated, disposable PostgreSQL schema for browser tests.
package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net/url"
	"os"
	"regexp"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

var allowedSchema = regexp.MustCompile(`^auth_e2e_[0-9a-f]{16}$`)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
func run() error {
	if len(os.Args) < 2 || len(os.Args) > 3 {
		return errors.New("usage: e2edb create|drop <private-env-file> OR e2edb verify")
	}
	raw := os.Getenv("DATABASE_URL")
	if raw == "" {
		return errors.New("DATABASE_URL required")
	}
	pool, err := pgxpool.New(context.Background(), raw)
	if err != nil {
		return err
	}
	defer pool.Close()
	if os.Args[1] == "verify" {
		if len(os.Args) != 2 || !allowedSchema.MatchString(os.Getenv("AUTH_E2E_SCHEMA")) {
			return errors.New("AUTH_E2E_SCHEMA required")
		}
		var current string
		if err := pool.QueryRow(context.Background(), "SELECT current_schema()").Scan(&current); err != nil {
			return err
		}
		if current != os.Getenv("AUTH_E2E_SCHEMA") {
			return errors.New("database URL does not select the isolated schema")
		}
		fmt.Println("isolated schema selected", current)
		return nil
	}
	if len(os.Args) != 3 {
		return errors.New("private environment file required")
	}
	path := os.Args[2]
	switch os.Args[1] {
	case "create":
		if _, err = os.Stat(path); err == nil {
			return errors.New("environment file already exists; drop it first")
		}
		if !os.IsNotExist(err) {
			return err
		}
		var random [8]byte
		if _, err = rand.Read(random[:]); err != nil {
			return err
		}
		schema := "auth_e2e_" + hex.EncodeToString(random[:])
		if _, err = pool.Exec(context.Background(), "CREATE SCHEMA "+schema); err != nil {
			return err
		}
		parsed, err := url.Parse(raw)
		if err != nil {
			pool.Exec(context.Background(), "DROP SCHEMA "+schema+" CASCADE")
			return err
		}
		q := parsed.Query()
		q.Set("options", "-csearch_path="+schema)
		parsed.RawQuery = q.Encode()
		content := "DATABASE_URL=" + shellQuote(parsed.String()) + "\nAUTH_E2E_SCHEMA=" + shellQuote(schema) + "\n"
		file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
		if err != nil {
			pool.Exec(context.Background(), "DROP SCHEMA "+schema+" CASCADE")
			return err
		}
		_, err = file.WriteString(content)
		closeErr := file.Close()
		if err == nil {
			err = closeErr
		}
		if err != nil {
			os.Remove(path)
			pool.Exec(context.Background(), "DROP SCHEMA "+schema+" CASCADE")
			return err
		}
		fmt.Println("created isolated schema", schema)
		return nil
	case "drop":
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		var schema string
		for _, line := range strings.Split(string(data), "\n") {
			if strings.HasPrefix(line, "AUTH_E2E_SCHEMA=") {
				schema = strings.Trim(strings.TrimPrefix(line, "AUTH_E2E_SCHEMA="), "'")
			}
		}
		if !allowedSchema.MatchString(schema) {
			return errors.New("refusing to drop an unexpected schema name")
		}
		if _, err = pool.Exec(context.Background(), "DROP SCHEMA "+schema+" CASCADE"); err != nil {
			return err
		}
		if err = os.Remove(path); err != nil {
			return err
		}
		fmt.Println("dropped isolated schema", schema)
		return nil
	default:
		return errors.New("usage: e2edb create|drop <private-env-file> OR e2edb verify")
	}
}
func shellQuote(v string) string { return "'" + strings.ReplaceAll(v, "'", "'\\''") + "'" }
