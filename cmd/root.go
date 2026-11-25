package cmd

import (
	"backend-go/internal/app"
	"backend-go/internal/app/database"
	"backend-go/internal/config"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "backend-go",
	Short: "Backend Go CLI and HTTP server",
}

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the HTTP API server",
	RunE: func(cmd *cobra.Command, args []string) error {
		if watchMode {
			return nil
		}
		bootstrap := app.NewAppBootstrap()
		bootstrap.Run()
		return nil
	},
}

var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Run GORM database migrations",
	RunE: func(cmd *cobra.Command, args []string) error {
		db, err := database.InitDB(config.UseApp())
		if err != nil {
			return fmt.Errorf("connect db: %w", err)
		}

		if err := database.Migrate(db); err != nil {
			return fmt.Errorf("migrate db: %w", err)
		}

		fmt.Println("Database migration completed successfully.")
		return nil
	},
}

var watchMode bool

func init() {
	serveCmd.Flags().BoolVarP(&watchMode, "watch", "w", false, "Watch file changes and restart server")

	serveCmd.PreRunE = func(cmd *cobra.Command, args []string) error {
		if !watchMode {
			return nil
		}
		cleanArgs := []string{"serve"}
		return runWithWatch(".", cleanArgs)
	}

	rootCmd.AddCommand(serveCmd)
	rootCmd.AddCommand(migrateCmd)
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
