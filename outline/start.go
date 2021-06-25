package outline

import (
	"container/list"
	"fmt"
	"log"
	"net"
	"time"

	"github.com/Jigsaw-Code/outline-ss-server/service"
	"github.com/Jigsaw-Code/outline-ss-server/service/metrics"
	ss "github.com/Jigsaw-Code/outline-ss-server/shadowsocks"
	"github.com/f-hj/fruit-outline-server/mongo"
)

// 59 seconds is most common timeout for servers that do not respond to invalid requests
const tcpReadTimeout time.Duration = 59 * time.Second

// A UDP NAT timeout of at least 5 minutes is recommended in RFC 4787 Section 4.3.
const defaultNatTimeout time.Duration = 5 * time.Minute

const defaultReplayHistory int = 10000

const defaultPort int = 6666

type ssPort struct {
	tcpService service.TCPService
	udpService service.UDPService
	cipherList service.CipherList
}

type SSServer struct {
	natTimeout  time.Duration
	m           metrics.ShadowsocksMetrics
	replayCache service.ReplayCache
	ports       map[int]*ssPort
}

func (s *SSServer) startPort(portNum int) error {
	listener, err := net.ListenTCP("tcp", &net.TCPAddr{Port: portNum})
	if err != nil {
		return fmt.Errorf("Failed to start TCP on port %v: %v", portNum, err)
	}
	packetConn, err := net.ListenUDP("udp", &net.UDPAddr{Port: portNum})
	if err != nil {
		return fmt.Errorf("Failed to start UDP on port %v: %v", portNum, err)
	}
	log.Println("Listening TCP and UDP on port", portNum)
	port := &ssPort{cipherList: service.NewCipherList()}
	// TODO: Register initial data metrics at zero.
	port.tcpService = service.NewTCPService(port.cipherList, &s.replayCache, s.m, tcpReadTimeout)
	port.udpService = service.NewUDPService(s.natTimeout, port.cipherList, s.m)
	s.ports[portNum] = port
	go port.tcpService.Serve(listener)
	go port.udpService.Serve(packetConn)
	return nil
}

func (s *SSServer) removePort(portNum int) error {
	port, ok := s.ports[portNum]
	if !ok {
		return fmt.Errorf("Port %v doesn't exist", portNum)
	}
	tcpErr := port.tcpService.Stop()
	udpErr := port.udpService.Stop()
	delete(s.ports, portNum)
	if tcpErr != nil {
		return fmt.Errorf("Failed to close listener on %v: %v", portNum, tcpErr)
	}
	if udpErr != nil {
		return fmt.Errorf("Failed to close packetConn on %v: %v", portNum, udpErr)
	}
	log.Println("Stopped TCP and UDP on port", portNum)
	return nil
}

func (s *SSServer) Start(users []*mongo.OutlineUser) error {
	if err := s.startPort(defaultPort); err != nil {
		return err
	}

	if err := s.Reload(users); err != nil {
		return err
	}

	return nil
}

func (s *SSServer) Reload(users []*mongo.OutlineUser) error {
	portCiphers := &list.List{}
	for _, user := range users {
		cipher, err := ss.NewCipher(user.Cipher, user.Secret)
		if err != nil {
			return fmt.Errorf("Failed to create cipher for key %v: %v", user.ID, err)
		}
		entry := service.MakeCipherEntry(user.ID, cipher, user.Secret)
		portCiphers.PushBack(&entry)
	}

	s.ports[defaultPort].cipherList.Update(portCiphers)

	return nil
}

// Stop serving on all ports.
func (s *SSServer) Stop() error {
	for portNum := range s.ports {
		if err := s.removePort(portNum); err != nil {
			return err
		}
	}
	return nil
}

func New() (*SSServer, error) {
	server := &SSServer{
		natTimeout:  defaultNatTimeout,
		replayCache: service.NewReplayCache(defaultReplayHistory),
		ports:       make(map[int]*ssPort),
	}

	return server, nil
}
