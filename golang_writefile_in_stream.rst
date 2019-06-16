流式小批量写文件方式
===========================

::

	func writefilestream(filename string, content chan *[]byte, perm os.FileMode) error{
		f, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, perm)
		if err != nil {
			return err
		}
		for c := range content{
			n, err := f.Write(*c)
			if err == nil && n < len(*c) {
				err = io.ErrShortWrite
			}
		}

		if err1 := f.Close(); err == nil {
			err = err1
		}
		return err
	}


流式读取文件方式
===========================

::

	const (
		BUFSIZE = 16
	)

	func fileReaderGen(filename string) chan []byte {
		fileReadChan := make(chan []byte, BUFSIZE)
		go func() {
			file, err := os.Open(filename)
			if err != nil {
				log.Fatal(err)
			} else {
				scan := bufio.NewScanner(file)
				for scan.Scan() {
					// Write to the channel we will return
					// We additionally have to copy the content
					// of the slice returned by scan.Bytes() into
					// a new slice (using append()) before sending
					// it to another go-routine since scan.Bytes()
					// will re-use the slice it returned for
					// subsequent scans, which will garble up data
					// later if we don't put the content in a new one.
					fileReadChan <- append([]byte(nil), scan.Bytes()...)
				}
				if scan.Err() != nil {
					log.Fatal(scan.Err())
				}
				close(fileReadChan)
				fmt.Println("Closed file reader channel")
			}
			file.Close()
		}()
	return fileReadChan
}
