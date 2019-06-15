流式小批量写文件方式
===========================

adsad ::

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
