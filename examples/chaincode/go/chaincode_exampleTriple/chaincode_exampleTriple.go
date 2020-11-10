package main

import(
	"fmt"
	"encoding/json"
	"encoding/pem"
	"crypto/x509"
	"bytes"
	"time"
	"strconv"
	"errors"
	"github.com/hyperledger/fabric/core/chaincode/shim"
	"github.com/hyperledger/fabric/protos/peer"
)


type TripleChaincode struct{
}

type Value struct{
	Object string `json:"object"`
	Score float64 `json:"score"`
}

type Trace struct{
	Object string `json:"object"`
	Supporter []string `json:"supporter"`
}

type Triple struct{
	Subject string `json:"subject"`
	Predicate string `json:"predicate"`
	IsSolid bool `json:"issolid"`
	Values []Value `json:"values"`
	Answer Trace `json:"answer"`
	Committer string `json:"committer"`
}

func (triple *Triple) addCommit(stub shim.ChaincodeStubInterface, objectT string, scoreT float64) (string,bool){
	if (triple.IsSolid==true){
		result := triple.Predicate +" of "+triple.Subject+" is solidified: "+triple.Answer.Object
		return result,true
	}else{
		creatorByte,_ := stub.GetCreator()
		certStart := bytes.IndexAny(creatorByte,"-----BEGIN")
		if certStart == -1{
			result := "GetCreator Failed: No Cert Found"
			return result,true
		}
		certText := creatorByte[certStart:]
		bl,_ := pem.Decode(certText)
		if bl==nil{
			result := "GetCreator Failed: Decode PEM Failed"
			return result,true
		}
		cert, err := x509.ParseCertificate(bl.Bytes)
		if err != nil{
			result := "GetCreator Failed: ParseCert Failed"
			return result,true
		}
		uname := cert.Subject.CommonName
		triple.Committer = uname
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
	//	triple.checkSolid()
	//	triple.isSolid = true
		result := "succeed voting " +strconv.FormatFloat(scoreT,'E',-1,32)+ " to "+objectT
		return result,false
	}
}

func (triple * Triple) checkSolid(stub shim.ChaincodeStubInterface){
	solidScoreThreshold := 3.0
	solidObjectThreshold := 5
	if (triple.IsSolid==true){
		return
	}else{
		scoreSum := 0.0
		answerT := Trace{}
		maxScore := 0.0
		for i := range(triple.Values){
			scoreSum += triple.Values[i].Score
			if triple.Values[i].Score > maxScore{
				maxScore = triple.Values[i].Score
				answerT.Object = triple.Values[i].Object
			}
		}
		if ((scoreSum>=solidScoreThreshold) || (len(triple.Values)>=solidObjectThreshold)){
			triple.IsSolid = true
			triple.Answer = answerT
			triple.trackTrace(stub)
		}
	}
}

func (triple * Triple) trackTrace(stub shim.ChaincodeStubInterface){
	fmt.Println("Start TrackTrace")
	subjectT := triple.Subject
	predicateT := triple.Predicate
	key,err := stub.CreateCompositeKey("SP",[]string{subjectT,predicateT})
	historyInfo,err := stub.GetHistoryForKey(key)
	if err != nil{
		return //shim.Error("getHistory failed")
	}
	if historyInfo == nil{
		return //shim.Error("key not found")
	}else{
		defer historyInfo.Close()
		supportMap := make(map[string]bool)
		isFirst := true;
		for historyInfo.HasNext(){
			queryResult,err := historyInfo.Next()
			if(err != nil){
				return //shim.Error(err.Error())
			}
			lastValue := Triple{}
			err = json.Unmarshal(queryResult.Value,&lastValue)
			if err!=nil {
				return
			}
			if (isFirst){
				isFirst = false
				supportMap[lastValue.Committer] = true
			}else{
				supportMap[lastValue.Committer] = true
			}
		}
		for  supportName := range supportMap{
			triple.Answer.Supporter = append(triple.Answer.Supporter,supportName)
		}
	}

}

func (t *TripleChaincode) Init(stub shim.ChaincodeStubInterface) peer.Response{
	return shim.Success(nil)
}

func (t *TripleChaincode) Invoke(stub shim.ChaincodeStubInterface) peer.Response{
	fn,args := stub.GetFunctionAndParameters()
	if fn=="commitAnswer"{
		return t.commitAnswer(stub,args)
	}else if fn=="queryAnswers"{
		return t.queryAnswers(stub,args)
	}else if fn=="getHistory"{
		return t.getHistory(stub,args)
	}
	return shim.Error("Invoke Wrong Method Name!")
}

//Chaincode CommitAnswer
//Before Commit, Query and CheckSolid
//Invoke Triple.CheckSolid 
func (t *TripleChaincode) commitAnswer(stub shim.ChaincodeStubInterface, args []string) peer.Response{
	fmt.Println("Start Commit Answer")
	triple := Triple{}
	subjectT := args[0]
	predicateT := args[1]
	objectT := args[2]
	IsSolidT := false
	scoreT,err := strconv.ParseFloat(args[3],32)
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
		triple = Triple{subjectT,predicateT,IsSolidT,[]Value{},Trace{},"NoBody"}
	}
	if (triple.IsSolid==true){
		return shim.Success([]byte("Commit Failed Because Triple Is Solid"))
//		return shim.Success([]byte("what the mother fucker"))
	}
	result,solid := triple.addCommit(stub,objectT,float64(scoreT))
	if solid==true{
		return shim.Success([]byte(result))
	}
	triple.checkSolid(stub)
	if triple.IsSolid==true{
		result = "isSolid what the fuck\n"+result
	}
	TripleAsBytes,err = json.Marshal(triple)
	if err != nil{
		return shim.Error("Commit Error Unmarshal")
	}
	err = stub.PutState(key,TripleAsBytes)
	if err != nil{
		return shim.Error("Commit Error Writing")
	}
	fmt.Println("End Commit Answer")
	return shim.Success([]byte(result))
}

//Function for CheckSolid to Invoke
//Compare Neighbour Logs For Diff
//Reutrn Addition of Latter Log and Error Information
func compareTriple(Triple last,Triple present) string,error{
	valueListL := last.Values
	valueListP := present.Values
	found := false
	result := string{}
	if len(valueListL)>len(valueListP){
		return "Something Wrong",error.New("Present List Longer Than Last")
	}
	if len(valueListL)==len(valueListP){
		for i := range(valueListL){
			if(valueListL[i].Object!=valueListP[i].Object){
				return "Something Wrong",error.New("Value List Changed")
			}
			if(valueListP[i].Score-valueListL[i].Score>0.001){
				if(found){
					return "Something Wrong",error.New("More Than One Diff")
				}
				result = valueListL[i].Object
				found = true
			}
		}
		return result,nil
	}else if len(valueListL)+1==len(valueListP){
		result = valueListP[len(valueListL)]
		return result,nil
	}else{
		return "Something Wrong",error.New("More Than One Diff")
	}
}

//Chaincode QueryAnswers to Respond to Query on Triples of Detailed Subject&Predicate
//If Solid, Return Answer&Supporter
//If Not, Return AnswerList&Score
func (t *TripleChaincode) queryAnswers(stub shim.ChaincodeStubInterface, args []string) peer.Response{
	fmt.Println("Start Query Answers")
	subjectT := args[0]
	predicateT := args[1]
	key,err := stub.CreateCompositeKey("SP",[]string{subjectT,predicateT})
	TripleAsBytes,err := stub.GetState(key)
	if err != nil{
		return shim.Error("Error Caused Query CompositeKey")
	}
	if TripleAsBytes==nil{
		return shim.Success([]byte("Query No Such Record"))
	}
	triple := Triple{}
	err = json.Unmarshal(TripleAsBytes,&triple)
	if err != nil{
		return shim.Error("Query Error Unmarshal")
	}
	var buffer bytes.Buffer
	if triple.IsSolid==false{
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
		buffer.WriteString("Subject: "+triple.Subject+", Predicate: "+triple.Predicate+", Object: "+triple.Answer.Object+", Supporter: ")
		isFirst := true
		for i := range triple.Answer.Supporter{
			if isFirst{
				buffer.WriteString(triple.Answer.Supporter[i])
				isFirst = false;
			}else{
				buffer.WriteString(", "+triple.Answer.Supporter[i])
			}
		}
		buffer.WriteString(" ]")
	}
	fmt.Println("End Query Answers")
	return shim.Success(buffer.Bytes())
}

//Chaincode GetHistory to Test GetHistoryForKey Function, Not In Use Currently
func (t *TripleChaincode) getHistory(stub shim.ChaincodeStubInterface, args []string) peer.Response{
	fmt.Println("Start GetHistory")
	subjectT := args[0]
	predicateT := args[1]
	key,err := stub.CreateCompositeKey("SP",[]string{subjectT,predicateT})
	historyInfo,err := stub.GetHistoryForKey(key)
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
		fmt.Println("historyInfo of %s of %s:\n%s\n",predicateT,subjectT,buffer.String())
		return shim.Success(buffer.Bytes())
	}
}

//Instantiate chaincode
func main(){
	err := shim.Start(new(TripleChaincode))
	if err != nil{
		fmt.Println("Triple Chaincode Start Error")
	}
}
