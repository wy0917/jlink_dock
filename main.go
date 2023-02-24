package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/wy0917/jlink_dock/controller"
	"github.com/wy0917/jlink_dock/model"

	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	_ "github.com/wy0917/jlink_dock/docs"
)

var (
	id  uuid.UUID
	log *logrus.Logger
)

// @title           Swagger Example API
// @version         1.0
// @description     This is a sample server celler server.
// @termsOfService  http://swagger.io/terms/

// @contact.name   API Support
// @contact.url    http://www.swagger.io/support
// @contact.email  support@swagger.io

// @license.name  Apache 2.0
// @license.url   http://www.apache.org/licenses/LICENSE-2.0.html

// @host      localhost:80
// @BasePath  /api/v1

// @securityDefinitions.basic  BasicAuth

// @externalDocs.description  OpenAPI
// @externalDocs.url          https://swagger.io/resources/open-api/
func main() {
	id = uuid.New()
	var configFile string
	flag.StringVar(&configFile, "config", "config.toml", "Path to the config file")

	cfg := model.LoadConfig(configFile)

	flag.StringVar(&cfg.ACM, "acm", cfg.ACM, "ACM TTY device")
	flag.StringVar(&cfg.TTY, "tty", cfg.TTY, "2nd TTY Device for debugging")
	flag.StringVar(&cfg.Serial, "serial", cfg.Serial, "Short Serial from STM32 device")
	flag.StringVar(&cfg.Type, "type", cfg.Type, "STM32 board type")
	flag.Parse()

	log = logrus.New()

	// Set file output for log
	logFile, err := os.CreateTemp("", fmt.Sprintf("log-%s.txt", id))
	if err != nil {
		log.Fatalln(err)
	}
	mw := io.MultiWriter(logFile, os.Stdout)
	log.SetOutput(mw)

	injectContext := func(cfg model.Config) gin.HandlerFunc {
		return func(c *gin.Context) {
			c.Set("config", &cfg)
			c.Set("logger", &log)
			c.Set("log_path", logFile.Name())
			c.Next()
		}
	}

	c := controller.NewController()

	r := gin.Default()
	v1 := r.Group("/api/v1")
	{
		info := v1.Group("/info")
		{
			info.GET("", injectContext(*cfg), c.GetInfo)
		}
		log_ := v1.Group("/log")
		{
			log_.GET("", injectContext(*cfg), c.GetLog)
		}
		kpi := v1.Group("/script")
		{
			kpi.POST("", injectContext(*cfg), c.RunScript)
		}
	}
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	log.Println("Starting http service on port 80")
	err = r.Run(fmt.Sprintf(cfg.Server))
	if err != nil {
		return
	}
}
