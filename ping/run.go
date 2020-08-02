package ping

import (
	"flag"
	"fmt"
	"io"
	"local.lc/log"
	"os"
	"strings"
)

func InitAgentBroker(config *AgentConfig) (*AgentBroker, error){
	return NewAgentBroker(config)
}

func InitScheduler(config *SchedulerConfig) (*Scheduler, error){
	return NewScheduler(config)
}


func SetLogger(prefix string, output io.Writer){
	log.SetPrefix(prefix)
	log.SetOutput(output)
}


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

func openLogFile(name string) (*os.File, error){
	if pathIsExist(name) {
		return os.OpenFile(name, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	}

	paths := strings.Split(name, "/")
	if len(paths) == 0 {
		return nil, fmt.Errorf("Invalied filename '%s'.", name)
	}

	will_to_be_created_path := ""
	for _, dir := range paths[0:len(paths)-1]{
		will_to_be_created_path += dir + "/"
		if !pathIsExist(will_to_be_created_path){
			if err := os.Mkdir(will_to_be_created_path, 0777); err != nil{
				return nil, err
			}
			if err := os.Chmod(will_to_be_created_path, 0777); err != nil{
				return nil, err
			}
		}
	}

	w, err := os.Create(name)
	if err != nil {
		return nil, err
	}
	if err := os.Chmod(name, 0666); err != nil{
		return nil, err
	}

	return w, nil
}

type Args struct {
	agent bool
	scheduler bool
	configfile string
}

var args = &Args{}

func initflag() {
	flag.BoolVar(&args.agent, "a", true, `Run as an agent.`)
	flag.BoolVar(&args.scheduler, "s", false, `Run as an scheduler.`)
	flag.StringVar(&args.configfile, "c", "config/config.conf", `Configuration filename.`)
	flag.Parse()
}

func Run(){
	initflag()

	if args.configfile == "" {
		log.Error("Configration file should not be ''.")
		return
	}

	if args.scheduler{
		config, err := InitSchedulerConfig(args.configfile)
		if err != nil {
			log.Error(err)
			return
		}

		w, err := openLogFile(config.Logger.LogFile)
		if err != nil {
			log.Error(err)
		}
		defer w.Close()

		SetLogger("Scheduler-", w)

		scheduler, err := InitScheduler(config)
		if err != nil{
			log.Error(err)
			return
		}

		scheduler.run()

	}else if args.agent{
		config, err := InitAgentConfig(args.configfile)
		if err != nil {
			log.Error(err)
			return
		}

		w, err := openLogFile(config.Logger.LogFile)
		if err != nil {
			log.Error(err)
		}
		defer w.Close()

		SetLogger("Agent-", w)

		agent, err := InitAgentBroker(config)
		if err != nil{
			log.Error(err)
			return
		}

		agent.Run()
	}
}
