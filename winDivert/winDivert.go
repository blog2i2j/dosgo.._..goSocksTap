//go:build windows
// +build windows

package winDivert

import (
	_ "embed"
	"encoding/binary"
	"log"
	"net"
	"os"
	"strconv"
	"time"
	"sync"

	"github.com/imgk/divert-go"
)

var winDivert *divert.Handle
var winDivertRun = false

var winDivertEvent *divert.Handle
var winDivertEventRun = false
//不处理的地址包括udp+tcp
var excludeOriginalAddr sync.Map
var divertDll = "WinDivert.dll"
var divertSys = "WinDivert32.sys"

func dllInit(_divertDll string) {
	_, err := os.Stat(_divertDll)
	if err != nil {
		log.Printf("not found :%s\r\n", _divertDll)
	}
}

/*windows转发*/
func RedirectDNS(dnsAddr string, _port string, sendStartPort int, sendEndPort int) {
	var err error
	_, err = os.Stat(divertDll)
	if err != nil {
		log.Printf("not found :%s\r\n", divertDll)
		return
	}
	winDivertRun = true
	var filter = "outbound and !loopback and !impostor and udp.DstPort=53"

	for i := sendStartPort; i <= sendEndPort; i++ {
		filter = filter + " and udp.SrcPort!=" + strconv.Itoa(i)
	}

	winDivert, err = divert.Open(filter, divert.LayerNetwork, divert.PriorityDefault, divert.FlagDefault)
	if err != nil {
		log.Printf("winDivert open failed: %v\r\n", err)
		return
	}

	rawbuf := make([]byte, 1500)
	var dnsRecvBuf []byte = make([]byte, 1500)
	var recvBuf []byte = make([]byte, 1500)
	addr := divert.Address{}
	var dnsConn net.Conn
	dnsConn, _ = net.DialTimeout("udp", dnsAddr+":"+_port, time.Second*15)
	var udpsize=0;
	var packetsize int=0;
	var ipheadlen int
	var recvLen uint
	var conRecvLen int
	var udpheadlen = 8
	for winDivertRun {
		if winDivert == nil {
			continue
		}
		recvLen, err = winDivert.Recv(recvBuf, &addr)
		if err != nil {
			log.Println(1, err)
			continue
		}
		ipv6 := recvBuf[0]>>4 == 6
		if ipv6 {
			ipheadlen = 40
		} else {
			ipheadlen = int(recvBuf[0]&0xF) * 4
		}
		dnsConn.SetWriteDeadline(time.Now().Add(10 * time.Second))
		dnsConn.Write(recvBuf[ipheadlen+udpheadlen : recvLen])
		dnsConn.SetReadDeadline(time.Now().Add(10 * time.Second))
		conRecvLen, err = dnsConn.Read(dnsRecvBuf)
		if err == nil {
			udpsize = conRecvLen +udpheadlen
			if ipv6 {
				copy(rawbuf, []byte{96, 12, 19, 68, 0, 98, 17, 128})
				packetsize = 40 + udpsize
				binary.BigEndian.PutUint16(rawbuf[4:], uint16(udpsize))
				copy(rawbuf[8:], recvBuf[24:40])
				copy(rawbuf[24:], recvBuf[8:24])
				copy(rawbuf[ipheadlen:], recvBuf[ipheadlen+2:ipheadlen+4])
				copy(rawbuf[ipheadlen+2:], recvBuf[ipheadlen:ipheadlen+2])
			} else {
				copy(rawbuf, []byte{69, 0, 1, 32, 141, 152, 64, 0, 64, 17, 150, 46})
				packetsize = 20 + udpsize
				binary.BigEndian.PutUint16(rawbuf[2:], uint16(packetsize))
				copy(rawbuf[12:], recvBuf[16:20])
				copy(rawbuf[16:], recvBuf[12:16])
				copy(rawbuf[20:], recvBuf[ipheadlen+2:ipheadlen+4])
				copy(rawbuf[22:], recvBuf[ipheadlen:ipheadlen+2])
				ipheadlen = 20
			}

			binary.BigEndian.PutUint16(rawbuf[ipheadlen+4:], uint16(udpsize))
			copy(rawbuf[ipheadlen+udpheadlen:], dnsRecvBuf[:conRecvLen])
			divert.CalcChecksums( rawbuf[:packetsize], &addr, 0)
			_, err = winDivert.Send( rawbuf[:packetsize], &addr)
			if err != nil {
				log.Println(1, err)
				return
			}
		} else {
			dnsConn.Close()
			dnsConn, _ = net.DialTimeout("udp", dnsAddr+":"+_port, time.Second*15)
		}
	}
	if dnsConn != nil {
		dnsConn.Close()
	}
}


/*windows转发*/
func RedirectDNSV1(dnsAddr string, _port string) {
	var err error
	_, err = os.Stat(divertDll)
	if err != nil {
		log.Printf("not found :%s\r\n", divertDll)
		return
	}
	winDivertRun = true
	var filter = "outbound and !loopback and !impostor and udp.DstPort=53"

	winDivert, err = divert.Open(filter, divert.LayerNetwork, divert.PriorityDefault, divert.FlagDefault)
	if err != nil {
		log.Printf("winDivert open failed: %v\r\n", err)
		return
	}

	rawbuf := make([]byte, 1500)
	var dnsRecvBuf []byte = make([]byte, 1500)
	var recvBuf []byte = make([]byte, 1500)
	addr := divert.Address{}
	var dnsConn net.Conn
	dnsConn, _ = net.DialTimeout("udp", dnsAddr+":"+_port, time.Second*15)
	var udpsize=0;
	var packetsize int=0;
	var ipheadlen int
	var recvLen uint
	var conRecvLen int
	var udpheadlen = 8
	for winDivertRun {
		if winDivert == nil {
			continue
		}
		recvLen, err = winDivert.Recv(recvBuf, &addr)
		if err != nil {
			log.Println(1, err)
			continue
		}
		//如果是本机的请求直接发送
		localIp := net.IP(addr.Socket().LocalAddress[:])
		_, ok := excludeOriginalAddr.Load("udp:"+localIp.String()+":"+strconv.Itoa(int(addr.Socket().LocalPort)))
		if(ok){
			_, err = winDivert.Send( recvBuf[:recvLen], &addr)
			continue;
		}
		ipv6 := recvBuf[0]>>4 == 6
		if ipv6 {
			ipheadlen = 40
		} else {
			ipheadlen = int(recvBuf[0]&0xF) * 4
		}
		dnsConn.SetWriteDeadline(time.Now().Add(10 * time.Second))
		dnsConn.Write(recvBuf[ipheadlen+udpheadlen : recvLen])
		dnsConn.SetReadDeadline(time.Now().Add(10 * time.Second))
		conRecvLen, err = dnsConn.Read(dnsRecvBuf)
		if err == nil {
			udpsize = conRecvLen +udpheadlen
			if ipv6 {
				copy(rawbuf, []byte{96, 12, 19, 68, 0, 98, 17, 128})
				packetsize = 40 + udpsize
				binary.BigEndian.PutUint16(rawbuf[4:], uint16(udpsize))
				copy(rawbuf[8:], recvBuf[24:40])
				copy(rawbuf[24:], recvBuf[8:24])
				copy(rawbuf[ipheadlen:], recvBuf[ipheadlen+2:ipheadlen+4])
				copy(rawbuf[ipheadlen+2:], recvBuf[ipheadlen:ipheadlen+2])
			} else {
				copy(rawbuf, []byte{69, 0, 1, 32, 141, 152, 64, 0, 64, 17, 150, 46})
				packetsize = 20 + udpsize
				binary.BigEndian.PutUint16(rawbuf[2:], uint16(packetsize))
				copy(rawbuf[12:], recvBuf[16:20])
				copy(rawbuf[16:], recvBuf[12:16])
				copy(rawbuf[20:], recvBuf[ipheadlen+2:ipheadlen+4])
				copy(rawbuf[22:], recvBuf[ipheadlen:ipheadlen+2])
				ipheadlen = 20
			}

			binary.BigEndian.PutUint16(rawbuf[ipheadlen+4:], uint16(udpsize))
			copy(rawbuf[ipheadlen+udpheadlen:], dnsRecvBuf[:conRecvLen])
			divert.CalcChecksums( rawbuf[:packetsize], &addr, 0)
			_, err = winDivert.Send( rawbuf[:packetsize], &addr)
			if err != nil {
				log.Println(1, err)
				return
			}
		} else {
			dnsConn.Close()
			dnsConn, _ = net.DialTimeout("udp", dnsAddr+":"+_port, time.Second*15)
		}
	}
	if dnsConn != nil {
		dnsConn.Close()
	}
}

/*网络事件监听*/
func NetEvent(pid string, ownPid bool) {
	var err error
	_, err = os.Stat(divertDll)
	if err != nil {
		log.Printf("not found :%s\r\n", divertDll)
		return
	}
	winDivertEventRun = true

	var filter = "outbound and udp and remotePort=53"
	if(ownPid){
		filter = filter + " and (processId=" +pid+" or  processId="+strconv.Itoa(os.Getpid())+")";
	}else{
		filter = filter + " and processId=" +pid;
	}

	winDivertEvent, err = divert.Open(filter, divert.LayerSocket, divert.PriorityDefault, divert.FlagSniff|divert.FlagRecvOnly )
	if err != nil {
		log.Printf("winDivert open failed: %v\r\n", err)
		return
	}

	var recvBuf []byte = make([]byte, 1)
	addr := divert.Address{}
	for winDivertEventRun {
		if winDivertEvent == nil {
			continue
		}
		_, err = winDivertEvent.Recv(recvBuf, &addr)
		switch addr.Event(){
			case divert.EventSocketBind:
				if(addr.Socket().Protocol==17){
					log.Printf("udp pid:%d  local Port:%d\r\n",addr.Socket().ProcessID,addr.Socket().LocalPort);
				}
			break;
			case divert.EventSocketClose:
				if(addr.Socket().Protocol==17){
					log.Printf("udp close pid:%d  local Port:%d\r\n",addr.Socket().ProcessID,addr.Socket().LocalPort);
				}
				break;
		}
	}
	
}

/*网络事件监听*/
func NetEventv1(pid string,ownPid bool) {
	var err error
	_, err = os.Stat(divertDll)
	if err != nil {
		log.Printf("not found :%s\r\n", divertDll)
		return
	}
	winDivertEventRun = true
	var filter = "outbound"
	if(ownPid){
		filter = filter + " and (processId=" +pid+" or  processId="+strconv.Itoa(os.Getpid())+")";
	}else{
		filter = filter + " and processId=" +pid;
	}
	winDivertEvent, err = divert.Open(filter, divert.LayerFlow, divert.PriorityDefault, divert.FlagSniff|divert.FlagRecvOnly )
	if err != nil {
		log.Printf("winDivert open failed: %v\r\n", err)
		return
	}

	var recvBuf []byte = make([]byte, 1)
	addr := divert.Address{}
	for winDivertEventRun {
		if winDivertEvent == nil {
			continue
		}
		_, err = winDivertEvent.Recv(recvBuf, &addr)
		localIp := net.IP(addr.Socket().LocalAddress[:])
		switch addr.Event(){
			case divert.EventFlowEstablished:
				if(addr.Socket().Protocol==17){
					log.Printf("udp pid:%d  local Port:%d rport:%d\r\n",addr.Socket().ProcessID,addr.Socket().LocalPort,addr.Socket().RemotePort);
					//udp只有53的才记录
					if(addr.Socket().RemotePort==53){
						excludeOriginalAddr.Store("udp:"+localIp.String()+":"+strconv.Itoa(int(addr.Socket().LocalPort)),int64(time.Now().Unix()))
					}
				}else{
					excludeOriginalAddr.Store("tcp:"+localIp.String()+":"+strconv.Itoa(int(addr.Socket().LocalPort)),int64(time.Now().Unix()))
				}
			break;
			case divert.EventFlowDeleted:
				if(addr.Socket().Protocol==17){
					log.Printf("udp close pid:%d  local Port:%d rport:%d\r\n",addr.Socket().ProcessID,addr.Socket().LocalPort,addr.Socket().RemotePort);
				}
				break;
		}
	}
	
}

func CloseWinDivert() {
	winDivertRun = false
	if winDivert != nil {
		winDivert.Close()
	}

	winDivertEventRun = false
	if winDivertEvent != nil {
		winDivertEvent.Close()
	}
}
