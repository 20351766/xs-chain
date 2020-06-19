#!/bin/bash +x
#
# Copyright IBM Corp. All Rights Reserved.
#
# SPDX-License-Identifier: Apache-2.0
#


#set -e

export FABRIC_ROOT=$PWD/../..
export FABRIC_CFG_PATH=$PWD
echo


## Generates Org certs using cryptogen tool
function generateCerts (){
    which cryptogen
    if [ "$?" -ne 0 ]; then
      echo "cryptogen tool not found. exiting"
      exit 1
    fi

	CRYPTOGEN=cryptogen

	echo
	echo "##########################################################"
	echo "##### Generate certificates using cryptogen tool #########"
	echo "##########################################################"
	$CRYPTOGEN generate --config=./crypto-config.yaml
	echo
}

#function generateIdemixMaterial (){
#    which idemixgen
#    if [ "$?" -ne 0 ]; then
#      echo "idemixgen tool not found. exiting"
#      exit 1
#    fi
#
#	IDEMIXGEN=idemixgen
#	CURDIR=`pwd`
#	IDEMIXMATDIR=$CURDIR/crypto-config/idemix
#
#	echo
#	echo "####################################################################"
#	echo "##### Generate idemix crypto material using idemixgen tool #########"
#	echo "####################################################################"
#
#	mkdir -p $IDEMIXMATDIR
#	cd $IDEMIXMATDIR
#
#	# Generate the idemix issuer keys
#	$IDEMIXGEN ca-keygen
#
#	# Generate the idemix signer keys
#	$IDEMIXGEN signerconfig -u OU1 -e OU1 -r 1
#
#	cd $CURDIR
#}

## Generate orderer genesis block , channel configuration transaction and anchor peer update transactions
function generateChannelArtifacts() {
    which configtxgen
    if [ "$?" -ne 0 ]; then
      echo "configtxgen tool not found. exiting"
      exit 1
    fi

	CONFIGTXGEN=configtxgen

	echo "##########################################################"
	echo "#########  Generating Orderer Genesis block ##############"
	echo "##########################################################"
	# Note: For some unknown reason (at least for now) the block file can't be
	# named orderer.genesis.block or the orderer will fail to launch!
	$CONFIGTXGEN -profile TwoOrgsOrdererGenesis -channelID e2e-orderer-syschan -outputBlock ./channel-artifacts/genesis.block

	echo
	echo "########################################################################################################3###"
	echo "### Generating channel configuration transaction 'firstchannel.tx' ###"
	echo "#################################################################"
	$CONFIGTXGEN -profile TwoOrgsChannel -outputCreateChannelTx ./channel-artifacts/firstchannel.tx -channelID firstchannel

	echo
	echo "#################################################################"
	echo "#######    Generating anchor peer update for Org1MSP   ##########"
	echo "#################################################################"
	$CONFIGTXGEN -profile TwoOrgsChannel -outputAnchorPeersUpdate ./channel-artifacts/Org1MSPanchorsFirst.tx -channelID firstchannel -asOrg Org1MSP

	echo
	echo "#################################################################"
	echo "#######    Generating anchor peer update for Org2MSP   ##########"
	echo "#################################################################"
	$CONFIGTXGEN -profile TwoOrgsChannel -outputAnchorPeersUpdate ./channel-artifacts/Org2MSPanchorsFirst.tx -channelID firstchannel -asOrg Org2MSP

    echo
	echo "############################################################################################################"
	echo "### Generating channel configuration transaction 'secondchannel.tx' ###"
	echo "#################################################################"
	$CONFIGTXGEN -profile TwoOrgsChannel -outputCreateChannelTx ./channel-artifacts/secondchannel.tx -channelID secondchannel

	echo
	echo "#################################################################"
	echo "#######    Generating anchor peer update for Org1MSP   ##########"
	echo "#################################################################"
	$CONFIGTXGEN -profile TwoOrgsChannel -outputAnchorPeersUpdate ./channel-artifacts/Org1MSPanchorsSecond.tx -channelID secondchannel -asOrg Org1MSP

	echo
	echo "#################################################################"
	echo "#######    Generating anchor peer update for Org2MSP   ##########"
	echo "#################################################################"
	$CONFIGTXGEN -profile TwoOrgsChannel -outputAnchorPeersUpdate ./channel-artifacts/Org2MSPanchorsSecond.tx -channelID secondchannel -asOrg Org2MSP
}

function resetAritificialDir() {
	rm -rf channel-artifacts/*
	rm -rf crypto-config
}
resetAritificialDir
generateCerts
#generateIdemixMaterial
generateChannelArtifacts
