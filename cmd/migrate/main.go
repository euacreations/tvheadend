package main

import (
	"fmt"
	"log"
	"os"

	"github.com/euacreations/tvheadend/internal/config"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/mysql"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func main() {
	cfg := config.LoadConfig()

	dsn := buildDSN(cfg)
	m, err := migrate.New(
		"file://internal/database/migrations",
		dsn,
	)
	if err != nil {
		log.Fatal(err)
	}

	cmd := "up"
	if len(os.Args) > 1 {
		cmd = os.Args[1]
	}

	switch cmd {
	case "up":
		if err := m.Up(); err != nil && err != migrate.ErrNoChange {
			log.Fatal(err)
		}
		log.Println("Migrations applied successfully")
	case "down":
		if err := m.Down(); err != nil && err != migrate.ErrNoChange {
			log.Fatal(err)
		}
		log.Println("Migrations rolled back successfully")
	default:
		log.Fatalf("Unknown command: %s", cmd)
	}
}

func buildDSN(cfg *config.Config) string {
	return fmt.Sprintf("mysql://%s:%s@tcp(%s:%d)/%s?multiStatements=true",
		cfg.DBUser,
		cfg.DBPassword,
		cfg.DBHost,
		cfg.DBPort,
		cfg.DBName,
	)
}
