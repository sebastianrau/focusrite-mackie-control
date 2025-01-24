package focusrite

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"net"
	"sync"
	"time"
)

const (
	SERVER_PORT int    = 49152
	SERVER_IP   string = "127.0.0.1"
)

// FocusriteClient stellt eine TCP-Verbindung zu einem Focusrite-Server her und empfängt Daten.
type FocusriteClient struct {
	connection  net.Conn
	isConnected bool
	mutex       sync.Mutex

	DataChannel      chan *FocusriteControl
	ConnectedChannel chan interface{}
	stopChannel      chan struct{}
	messageChannel   chan string
}

// NewFocusriteClient erstellt einen neuen FocusriteClient.
func NewFocusriteClient(ip string) *FocusriteClient {
	f := &FocusriteClient{
		DataChannel:      make(chan *FocusriteControl),
		ConnectedChannel: make(chan interface{}),
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
	conn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", SERVER_IP, SERVER_PORT))
	if err != nil {
		return fmt.Errorf("Fehler beim Verbinden: %v", err)
	}
	defer conn.Close()

	log.Printf("Verbindung zu %s:%d hergestellt.\n", SERVER_IP, SERVER_PORT)
	fc.setConnected(true)
	fc.setConnection(conn)

	decoder := xml.NewDecoder(conn)
	for {
		var data FocusriteControl
		err := decoder.Decode(&data)
		if err == io.EOF {
			log.Println("Verbindung geschlossen.")
			break
		}
		if err != nil {
			log.Printf("Fehler beim Dekodieren von XML: %v\n", err)
			break
		}

		// Empfange und sende Daten über den Channel
		fc.DataChannel <- &data

		go func(data FocusriteControl) {
			switch data.MessageType {
			case "status-update":
				fmt.Printf("Received status update: %+v\n", data)
				fc.messageChannel <- "Status update received."

			case "device-info":
				fmt.Printf("Received device info: %+v\n", data)
				fc.messageChannel <- fmt.Sprintf("Device Info: %s (Firmware: %s)", data.Device.Name, data.Device.FirmwareVersion)

			default:
				fmt.Printf("Received unknown message type: %+v\n", data)
				fc.messageChannel <- "Unknown message received."
			}
		}(data)

	}
	return nil
}

// SendData sends an XML-encoded FocusriteControl object to the server.
func (fc *FocusriteClient) SendControlData(data *FocusriteControl) error {
	fc.mutex.Lock()
	defer fc.mutex.Unlock()

	if fc.connection == nil {
		return fmt.Errorf("Not connected to the server")
	}

	// Encode the data as XML
	var buffer bytes.Buffer
	encoder := xml.NewEncoder(&buffer)
	err := encoder.Encode(data)
	if err != nil {
		return fmt.Errorf("Error encoding data: %v", err)
	}

	// Write the XML data to the connection
	_, err = fc.connection.Write(buffer.Bytes())
	if err != nil {
		return fmt.Errorf("Error sending data: %v", err)
	}

	log.Println("Data sent successfully.")
	return nil
}

// SendData sends an XML-encoded FocusriteControl object to the server.
func (fc *FocusriteClient) SendClientDetails(data *FocusriteControl) error {
	fc.mutex.Lock()
	defer fc.mutex.Unlock()

	if fc.connection == nil {
		return fmt.Errorf("Not connected to the server")
	}

	// Encode the data as XML
	var buffer bytes.Buffer
	encoder := xml.NewEncoder(&buffer)
	err := encoder.Encode(data.ClientDetails)
	if err != nil {
		return fmt.Errorf("Error encoding data: %v", err)
	}

	// Write the XML data to the connection
	_, err = fc.connection.Write(buffer.Bytes())
	if err != nil {
		return fmt.Errorf("Error sending data: %v", err)
	}

	log.Println("Data sent successfully.")
	return nil
}

// setConnected aktualisiert den Verbindungsstatus.
func (fc *FocusriteClient) setConnected(status bool) {
	fc.mutex.Lock()
	defer fc.mutex.Unlock()
	fc.isConnected = status
	fc.ConnectedChannel <- status
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
