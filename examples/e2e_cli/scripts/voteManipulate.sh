#!/bin/bash

PEER=${1}
FUNCTION=${2}
CHANNEL_NAME=mychannel

ORDERER_CA=/opt/gopath/src/github.com/hyperledger/fabric/peer/crypto/ordererOrganizations/example.com/orderers/orderer.example.com/msp/tlscacerts/tlsca.example.com-cert.pem

validateArgs(){
	if [ -z "${PEER}" ]; then
		echo "PEER NUMBER not mentioned, set to default 0"
		PEER=0
	fi
	if [ -z "${FUNCTION}" ]; then
		echo "FUNCTION NAME not mentioned, set to default 'getUserVote'"
		FUNCTION=getUserVote
	fi
}

setGlobals(){
	if [ $1 -eq 0 -o $1 -eq 1 ] ; then
		CORE_PEER_LOCALMSPID="Org1MSP"
		CORE_PEER_TLS_ROOTCERT_FILE=/opt/gopath/src/github.com/hyperledger/fabric/peer/crypto/peerOrganizations/org1.example.com/peers/peer0.org1.example.com/tls/ca.crt
		CORE_PEER_MSPCONFIGPATH=/opt/gopath/src/github.com/hyperledger/fabric/peer/crypto/peerOrganizations/org1.example.com/users/Admin@org1.example.com/msp
		if [ $1 -eq 0 ]; then
			CORE_PEER_ADDRESS=peer0.org1.example.com:7051
		else
			CORE_PEER_ADDRESS=peer1.org1.example.com:7051
		fi
	else
		CORE_PEER_LOCALMSPID="Org2MSP"
		CORE_PEER_TLS_ROOTCERT_FILE=/opt/gopath/src/github.com/hyperledger/fabric/peer/crypto/peerOrganizations/org2.example.com/peers/peer0.org2.example.com/tls/ca.crt
		CORE_PEER_MSPCONFIGPATH=/opt/gopath/src/github.com/hyperledger/fabric/peer/crypto/peerOrganizations/org2.example.com/users/Admin@org2.example.com/msp
		if [ $1 -eq 2 ]; then
			CORE_PEER_ADDRESS=peer0.org2.example.com:7051
		else
			CORE_PEER_ADDRESS=peer1.org2.example.com:7051
		fi
	fi
	env |grep CORE
}

chaincodeGetUserVote(){
	peer chaincode query -C $CHANNEL_NAME -n mycc -c '{"Args":["getUserVote"]}' >&log.txt
	echo
	cat log.txt
	if [ $? == 0 ] ; then
		echo "==========Query succeed=========="
	else
		echo "==========Query failed=========="
	fi
}

chaincodeGetHistory(){
	peer chaincode query -C $CHANNEL_NAME -n mycc -c '{"Args":["getHistory","'${1}'"]}' >&log.txt
	echo
	cat log.txt
	if [ $? -eq 0 ] ; then
		echo "==========GetHistory succeed=========="
	else
		echo "==========GetHistory failed=========="
	fi
}

chaincodeVoteUser(){
	if [ -z "$CORE_PEER_TLS_ENABLED" -o "$CORE_PEER_TLS_ENABLED" = "false" ] ; then
		peer chaincode invoke -C $CHANNEL_NAME -n mycc -c '{"Args":["voteUser","'${1}'"]}' >&log.txt
	else
		peer chaincode invoke --tls $CORE_PEER_CLS_ENABLED --cafile $ORDERER_CA -C $CHANNEL_NAME -n mycc -c '{"Args":["voteUser","'${1}'"]}' >&log.txt
	fi
	echo
	cat log.txt
	if [ $? -eq 0 ] ; then
		echo "==========Vote succeed=========="
	else
		echo "==========Vote failed=========="
	fi
}

chaincodeDelUser(){
	if [ -z "$CORE_PEER_TLS_ENABLED" -o "$CORE_PEER_TLS_ENABLED" = "false" ] ; then
		peer chaincode invoke -C $CHANNEL_NAME -n mycc -c '{"Args":["delUser","'${1}'"]}' >&log.txt
	else
		peer chaincode invoke --tls $CORE_PEER_CLS_ENABLED --cafile $ORDERER_CA -C $CHANNEL_NAME -n mycc -c '{"Args":["delUser","'${1}'"]}' >&log.txt
	fi
	echo
	cat log.txt
	if [ $? -eq 0 ] ; then
		echo "==========DelUser succeed=========="
	else
		echo "==========DelUser failed=========="
	fi
}


validateArgs
setGlobals $PEER
if [ "${FUNCTION}" == "getUserVote" ] ; then
	chaincodeGetUserVote
elif [ "${FUNCTION}" == "voteUser" ] ; then
	chaincodeVoteUser $3
elif [ "${FUNCTION}" == "getHistory" ] ; then
	chaincodeGetHistory $3
elif [ "${FUNCTION}" == "delUser" ] ; then
	chaincodeDelUser $3
fi
