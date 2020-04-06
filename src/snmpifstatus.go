package main

import (
	"database/sql"
	"flag"
	"fmt"
	_ "golang.org/x/go-sqlite3"
	"golang.org/x/gosnmp"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const (
	ifOperStatus_prefix = "1.3.6.1.2.1.2.2.1.8."
)

func main() {
	flag.Usage = func() {
		fmt.Printf("Usage:\n")
		fmt.Printf("   %s [-community=<community>] -database=<database>\n", filepath.Base(os.Args[0]))
		flag.PrintDefaults()
	}

	var database string
	var community string

	flag.StringVar(&database, "database", "", "the sqlite3 db file(with filepath).")
	flag.StringVar(&community, "community", "360buy", "the community string for device")

	flag.Parse()

	db, err := sql.Open("sqlite3", database)

	if err != nil {
		fmt.Printf("%v", err)
		os.Exit(1)
	}
	db.SetMaxOpenConns(1)
	defer db.Close()

	_, err = db.Exec(`DROP TABLE IF EXISTS failoverport`)
	if err != nil {
		fmt.Printf("%v", err)
		os.Exit(1)
	}

	create_table := `CREATE TABLE failoverport (code TEXT, devmgt TEXT, localport TEXT, localportidx TEXT, remotedev TEXT, remoteport TEXT, toalport INTEGER);`
	_, err = db.Exec(create_table)
	if err != nil {
		fmt.Printf("%v", err)
		os.Exit(1)
	}

	//Scan All hosts that to be retrieved.
	rows, err := db.Query(`SELECT DISTINCT devmgt FROM uplinks limit 10`)
	if err != nil {
		fmt.Printf("%v\n", err)
		os.Exit(1)
	}
	defer rows.Close()

	hosts := make([]string, 10)

	var devmgt sql.NullString

	for rows.Next() {
		err = rows.Scan(&devmgt)
		if err != nil {
			fmt.Printf("%v", err)
			continue
		}
		if !devmgt.Valid {
			continue
		} else {
			fmt.Println(devmgt)
			hosts = append(hosts, devmgt.String)
		}
	}

	//Init go concorote
	maxThread := 500
	threadchan := make(chan struct{}, maxThread)
	wait := sync.WaitGroup{}
	//fmt.Println(hosts)
	for _, host := range hosts {
		wait.Add(1)
		go func(target string) {
			threadchan <- struct{}{}
			err = RetrievePortStat(target, community, db)
			if err != nil {
				fmt.Print(err)
			}
			<-threadchan
			wait.Done()
		}(host)
	}

	wait.Wait()
}

func GetRetrievePortStat(snmpd *gosnmp.GoSNMP, oid string) (reslut string, err error) {
	reslut = ""
	err = nil
	_resp, err := snmpd.Get([]string{oid})
	if err != nil {
		err = fmt.Errorf("Get Error: %v\n", err)
		return reslut, err
	}

	resp := _resp.Variables[0]

	switch resp.Type {
	case gosnmp.Integer:
		if resp.Value != 1 {
			reslut = "down"
		} else {
			reslut = "up"
		}

	default:
		reslut = "unkown"
	}
	return reslut, err
}

func WriteLinkMsg(db *sql.Tx, code, target, localportidx, localport, remoteid, remoteport string, toalport int) error {

	_, err := db.Exec(`INSERT INTO failoverport (code, devmgt, localport, localportidx, remotedev, remoteport, toalport) VALUES (?, ?, ?, ?, ?), ?`, code, target, localport, localportidx, remoteid, remoteport, toalport)
	return err
}

type uplink struct {
	code         string
	devmgt       string
	localport    string
	localportidx string
	remotedev    string
	remoteport   string
}

func RetrievePortStat(target, community string, db *sql.DB) error {
	conn := &gosnmp.GoSNMP{
		Target:         target,
		Port:           uint16(161),
		Community:      community,
		Version:        gosnmp.Version2c,
		Retries:        1,
		Timeout:        time.Duration(3) * time.Second,
		MaxRepetitions: 10,
		//Logger:    log.New(os.Stdout, "", 0),
	}
	err := conn.Connect()
	if err != nil {
		return fmt.Errorf("[%s]ConnectFailed.\n", target)
	}
	defer conn.Conn.Close()

	sql := `SELECT * FROM uplinks WHERE devmgt = "` + target + `"`

	rows, err := db.Query(sql)
	if err != nil {
		return fmt.Errorf("[%s]%v.\n", err)
	}
	defer rows.Close()

	links := make([]uplink, 1, 1)
	for rows.Next() {
		item := new(uplink)
		err = rows.Scan(&item.code, &item.devmgt, &item.localport, &item.localportidx, &item.remotedev, &item.remoteport)
		if err != nil {
			return fmt.Errorf("[%s]%v.\n", err)
		}

		links = append(links, *item)
	}
	totallink := len(links)

	faillinks := make([]uplink, 1, 1)
	for _, link := range links {
		//fmt.Println(ifOperStatus_prefix + link.localportidx + link.devmgt)
		stat, err := GetRetrievePortStat(conn, ifOperStatus_prefix + link.localportidx)
		if err != nil {
			return fmt.Errorf("[%s]Failed Retrieved remote chassis id.\n", target)
		}
		if stat == "down" {
			faillinks = append(faillinks, link)
		}
	}

	if len(faillinks) == 0 {
		return nil
	}

	//tx, err := db.Begin()
	//if err != nil {
	//  return nil
	//}

	for _, faillink := range faillinks {

		fmt.Println(faillink.devmgt, totallink)
		/*
			err = WriteLinkMsg(tx, faillink.code, faillink.devmgt, faillink.localport, faillink.localportidx, faillink.remotedev, faillink.remoteport, totallink)
			if err != nil {
				tx.Rollback()
				return fmt.Errorf("[%s]%v. Rollbacked.\n", target, err)
			}*/
	}
	//tx.Commit()
	return nil
}