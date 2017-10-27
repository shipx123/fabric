/*
Licensed to the Apache Software Foundation (ASF) under one
or more contributor license agreements.  See the NOTICE file
distributed with this work for additional information
regarding copyright ownership.  The ASF licenses this file
to you under the Apache License, Version 2.0 (the
"License"); you may not use this file except in compliance
with the License.  You may obtain a copy of the License at

  http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing,
software distributed under the License is distributed on an
"AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
KIND, either express or implied.  See the License for the
specific language governing permissions and limitations
under the License.
*/

// ====CHAINCODE EXECUTION SAMPLES (CLI) ==================

// ==== Invoke logs ====
// peer chaincode invoke -C mychannel -n mycc -c '{"Args":["uploadLog","msc_20170613","900150983cd24fb0d6963f7d28e16f72","tom"]}'
// peer chaincode invoke -C mychannel -n mycc -c '{"Args":["deleteLog","msc_20170613"]}'

// ==== Query logs ====
// peer chaincode query -C mychannel -n mycc -c '{"Args":["readLog","msc_20170613"]}'

// Rich Query (Only supported if CouchDB is used as state database):
//   peer chaincode query -C mychannel -n mycc -c '{"Args":["queryLogsByUser","tom"]}'

//The following examples demonstrate creating indexes on CouchDB
//Example hostname:port configurations
//
//Docker or vagrant environments:
// http://couchdb:5984/
//
//Inside couchdb docker container
// http://127.0.0.1:5984/


package main

import (
	"encoding/json"
	"fmt"
	"bytes"
	"time"
	
	"github.com/hyperledger/fabric/core/chaincode/shim"
	pb "github.com/hyperledger/fabric/protos/peer"
)

// SimpleChaincode example simple Chaincode implementation
type SimpleChaincode struct {
}

type log struct {
	Name           string `json:"name"`    //the fieldtags are needed to keep case from bouncing around
	Logcontent      string `json:"logContent"`
	Uploadtime	   string `json:"uploadTime"`
	User           string `json:"user"`
}

// ===================================================================================
// Main
// ===================================================================================
func main() {
	err := shim.Start(new(SimpleChaincode))
	if err != nil {
		fmt.Printf("Error starting Simple chaincode: %s", err)
	}
}

// Init initializes chaincode
// ===========================
func (t *SimpleChaincode) Init(stub shim.ChaincodeStubInterface) pb.Response {
	return shim.Success(nil)
}

// Invoke - Our entry point for Invocations
// ========================================
func (t *SimpleChaincode) Invoke(stub shim.ChaincodeStubInterface) pb.Response {
	function, args := stub.GetFunctionAndParameters()
	fmt.Println("invoke is running " + function)

	// Handle different functions
	if function == "uploadLog" { //create a new log
		return t.uploadLog(stub, args)
	} else if function == "deleteLog" { //delete a log
		return t.deleteLog(stub, args)
	} else if function == "readLog" { //read a log
		return t.readLog(stub, args)
	} else if function == "queryLogsByUser" { //find log for user X using rich query
		return t.queryLogsByUser(stub, args)
	}

	fmt.Println("invoke did not find func: " + function) //error
	return shim.Error("Received unknown function invocation")
}

// ============================================================
// uploadLog - create a new log, store into chaincode state
// ============================================================
func (t *SimpleChaincode) uploadLog(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	var err error

	//        0                          1                     2 
	// "msc_20170613",  "900150983cd24fb0d6963f7d28e16f72",  "tom"
	if len(args) != 3 {
		return shim.Error("Incorrect number of arguments. Expecting 3")
	}

	// ==== Input sanitation ====
	fmt.Println("- start upload log")
	if len(args[0]) <= 0 {
		return shim.Error("1st argument must be a non-empty string")
	}
	if len(args[1]) <= 0 {
		return shim.Error("2nd argument must be a non-empty string")
	}
	if len(args[2]) <= 0 {
		return shim.Error("3rd argument must be a non-empty string")
	}

	logName := args[0]
	logContent := args[1]
	user := args[2]
	
	// ==== Check if log already exists ====
	logAsBytes, err := stub.GetState(logName)
	if err != nil {
		return shim.Error("Failed to get log: " + err.Error())
	} else if logAsBytes != nil {
		fmt.Println("This log already exists: " + logName)
		return shim.Error("This log already exists: " + logName)
	}
	
	utcNow := time.Now()
	utcTimestamp := utcNow.Unix()
	cstTimestamp := utcTimestamp+28800
	cstNow := time.Unix(cstTimestamp,0)
	uploadTime := cstNow.Format("2006-01-02 15:04:05")
	log := &log{logName, logContent, uploadTime, user}
	logJSONasBytes, err := json.Marshal(log)
	if err != nil {
		return shim.Error(err.Error())
	}
	//Alternatively, build the log json string manually if you don't want to use struct marshalling
	//logJSONasString := `{"name": "` + logName + `",  "logContent": "` + logContent + `",  "time": "` + uploadTime + `",  "user": "` + user + `"}`
	//logJSONasBytes := []byte(str)

	// === Save log to state ===
	err = stub.PutState(logName, logJSONasBytes)
	if err != nil {
		return shim.Error(err.Error())
	}

	//  An 'index' is a normal key/value entry in state.
	//  The key is a composite key, with the elements that you want to range query on listed first.
	//  In our case, the composite key is based on indexName~user~name.
	//  This will enable very efficient state range queries based on composite keys matching indexName~user~*
	indexName := "user~name"
	userNameIndexKey, err := stub.CreateCompositeKey(indexName, []string{log.User, log.Name})	
	if err != nil {
		return shim.Error(err.Error())
	}
	//  Save index entry to state. Only the key name is needed, no need to store a duplicate copy of the log.
	//  Note - passing a 'nil' value will effectively delete the key from state, therefore we pass null character as value
	value := []byte{0x00}
	stub.PutState(userNameIndexKey, value)
	fmt.Println("- end upload log")
	return shim.Success(nil)
}

// ===============================================
// readlog - read a log from chaincode state
// ===============================================
func (t *SimpleChaincode) readLog(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	var name, jsonResp string
	var err error

	if len(args) != 1 {
		return shim.Error("Incorrect number of arguments. Expecting name of the log to query")
	}

	name = args[0]
	valAsbytes, err := stub.GetState(name) //get the log from chaincode state
	if err != nil {
		jsonResp = "{\"Error\":\"Failed to get state for " + name + "\"}"
		return shim.Error(jsonResp)
	} else if valAsbytes == nil {
		jsonResp = "{\"Error\":\"log does not exist: " + name + "\"}"
		return shim.Error(jsonResp)
	}

	return shim.Success(valAsbytes)
}

// =======Rich queries =========================================================================
// Two examples of rich queries are provided below (parameterized query and ad hoc query).
// Rich queries pass a query string to the state database.
// Rich queries are only supported by state database implementations
//  that support rich query (e.g. CouchDB).
// The query string is in the syntax of the underlying state database.
// With rich queries there is no guarantee that the result set hasn't changed between
//  endorsement time and commit time, aka 'phantom reads'.
// Therefore, rich queries should not be used in update transactions, unless the
// application handles the possibility of result set changes between endorsement and commit time.
// Rich queries can be used for point-in-time queries against a peer.
// ============================================================================================

// ===== Example: Parameterized rich query =================================================
// queryLogsByUser queries for logs based on a passed in user.
// This is an example of a parameterized query where the query logic is baked into the chaincode,
// and accepting a single query parameter (user).
// Only available on state databases that support rich query (e.g. CouchDB)
// =========================================================================================
func (t *SimpleChaincode) queryLogsByUser(stub shim.ChaincodeStubInterface, args []string) pb.Response {

	//   0
	// "tom"
	if len(args) < 1 {
		return shim.Error("Incorrect number of arguments. Expecting 1")
	}

	user := args[0]

	queryString := fmt.Sprintf("{\"selector\":{\"user\":\"%s\"}}", user)

	queryResults, err := getQueryResultForQueryString(stub, queryString)
	if err != nil {
		return shim.Error(err.Error())
	}
	return shim.Success(queryResults)
}

// ==================================================
// delete - remove a log key/value pair from state
// ==================================================
func (t *SimpleChaincode) deleteLog(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	var jsonResp string
	var logJSON log
	if len(args) != 1 {
		return shim.Error("Incorrect number of arguments. Expecting 1")
	}
	logName := args[0]

	valAsbytes, err := stub.GetState(logName) //get the log from chaincode state
	if err != nil {
		jsonResp = "{\"Error\":\"Failed to get state for " + logName + "\"}"
		return shim.Error(jsonResp)
	} else if valAsbytes == nil {
		jsonResp = "{\"Error\":\"Log does not exist: " + logName + "\"}"
		return shim.Error(jsonResp)
	}

	err = json.Unmarshal([]byte(valAsbytes), &logJSON)
	if err != nil {
		jsonResp = "{\"Error\":\"Failed to decode JSON of: " + logName + "\"}"
		return shim.Error(jsonResp)
	}

	err = stub.DelState(logName) //remove the log from chaincode state
	if err != nil {
		return shim.Error("Failed to delete state:" + err.Error())
	}

	// maintain the index
	indexName := "user~name"
	userNameIndexKey, err := stub.CreateCompositeKey(indexName, []string{logJSON.User, logJSON.Name})
	if err != nil {
		return shim.Error(err.Error())
	}

	//  Delete index entry to state.
	err = stub.DelState(userNameIndexKey)
	if err != nil {
		return shim.Error("Failed to delete state:" + err.Error())
	}
	return shim.Success(nil)
}

// =========================================================================================
// getQueryResultForQueryString executes the passed in query string.
// Result set is built and returned as a byte array containing the JSON results.
// =========================================================================================
func getQueryResultForQueryString(stub shim.ChaincodeStubInterface, queryString string) ([]byte, error) {

	fmt.Printf("- getQueryResultForQueryString queryString:\n%s\n", queryString)

	resultsIterator, err := stub.GetQueryResult(queryString)
	if err != nil {
		return nil, err
	} 
	
	defer resultsIterator.Close()

	// buffer is a JSON array containing QueryRecords
	var buffer bytes.Buffer
	buffer.WriteString("[")

	bArrayMemberAlreadyWritten := false
	for resultsIterator.HasNext() {
		queryResponse, err := resultsIterator.Next()
		if err != nil {
			return nil, err
		}
		// Add a comma before array members, suppress it for the first array member
		if bArrayMemberAlreadyWritten == true {
			buffer.WriteString(",")
		}
		buffer.WriteString("{\"Key\":")
		buffer.WriteString("\"")
		buffer.WriteString(queryResponse.Key)
		buffer.WriteString("\"")

		buffer.WriteString(", \"Record\":")
		// Record is a JSON object, so we write as-is
		buffer.WriteString(string(queryResponse.Value))
		buffer.WriteString("}")
		bArrayMemberAlreadyWritten = true
	}
	buffer.WriteString("]")

	fmt.Printf("- getQueryResultForQueryString queryResult:\n%s\n", buffer.String())

	return buffer.Bytes(), nil
}

