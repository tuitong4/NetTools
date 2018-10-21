package main

import (
	"bufio"
	"flag"
	"fmt"
	"golang.org/x/gosnmp"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"
)

const (
	sysDescr = "1.3.6.1.2.1.1.1.0"
	sysName  = "1.3.6.1.2.1.1.5.0"
)

var swVendor = map[string]*regexp.Regexp{
	"H3C":    regexp.MustCompile("H3C"),
	"HUAWEI": regexp.MustCompile("Huawei"),
	"NEXUS":  regexp.MustCompile("Cisco NX"),
	"CISCO":  regexp.MustCompile("Cisco"),
	"RUIJIE": regexp.MustCompile("Ruijie"),
}

func getVendor(host string, community string, port uint16) (version string) {
	conn := &gosnmp.GoSNMP{
		Target:    host,
		Port:      port,
		Community: community,
		Version:   gosnmp.Version2c,
		Timeout:   time.Duration(3) * time.Second,
	}

	err := conn.Connect()
	if err != nil {
		return "ConnectFailed"
	}
	defer conn.Conn.Close()

	_resp, err := conn.Get([]string{sysDescr})

	if err != nil {
		return "Unkown"
	}

	resp := _resp.Variables[0]

	switch resp.Type {
	case gosnmp.OctetString:
		sysdescr := string(resp.Value.([]byte))
		for k, v := range swVendor {
			if v.MatchString(sysdescr) {
				return k
			}
		}

	default:
		return "Unkown"
	}

	return "Unkown"
}

func getSysname(host string, community string, port uint16) (sysname string) {
	conn := &gosnmp.GoSNMP{
		Target:    host,
		Port:      port,
		Community: community,
		Version:   gosnmp.Version2c,
		Timeout:   time.Duration(3) * time.Second,
	}

	err := conn.Connect()
	if err != nil {
		return "ConnectFailed"
	}
	defer conn.Conn.Close()

	_resp, err := conn.Get([]string{sysName})

	if err != nil {
		return "Unkown"
	}

	resp := _resp.Variables[0]

	switch resp.Type {
	case gosnmp.OctetString:
		sysname = string(resp.Value.([]byte))
		return sysname

	default:
		return "Unkown"
	}

	return "Unkown"
}

type Args struct {
	hostfile  string
	community string
	verdor    bool
	sysname   bool
	inorder   bool
}

func readlines(filename string) ([]string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	reader := bufio.NewReader(file)
	var lines []string
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			break
		}
		lines = append(lines, strings.TrimSpace(line))
	}

	return lines, nil
}

var args = Args{}

func initflag() {
	flag.StringVar(&args.hostfile, "f", "", `Target hosts list file, one ip on a separate line, for example:
'10.10.10.10'
'12.12.12.12'.`)
	flag.StringVar(&args.community, "c", "360buy", `Snmp community.`)
	flag.BoolVar(&args.verdor, "vendor", false, "Retrieve switch's vendor.")
	flag.BoolVar(&args.sysname, "sysname", false, `Retrieve switch's sysname.`)
	flag.BoolVar(&args.inorder, "order", false, `Print tagert respone in same order with hostfile.`)
	flag.Parse()
}
func main() {
	initflag()

	if args.hostfile == "" {
		fmt.Println("Traget hostfile is expected but got none. See help docs.")
		os.Exit(0)
	}

	if args.verdor && args.sysname {
		fmt.Println("Only one type retrieve is spported.")
		os.Exit(0)
	}

	if !args.verdor && !args.sysname {
		fmt.Println("You should specify the retrieve type. See help docs.")
		os.Exit(0)
	}

	hosts, err := readlines(args.hostfile)
	if err != nil {
		fmt.Printf("Open file failed, %v\n", err)
	}

	maxThread := 500
	threadchan := make(chan struct{}, maxThread)
	wait := sync.WaitGroup{}

	if args.verdor {
		if args.inorder {
			//respchan := make(chan map[string]string, len(hosts))
			resps := make(map[string]string)
			for _, host := range hosts {
				wait.Add(1)
				resps[host] = ""
				go func(host string) {
					threadchan <- struct{}{}
					resps[host] = getVendor(host, args.community, 161)
					<-threadchan
					wait.Done()
				}(host)
			}
			wait.Wait()
			for k, v := range resps {
				fmt.Printf("%s\t%s\n", k, v)
			}

		} else {
			for _, host := range hosts {
				wait.Add(1)
				go func(host string) {
					threadchan <- struct{}{}
					fmt.Printf("%s\t%s\n", host, getVendor(host, args.community, 161))
					<-threadchan
					wait.Done()
				}(host)
			}
			wait.Wait()
		}
	}

	if args.sysname {
		if args.inorder {
			resps := make(map[string]string)
			for _, host := range hosts {
				wait.Add(1)
				resps[host] = ""
				go func(host string) {
					threadchan <- struct{}{}
					resps[host] = getSysname(host, args.community, 161)
					<-threadchan
					wait.Done()
				}(host)
			}
			wait.Wait()
			for k, v := range resps {
				fmt.Printf("%s\t%s\n", k, v)
			}

		} else {
			for _, host := range hosts {
				wait.Add(1)
				go func(host string) {
					threadchan <- struct{}{}
					fmt.Printf("%s\t%s\n", host, getSysname(host, args.community, 161))
					<-threadchan
					wait.Done()
				}(host)
			}
			wait.Wait()
		}
	}
}
