package nqas

import (
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

func initLogger(logFile, prefix string) (*log.Logger, error, func() error){
	fd, err := openLogFile(logFile)
	if err != nil{
		return nil, err, nil
	}
	return log.New(fd, prefix, log.LstdFlags), nil, fd.Close
}

func convertStringToMap(s string) (map[string]string, error){
	// Format of s is 'BJ03:BJ04;BJ04:BJ03'
	m := make(map[string]string)
	ss := strings.Split(s, ";")
	for _, section := range ss{
		if section == ""{
			continue
		}

		key_val := strings.Split(section, ":")
		if len(key_val) != 2{
			return nil, fmt.Errorf("Unavalible content about '%s'. Only one ':' is needed.", section)
		}

		key := strings.TrimSpace(key_val[0])
		m[key] = strings.TrimSpace(key_val[1])
	}

	return m, nil
}
