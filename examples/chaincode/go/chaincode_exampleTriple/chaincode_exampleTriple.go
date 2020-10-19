package main

import(
	"fmt"
	"encoding/json"
	"bytes"
//	"time"
	"strconv"
	"github.com/hyperledger/fabric/core/chaincode/shim"
	"github.com/hyperledger/fabric/protos/peer"
)


type TripleChaincode struct{
}

/*
type VoteChaincode struct {
}
*/

type Value struct{
	Object string `json:"object"`
	Score float32 `json:"score"`
}

type Trace struct{
	Object string `json:"object"`
	Supporter []string `json:"supporter"`
}

type old_Triple struct{
	Subject string `json:"subject"`
	Predicate string `json:"predicate"`
	Values []Value `json:"values"`
}

type Triple struct{
	Subject string `json:"subject"`
	Predicate string `json:"predicate"`
	isSolid bool `json:"issolid"`
	Values []Value `json:"values"`
	Answer Trace `json:"answer"`
}

func (triple *Triple) old_addCommit(objectT string, scoreT float32){
	found := false
	for i := range (triple.Values){
		if triple.Values[i].Object == objectT{
			triple.Values[i].Score += scoreT
			found = true
			break
		}
	}
	if found == false{
		valueT := Value{objectT,scoreT}
		triple.Values = append(triple.Values,valueT)
	}
}

func (triple *Triple) addCommit(objectT string, scoreT float32) (string,bool){
	if triple.isSolid==true{
		result := triple.Predicate +" of "triple.Subject+" is solidified: "+triple.Answer.Object
		return result,true
	}
	found := false
	for i := range (triple.Values){
		if triple.Values[i].Object == objectT{
			triple.Values[i].Score += scoreT
			found = true
			break
		}
	}
	if found == false{
		valueT := Value{objectT,scoreT}
		triple.Values = append(triple.Values,valueT)
	}
	result := "succeed voting " +strconv.FormatFloat(scoreT,'E',-1,32)+ " to "+objectT
	return result,false
}

func (t *TripleChaincode) Init(stub shim.ChaincodeStubInterface) peer.Response{
	return shim.Success(nil)
}

/*
type Vote struct {
	Username string `json:"username"`
	Votenum int `json:"votenum"`
}
func (t *VoteChaincode) Init(stub shim.ChaincodeStubInterface) peer.Response {
	return shim.Success(nil)
}
func (t *VoteChaincode) Invoke(stub shim.ChaincodeStubInterface) peer.Response {

	fn , args := stub.GetFunctionAndParameters()

	if fn == "voteUser" {
		return t.voteUser(stub,args)
	} else if fn == "getUserVote" {
		return t.getUserVote(stub,args)
	} else if fn == "getHistory" {
		return t.getHistory(stub,args)
	} else if fn == "delUser" {
		return t.delUser(stub,args)
	}

	return shim.Error("Invoke 调用方法有误！")
}
*/
func (t *TripleChaincode) Invoke(stub shim.ChaincodeStubInterface) peer.Response{
	fn,args := stub.GetFunctionAndParameters()
	if fn=="commitAnswer"{
		return t.commitAnswer(stub,args)
	}else if fn=="queryAnswers"{
		return t.queryAnswers(stub,args)
	}
	return shim.Error("Invoke Wrong Method Name!")
}

/*
func (t *VoteChaincode) voteUser(stub shim.ChaincodeStubInterface , args []string) peer.Response{
	// 查询当前用户的票数，如果用户不存在则新添一条数据，如果存在则给票数加1
	fmt.Println("start voteUser")
	vote := Vote{}
	username := args[0]
	voteAsBytes, err := stub.GetState(username)
	if err != nil {
		shim.Error("voteUser 获取用户信息失败！")
	}

	if voteAsBytes != nil {
		err = json.Unmarshal(voteAsBytes, &vote)
		if err != nil {
			shim.Error(err.Error())
		}
		vote.Votenum += 1
	}else {
		vote = Vote{ Username: args[0], Votenum: 1}
	}
	//将 Vote 对象 转为 JSON 对象
	voteJsonAsBytes, err := json.Marshal(vote)
	if err != nil {
		shim.Error(err.Error())
	}
	err = stub.PutState(username,voteJsonAsBytes)
	if err != nil {
		shim.Error("voteUser 写入账本失败！")
	}
	fmt.Println("end voteUser")
	return shim.Success(nil)
}
*/

func (t *TripleChaincode) old_commitAnswer(stub shim.ChaincodeStubInterface, args []string) peer.Response{
	fmt.Println("Start Commit Answer")
	triple := Triple{}
	subjectT := args[0]
	predicateT := args[1]
	objectT := args[2]
	scoreT,err := strconv.ParseFloat(args[3],32)
//	key,err := stub.CreateCompositeKey(subjectT,[]string{predicateT})
	key,err := stub.CreateCompositeKey("SP",[]string{subjectT,predicateT})
	TripleAsBytes,err := stub.GetState(key)
	if err!=nil{
		shim.Error("Commit Error One")
	}
	if TripleAsBytes!=nil{
		err = json.Unmarshal(TripleAsBytes,&triple)
		if err!=nil{
			shim.Error("Commit Error Unmarshal")
		}
	}else{
		triple = Triple{subjectT,predicateT,[]Value{}}
	}
	triple.addCommit(objectT,float32(scoreT))
	TripleAsBytes,err = json.Marshal(triple)
	if err != nil{
		shim.Error("Commit Error Unmarshal")
	}
	err = stub.PutState(key,TripleAsBytes)
	if err != nil{
		shim.Error("Commit Error Writing")
	}
	fmt.Println("End Commit Answer")
	return shim.Success(nil)
}

func (t *TripleChaincode) commitAnswer(stub shim.ChaincodeStubInterface, args []string) peer.Response{
	fmt.Println("Start Commit Answer")
	triple := Triple{}
	subjectT := args[0]
	predicateT := args[1]
	objectT := args[2]
	scoreT,err := strconv.ParseFloat(args[3],32)
//	key,err := stub.CreateCompositeKey(subjectT,[]string{predicateT})
	key,err := stub.CreateCompositeKey("SP",[]string{subjectT,predicateT})
	TripleAsBytes,err := stub.GetState(key)
	if err!=nil{
		shim.Error("Commit Error One")
	}
	if TripleAsBytes!=nil{
		err = json.Unmarshal(TripleAsBytes,&triple)
		if err!=nil{
			shim.Error("Commit Error Unmarshal")
		}
	}else{
		triple = Triple{subjectT,predicateT,[]Value{}}
	}
	ans,solid := triple.addCommit(objectT,float32(scoreT))
	if solid==true{
		return shim.Success(result)
	}
	TripleAsBytes,err = json.Marshal(triple)
	if err != nil{
		shim.Error("Commit Error Unmarshal")
	}
	err = stub.PutState(key,TripleAsBytes)
	if err != nil{
		shim.Error("Commit Error Writing")
	}
	fmt.Println("End Commit Answer")
	return shim.Success(result)
}

func (t *TripleChaincode) queryAnswers(stub shim.ChaincodeStubInterface, args []string) peer.Response{
	fmt.Println("Start Query Answers")
	subjectT := args[0]
	predicateT := args[1]
//	logger := shim.NewLogger("myLogger")
//	logger.Info(subjectT+"\t"+predicateT)
//	return shim.Error(subjectT)
//	fmt.Printf("%s\t%s",subjectT,predicateT)
	key,err := stub.CreateCompositeKey("SP",[]string{subjectT,predicateT})
	TripleAsBytes,err := stub.GetState(key)
	if err != nil{
		shim.Error("Query Error One")
	}
	if TripleAsBytes==nil{
		shim.Error("Query No Such Record")
	}
	triple := Triple{}
	err = json.Unmarshal(TripleAsBytes,&triple)
	if err != nil{
		shim.Error("Commit Error Unmarshal")
	}
	var buffer bytes.Buffer
	if triple.isSolid==false{
		buffer.WriteString("[")
		buffer.WriteString("Subject: "+triple.Subject+", Predicate: "+triple.Predicate+"\n")
		isWritten := false
		for i := range triple.Values{
			if isWritten == true{
				buffer.WriteString(",")
			}
			buffer.WriteString(triple.Values[i].Object+":"+strconv.FormatFloat(float64(triple.Values[i].Score),'E',-2,64))
			isWritten = true
		}
		buffer.WriteString("]")
	}else{
		buffer.WriteString("[")
		//todo: write if issolid==true
	}
	fmt.Println("End Query Answers")
	return shim.Success(buffer.Bytes())
}

/*
func (t * VoteChaincode) delUser(stub shim.ChaincodeStubInterface, args []string) peer.Response{
	fmt.Println("start delUser")
	if len(args)!=1{
		return shim.Error("delUser expecting 1 parameter")
	}
	err := stub.DelState(args[0])
	if err != nil{
		return shim.Error("failed to delUser "+args[0])
	}
	return shim.Success(nil)
}
func (t *VoteChaincode) getHistory(stub shim.ChaincodeStubInterface, args []string) peer.Response{
	fmt.Println("start getHistory")
	userName := args[0]
	historyInfo,err := stub.GetHistoryForKey(userName)
	if err != nil{
		return shim.Error("getHistory failed")
	}
	if historyInfo == nil{
		return shim.Error("key not found")
	} else{
		defer historyInfo.Close()
		var buffer bytes.Buffer
		buffer.WriteString("[")
		isWritten := false
		for historyInfo.HasNext(){
			queryResult,err := historyInfo.Next()
			if err != nil{
				return shim.Error(err.Error())
			}
			if isWritten == true{
				buffer.WriteString("\n")
			}
			buffer.WriteString("TxID:"+queryResult.TxId+",")
			buffer.WriteString("Value:"+string(queryResult.Value)+",")
			buffer.WriteString("IsDelete:"+strconv.FormatBool(queryResult.IsDelete)+",")
			buffer.WriteString("Timestamp:"+time.Unix(queryResult.Timestamp.Seconds,int64(queryResult.Timestamp.Nanos)).String())
			isWritten = true
		}
		buffer.WriteString("]")
		fmt.Println("historyInfo of %s:\n%s\n",userName,buffer.String())
		return shim.Success(buffer.Bytes())
	}
}
func (t *VoteChaincode) getUserVote(stub shim.ChaincodeStubInterface, args []string) peer.Response{
	fmt.Println("start getUserVote")
	// 获取所有用户的票数
	resultIterator, err := stub.GetStateByRange("","")
	if err != nil {
		return shim.Error("获取用户票数失败！")
	}
	defer resultIterator.Close()

	var buffer bytes.Buffer
	buffer.WriteString("[")

	isWritten := false

	for resultIterator.HasNext() {
		queryResult , err := resultIterator.Next()
		if err != nil {
			return shim.Error(err.Error())
		}

		if isWritten == true {
			buffer.WriteString(",")
		}

		buffer.WriteString(string(queryResult.Value))
		isWritten = true
	}

	buffer.WriteString("]")

	fmt.Printf("查询结果：\n%s\n",buffer.String())
	fmt.Println("end getUserVote")
	return shim.Success(buffer.Bytes())
}
func main(){
	err := shim.Start(new(VoteChaincode))
	if err != nil {
		fmt.Println("vote chaincode start err")
	}
}
*/
func main(){
	err := shim.Start(new(TripleChaincode))
	if err != nil{
		fmt.Println("Triple Chaincode Start Error")
	}
}
