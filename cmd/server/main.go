package main

import (
	"log"
	"os"

	"github.com/liondadev/sx-host/internal/config"
	"github.com/liondadev/sx-host/internal/database"
	"github.com/liondadev/sx-host/internal/server"
)

func main() {
	db, err := database.OpenSqlxDatabase()
	if err != nil {
		panic(err)
	}

	var configPath = "./config.json"
	p, ok := os.LookupEnv("SX_UPLOAD_CONFIG_PATH")
	if ok {
		configPath = p
	}

	conf, err := config.FromFile(configPath)
	if err != nil {
		panic(err)
	}

	s := server.NewServer(db, conf)

	log.Fatalln(s.Start(8080))
}
