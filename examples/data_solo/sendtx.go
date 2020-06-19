
package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"github.com/hyperledger/fabric/bccsp/factory"
	"github.com/hyperledger/fabric/common/crypto"
	"github.com/hyperledger/fabric/common/localmsp"
	"github.com/hyperledger/fabric/common/tools/protolator"
	"github.com/hyperledger/fabric/common/util"
	"github.com/hyperledger/fabric/msp"
	"github.com/hyperledger/fabric/msp/mgmt"
	"github.com/hyperledger/fabric/peer/chaincode"
	"github.com/hyperledger/fabric/peer/common"
	"github.com/hyperledger/fabric/peer/common/api"
	com "github.com/hyperledger/fabric/protos/common"
	"github.com/hyperledger/fabric/protos/orderer"
	pb "github.com/hyperledger/fabric/protos/peer"
	"github.com/hyperledger/fabric/protos/utils"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
	"gopkg.in/alecthomas/kingpin.v2"
	"math"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

var (
	oldest  = &orderer.SeekPosition{Type: &orderer.SeekPosition_Oldest{Oldest: &orderer.SeekOldest{}}}
	newest  = &orderer.SeekPosition{Type: &orderer.SeekPosition_Newest{Newest: &orderer.SeekNewest{}}}
	maxStop = &orderer.SeekPosition{Type: &orderer.SeekPosition_Specified{Specified: &orderer.SeekSpecified{Number: math.MaxUint64}}}

	app = kingpin.New("fabtool", "test fabric tps and send tx")
	send = app.Command("send", "send tx to fabric system")
	SendMspId = send.Flag("mspid", "mspid").Required().String()
	InvokeArgs = send.Flag("arg", "such as {\"Args\":[\"invoke\",\"a\",\"b\",\"1\"]}").Required().String()
	SendChannel = send.Flag("chan", "channel name").Required().String()
	CCName = send.Flag("cc", "chaincode name").Required().String()
	EndorsePeers = send.Flag("peer", "endorser_peer1;endorser_peer2").Required().String()
	SendCert = send.Flag("cert", "msp cert directory").Required().String()
	Orderer = send.Flag("orderer", "orderer address").Required().String()
	Routine = send.Flag("routine", "goroutine count").Default("1").Int()
	Count = send.Flag("count", "the num of txs each gou").Default("1").Int()

	monitor = app.Command("monitor", "monitor specified peer tps")
	MonitorPeer = monitor.Flag("peer", "monitor peer addr").Required().String()
	MonitorCert = monitor.Flag("cert", "msp cert directory").Required().String()
	MonitorMspId = monitor.Flag("mspid", "mspid").Required().String()
	MonitorChannel = monitor.Flag("chan", "channel name").Required().String()
)

var (
	localMsp msp.MSP
)

func main() {
	switch kingpin.MustParse(app.Parse(os.Args[1:])) {
	case send.FullCommand():
		concurrentSendTx(*Routine, *Count)
	case monitor.FullCommand():
		checkTps()
	}
}

func concurrentSendTx(rCount, sendCount int) {
	viper.Set("orderer.address", Orderer)
	opts := factory.GetDefaultOpts()
	opts.SwOpts.FileKeystore = &factory.FileKeystoreOpts{}
	err := mgmt.LoadLocalMspWithType(*SendCert, opts, *SendMspId, "bccsp")
	if err != nil {
		fmt.Println("LoadLocalMspWithType error :", err)
		return
	}
	localMsp = mgmt.GetLocalMSP()

	allStartTime := time.Now()
	var tmpEndTime time.Time
	tmpStartTime := time.Now()
	var preSuccessCount int32
	var preFailedCount int32
	successCount := int32(0)
	failedCount := int32(0)
	wg := sync.WaitGroup{}
	for i := 0; i < rCount; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			sendTx(sendCount, index*sendCount, &successCount, &failedCount)
		}(i)
	}

	for i:=0; i<6; i++{
		time.Sleep(1*time.Second)
		tmpEndTime = time.Now()
		curSuccessCount := successCount
		curFailedCount := failedCount
		interval := tmpEndTime.Sub(tmpStartTime).Seconds()
		fmt.Printf("interval %.2fs, successCount %d, failedCount %d\r\n",
			interval, curSuccessCount-preSuccessCount, curFailedCount-preFailedCount)
		if curSuccessCount != preSuccessCount || curFailedCount != preFailedCount {
			i = 0
		}
		tmpStartTime = tmpEndTime
		preSuccessCount = curSuccessCount
		preFailedCount = curFailedCount
	}

	wg.Wait()
	fmt.Printf("send over, success count:%d, failed count:%d, totalUsedTime:%.2fs\r\n",
		successCount, failedCount,time.Now().Sub(allStartTime).Seconds()-5.0)
}

func sendTx(count ,startIndex int, successNum, failedNum *int32) {
	signer, err := localMsp.GetDefaultSigningIdentity()
	if err != nil {
		fmt.Println("GetDefaultSigningIdentity error : ", err)
		return
	}

	clis, err := GetAllClients(*EndorsePeers)
	if err != nil {
		fmt.Println("GetAllClients error : ", err)
		return
	}
	var tlsCert tls.Certificate

	for i:=0; i<count; i++ {
		//构造交易
		//tmpInvokeArgs := fmt.Sprintf("{\"Args\":[\"invoke\",\"%d\",\"0\"]}", i+startIndex)
		//spec, err := getChaincodeSpec(tmpInvokeArgs)
		spec, err := getChaincodeSpec(*InvokeArgs)
		if err != nil {
			fmt.Println("getChaincodeSpec error : ", err)
			return
		}

		resp, err := chaincode.ChaincodeInvokeOrQuery(spec, *SendChannel, "", true, signer, tlsCert, clis.endorser, clis.deliverClients, clis.broadClient)

		if err != nil {
			fmt.Println("ChaincodeInvokeOrQuery error :", err)
			atomic.AddInt32(failedNum, 1)
		} else if resp.Response == nil {
			fmt.Println("resp.Response is nil, err :", err)
			atomic.AddInt32(failedNum, 1)
		} else if resp.Response.Status != 200 {
			fmt.Println("tx index ", i, ", response:", resp.Response.Status, resp.Response.Message)
			atomic.AddInt32(failedNum, 1)
		} else {
			atomic.AddInt32(successNum, 1)
		}
	}
}

type chaincodeInput struct {
	pb.ChaincodeInput
}

// UnmarshalJSON converts the string-based REST/JSON input to
// the []byte-based current ChaincodeInput structure.
func (c *chaincodeInput) UnmarshalJSON(b []byte) error {
	sa := struct {
		Function string
		Args     []string
	}{}
	err := json.Unmarshal(b, &sa)
	if err != nil {
		return err
	}
	allArgs := sa.Args
	if sa.Function != "" {
		allArgs = append([]string{sa.Function}, sa.Args...)
	}
	c.Args = util.ToChaincodeArgs(allArgs...)
	return nil
}

// getChaincodeSpec get chaincode spec from the cli cmd pramameters
func getChaincodeSpec(chaincodeCtorJSON string) (*pb.ChaincodeSpec, error) {
	spec := &pb.ChaincodeSpec{}

	//chaincodeCtorJSON = "{\"Args\":[\"invoke\",\"a\",\"b\",\"1\"]}"
	// Build the spec
	input := chaincodeInput{}
	if err := json.Unmarshal([]byte(chaincodeCtorJSON), &input); err != nil {
		return spec, errors.Wrap(err, "chaincode argument error")
	}

	chaincodeLang := strings.ToUpper("golang")
	spec = &pb.ChaincodeSpec{
		Type:        pb.ChaincodeSpec_Type(pb.ChaincodeSpec_Type_value[chaincodeLang]),
		ChaincodeId: &pb.ChaincodeID{Path: "", Name: *CCName, Version: ""},
		Input:       &input.ChaincodeInput,
	}
	return spec, nil
}

type ChaincodeClients struct {
	endorser []pb.EndorserClient
	deliverClients []api.PeerDeliverClient
	broadClient common.BroadcastClient
}

func GetAllClients(peerAddrs string) (*ChaincodeClients, error) {
	var err error
	var endorserClients []pb.EndorserClient
	var deliverClients []api.PeerDeliverClient

	peerAddresses := strings.Split(peerAddrs, ";")

	for _, address := range peerAddresses {
		var tlsRootCertFile string

		endorserClient, err := common.GetEndorserClientFnc(address, tlsRootCertFile)
		if err != nil {
			return nil, errors.Wrapf(err, "error getting endorser client for %s", peerAddrs)
		}
		endorserClients = append(endorserClients, endorserClient)
	}
	if len(endorserClients) == 0 {
		return nil, errors.New("no endorser clients retrieved - this might indicate a bug")
	}

	var broadcastClient common.BroadcastClient
	broadcastClient, err = common.GetBroadcastClientFnc()

	if err != nil {
		return nil, errors.WithMessage(err, "error getting broadcast client")
	}

	return &ChaincodeClients{endorser:endorserClients, deliverClients:deliverClients, broadClient:broadcastClient}, nil
}

type deliverClient interface {
	Send(envelope *com.Envelope) error
	Recv() (*pb.DeliverResponse, error)
}

type eventsClient struct {
	client      deliverClient
	signer      crypto.LocalSigner
	tlsCertHash []byte
}

func (r *eventsClient) readEventsStream() {
	allCount := 0
	startTime := time.Now()
	totalCount := 0
	maxTps := float64(0)

	for {
		msg, err := r.client.Recv()
		if err != nil {
			fmt.Println("Error receiving:", err)
			return
		}

		switch t := msg.Type.(type) {
		case *pb.DeliverResponse_Status:
			fmt.Println("Got status ", t)
			return
		case *pb.DeliverResponse_Block:
			fmt.Println("Received block: ")
			err := protolator.DeepMarshalJSON(os.Stdout, t.Block)
			if err != nil {
				fmt.Printf("  Error pretty printing block: %s", err)
			}
		case *pb.DeliverResponse_FilteredBlock:
			txCount := len(t.FilteredBlock.FilteredTransactions)
			allCount += txCount
			//fmt.Println("Received filtered block: ", t.FilteredBlock.Number, txCount)
			totalCount += txCount
			tmpTime := time.Now()
			if tmpTime.Sub(startTime).Seconds() >= 1.0 || t.FilteredBlock.Number == 166{
				interval := tmpTime.Sub(startTime).Seconds()
				curTps := float64(totalCount) / interval
				if curTps > maxTps {
					maxTps = curTps
				}
				fmt.Printf("curTps %.3f, maxTps %.3f, totalTx %d, time %f\n", curTps, maxTps, allCount, tmpTime.Sub(startTime).Seconds())
				startTime = tmpTime
				totalCount = 0
			}
			//err := protolator.DeepMarshalJSON(os.Stdout, t.FilteredBlock)
			//if err != nil {
			// fmt.Printf("  Error pretty printing filtered block: %s", err)
			//}
		}
	}
}

func (r *eventsClient) seekHelper(start *orderer.SeekPosition, stop *orderer.SeekPosition) *com.Envelope {
	env, err := utils.CreateSignedEnvelopeWithTLSBinding(com.HeaderType_DELIVER_SEEK_INFO, *MonitorChannel, r.signer, &orderer.SeekInfo{
		Start:    start,
		Stop:     stop,
		Behavior: orderer.SeekInfo_BLOCK_UNTIL_READY,
	}, 0, 0, r.tlsCertHash)
	if err != nil {
		panic(err)
	}
	return env
}

func (r *eventsClient) seekOldest() error {
	return r.client.Send(r.seekHelper(oldest, maxStop))
}

func (r *eventsClient) seekNewest() error {
	return r.client.Send(r.seekHelper(newest, maxStop))
}

func checkTps() {
	opts := factory.GetDefaultOpts()
	opts.SwOpts.FileKeystore = &factory.FileKeystoreOpts{}
	err := mgmt.LoadLocalMspWithType(*MonitorCert, opts, *MonitorMspId, "bccsp")
	if err != nil {
		fmt.Println("LoadLocalMspWithType error :", err)
		return
	}
	localMsp = mgmt.GetLocalMSP()

	dcs, err := GetDeliverClients(*MonitorPeer)
	if err != nil {
		fmt.Println("GetDeliverClients error :", err)
		return
	}
	if len(dcs) == 0 {
		fmt.Println("there is no deliverClients")
		return
	}

	client, err := dcs[0].DeliverFiltered(context.Background())
	if err != nil {
		fmt.Println("create DeliverFiltered error :", err)
		return
	}

	event := eventsClient{client:client, signer:localmsp.NewSigner()}
	err = event.seekOldest()
	if err != nil {
		fmt.Println("seek error : ", err)
		return
	}
	event.readEventsStream()

}

func GetDeliverClients(peerAddress string) ([]api.PeerDeliverClient, error) {
	var deliverClients []api.PeerDeliverClient
	peerAddresses := strings.Split(peerAddress, ";")
	for _, address := range peerAddresses {
		var tlsRootCertFile string

		deliverClient, err := common.GetPeerDeliverClientFnc(address, tlsRootCertFile)
		if err != nil {
			return nil, fmt.Errorf("error getting deliver client error %s", err.Error())
		}
		deliverClients = append(deliverClients, deliverClient)
	}
	return deliverClients, nil
}
