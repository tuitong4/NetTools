package ping

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"
)

type Target struct {
	SrcIP     string
	SrcBind   bool
	DstIP     string
	Count     int
	TimeoutMs time.Duration
	Interval  int
}

type PingResponse struct {
	SrcIP      string
	DstIP      string
	Timestamp  int64
	AvgRTT     time.Duration
	MaxRTT     time.Duration
	MinRTT     time.Duration
	TTL        int
	PacketLoss int
}

func Pinger(target *Target, xid int) (*PingResponse, error) {
	xseq := 1
	rc := 0
	var conn net.Conn
	var err error
	rttSum := time.Duration(0)
	if target.SrcBind {
		laddr := net.IPAddr{IP: net.ParseIP(target.SrcIP)}
		raddr, _ := net.ResolveIPAddr("ip", target.DstIP)
		conn, err = net.DialIP("ip4:icmp", &laddr, raddr)
	} else {
		conn, err = net.Dial("ip4:icmp", target.DstIP)
	}
	if err != nil {
		errMsg := fmt.Sprintf("%s<0x%0x> Dial icmp error! %s.", target.DstIP, xid, err.Error())
		return nil, errors.New(errMsg)
	}
	defer conn.Close()
	pr := new(PingResponse)
	pr.SrcIP = target.SrcIP
	pr.DstIP = target.DstIP
	pr.Timestamp = time.Now().Unix() * 1000
	pr.MaxRTT = time.Duration(0)
	pr.MinRTT = time.Duration(0)

	for i := target.Count; i > 0; i-- {
		if i != target.Count {
			time.Sleep(time.Duration(target.Interval) * time.Millisecond)
		}
		msg := &ICMPMessage{
			Type: ICMPv4EchoRequest,
			Code: 0,
			Body: &ICMPEcho{
				ID:        xid,
				Seq:       xseq + i,
				Timestamp: time.Now(),
				Data:      bytes.Repeat([]byte(strings.Repeat("G", 21)), 2),
			},
		}

		bs, err := msg.Marshal()
		if err != nil {
			errMsg := fmt.Sprintf("%s<0x%0x> ICMP message marshal err! %s.", target.DstIP, xid, err.Error())
			return nil, errors.New(errMsg)
		}
		_, err = conn.Write(bs)
		//log.Debug("%s ICMP count %d,send time %s,src %s", logHeader, target.Count-i, time.Now(), target.SrcIP)
		if err != nil {
			errMsg := fmt.Sprintf("%s<0x%0x> ICMP conn write err! %s.", target.DstIP, xid, err.Error())
			return nil, errors.New(errMsg)
		}
		conn.SetReadDeadline(time.Now().Add(time.Duration(target.TimeoutMs) * time.Millisecond))

		buf := make([]byte, 1024)
		_, err = conn.Read(buf)
		if err != nil {
			//log.Debug("%s Read timeouts", logHeader)
			break
		}

		h, data, _ := ParseHeader(buf)
		//log.Debug("%s Parse ICMP Message %s -> %s", logHeader, h.Src.String(), h.Dst.String())
		bufmsg, err := ParseICMPMessage(data)
		if err != nil {
			errMsg := fmt.Sprintf("%s<0x%0x> Parse ICMP Message error! %s.", target.DstIP, xid, err.Error())
			return nil, errors.New(errMsg)
		}
		switch bufmsg.Type {
		case ICMPv4EchoReply:
			if target.SrcBind { //make sure src ip
				if h.Src.String() != target.DstIP || h.Dst.String() != target.SrcIP {
					//log.Debug("%s mistake receiving: src m: %s, dst: %s, src_in_ip: %s, dst_in_ip: %s, id: %d, the wrong id: %d\n",
					//	logHeader, target.SrcIP, target.DstIP, h.Dst.String(), h.Src.String(), xid, msg.Body.(*ICMPEcho).ID)
					continue
				}
			} else { //do not know src ip
				if h.Src.String() != target.DstIP {
					//log.Debug("%s mistake receiving: src m: %s, dst: %s, src_in_ip: %s, dst_in_ip: %s, id: %d, the wrong id: %d\n",
					//	logHeader, target.SrcIP, target.DstIP, h.Dst.String(), h.Src.String(), xid, msg.Body.(*ICMPEcho).ID)
					continue
				}
			}
			if xid != bufmsg.Body.(*ICMPEcho).ID {
				//log.Debug("%s xid wrong. ID is %d, the wrong id is %d\n", logHeader, xid, msg.Body.(*ICMPEcho).ID)
				continue
			}
			t := time.Now()
			rtt := t.Sub(bufmsg.Body.(*ICMPEcho).Timestamp)
			//log.Debug("%s send time:%s, receive time:%s, rtt:%d", logHeader, msg.Body.(*ICMPEcho).Timestamp, t, rtt)
			rc++
			if rc == 1 {
				pr.TTL = h.TTL
				pr.MaxRTT = rtt
				pr.MinRTT = rtt
			} else {
				if rtt > pr.MaxRTT {
					pr.MaxRTT = rtt
				}
				if rtt < pr.MinRTT {
					pr.MinRTT = rtt
				}
			}
			rttSum = rttSum + rtt
			//log.Debug("%s receive ok. %s -> %s", logHeader, h.Src.String(), h.Dst.String())
		}
	}
	pr.PacketLoss = (target.Count - rc) * 100 / target.Count
	if rc == 0 {
		pr.AvgRTT = time.Duration(0)
	} else {
		pr.AvgRTT = rttSum / time.Duration(rc)
	}
	return pr, nil
}
