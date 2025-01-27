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

	DeviceList    DeviceList
	ClientDetails focusritexml.ClientDetails

	ConnectedChannel chan bool
	DataChannel      chan *focusritexml.Device
	ApprovalChannel  chan bool

	stopChannel chan struct{}
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
		port: port,
		ClientDetails: focusritexml.ClientDetails{
			Hostname:  "Monitor Controller",
			ClientKey: "123456789",
		},

		DeviceList:       make(DeviceList),
		DataChannel:      make(chan *focusritexml.Device),
		ApprovalChannel:  make(chan bool),
		ConnectedChannel: make(chan bool),
		stopChannel:      make(chan struct{}),
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
			switch dd := d.(type) {

			case focusritexml.KeepAlive:
				//TODO add Keep Alive timer
				//TODO reset keep alive timer

			case focusritexml.Set:
				fc.DeviceList.UpdateSet(dd)
				log.Printf("Device Updated with ID: %d \n\n", dd.DevID)
				device, ok := fc.DeviceList.GetDevice(dd.DevID)
				if ok {
					fc.DataChannel <- device
				}

			case focusritexml.DeviceArrival:
				fc.DeviceList.AddDevice(&dd.Device)
				fc.SendSubscribe(dd.Device.ID, true)
				device, ok := fc.DeviceList.GetDevice(dd.Device.ID)
				if ok {
					fc.DataChannel <- device
				}
				log.Printf("New Device: %s, with ID: %d \n\n", dd.Device.Model, dd.Device.ID)

			case focusritexml.DeviceRemoval:
				fc.DeviceList.Remove(dd.Id)

			case focusritexml.ClientDetails:
				fc.ClientDetails.Id = dd.Id
				log.Printf("New Cleint Details: %s, with ID: %s \n\n", dd.ClientKey, dd.Id)

			case focusritexml.Approval:
				fc.ApprovalChannel <- dd.Authorised

			default:
				fmt.Printf("UNKNOWN data: %+v\n\n", d)
			}
		}

	}
	return nil
}

// SendData sends an XML-encoded FocusriteControl object to the server.
func (fc *FocusriteClient) SendClientDetails() error {

	return fc.sendXML(fc.ClientDetails)
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
