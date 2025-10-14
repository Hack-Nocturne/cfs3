package main

import (
	"fmt"
	"os"

	"github.com/Hack-Nocturne/cfs3"
	"github.com/Hack-Nocturne/cfs3/vars"
	_ "github.com/joho/godotenv/autoload"
)

func init() {
	os.MkdirAll(vars.UPLOAD_BASE_DIR, 0o755)
}

func main() {
	defer func() { os.RemoveAll(vars.UPLOAD_BASE_DIR) }()

	configFile := "cfs3.config.json"
	if len(os.Args) > 1 {
		configFile = os.Args[1]
	}

	config, cfgErr := cfs3.NewCFS3ConfigFromFile(configFile)
	if cfgErr != nil {
		fmt.Println("❌ Failure loading config:", cfgErr)
		return
	}

	if procErr := config.Process(); procErr != nil {
		fmt.Println("❌ Failure processing config:", procErr)
		return
	}

	if apErr := config.Apply(); apErr != nil {
		fmt.Println("❌ Failure processing config:", apErr)
	}
}
