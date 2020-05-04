package ping

import (
	"fmt"
	"testing"
)


func TestUtilConverInt(t *testing.T){
	buf := []byte{0x00, 0x001, 0x002, 0x003}
	//a := binary.ByteOrder.Uint16(buf)
	a := nativeEndian.Uint16(buf)
	fmt.Println("Bytes: ", buf, a)
}