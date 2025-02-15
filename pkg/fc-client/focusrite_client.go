package focusriteclient

import (
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	focusritexml "github.com/sebastianrau/focusrite-mackie-control/pkg/fc-xml"
	"github.com/sebastianrau/focusrite-mackie-control/pkg/logger"
)

var log *logger.CustomLogger = logger.WithPackage("focusriteclient")

type State int

const (
	Discover = iota
	Connected
	Waiting
)

const (
	SERVER_IP        string        = "localhost"
	KEEP_ALIVE_TIME  time.Duration = 3 * time.Second
	RECONNECT_TIME_S time.Duration = 5 * time.Second

	FC_SEND_INTERVAL time.Duration = 75 * time.Millisecond
)

type FocusriteClientMode int

const (
	UpdateDevice FocusriteClientMode = iota
	UpdateRaw
	UpdateBoth
)

// FocusriteClient stellt eine TCP-Verbindung zu einem Focusrite-Server her und empfängt Daten.
type FocusriteClient struct {
	connectionMutex sync.Mutex
	state           State
	port            int
	connection      net.Conn
	isConnected     bool

	DeviceList    DeviceList
	ClientDetails focusritexml.ClientDetails

	ToFocusrite   chan focusritexml.Set
	FromFocusrite chan interface{}

	sendMutex sync.Mutex
	sendQueue map[int]focusritexml.Set

	Mode FocusriteClientMode
}

// NewFocusriteClient erstellt einen neuen FocusriteClient.
func NewFocusriteClient(mode FocusriteClientMode) *FocusriteClient {
	f := &FocusriteClient{
		connectionMutex: sync.Mutex{},

		state: Discover,

		ClientDetails: focusritexml.ClientDetails{
			Hostname:  "Monitor Controller",
			ClientKey: "123456789",
		},
		DeviceList: make(DeviceList),

		ToFocusrite:   make(chan focusritexml.Set, 100),
		FromFocusrite: make(chan interface{}, 100),

		Mode: mode,

		sendMutex: sync.Mutex{},
		sendQueue: make(map[int]focusritexml.Set),
	}
	go f.runConnection()
	go f.runKeepalive()
	go f.runCommandHandling()
	go f.runSendQueue()

	return f
}

// Start stellt eine Verbindung zum Focusrite-Server her und empfängt Daten.
func (fc *FocusriteClient) runConnection() {
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

func (fc *FocusriteClient) runKeepalive() {
	t := time.NewTicker(KEEP_ALIVE_TIME)
	defer t.Stop()

	for range t.C {
		if fc.isConnected {
			err := fc.sendXML(focusritexml.KeepAlive{})
			if err != nil {
				log.Error(err.Error())
			}
		}
	}
}

func (fc *FocusriteClient) runCommandHandling() {

	for set := range fc.ToFocusrite {
		if set.DevID != 0 && len(set.Items) > 0 {
			fc.sendMutex.Lock()
			q, ok := fc.sendQueue[set.DevID]
			if !ok { //new set to send
				fc.sendQueue[set.DevID] = set
			} else {
				for _, newItem := range set.Items {
					updated := false
					for qItemId, qItem := range q.Items {
						//set contains Item --> Update Item
						if qItem.ID == newItem.ID {
							log.Debugf("Updating Value: %d from %s to %s", qItem.ID, qItem.Value, newItem.Value)
							q.Items[qItemId].Value = newItem.Value
							updated = true
						}
					}
					if !updated {
						q.Items = append(q.Items, newItem)
					}
				}
			}
			fc.sendMutex.Unlock()
		}
	}
}

func (fc *FocusriteClient) runSendQueue() {

	t := time.NewTicker(FC_SEND_INTERVAL)
	for range t.C {
		fc.sendMutex.Lock()
		for qID, q := range fc.sendQueue {
			if len(q.Items) >= 0 || q.DevID != 0 {

				log.Debugf("Sending to Focusrite %d items\n", len(q.Items))
				err := fc.sendSet(q)
				if err != nil {
					log.Error(err)
				}
				//reset Buffer
				delete(fc.sendQueue, qID)

			}
		}
		fc.sendMutex.Unlock()

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

func (fc *FocusriteClient) handleXmlPacket(packet string) {
	d, err := focusritexml.ParseFromXML(packet)
	if err != nil {
		log.Errorln(err.Error())
	}

	switch dd := d.(type) {
	case focusritexml.Set:
		device, ok := fc.DeviceList.GetDevice(dd.DevID)
		if !ok {
			log.Warningf("Unknown device to Update with ID: %d (%d Items)\n", dd.DevID, len(dd.Items))
			return
		}
		if fc.Mode == UpdateDevice || fc.Mode == UpdateBoth {
			fc.DeviceList.UpdateSet(dd)
			fc.FromFocusrite <- DeviceUpdateMessage(*device)
		}
		if fc.Mode == UpdateRaw || fc.Mode == UpdateBoth {
			fc.FromFocusrite <- RawUpdateMessage(dd)
		}
		return

	case focusritexml.DeviceArrival:
		device := fc.DeviceList.AddDevice(&dd.Device)
		device.UpdateMap()
		err := fc.SendSubscribe(device.ID, true)
		if err != nil {
			log.Error(err.Error())
		}
		fc.FromFocusrite <- DeviceArrivalMessage(*device)
		log.Infof("New Device: %s, with ID: %d \n", dd.Device.Model, dd.Device.ID)
		return

	case focusritexml.DeviceRemoval:
		fc.FromFocusrite <- DeviceRemovalMessage(dd.Id)
		fc.DeviceList.Remove(dd.Id)
		return

	case focusritexml.ClientDetails:
		fc.ClientDetails.Id = dd.Id
		log.Debugf("New Cleint Details: %s, with ID: %s \n", dd.ClientKey, dd.Id)
		return

	case focusritexml.Approval:
		fc.FromFocusrite <- ApprovalMessasge(dd.Authorised)
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
	fc.connectionMutex.Lock()
	defer fc.connectionMutex.Unlock()
	fc.isConnected = status
	fc.FromFocusrite <- ConnectionStatusMessage(status)
}

// setConnected aktualisiert den Verbindungsstatus.
func (fc *FocusriteClient) Connected() bool {
	fc.connectionMutex.Lock()
	defer fc.connectionMutex.Unlock()
	return fc.isConnected
}

// setConnection sets the active connection.
func (fc *FocusriteClient) setConnection(conn net.Conn) {
	fc.connectionMutex.Lock()
	defer fc.connectionMutex.Unlock()
	fc.connection = conn
}

func (fc *FocusriteClient) sendXML(data interface{}) error {
	fc.connectionMutex.Lock()
	defer fc.connectionMutex.Unlock()

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

func (fc *FocusriteClient) sendSet(set focusritexml.Set) error {

	var cleanSet []focusritexml.Item

	for _, v := range set.Items {
		if v.ID != 0 {
			cleanSet = append(cleanSet, v)
		}
	}
	set.Items = cleanSet

	dev, ok := fc.DeviceList.GetDevice(set.DevID)
	if ok && len(set.Items) > 0 {
		dev.UpdateSet(set)
		return fc.sendXML(set)
	}
	return nil
}
