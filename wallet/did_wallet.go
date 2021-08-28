package wallet

import (
	"bytes"
	"sync"

	"github.com/dappley/go-dappley/core/account"
	accountpb "github.com/dappley/go-dappley/core/account/pb"
	"github.com/dappley/go-dappley/storage"
	laccountpb "github.com/dappley/go-dappley/wallet/pb"
	"github.com/golang/protobuf/proto"
	logger "github.com/sirupsen/logrus"
)

type DIDManager struct {
	SystemAddress account.Address
	DIDSets       []*account.DIDSet
	fileLoader    storage.FileStorage
	PassPhrase    []byte
	mutex         sync.Mutex
}

const didDataPath = "../bin/did.dat"

//GetDIDFilePath return account file Path
func GetDIDFilePath() string {
	createWalletFile(didDataPath)
	return didDataPath
}

func NewDIDManager(fileLoader storage.FileStorage) *DIDManager {
	return &DIDManager{
		fileLoader: fileLoader,
	}
}

func (dm *DIDManager) LoadFromFile() error {
	dm.mutex.Lock()
	defer dm.mutex.Unlock()
	fileContent, err := dm.fileLoader.ReadFromFile()

	if err != nil {
		return err
	}
	dmProto := &laccountpb.DIDManager{}
	err = proto.Unmarshal(fileContent, dmProto)

	if err != nil {
		return err
	}

	dm.FromProto(dmProto)
	return nil
}

func (dm *DIDManager) IsEmpty() bool {
	return len(dm.DIDSets) == 0
}

// SaveDIDsToFile saves DIDs to a file
func (dm *DIDManager) SaveDIDsToFile() {
	var content bytes.Buffer
	dm.mutex.Lock()
	defer dm.mutex.Unlock()
	rawBytes, err := proto.Marshal(dm.ToProto())
	if err != nil {
		logger.WithError(err).Error("AccountManager: Save account to file failed")
		return
	}
	content.Write(rawBytes)
	dm.fileLoader.SaveToFile(content)
}

func (dm *DIDManager) AddDID(did *account.DIDSet) {
	dm.mutex.Lock()
	dm.DIDSets = append(dm.DIDSets, did)
	dm.mutex.Unlock()
}

func (dm *DIDManager) ToProto() proto.Message {
	pbDIDs := []*accountpb.DIDSet{}
	for _, did := range dm.DIDSets {
		pbDIDs = append(pbDIDs, did.ToProto().(*accountpb.DIDSet))
	}

	sysAddr := &accountpb.Address{
		Address: dm.SystemAddress.String(),
	}

	return &laccountpb.DIDManager{
		Dids:          pbDIDs,
		PassPhrase:    dm.PassPhrase,
		SystemAddress: sysAddr,
	}
}

func (dm *DIDManager) FromProto(pb proto.Message) {
	didSets := []*account.DIDSet{}
	for _, didPb := range pb.(*laccountpb.DIDManager).Dids {
		didSet := &account.DIDSet{}
		didSet.FromProto(didPb)
		didSets = append(didSets, didSet)
	}

	dm.DIDSets = didSets
	dm.PassPhrase = pb.(*laccountpb.DIDManager).PassPhrase
	address := account.Address{}
	if pb.(*laccountpb.DIDManager).SystemAddress != nil {
		address.FromProto(pb.(*laccountpb.DIDManager).SystemAddress)
	}
	dm.SystemAddress = address
}
