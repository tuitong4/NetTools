package ping

import (
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"runtime"
	"syscall"
	"time"
	"unsafe"
)

const (
	Version            = 4  // protocol version
	HeaderLen          = 20 // header length without extension headers
	maxHeaderLen       = 60 // sensible default, revisit if later RFCs define new usage of version and header length fields
	ICMPv4EchoRequest  = 8
	ICMPv4EchoReply    = 0
	ICMPv4TimeExceeded = 11
	ICMPv6EchoRequest  = 128
	ICMPv6EchoReply    = 129
	ICMPv6TimeExceeded = 3
)

type HeaderFlags int

const (
	MoreFragments HeaderFlags = 1 << iota // more fragments flag
	DontFragment                          // don't fragment flag
)

var (
	errMissingAddress           = errors.New("missing address")
	errMissingHeader            = errors.New("missing header")
	errHeaderTooShort           = errors.New("header too short")
	errBufferTooShort           = errors.New("buffer too short")
	errInvalidConnType          = errors.New("invalid conn type")
	errOpNoSupport              = errors.New("operation not supported")
	errNoSuchInterface          = errors.New("no such interface")
	errNoSuchMulticastInterface = errors.New("no such multicast interface")
	freebsdVersion              uint32

	nativeEndian binary.ByteOrder
)

// init parameter 'nativeEndian' base on your arch.
func init() {
	i := uint32(1)
	b := (*[4]byte)(unsafe.Pointer(&i))
	if b[0] == 1 {
		nativeEndian = binary.LittleEndian
	} else {
		nativeEndian = binary.BigEndian
	}
}


// A Header represents an IPv4 header.
type Header struct {
	Version  int         // protocol version
	Len      int         // header length
	TOS      int         // type-of-service
	TotalLen int         // packet total length
	ID       int         // identification
	Flags    HeaderFlags // flags
	FragOff  int         // fragment offset
	TTL      int         // time-to-live
	Protocol int         // next protocol
	Checksum int         // checksum
	Src      net.IP      // source address
	Dst      net.IP      // destination address
	Options  []byte      // options, extension headers
}

func (h *Header) String() string {
	if h == nil {
		return "<nil>"
	}
	return fmt.Sprintf("ver=%d hdrlen=%d tos=%#x totallen=%d id=%#x flags=%#x fragoff=%#x ttl=%d proto=%d cksum=%#x src=%v dst=%v", h.Version, h.Len, h.TOS, h.TotalLen, h.ID, h.Flags, h.FragOff, h.TTL, h.Protocol, h.Checksum, h.Src, h.Dst)
}

// Marshal returns the binary encoding of the IPv4 header h.
func (h *Header) Marshal() ([]byte, error) {
	if h == nil {
		return nil, syscall.EINVAL
	}
	if h.Len < HeaderLen {
		return nil, errHeaderTooShort
	}
	hdrlen := HeaderLen + len(h.Options)
	b := make([]byte, hdrlen)
	b[0] = byte(Version<<4 | (hdrlen >> 2 & 0x0f))
	b[1] = byte(h.TOS)
	flagsAndFragOff := (h.FragOff & 0x1fff) | int(h.Flags<<13)
	switch runtime.GOOS {
	case "darwin", "dragonfly", "netbsd":
		nativeEndian.PutUint16(b[2:4], uint16(h.TotalLen))
		nativeEndian.PutUint16(b[6:8], uint16(flagsAndFragOff))
	case "freebsd":
		if freebsdVersion < 1100000 {
			nativeEndian.PutUint16(b[2:4], uint16(h.TotalLen))
			nativeEndian.PutUint16(b[6:8], uint16(flagsAndFragOff))
		} else {
			binary.BigEndian.PutUint16(b[2:4], uint16(h.TotalLen))
			binary.BigEndian.PutUint16(b[6:8], uint16(flagsAndFragOff))
		}
	default:
		binary.BigEndian.PutUint16(b[2:4], uint16(h.TotalLen))
		binary.BigEndian.PutUint16(b[6:8], uint16(flagsAndFragOff))
	}
	binary.BigEndian.PutUint16(b[4:6], uint16(h.ID))
	b[8] = byte(h.TTL)
	b[9] = byte(h.Protocol)
	binary.BigEndian.PutUint16(b[10:12], uint16(h.Checksum))
	if ip := h.Src.To4(); ip != nil {
		copy(b[12:16], ip[:net.IPv4len])
	}
	if ip := h.Dst.To4(); ip != nil {
		copy(b[16:20], ip[:net.IPv4len])
	} else {
		return nil, errMissingAddress
	}
	if len(h.Options) > 0 {
		copy(b[HeaderLen:], h.Options)
	}
	return b, nil
}

// ParseHeader parses b as an IPv4 header.
func ParseHeader(b []byte) (*Header, []byte, error) {
	var data []byte
	if len(b) < HeaderLen {
		return nil, data, errHeaderTooShort
	}
	hdrlen := int(b[0]&0x0f) << 2
	if hdrlen > len(b) {
		return nil, data, errBufferTooShort
	}
	h := &Header{
		Version:  int(b[0] >> 4),
		Len:      hdrlen,
		TOS:      int(b[1]),
		ID:       int(binary.BigEndian.Uint16(b[4:6])),
		TTL:      int(b[8]),
		Protocol: int(b[9]),
		Checksum: int(binary.BigEndian.Uint16(b[10:12])),
		Src:      net.IPv4(b[12], b[13], b[14], b[15]),
		Dst:      net.IPv4(b[16], b[17], b[18], b[19]),
	}
	switch runtime.GOOS {
	case "darwin", "dragonfly", "netbsd":
		h.TotalLen = int(nativeEndian.Uint16(b[2:4])) + hdrlen
		h.FragOff = int(nativeEndian.Uint16(b[6:8]))
	case "freebsd":
		if freebsdVersion < 1100000 {
			h.TotalLen = int(nativeEndian.Uint16(b[2:4]))
			if freebsdVersion < 1000000 {
				h.TotalLen += hdrlen
			}
			h.FragOff = int(nativeEndian.Uint16(b[6:8]))
		} else {
			h.TotalLen = int(binary.BigEndian.Uint16(b[2:4]))
			h.FragOff = int(binary.BigEndian.Uint16(b[6:8]))
		}
	default:
		h.TotalLen = int(binary.BigEndian.Uint16(b[2:4]))
		h.FragOff = int(binary.BigEndian.Uint16(b[6:8]))
	}
	h.Flags = HeaderFlags(h.FragOff&0xe000) >> 13
	h.FragOff = h.FragOff & 0x1fff
	if hdrlen-HeaderLen > 0 {
		h.Options = make([]byte, hdrlen-HeaderLen)
		copy(h.Options, b[HeaderLen:])
	}
	data = b[hdrlen:]
	return h, data, nil
}

type ICMPMessageBody interface {
	Len() int
	Marshal() ([]byte, error)
}

type ICMPMessage struct {
	Type     int             // type
	Code     int             // code
	Checksum int             // checksum
	Body     ICMPMessageBody // body
}

func (m *ICMPMessage) Marshal() (bs []byte, err error) {
	bs = []byte{byte(m.Type), byte(m.Code), 0, 0}
	if m.Body != nil && m.Body.Len() != 0 {
		mb, err := m.Body.Marshal()
		if err != nil {
			return nil, err
		}
		bs = append(bs, mb...)
	}
	switch m.Type {
	case ICMPv6EchoRequest, ICMPv6EchoReply:
		return
	}
	csumcv := len(bs) - 1 // checksum coverage
	s := uint32(0)
	for i := 0; i < csumcv; i += 2 {
		s += uint32(bs[i+1])<<8 | uint32(bs[i])
	}
	if csumcv&1 == 0 {
		s += uint32(bs[csumcv])
	}
	s = s>>16 + s&0xffff
	s = s + s>>16
	// Place checksum back in header; using ^= avoids the
	// assumption the checksum bytes are zero.
	bs[2] ^= byte(^s & 0xff)
	bs[3] ^= byte(^s >> 8)
	return
}

func ParseICMPMessage(bs []byte) (msg *ICMPMessage, err error) {
	msglen := len(bs)
	if msglen < 4 {
		err = errors.New("message too short")
		return
	}
	msg = &ICMPMessage{
		Type:     int(bs[0]),
		Code:     int(bs[1]),
		Checksum: int(bs[2])<<8 | int(bs[3]),
	}
	if msglen > 4 {
		switch msg.Type {
		case ICMPv4EchoRequest, ICMPv4EchoReply, ICMPv6EchoRequest, ICMPv6EchoReply:
			msg.Body, err = ParseICMPEcho(bs[4:])
			if err != nil {
				return
			}
		case ICMPv4TimeExceeded:
			msg.Body, err = ParseICMPTimeExceeded(bs[4:])
			if err != nil {
				return
			}
		}
	}
	return
}

type ICMPEcho struct {
	ID        int // identifier
	Seq       int // sequence number
	Timestamp time.Time
	Data      []byte // data
}

func (m *ICMPEcho) Len() (l int) {
	if m == nil {
		l = 0
		return
	}
	l = 4 + 14 + len(m.Data)
	return
}

func (m *ICMPEcho) Marshal() (bs []byte, err error) {
	bs = make([]byte, m.Len())
	bs[0], bs[1] = byte(m.ID>>8), byte(m.ID&0xff)
	bs[2], bs[3] = byte(m.Seq>>8), byte(m.Seq&0xff)
	//	binary.BigEndian.PutUint64(ts, uint64(m.Timestamp.UnixNano()))
	ts, _ := m.Timestamp.MarshalBinary()
	copy(bs[4:19], ts)
	copy(bs[20:], m.Data)
	return
}

func ParseICMPEcho(bs []byte) (msg *ICMPEcho, err error) {
	bodylen := len(bs)
	msg = &ICMPEcho{
		ID:  int(bs[0])<<8 | int(bs[1]),
		Seq: int(bs[2])<<8 | int(bs[3]),
	}
	if bodylen > 4 {
		err = msg.Timestamp.UnmarshalBinary(bs[4:19])
		msg.Data = make([]byte, bodylen-20)
		copy(msg.Data, bs[20:])
	}
	return
}

type ICMPTimeExceeded struct {
	ID    int
	Seq   int
	SrcIP string
	DstIP string
}

func (m *ICMPTimeExceeded) Len() (l int) {
	return
}

func (m *ICMPTimeExceeded) Marshal() (bs []byte, err error) {
	return
}

func ParseICMPTimeExceeded(bs []byte) (*ICMPTimeExceeded, error) {
	bodylen := len(bs)
	msg := &ICMPTimeExceeded{}
	if bodylen > 4 {
		msg.SrcIP = net.IPv4(bs[16], bs[17], bs[18], bs[19]).String()
		msg.DstIP = net.IPv4(bs[20], bs[21], bs[22], bs[23]).String()
		msg.ID = int(bs[28])<<8 | int(bs[29])
		msg.Seq = int(bs[30])<<8 | int(bs[31])
	}
	return msg, nil
}

// Return the first non-loopback address as a 4 byte IP address. This address
// is used for sending packets out.
func socketAddr() (addr [4]byte, err error) {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return
	}

	for _, a := range addrs {
		if ipnet, ok := a.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if len(ipnet.IP.To4()) == net.IPv4len {
				copy(addr[:], ipnet.IP.To4())
				return
			}
		}
	}
	err = errors.New("You do not appear to be connected to the Internet")
	return
}

func GetInterface(ip net.IP) (string, error) {
	ifs, err := net.Interfaces()
	if err != nil {
		return "", err
	}
	for _, i := range ifs {
		addrs, err := i.Addrs()
		if err != nil {
			return "", err
		}
		for _, a := range addrs {
			if ipnet, ok := a.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
				if ipnet.IP.To4().String() == ip.String() {
					return i.Name, nil
				}
			}
		}
	}
	err = errors.New("You do not appear to be connected to the Internet")
	return "", err
}

// Given a host name convert it to net.IP struct.
func hostToIP(host string) (net.IP, error) {
	addrs, err := net.LookupHost(host)
	if err != nil {
		return nil, err
	}
	addr := addrs[0]

	ipAddr, err := net.ResolveIPAddr("ip", addr)
	if err != nil {
		return nil, err
	}
	return ipAddr.IP, nil
}

// Given a net.IP struct convert it to a 4 byte IP address.
func ipToBytes(addr net.IP) [4]byte {
	var ip [4]byte
	copy(ip[:], addr.To4())
	return ip
}
