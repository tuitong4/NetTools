package main

import (
    "bufio"
    "flag"
    "fmt"
    "io/ioutil"
    "log"
    "nwssh"
    "os"
    "strings"
    "sync"
    "time"
)

type Args struct {
    hostfile     string
    host         string
    cmd          string
    swvendor     string
    username     string
    password     string
    port         string
    timeout      int
    readwaittime int
    cmdtimeout   int
    logdir       string
    help         bool
    patch        string
    upgradetime  int
}

var args = Args{}

func initflag() {
    flag.StringVar(&args.hostfile, "f", "", `Target hosts list file, one ip on a separate line, for example:
'10.10.10.10'
'12.12.12.12'.`)
    flag.StringVar(&args.swvendor, "V", "", `Vendor of target host, if not spicified, it will be checked 
automatically.`)
    flag.StringVar(&args.username, "u", "", "Username for login.")
    flag.StringVar(&args.password, "p", "", "Password for login.")
    flag.StringVar(&args.port, "port", "22", "Target SSH port to connect.")
    flag.IntVar(&args.timeout, "timeout", 10, "SSH connection timeout in seconds.")
    flag.IntVar(&args.readwaittime, "readwaittime", 500, `The time to wait ssh channel return the respone, if time reached,
end the wait. In Millisecond.`)
    flag.StringVar(&args.logdir, "logpath", "", "Log command output to /<path>/<ip_addr> instead of stdout.")
    flag.IntVar(&args.cmdtimeout, "cmdtimeout", 10, `The wait time for executing commands remotely, if timeout reached, 
means execution is failed.`)
    flag.BoolVar(&args.help, "help", false, `Usage of CLI.`)
    flag.StringVar(&args.patch, "patch", "", `Patch files use ';' as dilimiter.`)
    flag.IntVar(&args.upgradetime, "upgradetime", 600, `Time of waiting the upgrade complate, in second.`)
    flag.Parse()
}

func install_h3c(host, port string, args Args, sshoptions nwssh.SSHOptions) (string, bool) {

    var devssh *nwssh.SSHBase
    var err error

    username := args.username
    password := args.password

    devssh, err = nwssh.SSH(host, port, username, password, time.Duration(10)*time.Second, sshoptions)
    defer devssh.Close()
    if err != nil {
        log.Printf("[%s]%v\n", host, err)
        return "", false
    }
    if err = devssh.Connect(); err != nil {
        log.Printf("[%s]%v\n", host, err)
        return "", false
    }

    var device nwssh.SSHBASE

    device = &nwssh.H3cSSH{devssh}

    if !device.SessionPreparation() {
        log.Printf("[%s]Failed init execute envirment. Try to exectue command directly.", host)
    }
    patches := strings.Split(args.patch, ";")

    output := ""
    for _, patch := range patches {
        cmd := `install activate patch flash:/` + patch + ` all`

        o, err := device.ExecCommandExpect(cmd, "Y/N]:", time.Second*10)
        if err != nil {
            log.Printf("[%s]%v\n", host, err)
            return output + o, false
        }
        output += o
        o, err = device.ExecCommandExpect("Y", ">", time.Second*time.Duration(args.upgradetime))
        if err != nil {
            log.Printf("[%s]%v\n", host, err)
            return output + o, false
        }
        output += o
    }

    return output, false
}

func install_huawei(host, port string, args Args, sshoptions nwssh.SSHOptions) (string, bool) {

    var devssh *nwssh.SSHBase
    var err error

    username := args.username
    password := args.password

    devssh, err = nwssh.SSH(host, port, username, password, time.Duration(10)*time.Second, sshoptions)
    defer devssh.Close()
    if err != nil {
        log.Printf("[%s]%v\n", host, err)
        return "", false
    }
    if err = devssh.Connect(); err != nil {
        log.Printf("[%s]%v\n", host, err)
        return "", false
    }

    var device nwssh.SSHBASE

    device = &nwssh.HuaweiSSH{devssh}

    if !device.SessionPreparation() {
        log.Printf("[%s]Failed init execute envirment. Try to exectue command directly.", host)
    }
    patches := strings.Split(args.patch, ";")

    output := ""
    for _, patch := range patches {
        cmd := `patch load flash:/` + patch + ` all run`

        o, err := device.ExecCommandExpect(cmd, ">", time.Second*time.Duration(args.upgradetime))
        if err != nil {
            log.Printf("[%s]%v\n", host, err)
            return output + o, false
        }

        if !strings.Contains(o, "Succeeded") {
            log.Printf("[%s]Failed install patch %s.\n", host, patch)
            return output + o, false
        }
        output += o
    }

    return output, false
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

func writefile(file, conntent string) error {
    return ioutil.WriteFile(file, []byte(conntent), 0666)
}

func main() {
    initflag()
    if args.help {
        fmt.Println("Usage of CLI:")
        flag.PrintDefaults()
        os.Exit(0)
    }

    if args.hostfile == "" {
        fmt.Println("Traget host is expected but got none. See help docs.")
        os.Exit(0)
    }

    if args.patch == "" {
        fmt.Println("Traget patch file is expected but got none. See help docs.")
        os.Exit(0)
    }

    if args.logdir == "" {
        fmt.Println("Output directory is expected but got none. See help docs.")
        os.Exit(0)
    }

    if args.swvendor == "" {
        fmt.Println("Switch vendor file is expected but got none. See help docs.")
        os.Exit(0)
    }

    if args.username == "" {
        fmt.Println("Username is expected but got none. See help docs.")
        os.Exit(0)
    }

    if args.password == "" {
        fmt.Printf("Please input the password:")
        fmt.Scanf("%s", &args.password)
    }

    sshoptions := nwssh.SSHOptions{
        IgnorHostKey: true,
        BannerCallback: func(msg string) error {
            return nil
        },
        TermType:     "vt100",
        TermHeight:   560,
        TermWidht:    480,
        ReadWaitTime: time.Duration(500) * time.Millisecond, //Read data from a ssh channel timeout
    }

    var err error

    hostfile := args.hostfile
    hosts, err := readlines(hostfile)
    if err != nil {
        log.Fatal("%v", err)
    }

    maxThread := 500
    threadchan := make(chan struct{}, maxThread)

    wait := sync.WaitGroup{}
    vendor := strings.ToLower(args.swvendor)
    for _, host := range hosts {
        wait.Add(1)
        go func(host string) {
            threadchan <- struct{}{}
            output := ""
            if vendor == "h3c" {
                output, _ = install_h3c(host, args.port, args, sshoptions)
            }else if vendor == "huawei" {
                output, _ = install_huawei(host, args.port, args, sshoptions)
            }else{
                fmt.Printf("unspported vendor %s.\n", vendor)
            }
            writefile(args.logdir+host, output)
            <-threadchan
            wait.Done()
        }(host)
    }
    wait.Wait()
}
