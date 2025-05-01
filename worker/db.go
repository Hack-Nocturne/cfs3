package worker

import (
	"fmt"
	"os"

	"github.com/Hack-Nocturne/cfs3/vars"
	"github.com/kofj/gorm-driver-d1/gormd1"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var db *gorm.DB

func init() {
	vars.CF_ACCOUNT_ID = os.Getenv("CF_ACCOUNT_ID")
	vars.CF_API_TOKEN = os.Getenv("CF_API_TOKEN")
	cfDBId := os.Getenv("CF_DATABASE_ID")

	if vars.CF_ACCOUNT_ID == "" || vars.CF_API_TOKEN == "" || cfDBId == "" {
		fmt.Println("❌ Missing either CF_ACCOUNT_ID, CF_API_TOKEN or CF_DATABASE_ID environment variables")
		os.Exit(1)
	}

	var err error
	// Initialize the database connection
	d1Dialect := gormd1.Open(fmt.Sprintf("d1://%s:%s@%s", vars.CF_ACCOUNT_ID, vars.CF_ACCOUNT_ID, cfDBId))
	db, err = gorm.Open(d1Dialect, &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		fmt.Println("❌ Failed to connect to the database:", err)
		os.Exit(1)
	}

	// Migrate the schema
	migErr := db.AutoMigrate(&Object{})
	if migErr != nil {
		fmt.Println("❌ Failed to migrate the database schema:", migErr)
		os.Exit(1)
	}
}
