package main

import (
	"crypto/md5"
	"database/sql"
	"encoding/hex"
	"flag"
	"fmt"
	_ "golang.org/x/go-sqlite3"
	"golang.org/x/gosnmp"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)

const (
	lldpRemChassisID = "1.0.8802.1.1.2.1.4.1.1.5"
	lldpRemPortID    = "1.0.8802.1.1.2.1.4.1.1.7"
	ifDescr          = "1.3.6.1.2.1.2.2.1.2"
	lldpLocPortID    = "1.0.8802.1.1.2.1.3.7.1.3"
)

var unviewchar = regexp.MustCompile(`\s`)

func main() {
	flag.Usage = func() {
		fmt.Printf("Usage:\n")
		fmt.Printf("   %s [-community=<community>] -database=<database>\n", filepath.Base(os.Args[0]))
		//fmt.Printf("           - the host to walk/scan\n")
		//fmt.Printf("     oid       - the MIB/Oid defining a subtree of values\n\n")
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

	_, err = db.Exec(`DROP TABLE IF EXISTS lldpneighbour`)
	if err != nil {
		fmt.Printf("%v", err)
		os.Exit(1)
	}

	create_table := `CREATE TABLE lldpneighbour (code TEXT, devmgt TEXT, localportidx TEXT, localport TEXT, remoteid TEXT, remoteport TEXT);`
	_, err = db.Exec(create_table)
	if err != nil {
		fmt.Printf("%v", err)
		os.Exit(1)
	}

	//Scan All hosts that to be retrieved.
	//rows, err := db.Query(`SELECT DISTINCT devmgt  FROM devices WHERE role="T0" and devmgt !="" and service NOT LIKE "%ACS%"`)
	rows, err := db.Query(`SELECT DISTINCT devmgt  FROM devices WHERE devmgt = "172.19.1.234" or devmgt = "172.19.1.68"`)
	if err != nil {
		fmt.Printf("%v\n", err)
		os.Exit(1)
	}
	defer rows.Close()

	var devmgt sql.NullString

	hosts := []string{}
	for rows.Next() {
		err = rows.Scan(&devmgt)
		if err != nil {
			fmt.Printf("%v", err)
			continue
		}

		if !devmgt.Valid {
			continue
		} else {
			hosts = append(hosts, devmgt.String)
		}
	}

	//Init go concorote
	maxThread := 500
	threadchan := make(chan struct{}, maxThread)
	wait := sync.WaitGroup{}

	for _, host := range hosts {
		wait.Add(1)
		go func(target string) {
			threadchan <- struct{}{}
			err = RetrieveLLDP(target, community, db)
			if err != nil {
				fmt.Print(err)
			}
			<-threadchan
			wait.Done()
		}(host)
	}

	wait.Wait()
}

func BulkRetrieveRemChassisID(snmpd *gosnmp.GoSNMP, oid string) (reslut map[string]string, err error) {
	reslut = make(map[string]string)
	err = nil
	resp, err := snmpd.BulkWalkAll(oid)
	if err != nil {
		err = fmt.Errorf("Walk Error: %v\n", err)
		return reslut, err
	}

	for _, pdu := range resp {
		parts := strings.Split(pdu.Name, ".")
		index := parts[len(parts)-2]

		switch pdu.Type {
		case gosnmp.OctetString:
			h := hex.EncodeToString(pdu.Value.([]byte))
			if len(h) == 12 {
				reslut[index] = h
			}
		default:
			reslut[index] = ""
		}
	}
	return reslut, err
}

func BulkRetrieveLocalPortID(snmpd *gosnmp.GoSNMP, oid string) (reslut map[string]string, err error) {
	reslut = make(map[string]string)
	err = nil

	resp, err := snmpd.BulkWalkAll(oid)
	if err != nil {
		err = fmt.Errorf("Walk Error: %v\n", err)
		return reslut, err
	}

	for _, pdu := range resp {
		parts := strings.Split(pdu.Name, ".")
		index := parts[len(parts)-1]

		switch pdu.Type {
		case gosnmp.OctetString:
			reslut[index] = string(pdu.Value.([]byte))

		default:
			reslut[index] = ""
		}
	}
	return reslut, err
}

func BulkRetrieveRemPortID(snmpd *gosnmp.GoSNMP, oid string) (reslut map[string]string, err error) {
	reslut = make(map[string]string)
	err = nil

	resp, err := snmpd.BulkWalkAll(oid)
	if err != nil {
		err = fmt.Errorf("Walk Error: %v\n", err)
		return reslut, err
	}

	for _, pdu := range resp {
		parts := strings.Split(pdu.Name, ".")
		index := parts[len(parts)-2]

		switch pdu.Type {
		case gosnmp.OctetString:
			reslut[index] = string(pdu.Value.([]byte))

		default:
			reslut[index] = ""
		}
	}
	return reslut, err
}

func BulkRetrievePortDescr(snmpd *gosnmp.GoSNMP, oid string) (reslut map[string]string, err error) {
	reslut = make(map[string]string)
	err = nil

	resp, err := snmpd.BulkWalkAll(oid)
	if err != nil {
		err = fmt.Errorf("Walk Error: %v\n", err)
		return reslut, err
	}

	for _, pdu := range resp {
		parts := strings.Split(pdu.Name, ".")
		index := parts[len(parts)-1]

		switch pdu.Type {
		case gosnmp.OctetString:
			v := string(pdu.Value.([]byte))
			reslut[v] = index

		default:
			continue
		}
	}
	return reslut, err
}

func WriteLLDPMsg(db *sql.Tx, target, localportidx, localport, remoteid, remoteport string) error {
	//Table name lldpneighbour
	md5 := md5.New()
	md5.Write([]byte(target + localportidx + localport + remoteid + remoteport))
	encode_s := hex.EncodeToString(md5.Sum(nil))
	_, err := db.Exec(`INSERT INTO lldpneighbour (code, devmgt, localportidx, localport, remoteid, remoteport) VALUES (?, ?, ?, ?, ?, ?)`, encode_s, target, localportidx, localport, remoteid, remoteport)
	return err
}

func RetrieveLLDP(target, community string, db *sql.DB) error {
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

	remoteid, err := BulkRetrieveRemChassisID(conn, lldpRemChassisID)
	if err != nil {
		return fmt.Errorf("[%s]Failed Retrieved remote chassis id.\n", target)
	}

	remoteport, err := BulkRetrieveRemPortID(conn, lldpRemPortID)
	if err != nil {
		return fmt.Errorf("[%s]Failed Retrieved remote port id.\n", target)
	}

	localport, err := BulkRetrieveLocalPortID(conn, lldpLocPortID)
	if err != nil {
		return fmt.Errorf("[%s]Failed Retrieved local port id.\n", target)
	}

	localportdescr, err := BulkRetrievePortDescr(conn, ifDescr)
	if err != nil {
		return fmt.Errorf("[%s]Failed Retrieved port descr.\n", target)
	}

	tx, err := db.Begin()
	if err != nil {
		return nil
	}

	for idx, remid := range remoteid {
		rport := remoteport[idx]
		if unviewchar.MatchString(rport) {
			fmt.Println(rport)
			continue
		}
		err = WriteLLDPMsg(tx, target, localportdescr[rport], localport[idx], remid, rport)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("[%s]%v. Rollbacked.\n", target, err)
		}
	}
	tx.Commit()
	return nil
}
