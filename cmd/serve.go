package cmd

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/shidaxi/go-webhook/docs" // swagger docs

	"github.com/shidaxi/go-webhook/internal/config"
	"github.com/shidaxi/go-webhook/internal/engine"
	"github.com/shidaxi/go-webhook/internal/logger"
	"github.com/shidaxi/go-webhook/internal/server"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the webhook server",
	Long:  "Starts both the business webhook server and the admin server.",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Load config
		cfg, err := config.InitConfig(cfgFile)
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		// Initialize logger
		if err := logger.Init(cfg.Log.Format); err != nil {
			return fmt.Errorf("failed to init logger: %w", err)
		}
		defer logger.Sync()

		// Load and compile rules
		store := engine.NewRuleStore()
		if err := store.LoadAndCompile(cfg.Rules.Path); err != nil {
			return fmt.Errorf("failed to load rules: %w", err)
		}

		rules := store.GetRules()
		logger.L().Info("rules loaded",
			zap.Int("total", len(rules)),
		)

		// Start watching rules for hot reload
		stopWatch, err := store.WatchRules(cfg.Rules.Path)
		if err != nil {
			logger.L().Warn("failed to start rules watcher", zap.Error(err))
		} else {
			defer stopWatch()
		}

		// Create servers
		webhookEngine := server.NewWebhookEngine(store, "")
		adminEngine := server.NewAdminEngine(store, cfg)

		webhookSrv := &http.Server{
			Addr:    fmt.Sprintf(":%d", cfg.Server.Port),
			Handler: webhookEngine,
		}
		adminSrv := &http.Server{
			Addr:    fmt.Sprintf(":%d", cfg.Admin.Port),
			Handler: adminEngine,
		}

		// Start servers in goroutines
		go func() {
			logger.L().Info("webhook server starting", zap.Int("port", cfg.Server.Port))
			if err := webhookSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				logger.L().Fatal("webhook server failed", zap.Error(err))
			}
		}()
		go func() {
			logger.L().Info("admin server starting", zap.Int("port", cfg.Admin.Port))
			if err := adminSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				logger.L().Fatal("admin server failed", zap.Error(err))
			}
		}()

		// Graceful shutdown
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		<-quit

		logger.L().Info("shutting down servers...")

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := webhookSrv.Shutdown(ctx); err != nil {
			logger.L().Error("webhook server shutdown error", zap.Error(err))
		}
		if err := adminSrv.Shutdown(ctx); err != nil {
			logger.L().Error("admin server shutdown error", zap.Error(err))
		}

		logger.L().Info("servers stopped")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(serveCmd)
}
