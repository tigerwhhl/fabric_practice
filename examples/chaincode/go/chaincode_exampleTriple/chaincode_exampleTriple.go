package main

import(
	"fmt"
	"encoding/json"
	"encoding/pem"
	"crypto/x509"
	"bytes"
	"time"
	//"reflect"
	"strconv"
	"errors"
	"github.com/hyperledger/fabric/core/chaincode/shim"
	"github.com/hyperledger/fabric/protos/peer"
)


type TripleChaincode struct{
}

//Candidate Objects and Scores
type Value struct{
	Object string `json:"object"`
	Score float64 `json:"score"`
}

//Trace Information of Answer
type Trace struct{
	Object string `json:"object"`
	Supporter []string `json:"supporter"`
}

/*
Triple Structure
IsSolid Means No More Change
Committer Means Committer of this Log
IsSolid==1 return Answer ; IsSolid==0 return Values
*/
type Triple struct{
	Subject string `json:"subject"`
	Predicate string `json:"predicate"`
	IsSolid bool `json:"issolid"`
	Values []Value `json:"values"`
	Answer Trace `json:"answer"`
	Committer string `json:"committer"`
}

/*
Copy for Slice
Big Problem
Disaster
*/
func (triple *Triple) copy() Triple{
	newT := Triple{}
	newT.Subject = triple.Subject
	newT.Predicate = triple.Predicate
	newT.IsSolid = triple.IsSolid
	newT.Values = make([]Value,len(triple.Values))
	copy(newT.Values,triple.Values)
	newT.Answer = Trace{}
	newT.Answer.Object = triple.Answer.Object
	newT.Answer.Supporter =make([]string,len(triple.Answer.Supporter))
	copy(newT.Answer.Supporter,triple.Answer.Supporter)
	newT.Committer = triple.Committer
	return newT
}

/*
Invoked by CommitAnswer
Require Object and Score
Return result and whether solid
*/
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

/*
Check If Triple Satisfy the Condition to be Solid
If Number of Candidate Answers or Sum of Score >= Threshold -> Solid It
*/
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

/*
Invoke by CheckSolid
Solid the Answer and Supporter via Comparison of History(Log&Committer)
*/
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
		isFirst := true
		lastValue := Triple{}
		presentValue := Triple{}
		for historyInfo.HasNext(){
			queryResult,err := historyInfo.Next()
			if(err != nil){
				return //shim.Error(err.Error())
			}
			//lastValue := Triple{}
			//presentValue := Triple{}
			//lastValue = presentValue
			//copy(lastValue,presentValue)
			lastValue = presentValue.copy()
			err = json.Unmarshal(queryResult.Value,&presentValue)
			if err!=nil {
				return
			}
			if (isFirst){
				isFirst = false
				//lastValue = queryResult
				//supportMap[presentValue.Committer] = true
				if presentValue.Values[0].Object==triple.Answer.Object{
					supportMap[presentValue.Committer] = true
					//triple.Answer.Supporter = append(triple.Answer.Supporter,presentValue.Committer)
				}
				//supportMap[lastValue.Committer] = true
			}else{
				//supportMap[presentValue.Committer] = true
				logChange,err := compareTriple(lastValue,presentValue)
				//triple.Answer.Supporter = append(triple.Answer.Supporter,logChange)
				//supportMap[logChange] = true
				//triple.Answer.Supporter = append(triple.Answer.Supporter,logChange)
				//triple.Answer.Supporter = append(triple.Answer.Supporter,logChange)
				if ((err==nil)&&(logChange==triple.Answer.Object)){
					supportMap[presentValue.Committer] = true
				}
			}
			//supportMap[presentValue.Committer] = true
		}
		for  supportName := range supportMap{
			triple.Answer.Supporter = append(triple.Answer.Supporter,supportName)
		}
	}

}

func (t *TripleChaincode) Init(stub shim.ChaincodeStubInterface) peer.Response{
	return shim.Success(nil)
}

/*
Invoke Different Method
*/
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
//Something Wrong Here, Always Return Empty Result String
func compareTriple(last Triple,present Triple) (string,error){
	valueListL := last.Values
	valueListP := present.Values
	found := false
	result := "wtf"
	if len(valueListL)>len(valueListP){
		return "Something Wrong 1",errors.New("Last List Longer Than Present")
	}
	if len(valueListL)==len(valueListP){
		for i := range(valueListL){
			if (valueListL[i].Object!=valueListP[i].Object){
				//return "Something Wrong 2"+valueListL[i].Object+valueListP[i].Object,errors.New("Value List Changed")
				return "Something Wrong 2",errors.New("Value List Changed")
			}
			//result += valueListL[i].Object
			//result += reflect.TypeOf(valueListP[i].Score)
			//result += strconv.FormatFloat(valueListL[i].Score,'g',1,64)
			//result += strconv.FormatFloat(valueListP[i].Score,'g',1,64)
			//result += "\t"
			if (float64(valueListP[i].Score-valueListL[i].Score)>0.01){
				if(found){
					return "Something Wrong 3 MTOD",errors.New("More Than One Diff")
				}
				//return valueListL[i].Object,nil
				result = valueListL[i].Object
				found = true
			}
		}
		return result,nil
	}else if (len(valueListL)+1)==len(valueListP){
		result = valueListP[len(valueListL)].Object
		return result,nil
	}else{
		//return "Something Wrong 4 MTOD"+strconv.Itoa(len(valueListL))+" "+strconv.Itoa(len(valueListP)),errors.New("More Than One Diff")
		return "Something Wrong 4 MTOD",errors.New("More Than One Diff")
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
