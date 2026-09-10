package main

import (
	"context"
	"flag"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"edge-rule-engine/internal/action"
	"edge-rule-engine/internal/api"
	"edge-rule-engine/internal/cleanup"
	"edge-rule-engine/internal/config"
	"edge-rule-engine/internal/engine"
	"edge-rule-engine/internal/store"
	"edge-rule-engine/ui"

	_ "modernc.org/sqlite"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func main() {
	configPath := flag.String("config", "config.yaml", "Path to config file")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	sseBroker := api.NewSSEBroker()

	var zLog *zap.Logger
	if cfg.Logging.Level == "debug" {
		zLog, _ = zap.NewDevelopment()
	} else {
		zLog, _ = zap.NewProduction()
	}
	
	sseLogSync := zapcore.AddSync(&api.SSELogWriter{Broker: sseBroker})
	zLog = zLog.WithOptions(zap.WrapCore(func(c zapcore.Core) zapcore.Core {
		encoderConfig := zap.NewProductionEncoderConfig()
		encoderConfig.TimeKey = "ts"
		encoderConfig.LevelKey = "level"
		encoderConfig.MessageKey = "msg"
		sseEncoder := zapcore.NewJSONEncoder(encoderConfig)
		sseCore := zapcore.NewCore(sseEncoder, sseLogSync, zap.DebugLevel)
		return zapcore.NewTee(c, sseCore)
	}))
	defer zLog.Sync()

	db, err := store.NewDB(cfg.Database.Path)
	if err != nil {
		zLog.Fatal("failed to connect to database", zap.Error(err))
	}
	defer db.Close()

	if err := store.RunMigrations(db); err != nil {
		zLog.Fatal("failed to run migrations", zap.Error(err))
	}

	ruleStore := store.NewRuleStore(db)
	stateStore := store.NewStateStore(db)
	eventStore := store.NewEventStore(db)
	execStore := store.NewExecutionStore(db)

	fileExecutor, err := action.NewFileExecutor(cfg.Actions.Directory, zLog)
	if err != nil {
		zLog.Fatal("failed to init file executor", zap.Error(err))
	}

	ruleEngine := engine.NewEngine(fileExecutor, execStore, sseBroker, zLog)

	rules, err := ruleStore.List()
	if err != nil {
		zLog.Fatal("failed to load rules", zap.Error(err))
	}
	for _, r := range rules {
		if err := ruleEngine.RegisterRule(*r); err != nil {
			zLog.Error("failed to register rule on startup", zap.String("ruleId", r.ID), zap.Error(err))
		}
	}
	zLog.Info("loaded rules", zap.Int("count", len(rules)))

	cleaner := cleanup.NewCleaner(stateStore, eventStore, cfg.Engine.StateRetentionHours, cfg.Engine.EventRetentionHours, zLog)
	cleanupCtx, cleanupCancel := context.WithCancel(context.Background())
	go cleaner.Start(cleanupCtx)

	distFS, err := fs.Sub(ui.Files, "dist")
	if err != nil {
		zLog.Fatal("failed to load embedded ui", zap.Error(err))
	}
	server := api.NewServer(ruleEngine, ruleStore, stateStore, eventStore, execStore, sseBroker, zLog, http.FS(distFS))

	httpServer := &http.Server{
		Addr:    cfg.Server.Address,
		Handler: server,
	}

	go func() {
		zLog.Info("starting server", zap.String("address", cfg.Server.Address))
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			zLog.Fatal("server error", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	zLog.Info("shutting down server...")

	cleanupCancel()

	ctxShutdown, cancelShutdown := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelShutdown()

	if err := httpServer.Shutdown(ctxShutdown); err != nil {
		zLog.Fatal("server shutdown failed", zap.Error(err))
	}
	zLog.Info("server exited")
}
