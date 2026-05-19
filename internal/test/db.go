package test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"deps-api/internal/db"
)

func DB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		dsn = "host=localhost user=postgres password=postgres dbname=deps_api_test port=5432 sslmode=disable"
	}
	database, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Skipf("skipping: cannot connect to test database: %v", err)
		return nil
	}
	require.NoError(t, db.Migrate(database))
	return database
}

func Truncate(t *testing.T, database *gorm.DB) {
	t.Helper()
	err := database.Exec("TRUNCATE TABLE employees, departments RESTART IDENTITY CASCADE").Error
	require.NoError(t, err)
}
