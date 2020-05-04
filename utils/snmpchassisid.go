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
	"sync"
	"time"
)

const (
	lldpLocChassisID      = "1.0.8802.1.1.2.1.3.2.0"
	lldpLocChassisIDNexus = "1.3.6.1.2.1.2.2.1.6"
)

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

	_, err = db.Exec(`DROP TABLE IF EXISTS lldpdevices`)
	if err != nil {
		fmt.Printf("%v", err)
		os.Exit(1)
	}

	create_table := `CREATE TABLE lldpdevices (devmgt TEXT, chassisid TEXT, role TEXT);`
	_, err = db.Exec(create_table)
	if err != nil {
		fmt.Printf("%v", err)
		os.Exit(1)
	}

	//Scan All hosts that to be retrieved.
	rows, err := db.Query(`SELECT DISTINCT devmgt, brand, role FROM devices WHERE role="T1" and devmgt !=""`)
	if err != nil {
		fmt.Printf("%v\n", err)
		os.Exit(1)
	}
	defer rows.Close()


	type sw struct {
		mgt string;
		brand string;
		role string;
	}

	hosts := make([]sw, 10, 10)

	var devmgt sql.NullString
	var brand sql.NullString
	var role sql.NullString

	for rows.Next() {
		err = rows.Scan(&devmgt, &brand)
		if err != nil {
			fmt.Printf("%v", err)
			continue
		}
		if !devmgt.Valid {
			continue
		}else if !brand.Valid{
			brand.String = ""
		}else if !role.Valid{
			role.String = ""
		}

		hosts = append(hosts, sw{devmgt.String, brand.String, role.String})
	}

	//Init go concorote
	maxThread := 500
	threadchan := make(chan struct{}, maxThread)
	wait := sync.WaitGroup{}

	for _, host := range hosts {
		wait.Add(1)
		go func(target sw) {
			threadchan <- struct{}{}
			err = RetrieveChassisID(target.mgt, target.brand, target.role, community, db)
			if err != nil {
				fmt.Print(err)
			}
			<-threadchan
			wait.Done()
		}(host)
	}

	wait.Wait()
}

func BulkRetrieveNexusChassisID(snmpd *gosnmp.GoSNMP, oid string) (reslut map[string]string, err error) {
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
			reslut[index] = hex.EncodeToString(pdu.Value.([]byte))

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

func WriteMsg(db *sql.Tx, target, chassisid, role string) error {
	//Table name lldpneighbour
	//fmt.Println(target, localid, localport, remoteid, remoteport)
	_, err := db.Exec(`INSERT INTO lldpdevices (devmgt, chassisid, role) VALUES (?, ?, ?)`, target, chassisid, role)
	return err
}

func RetrieveChassisID(target, brand, role, community string, db *sql.DB) error {
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

	if strings.Contains(brand, "Nexus"){
		chassisid, err := BulkRetrieveNexusChassisID(conn, lldpLocChassisIDNexus)
		if err != nil {
			return fmt.Errorf("[%s]Failed Retrieved local chassis id.\n", target)
		}


		tx, err := db.Begin()
		if err != nil{
			return nil
		}

		for _, cid := range chassisid {
			err = WriteMsg(tx, target, cid, role)
			if err != nil {
				tx.Rollback()
				return fmt.Errorf("[%s]%v. Rollbacked.\n", target, err)
			}
		}
		tx.Commit()
		return nil

	} else{

		chassisid, err := GetRetrieveLocalChassisID(conn, lldpLocChassisID)
		if err != nil {
			return fmt.Errorf("[%s]Failed Retrieved local chassis id.\n", target)
		}

		tx, err := db.Begin()
		if err != nil{
			return nil
		}
		err = WriteMsg(tx, target, chassisid, role)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("[%s]%v. Rollbacked.\n", target, err)
		}
		tx.Commit()
		return nil
	}

}
