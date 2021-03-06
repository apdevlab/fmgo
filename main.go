package main

import (
	"context"
	"flag"
	"fmgo/common/config"
	"fmgo/common/data"
	"fmgo/common/data/model"
	"fmgo/module/friend"
	"fmgo/module/notification"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/gin-gonic/contrib/static"
	"github.com/gin-gonic/gin"
	"github.com/golang/glog"
	"github.com/szuecs/gin-glog"
)

var (
	appName = "fmgo"
	version = "development"

	showVersion            bool
	runMigration           bool
	configuration          config.Configuration
	dbFactory              *data.DBFactory
	friendController       *friend.Controller
	notificationController *notification.Controller
)

func init() {
	flag.BoolVar(&showVersion, "version", false, "print version information")
	flag.BoolVar(&runMigration, "migrate", false, "run db migration before starting app")
	flag.Parse()

	if showVersion {
		fmt.Printf("%s version %s\n", appName, version)
		os.Exit(0)
	}

	glog.V(2).Info("Initializing configuration...")
	cfg, err := config.New()
	if err != nil {
		glog.Fatalf("Failed to load configuration: %s", err)
	}

	configuration = *cfg
	dbFactory = data.NewDbFactory(cfg.Database)

	if runMigration {
		glog.Info("Running db migration")
		err := retry(5, 2*time.Second, func() error {
			db, err := dbFactory.DBConnection()
			if err != nil {
				return err
			}
			defer db.Close()

			db.AutoMigrate(&model.User{})
			return nil
		})

		if err != nil {
			glog.Fatalf("Failed to open database connection after 5 retries: %s", err)
		}

		glog.Info("Done running db migration")
	}

	friendController = friend.NewController(dbFactory)
	notificationController = notification.NewController(dbFactory)
}

func setupRouter() *gin.Engine {
	router := gin.New()
	logDuration := time.Duration(configuration.Server.LogDuration) * time.Second

	router.Use(ginglog.Logger(logDuration), gin.Recovery())
	router.Use(static.Serve("/", static.LocalFile("./public", true)))

	router.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"version":    version,
			"serverTime": time.Now(),
		})
	})

	api := router.Group("/api")
	{
		api.POST("/friend/connect", friendController.Connect)
		api.POST("/friend/list", friendController.GetFriends)
		api.POST("/friend/common", friendController.GetCommons)

		api.POST("/notification/subscribe", notificationController.Subscribe)
		api.POST("/notification/block", notificationController.Block)
		api.POST("/notification/list", notificationController.GetNotificationList)
	}

	return router
}

func main() {
	glog.V(2).Infof("Setting up server mode to %s", configuration.Server.Mode)
	gin.SetMode(configuration.Server.Mode)

	glog.V(2).Info("Setting up server side routing...")
	r := setupRouter()

	srv := &http.Server{
		Addr:    configuration.Server.Addr,
		Handler: r,
	}

	go func() {
		glog.Infof("Starting %s server version %s at %s", appName, version, configuration.Server.Addr)
		if err := srv.ListenAndServe(); err != nil {
			if err.Error() != "http: Server closed" {
				glog.Fatalf("Failed to start server: %s", err)
			}
		}
	}()

	// wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal)
	signal.Notify(quit, os.Interrupt)
	<-quit

	shutdownTimeout := time.Duration(configuration.Server.ShutdownTimeout) * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	glog.Info("Shutting down server...")
	if err := srv.Shutdown(ctx); err != nil {
		glog.Errorf("Failed to shutdown server gracefully: %s", err)
	}

	glog.Info("Server shutted down")
}

type stop struct {
	error
}

func retry(attempts int, sleep time.Duration, fn func() error) error {
	if err := fn(); err != nil {
		if s, ok := err.(stop); ok {
			return s.error
		}

		if attempts--; attempts > 0 {
			time.Sleep(sleep)
			return retry(attempts, 2*sleep, fn)
		}

		return err
	}

	return nil
}
