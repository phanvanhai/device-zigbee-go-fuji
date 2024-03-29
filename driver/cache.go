package driver

import (
	"strconv"
	"sync"

	sdk "github.com/edgexfoundry/device-sdk-go"
	"github.com/edgexfoundry/go-mod-core-contracts/models"
)

var (
	initOnce sync.Once
	oc       *objectCache
)

// type
const (
	DEVICETYPE   = "DeviceType"
	GROUPTYPE    = "GroupType"
	SCENARIOTYPE = "ScenarioType"
)

const (
	nameNetworkProtocol  = "Network"
	nameMACProperty      = "MAC"
	namePANProperty      = "PAN"
	nameAddressProperty  = "Address"
	nameEndpointProperty = "Endpoint"
	nameTypeProperty     = "Type" // callback add Device phai them property Type: 'D', 'G', 'S'

	nameProfileID   = "profileID"
	nameClusterID   = "clusterID"
	nameAttributeID = "attributeID"
	nameValueType   = "valueType"

	managerProfileNameConst = "ManagerProfile"
)

type objectCache struct {
	nameIDObject     map[string]string
	idNameObject     map[string]string
	resAttMap        map[string]AttributeInfo
	attResMap        map[AttributeInfo]models.DeviceResource
	addrIDObjectMap  map[ObjectAddress]string
	idInfoObjectMap  map[string]ObjectInfo
	nameMasterDevice string
	mutex            sync.Mutex
}

type ObjectCache interface {
	Lock()
	Unlock()
	UpdateObjectWhithoutSync(d models.Device)
	UpdateObject(d models.Device)
	DeleteObject(nameObject string)
	ConvertNameToIDObject(nameOb string) (string, bool)
	ConvertIDToNameObject(idOb string) (string, bool)
	ConvertAttToRes(a AttributeInfo) (models.DeviceResource, bool)
	ConvertResToAtt(resName string) (AttributeInfo, bool)
	ConvertAddrToIDObject(addr ObjectAddress) (string, bool)
	ConvertIDToObjectInfo(id string) (ObjectInfo, bool)
	GetMasterDeviceName() string
}

type AttributeInfo struct {
	ProfileID   uint16 `json:"pro"`
	ClusterID   uint16 `json:"clu"`
	AttributeID uint16 `json:"att"`
	ValueType   uint8  `json:"vltp"`
}

type AttributeValue struct {
	AttributeInfo
	Value interface{} `json:"val"`
}

type ResourceValue struct {
	ResourceName string
	Value        interface{} `json:"val"`
}

type ObjectAddress struct {
	Address  uint16 `json:"addr"`
	Type     uint8  `json:"type"`
	Endpoint uint8  `json:"endp"`
}
type AddressEUI64 struct {
	MAC string `json:"MAC,omitempty"`
	PAN uint16 `json:"PAN,omitempty"`
}
type ObjectInfo struct {
	AddressEUI64
	ObjectAddress
}

func getObjectAddressFromProtocol(p map[string]models.ProtocolProperties) (ob ObjectAddress, ok bool) {
	pp, ok := p[nameNetworkProtocol]
	if !ok {
		return
	}
	addr, ok := pp[nameAddressProperty]
	if !ok {
		return
	}
	addrint, err := strconv.ParseUint(addr, 10, 16)
	if err != nil {
		return ob, false
	}
	addr16 := uint16(addrint)

	tp, ok := pp[nameTypeProperty]
	if !ok {
		return
	}
	tpint, err := strconv.ParseUint(tp, 10, 8)
	if err != nil {
		return ob, false
	}
	tp8 := uint8(tpint)

	ep, ok := pp[nameEndpointProperty]
	if !ok {
		return
	}
	epint, err := strconv.ParseUint(ep, 10, 16)
	if err != nil {
		return ob, false
	}
	ep8 := uint8(epint)

	ob.Address = addr16
	ob.Endpoint = ep8
	ob.Type = tp8
	return ob, true
}

func getObjectInfoFromProtocol(p map[string]models.ProtocolProperties) (ob ObjectInfo, ok bool) {
	ob.ObjectAddress, ok = getObjectAddressFromProtocol(p)
	if !ok {
		return
	}
	pp, ok := p[nameNetworkProtocol]
	mac, ok := pp[nameMACProperty]
	if !ok {
		return
	}
	pan, ok := pp[namePANProperty]
	if !ok {
		return
	}

	pan64, err := strconv.ParseUint(pan, 10, 16)
	if err != nil {
		return ob, false
	}
	pan16 := uint16(pan64)

	ob.MAC = mac
	ob.PAN = pan16
	return ob, true
}

func getAttributeFromMap(att map[string]string) (attInfo AttributeInfo, ok bool) {
	profile, ok := att[nameProfileID]
	if !ok {
		return
	}
	profileint, err := strconv.ParseUint(profile, 10, 16)
	if err != nil {
		return attInfo, false
	}
	profile16 := uint16(profileint)

	cluster, ok := att[nameClusterID]
	if !ok {
		return
	}
	clusterint, err := strconv.ParseUint(cluster, 10, 16)
	if err != nil {
		return attInfo, false
	}
	cluster16 := uint16(clusterint)

	at, ok := att[nameAttributeID]
	if !ok {
		return
	}
	atint, err := strconv.ParseUint(at, 10, 16)
	if err != nil {
		return attInfo, false
	}
	at16 := uint16(atint)

	valueType, ok := att[nameValueType]
	if !ok {
		return
	}
	typeInt, err := strconv.ParseUint(valueType, 10, 8)
	if err != nil {
		return attInfo, false
	}
	vltp := uint8(typeInt)

	attInfo.ProfileID = profile16
	attInfo.ClusterID = cluster16
	attInfo.AttributeID = at16
	attInfo.ValueType = vltp

	return attInfo, true
}

func (oc *objectCache) Lock() {
	oc.mutex.Lock()
}

func (oc *objectCache) Unlock() {
	oc.mutex.Unlock()
}

func (oc *objectCache) UpdateObjectWhithoutSync(d models.Device) {
	id := d.Id
	profile := d.Profile
	oldName, ok := oc.idNameObject[id]
	if ok {
		delete(oc.nameIDObject, oldName)
	}
	oc.idNameObject[id] = d.Name
	oc.nameIDObject[d.Name] = d.Id

	if d.Profile.Name == managerProfileNameConst {
		oc.nameMasterDevice = d.Name
	}
	obAddr, ok := getObjectAddressFromProtocol(d.Protocols)
	if ok {
		oc.addrIDObjectMap[obAddr] = id
	}
	obInfo, ok := getObjectInfoFromProtocol(d.Protocols)
	if ok {
		oc.idInfoObjectMap[id] = obInfo
	}

	for _, res := range profile.DeviceResources {
		atInfo, ok := getAttributeFromMap(res.Attributes)
		if ok {
			oc.resAttMap[res.Name] = atInfo
			oc.attResMap[atInfo] = res
		}
	}
}

func (oc *objectCache) UpdateObject(d models.Device) {
	oc.mutex.Lock()
	defer oc.mutex.Unlock()

	oc.UpdateObjectWhithoutSync(d)
}

func (oc *objectCache) DeleteObject(nameObject string) {
	oc.mutex.Lock()
	defer oc.mutex.Unlock()

	id, ok := oc.nameIDObject[nameObject]
	if ok {
		delete(oc.nameIDObject, nameObject)
		delete(oc.idNameObject, id)
		obInfo, ok := oc.idInfoObjectMap[id]
		if ok {
			delete(oc.addrIDObjectMap, obInfo.ObjectAddress)
			delete(oc.idInfoObjectMap, id)
		}
	}
}

func (oc *objectCache) ConvertNameToIDObject(nameOb string) (string, bool) {
	oc.mutex.Lock()
	defer oc.mutex.Unlock()

	r, ok := oc.nameIDObject[nameOb]
	return r, ok
}

func (oc *objectCache) ConvertIDToNameObject(idOb string) (string, bool) {
	oc.mutex.Lock()
	defer oc.mutex.Unlock()

	r, ok := oc.idNameObject[idOb]
	return r, ok
}

func (oc *objectCache) ConvertAttToRes(a AttributeInfo) (models.DeviceResource, bool) {
	oc.mutex.Lock()
	defer oc.mutex.Unlock()

	r, ok := oc.attResMap[a]
	return r, ok
}

func (oc *objectCache) ConvertResToAtt(resName string) (AttributeInfo, bool) {
	oc.mutex.Lock()
	defer oc.mutex.Unlock()

	r, ok := oc.resAttMap[resName]
	return r, ok
}

func (oc *objectCache) ConvertAddrToIDObject(addr ObjectAddress) (string, bool) {
	oc.mutex.Lock()
	defer oc.mutex.Unlock()

	r, ok := oc.addrIDObjectMap[addr]
	return r, ok
}

func (oc *objectCache) ConvertIDToObjectInfo(id string) (ObjectInfo, bool) {
	oc.mutex.Lock()
	defer oc.mutex.Unlock()

	r, ok := oc.idInfoObjectMap[id]
	return r, ok
}

func (oc *objectCache) GetMasterDeviceName() string {
	oc.mutex.Lock()
	defer oc.mutex.Unlock()

	return oc.nameMasterDevice
}

// Init basic state for cache

func initCache() {
	initOnce.Do(func() {
		svc := sdk.RunningService()
		ds := svc.Devices()

		defaultSize := len(ds) * 2
		idNameObject := make(map[string]string, defaultSize)
		nameIDObject := make(map[string]string, defaultSize)
		resAttMap := make(map[string]AttributeInfo, len(ds))
		attResMap := make(map[AttributeInfo]models.DeviceResource, len(ds))
		addrIDObjectMap := make(map[ObjectAddress]string, defaultSize)
		idInfoObjectMap := make(map[string]ObjectInfo, defaultSize)

		oc = &objectCache{
			nameIDObject:     nameIDObject,
			idNameObject:     idNameObject,
			resAttMap:        resAttMap,
			attResMap:        attResMap,
			addrIDObjectMap:  addrIDObjectMap,
			idInfoObjectMap:  idInfoObjectMap,
			nameMasterDevice: "",
		}
		for _, d := range ds {
			oc.UpdateObject(d)
		}
	})
}

func Cache() ObjectCache {
	if oc == nil {
		initCache()
	}
	return oc
}
