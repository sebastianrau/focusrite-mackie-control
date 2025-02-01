package focusriteclient

import (
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	focusritexml "github.com/sebastianrau/focusrite-mackie-control/pkg/focusrite-xml"
	"github.com/sebastianrau/focusrite-mackie-control/pkg/logger"
	"github.com/sirupsen/logrus"
)

var log *logrus.Entry = logger.WithPackage("focusriteclient")

type State int

const (
	Discover = iota
	Connected
	Waiting
)

const (
	SERVER_IP        string        = "localhost"
	KEEP_ALIVE_S     time.Duration = 3 * time.Second
	RECONNECT_TIME_S time.Duration = 5 * time.Second
)

type FocusriteClientMode int

const (
	UpdateDevice FocusriteClientMode = iota
	UpdateRaw
	UpdateBoth
)

// FocusriteClient stellt eine TCP-Verbindung zu einem Focusrite-Server her und empfängt Daten.
type FocusriteClient struct {
	mutex       sync.Mutex
	state       State
	port        int
	connection  net.Conn
	isConnected bool

	DeviceList    DeviceList
	ClientDetails focusritexml.ClientDetails

	ConnectedChannel chan bool
	ApprovalChannel  chan bool

	DeviceArrivalChannel chan *focusritexml.Device
	DeviceUpdateChannel  chan *focusritexml.Device
	RawUpdateChannel     chan *focusritexml.Set

	Mode FocusriteClientMode
}

// NewFocusriteClient erstellt einen neuen FocusriteClient.
func NewFocusriteClient(mode FocusriteClientMode) *FocusriteClient {
	f := &FocusriteClient{
		state: Discover,
		ClientDetails: focusritexml.ClientDetails{
			Hostname:  "Monitor Controller",
			ClientKey: "123456789",
		},
		DeviceList:           make(DeviceList),
		DeviceUpdateChannel:  make(chan *focusritexml.Device),
		DeviceArrivalChannel: make(chan *focusritexml.Device),
		RawUpdateChannel:     make(chan *focusritexml.Set),
		ApprovalChannel:      make(chan bool),
		ConnectedChannel:     make(chan bool),
		Mode:                 mode,
	}
	go f.run()
	go f.runKeepalive()

	return f
}

// Start stellt eine Verbindung zum Focusrite-Server her und empfängt Daten.
func (fc *FocusriteClient) run() {
	for {
		switch fc.state {
		case Discover:
			p, err := DiscoverServer()
			if err != nil {
				log.Warn(err.Error())
				fc.state = Waiting
			}
			log.Infof("Port discovered: %d", fc.port)
			fc.port = p
			fc.state = Connected

		case Connected:
			err := fc.connectAndListen()
			if err != nil {
				log.Info("connect and listen: " + err.Error())
			}
			fc.setConnection(nil)
			fc.setConnected(false)
			fc.state = Waiting

		case Waiting:
			time.Sleep(RECONNECT_TIME_S)
			fc.state = Discover
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
	err = fc.SendClientDetails()
	if err != nil {
		return err
	}

	for {
		buf := make([]byte, 65536)
		n, err := conn.Read(buf)
		if err == io.EOF {
			continue
		}
		if err != nil {
			return err
		}
		packet := string(buf[:n])
		if packet != "" {
			fc.handleXmlPacket(packet)
		}
	}
}

func (fc *FocusriteClient) runKeepalive() {
	for {
		if fc.isConnected {
			err := fc.sendXML(focusritexml.KeepAlive{})
			if err != nil {
				log.Error(err.Error())
			}
		}
		time.Sleep(KEEP_ALIVE_S)
	}
}

func (fc *FocusriteClient) handleXmlPacket(packet string) {
	d, err := focusritexml.ParseFromXML(packet)
	if err != nil {
		log.Errorln(err.Error())
	}

	switch dd := d.(type) {
	case focusritexml.Set:
		log.Debugf("Got device Update with ID: %d (%d Items)\n", dd.DevID, len(dd.Items))
		device, ok := fc.DeviceList.GetDevice(dd.DevID)
		if !ok {
			log.Warningf("Unknown device to Update with ID: %d (%d Items)\n", dd.DevID, len(dd.Items))
			return
		}
		if fc.Mode == UpdateDevice || fc.Mode == UpdateBoth {
			fc.DeviceList.UpdateSet(dd)
			fc.DeviceUpdateChannel <- device
		}
		if fc.Mode == UpdateRaw || fc.Mode == UpdateBoth {
			fc.RawUpdateChannel <- &dd
		}
		return

	case focusritexml.DeviceArrival:
		device := fc.DeviceList.AddDevice(&dd.Device)
		device.UpdateMap()
		err := fc.SendSubscribe(device.ID, true)
		if err != nil {
			log.Error(err.Error())
		}
		fc.DeviceArrivalChannel <- device
		log.Infof("New Device: %s, with ID: %d \n", dd.Device.Model, dd.Device.ID)
		return

	case focusritexml.DeviceRemoval:
		fc.DeviceList.Remove(dd.Id)
		return

	case focusritexml.ClientDetails:
		fc.ClientDetails.Id = dd.Id
		log.Debugf("New Cleint Details: %s, with ID: %s \n", dd.ClientKey, dd.Id)
		return

	case focusritexml.Approval:
		fc.ApprovalChannel <- dd.Authorised
		return

	//Ignoring
	case focusritexml.KeepAlive:
	default:
		log.Warnf("UNKNOWN data: %+v\n\n", d)
	}
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

func (fc *FocusriteClient) SendSet(set focusritexml.Set) error {
	dev, ok := fc.DeviceList.GetDevice(set.DevID)
	if ok {
		dev.UpdateSet(set)
	}
	return fc.sendXML(set)
}
