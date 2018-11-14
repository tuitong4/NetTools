package main

import (
	"database/sql"
	"encoding/hex"
	"flag"
	"fmt"
	_ "golang.org/x/go-sqlite3"
	"golang.org/x/gosnmp"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	lldpRemChassisID = "1.0.8802.1.1.2.1.4.1.1.5"
	lldpRemPortID    = "1.0.8802.1.1.2.1.4.1.1.7"
	lldpLocChassisID = "1.0.8802.1.1.2.1.3.2.0"
	lldpLocPortID    = "1.0.8802.1.1.2.1.3.7.1.3"
)

func main() {
	flag.Usage = func() {
		fmt.Printf("Usage:\n")
		fmt.Printf("   %s [-community=<community>] host [oid]\n", filepath.Base(os.Args[0]))
		//fmt.Printf("     host      - the host to walk/scan\n")
		//fmt.Printf("     oid       - the MIB/Oid defining a subtree of values\n\n")
		flag.PrintDefaults()
	}

	var community string
	flag.StringVar(&community, "community", "public", "the community string for device")

	flag.Parse()

	if len(flag.Args()) < 1 {
		flag.Usage()
		os.Exit(1)
	}
	target := flag.Args()[0]
	//var oid string
	//if len(flag.Args()) > 1 {
	//  oid = flag.Args()[1]
	//}

	conn := &gosnmp.GoSNMP{
		Target:             target,
		Port:               uint16(161),
		Community:          community,
		Version:            gosnmp.Version2c,
		ExponentialTimeout: false,
		Retries:            5,
		Timeout:            time.Duration(3) * time.Second,
		MaxRepetitions:     10,
		//Logger:    log.New(os.Stdout, "", 0),
	}

	err := conn.Connect()
	if err != nil {
		fmt.Printf("ConnectFailed")
		os.Exit(1)
	}
	defer conn.Conn.Close()

	remoteid, err := BulkRetrieveRemChassisID(conn, lldpRemChassisID)
	if err != nil {
		fmt.Printf("Failed Retrieved remote chassis id.")
		os.Exit(1)
	}

	remoteport, err := BulkRetrievePortID(conn, lldpRemPortID)
	if err != nil {
		fmt.Printf("Failed Retrieved remote port id.")
		os.Exit(1)
	}

	localport, err := BulkRetrievePortID(conn, lldpLocPortID)
	if err != nil {
		fmt.Printf("Failed Retrieved local port id.")
		os.Exit(1)
	}

	localid, err := GetRetrieveLocalChassisID(conn, lldpLocChassisID)
	if err != nil {
		fmt.Printf("Failed Retrieved local chassis id.")
		os.Exit(1)
	}

	db, err := sql.Open("sqlite3", "./lldp.db")
	createtablesql := `CREATE TABLE lldpneighbour ("devmgt" TEXT NULL, "localid" TEXT NULL, "localport" TEXT NULL, "remoteid" TEXT NULL, "remoteport" TEXT NULL)`
	_, err = db.Query(createtablesql)
	if err != nil {
		fmt.Printf("%v", err)
		os.Exit(1)
	}
	lldpneighbour
	for idx, remid := range remoteid {
		err = WriteLLDPMsg(db, target, localid, localport[idx], remid, remoteport[idx])
		if err != nil {
			fmt.Printf("%v.", err)
			os.Exit(1)
		}
		//fmt.Printf("%s\t%s\t%s\t%s\t%v\n", target, localid, localport[idx], remid, remoteport[idx])
	}

}

func printValue(pdu gosnmp.SnmpPDU) error {
	fmt.Printf("%s = ", pdu.Name)

	switch pdu.Type {
	case gosnmp.OctetString:
		b := pdu.Value.([]byte)
		fmt.Printf("STRING: %x\n", string(b))

	default:
		fmt.Printf("TYPE %d: %d\n", pdu.Type, gosnmp.ToBigInt(pdu.Value))
	}
	return nil
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
			reslut[index] = hex.EncodeToString(pdu.Value.([]byte))

		default:
			reslut[index] = ""
		}
	}
	return reslut, err
}

func BulkRetrievePortID(snmpd *gosnmp.GoSNMP, oid string) (reslut map[string]string, err error) {
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

func GetRetrieveLocalChassisID(snmpd *gosnmp.GoSNMP, oid string) (reslut string, err error) {
	reslut = ""
	err = nil
	_resp, err := snmpd.Get([]string{oid})
	if err != nil {
		err = fmt.Errorf("Walk Error: %v\n", err)
		return reslut, err
	}

	resp := _resp.Variables[0]

	switch resp.Type {
	case gosnmp.OctetString:
		reslut = hex.EncodeToString(resp.Value.([]byte))

	default:
		reslut = ""
	}
	return reslut, err
}

func WriteLLDPMsg(db *sql.DB, target, localid, localport, remoteid, remoteport string) error {
	//Table name lldpneighbour
	stmt, err := db.Prepare(`INSERT INTO lldpneighbour VALUES (?, ?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	_, err = stmt.Exec(target, localid, localport, remoteid, remoteport)

	if err != nil {
		return err
	}
	return nil
}
