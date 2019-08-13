package main

import (
    "bufio"
    "encoding/json"
    "flag"
    "fmt"
    "io"
    "log"
    "net/http"
    "nwssh"
    "os"
    "regexp"
    "strings"
    "sync"
    "time"
)

func getVersionH3c(host, port string, sshoptions nwssh.SSHOptions, args Args) (string, string, string, bool) {

    var devssh *nwssh.SSHBase
    var err error

    username := args.username
    password := args.password

    devssh, err = nwssh.SSH(host, port, username, password, time.Duration(10)*time.Second, sshoptions)
    defer devssh.Close()
    if err != nil {
        log.Printf("[%s]%v\n", host, err)
        return "Failed", "Failed", "Failed", false
    }
    if err = devssh.Connect(); err != nil {
        log.Printf("[%s]%v\n", host, err)
        return "Failed", "Failed", "Failed", false
    }

    var device nwssh.SSHBASE

    device = &nwssh.H3cSSH{devssh}

    if !device.SessionPreparation() {
        log.Printf("[%s]Failed init execute envirment. Try to exectue command directly.", host)
    }

    version := "None"
    platform := "None"
    patch := "None"

    cmd := "display device"

    output, err := device.ExecCommandExpectPrompt(cmd, time.Second*5)
    if err != nil {
        log.Printf("[%s]%v\n", host, err)
        return "Failed", "Failed", "Failed", false
    }

    r := regexp.MustCompile(`Master\s{2,}\d\s{2,}(\S+\s?\S*)\s{2,}(\S+\s?\S*)\s+`)
    matched := r.FindStringSubmatch(output)
    if matched == nil {
        return "Failed", "Failed", "Failed", false
    }

    version = matched[1]
    patch = matched[2]

    cmd = "display version | in uptime"
    output, err = device.ExecCommandExpectPrompt(cmd, time.Second*5)
    if err != nil {
        log.Printf("[%s]%v\n", host, err)
        return "Failed", version, patch, false
    }

    r = regexp.MustCompile(`H3C (.+) uptime`)
    matched = r.FindStringSubmatch(output)
    if matched == nil {
        return "Failed", version, patch, false
    }
    platform = matched[1]
    return platform, version, patch, true
}

func getVersionHuawei(host, port string, sshoptions nwssh.SSHOptions, args Args) (string, string, string, bool) {

    var devssh *nwssh.SSHBase
    var err error

    username := args.username
    password := args.password

    devssh, err = nwssh.SSH(host, port, username, password, time.Duration(10)*time.Second, sshoptions)
    defer devssh.Close()
    if err != nil {
        log.Printf("[%s]%v\n", host, err)
        return "Failed", "Failed", "Failed", false
    }
    if err = devssh.Connect(); err != nil {
        log.Printf("[%s]%v\n", host, err)
        return "Failed", "Failed", "Failed", false
    }

    var device nwssh.SSHBASE

    device = &nwssh.HuaweiSSH{devssh}

    if !device.SessionPreparation() {
        log.Printf("[%s]Failed init execute envirment. Try to exectue command directly.", host)
    }

    cmd := "display version"

    output, err := device.ExecCommandExpectPrompt(cmd, time.Second*5)
    if err != nil {
        log.Printf("[%s]%v\n", host, err)
        return "Failed", "Failed", "Failed", false
    }

    version := "None"
    platform := "None"
    patch := "None"

    ver_r := regexp.MustCompile(`VRP.+ (V\d+R\d+.+)\)`)
    ver_matched := ver_r.FindStringSubmatch(output)

    if ver_matched != nil {
        version = ver_matched[1]
    }

    plat_r := regexp.MustCompile(`HUAWEI (.+) uptime`)
    plat_matched := plat_r.FindStringSubmatch(output)

    if plat_matched != nil {
        platform = plat_matched[1]
    }

    patch_r := regexp.MustCompile(`Patch Version: (.+)`)
    patch_matched := patch_r.FindStringSubmatch(output)

    if patch_matched != nil {
        patch = patch_matched[1]
    }

    return platform, version, patch, true
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

/*
* 用于解析API数据的结构体
 */

type RespBody struct {
    Code    int64
    Data    DataBlock
    Message string
}

type DataBlock struct {
    List       []ListBlock
    TotalCount float64
}

type ListBlock struct {
    ID                  string      `json:"id"`
    SID                 string      `json:"sid"`
    Name                string      `json:"name"`
    Describe            string      `json:"describe"`
    Type                string      `json:"type"`
    SN                  string      `json:"sn"`
    AssetId             string      `json:"asset_id"`
    Role                string      `json:"role"`
    StackRole           string      `json:"stack_role"`
    MemberId            int64       `json:"member_id"`
    ServiceStatus       string      `json:"service_status"`
    State               string      `json:"state"`
    RaState             string      `json:"ra_state"`
    MonitorState        string      `json:"monitor_state"`
    Constructed         int64       `json:"constructed"`
    Business            string      `json:"business"`
    Service             string      `json:"service"`
    ManagementIpId      string      `json:"management_ip_id"`
    ManagementIp        string      `json:"management_ip"`
    OutofbandIpId       string      `json:"outofband_ip_id"`
    OutofbandIp         string      `json:"outofband_ip"`
    SoftVersion         string      `json:"soft_version"`
    PatchVersion        string      `json:"patch_version"`
    Manufacturer        string      `json:"manufacturer"`
    Brand               string      `json:"brand"`
    Model               string      `json:"model"`
    EndofLife           string      `json:"end_of_life"`
    DatacenterId        string      `json:"datacenter_id"`
    DatacenterName      string      `json:"datacenter_name"`
    DatacenterShortName string      `json:"datacenter_short_name"`
    RoomID              string      `json:"room_id"`
    RoomName            string      `json:"room_name"`
    RackId              string      `json:"rack_id"`
    RackName            string      `json:"rack_name"`
    PodId               string      `json:"pod_id"`
    PodName             string      `json:"pod_name"`
    PodMode             string      `json:"pod_mode"`
    StackId             string      `json:"stack_id"`
    Extra               interface{} `json:"extra"`
    CreatedTime         string      `json:"created_time"`
    UpdatedTime         string      `json:"updated_time"`
}

/*
*  抓取所有NetNode节点
 */

type NetNode struct {
    Mgt    string
    Vendor string
}

const MaxNetNodes = 1000

func GetNetNode(url string) ([]*NetNode, error) {
    /*
    * url is the NetNode api.
     */

    var nodes = make([]*NetNode, 0, MaxNetNodes)
    resp, err := http.Get(url)
    if err != nil {
        return nil, err
    }

    defer resp.Body.Close()
    var r = &RespBody{}
    if err := json.NewDecoder(resp.Body).Decode(r); err != nil {
        return nil, err
    }
    if r.Code != 2000 {
        return nil, fmt.Errorf("Failed retrive data for Code is %d", r.Code)
    }

    if r.Message != "OK" {
        return nil, fmt.Errorf("Failed retrive data for Message is %s", r.Message)
    }
    for _, node := range r.Data.List {

        switch node.Manufacturer {
        case "H3C":
        case "Huawei":
        default:
            continue
        }
        var mgt string
        if node.ManagementIp == "" {
            if node.OutofbandIp != "" {
                mgt = node.OutofbandIp
            } else {
                continue
            }
        } else {
            mgt = node.ManagementIp
        }

        nodes = append(nodes, &NetNode{
            Mgt:    mgt,
            Vendor: strings.ToLower(node.Manufacturer),
        })
    }
    return nodes, nil
}

type Args struct {
    hostfile string
    username string
    password string
    output   string
    vendor   string
    apiurl   string
}

var args = Args{}

func initflag() {
    flag.StringVar(&args.hostfile, "f", "", `Target hosts list file, each ip on a separate line, for example:
'10.10.10.10'
'12.12.12.12'.`)
    flag.StringVar(&args.username, "u", "", "Username for login.")
    flag.StringVar(&args.password, "p", "", "Password for login.")
    flag.StringVar(&args.output, "o", "", "Output filename.")
    flag.StringVar(&args.vendor, "V", "", "Device vendor.")
    flag.StringVar(&args.apiurl, "url", "http://api.joybase.jd.com/network_devices", "Device information api.")

    flag.Parse()
}

func main() {

    initflag()

    if args.username == "" {
        fmt.Println("Username is expected but got none. See help docs.")
        os.Exit(0)
    }

    if args.password == "" {
        fmt.Printf("Please input the password:")
        fmt.Scanf("%s", &args.password)
    }

    var err error
    var nodes = make([]*NetNode, 0)
    if args.hostfile != "" {
        hostfile := args.hostfile
        hosts, err := readlines(hostfile)
        if err != nil {
            log.Fatal("%v", err)
        }

        if args.vendor == "" {
            log.Fatal("Device vendor expcected.")
        }
        vendor := strings.ToLower(args.vendor)
        for _, host := range hosts {
            nodes = append(nodes, &NetNode{host, vendor})
        }

    } else {
        nodes, err = GetNetNode(args.apiurl)
        if err != nil {
            log.Fatal("%v", err)
        }
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

    maxThread := 500
    threadchan := make(chan struct{}, maxThread)

    wait := sync.WaitGroup{}
    if args.output != "" {
        outputchan := make(chan string, len(nodes))
        for _, node := range nodes {
            wait.Add(1)
            go func(netnode *NetNode) {
                threadchan <- struct{}{}
                switch netnode.Vendor {
                case "h3c":
                    {
                        plat, ver, patch, _ := getVersionH3c(netnode.Mgt, "22", sshoptions, args)
                        o := fmt.Sprintf("%s\t%s\t%s\t%s\n", netnode.Mgt, plat, ver, patch)
                        fmt.Printf(o)
                        outputchan <- o
                    }
                case "huawei":
                    {
                        plat, ver, patch, _ := getVersionHuawei(netnode.Mgt, "22", sshoptions, args)
                        o := fmt.Sprintf("%s\t%s\t%s\t%s\n", netnode.Mgt, plat, ver, patch)
                        fmt.Printf(o)
                        outputchan <- o
                    }
                default:
                    log.Printf("Unsupported vendor %s.\n", netnode.Vendor)
                }
                <-threadchan
                wait.Done()
            }(node)
        }
        wait.Wait()
        close(outputchan)
        f, err := os.OpenFile(args.output, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 666)
        if err != nil {
            log.Fatal("%v", err)
        }
        /*for {
            select {
            case c := <-outputchan:
                {
                    _c := []byte(c)
                    n, err := f.Write(_c)
                    if err == nil && n < len(_c) {
                        err = io.ErrShortWrite
                    }
                fmt.Println("Got")
                }
            default:
                fmt.Println("LooP")
                break

            }
        } */
        for c := range outputchan{
            _c := []byte(c)
            n, err := f.Write(_c)
            if err == nil && n < len(_c) {
                err = io.ErrShortWrite
            }
        }

        if err1 := f.Close(); err == nil {
            err = err1
        }
        if err != nil{
        log.Fatal(err)
    }

    } else {
        for _, node := range nodes {
            wait.Add(1)
            go func(netnode *NetNode) {
                threadchan <- struct{}{}
                switch netnode.Vendor {
                case "h3c":
                    {
                        plat, ver, patch, _ := getVersionH3c(netnode.Mgt, "22", sshoptions, args)
                        fmt.Printf("%s\t%s\t%s\t%s\n", netnode.Mgt, plat, ver, patch)
                    }
                case "huawei":
                    {
                        plat, ver, patch, _ := getVersionHuawei(netnode.Mgt, "22", sshoptions, args)
                        fmt.Printf("%s\t%s\t%s\t%s\n", netnode.Mgt, plat, ver, patch)
                    }
                default:
                    log.Printf("Unsupported vendor %s.\n", netnode.Vendor)
                }
                <-threadchan
                wait.Done()
            }(node)
        }
        wait.Wait()
    }

}
