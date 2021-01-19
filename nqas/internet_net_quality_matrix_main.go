package nqas

import (
	"flag"
	"fmt"
	"io"
	"local.lc/log"
	"os"
	"strings"
)

func pathIsExist(path string) bool {
	_, err := os.Stat(path)
	if err != nil {
		if os.IsExist(err) {
			return true
		}
		return false
	}
	return true
}

func openLogFile(name string) (*os.File, error) {
	if pathIsExist(name) {
		return os.OpenFile(name, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	}

	paths := strings.Split(name, "/")
	if len(paths) == 0 {
		return nil, fmt.Errorf("Invalied filename '%s'.", name)
	}

	will_to_be_created_path := ""
	for _, dir := range paths[0 : len(paths)-1] {
		will_to_be_created_path += dir + "/"
		if !pathIsExist(will_to_be_created_path) {
			if err := os.Mkdir(will_to_be_created_path, 0777); err != nil {
				return nil, err
			}
			if err := os.Chmod(will_to_be_created_path, 0777); err != nil {
				return nil, err
			}
		}
	}

	w, err := os.Create(name)
	if err != nil {
		return nil, err
	}
	if err := os.Chmod(name, 0666); err != nil {
		return nil, err
	}

	return w, nil
}

func SetLogger(prefix string, output io.Writer) {
	log.SetPrefix(prefix)
	log.SetOutput(output)
}

type Args struct {
	configFile string
}

var args = &Args{}

func initFlag() {
	flag.StringVar(&args.configFile, "c", "config/config.conf", `Configuration filename.`)
	flag.Parse()
}

func Run() {
	initFlag()
	if args.configFile == "" {
		log.Error("Configuration file should not be empty.")
		return
	}
	config, err := InitConfig(args.configFile)
	if err != nil {
		log.Error(err)
		return
	}

	w, err := openLogFile(config.LoggerConfig.LogFile)
	if err != nil {
		log.Error(err)
	}
	defer w.Close()

	SetLogger("matrix-", w)

	//初始化全局变量
	internetNetQualityDataSource = config.DruidConfig.DataSource

}
