package ping

import (
	"flag"
	"local.lc/log"
)

func InitAgentBroker(config *AgentConfig) (*AgentBroker, error){
	return NewAgentBroker(config)
}

func InitScheduler(config *SchedulerConfig) (*Scheduler, error){
	return NewSheduler(config)
}

type Args struct {
	agent bool
	scheduler bool
	configfile string
}

var args = &Args{}

func initflag() {
	flag.BoolVar(&args.agent, "a", true, `Run as a agent.`)
	flag.BoolVar(&args.scheduler, "s", false, `Run as a scheduler.`)
	flag.StringVar(&args.configfile, "c", "config/config.ini", `configuration filename.`)
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

		scheduler, err := InitScheduler(config)
		if err != nil{
			log.Error(err)
			return
		}
		scheduler.Run()

	}else if args.agent{
		config, err := InitAgentConfig(args.configfile)
		if err != nil {
			log.Error(err)
			return
		}
		agent, err := InitAgentBroker(config)
		if err != nil{
			log.Error(err)
			return
		}
		agent.Run()
	}
}
