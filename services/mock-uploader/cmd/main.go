package main

import (
	"log"
	"mock-uploader/config"
	"net/http"

	"github.com/gin-gonic/gin"
)


type Application struct {
	Config *config.Config
}

func main() {
	cfg := config.LoadConfig()
	app := &Application{Config: cfg}

	r := gin.Default()

	r.GET("/info", app.handleInfo)
	log.Printf("%s", "API is running on port: " + cfg.Port)
	r.Run(":" + cfg.Port)
}

func (app *Application) handleInfo(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"environment": app.Config.Environment,
		"status":      "running",
	})
}