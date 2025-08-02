package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/kdimtricp/vshazam/internal/database"
)

func main() {
	var (
		dbType         = flag.String("db", "postgres", "Database type (postgres or sqlite)")
		host           = flag.String("host", "localhost", "Database host")
		port           = flag.Int("port", 5432, "Database port")
		user           = flag.String("user", "vshazam", "Database user")
		password       = flag.String("password", "vshazam_dev", "Database password")
		dbName         = flag.String("name", "vshazam", "Database name")
		migrationsPath = flag.String("migrations", "./migrations", "Path to migrations directory")
		status         = flag.Bool("status", false, "Show migration status only")
	)
	flag.Parse()

	// Build database config
	config := database.Config{
		Type:     *dbType,
		Host:     *host,
		Port:     *port,
		User:     *user,
		Password: *password,
		Name:     *dbName,
	}

	// Override with environment variables if set
	if env := os.Getenv("DB_TYPE"); env != "" {
		config.Type = env
	}
	if env := os.Getenv("DB_HOST"); env != "" {
		config.Host = env
	}
	if env := os.Getenv("DB_USER"); env != "" {
		config.User = env
	}
	if env := os.Getenv("DB_PASSWORD"); env != "" {
		config.Password = env
	}
	if env := os.Getenv("DB_NAME"); env != "" {
		config.Name = env
	}

	// Connect to database
	db, err := database.NewDB(config)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer db.Close()

	migrator := database.NewMigrator(db.Conn(), config.Type)

	if *status {
		// Show migration status
		if err := migrator.Initialize(); err != nil {
			log.Fatal("Failed to initialize migrator:", err)
		}

		applied, err := migrator.GetAppliedMigrations()
		if err != nil {
			log.Fatal("Failed to get applied migrations:", err)
		}

		migrations, err := migrator.LoadMigrations(*migrationsPath)
		if err != nil {
			log.Fatal("Failed to load migrations:", err)
		}

		fmt.Println("Migration Status:")
		fmt.Println("=================")
		for _, m := range migrations {
			status := "pending"
			if applied[m.Version] {
				status = "applied"
			}
			fmt.Printf("%s - %s [%s]\n", m.Version, m.Name, status)
		}
	} else {
		// Run migrations
		fmt.Printf("Running migrations from %s...\n", *migrationsPath)
		if err := db.RunMigrations(*migrationsPath); err != nil {
			log.Fatal("Failed to run migrations:", err)
		}
		fmt.Println("Migrations completed successfully!")
	}
}
