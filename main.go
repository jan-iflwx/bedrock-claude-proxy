package main

import (
	"bedrock-claude-proxy/pkg"
	"flag"
	"runtime"

	"github.com/joho/godotenv"
)

func main() {
	conf_path := flag.String("c", "config.json", "config json file")
	flag.Parse()

	runtime.GOMAXPROCS(runtime.NumCPU())

	var conf *pkg.Config
	var err error
	if len(*conf_path) > 0 {
		conf, err = pkg.NewConfigFromLocal(*conf_path)
		if err != nil {
			pkg.Log.Error(err)
			conf = &pkg.Config{}
		}
	} else {
		conf = &pkg.Config{}
	}

	// Load .env file
	err = godotenv.Load()
	if err != nil {
		pkg.Log.Fatal("Error loading .env file")
	}

	conf.MarginWithENV()

	pkg.InitLogger()
	pkg.Log.Debug("show config detail:")
	pkg.Log.Debug(conf.ToJSON())

	service := pkg.NewHttpService(conf)
	service.Start()
}
