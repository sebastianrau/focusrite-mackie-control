package focusriteclient

import (
	"fmt"
	"io"
	"log"
	"net"
	"sync"
	"time"

	focusritexml "github.com/sebastianrau/focusrite-mackie-control/pkg/focusrite-xml"
)

const (
	SERVER_IP    string        = "localhost"
	KEEP_ALIVE_S time.Duration = 3 * time.Second
)

// FocusriteClient stellt eine TCP-Verbindung zu einem Focusrite-Server her und empfängt Daten.
type FocusriteClient struct {
	port        int
	connection  net.Conn
	isConnected bool
	mutex       sync.Mutex

	DataChannel      chan interface{}
	ConnectedChannel chan bool
	stopChannel      chan struct{}
	messageChannel   chan string
}

// NewFocusriteClient erstellt einen neuen FocusriteClient.
func NewFocusriteClientAutoDiscover() (*FocusriteClient, error) {
	port, err := DiscoverServer()
	if err != nil {
		return nil, err
	}
	return NewFocusriteClient(port), nil
}

// NewFocusriteClient erstellt einen neuen FocusriteClient.
func NewFocusriteClient(port int) *FocusriteClient {
	f := &FocusriteClient{
		port:             port,
		DataChannel:      make(chan interface{}),
		ConnectedChannel: make(chan bool),
		stopChannel:      make(chan struct{}),
		messageChannel:   make(chan string),
	}
	go f.start()

	return f
}

// Start stellt eine Verbindung zum Focusrite-Server her und empfängt Daten.
func (fc *FocusriteClient) start() error {
	for {
		err := fc.connectAndListen()
		if err != nil {
			log.Printf("Verbindungsfehler: %v\n", err)
			fc.setConnected(false)

			// Reconnect-Logik
			select {
			case <-fc.stopChannel:
				return nil
			default:
				log.Println("Versuche erneut zu verbinden...")
				time.Sleep(5 * time.Second)
				continue
			}
		}
	}
}

// connectAndListen stellt die Verbindung her und verarbeitet eingehende Daten.
func (fc *FocusriteClient) connectAndListen() error {
	conn, err := net.Dial("tcp4", fmt.Sprintf("%s:%d", SERVER_IP, fc.port))
	if err != nil {
		return err
	}
	defer conn.Close()

	fc.setConnected(true)
	fc.setConnection(conn)
	fc.SendClientDetails()

	// Send keep alive
	go func(fc *FocusriteClient) {
		for {
			if !fc.Connected() {
				return
			}
			fc.sendXML(focusritexml.KeepAlive{})
			time.Sleep(KEEP_ALIVE_S)
		}
	}(fc)

	for {
		buf := make([]byte, 65536)
		n, err := conn.Read(buf) // Liest Daten in den Puffer
		if err == io.EOF {
			log.Println("Verbindung geschlossen.")
			time.Sleep(5 * time.Second)
			break
		}
		if err != nil {
			log.Printf("Fehler beim Lesen des Servers: %v\n", err)
			break
		}

		packet := string(buf[:n])

		// Empfange und sende Daten über den Channel
		if packet != "" {
			d, err := focusritexml.ParseFromXML(packet)
			if err != nil {
				fmt.Println(err.Error())
			}
			fc.DataChannel <- d
		}

	}
	return nil
}

// SendData sends an XML-encoded FocusriteControl object to the server.
func (fc *FocusriteClient) SendClientDetails() error {

	return fc.sendXML(focusritexml.ClientDetails{
		Hostname: "Monitor Controller",
		//Hostname:  "Focusrite Midi Control",
		ClientKey: "123456789",
	})
}

func (fc *FocusriteClient) SendSubscribe(id int, subscribe bool) error {
	return fc.sendXML(focusritexml.SubscribeMessage{
		DeviceId:  id,
		Subscribe: subscribe,
	})
}

// setConnected aktualisiert den Verbindungsstatus.
func (fc *FocusriteClient) setConnected(status bool) {
	fc.mutex.Lock()
	defer fc.mutex.Unlock()
	fc.isConnected = status
	fc.ConnectedChannel <- status
}

// setConnected aktualisiert den Verbindungsstatus.
func (fc *FocusriteClient) Connected() bool {
	fc.mutex.Lock()
	defer fc.mutex.Unlock()
	return fc.isConnected
}

// setConnection sets the active connection.
func (fc *FocusriteClient) setConnection(conn net.Conn) {
	fc.mutex.Lock()
	defer fc.mutex.Unlock()
	fc.connection = conn
}

// Stop beendet den Client und die Reconnect-Logik.
func (fc *FocusriteClient) Stop() {
	close(fc.stopChannel)
}

func (fc *FocusriteClient) sendXML(data interface{}) error {
	fc.mutex.Lock()
	defer fc.mutex.Unlock()

	if fc.connection == nil {
		return fmt.Errorf("not connected to the server")
	}

	msg, err := focusritexml.ParseToXML(data)
	if err != nil {
		return err
	}

	_, err = fc.connection.Write([]byte(msg))
	if err != nil {
		return err
	}
	return nil
}
