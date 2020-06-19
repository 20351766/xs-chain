export CHANNEL_NAME=firstchannel
export MYCC=mycc1
export VERSION=v1
peer channel create -o orderer.example.com:7050 -c $CHANNEL_NAME -f ./channel-artifacts/$CHANNEL_NAME.tx
peer channel join -b $CHANNEL_NAME.block
peer chaincode install -n $MYCC -v $VERSION -p github.com/hyperledger/fabric/examples/chaincode/go/chaincode_simple
peer chaincode instantiate -o orderer.example.com:7050 -C $CHANNEL_NAME -n $MYCC -v $VERSION -c '{"Args":["init","a","100","b","200"]}' -P "OR ('Org1MSP.peer','Org2MSP.peer')"


export CHANNEL_NAME=firstchannel
export MYCC=mycc
peer chaincode query -C $CHANNEL_NAME -n $MYCC -c '{"Args":["query","a"]}'
peer chaincode invoke -o orderer.example.com:7050 -C $CHANNEL_NAME -n $MYCC --peerAddresses peer0.org1.example.com:7051 -c '{"Args":["invoke","c","1"]}'
peer chaincode query -C $CHANNEL_NAME -n $MYCC -c '{"Args":["query","c"]}'
