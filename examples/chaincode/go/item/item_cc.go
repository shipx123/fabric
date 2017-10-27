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

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/hyperledger/fabric/core/chaincode/shim"
	pb "github.com/hyperledger/fabric/protos/peer"
)

// SimpleChaincode example simple Chaincode implementation
type SimpleChaincode struct {
}

type item struct {
	ObjectType string `json:"docType"` //docType is used to distinguish the various types of objects in state database
	Name       string `json:"name"`    //the fieldtags are needed to keep case from bouncing around
	Property      string `json:"property"`
	Price       int    `json:"price"`
	Owner      string `json:"owner"`
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
	if function == "initItem" { //create a new item
		return t.initItem(stub, args)
	} else if function == "transferItem" { //change owner of a specific item
		return t.transferItem(stub, args)
	} else if function == "queryBalance" { //query owner's balance
		return t.queryBalance(stub, args)
	} else if function == "deleteItem" { //delete a item
		return t.deleteItem(stub, args)
	} else if function == "queryItemsByItemPropertyOwner" { //find items for owner X using rich query
		return t.queryItemsByOwner(stub, args)
	} else if function == "initUser" { //create a new user
		return t.initUser(stub, args)
	}

	fmt.Println("invoke did not find func: " + function) //error
	return shim.Error("Received unknown function invocation")
}

// ============================================================
// initItem - create a new item, store into chaincode state
// ============================================================
func (t *SimpleChaincode) initItem(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	var err error

	//   0                 1                  2      3
	// "IPR", "Intellectual property right", "35", "tom"
	if len(args) != 4 {
		return shim.Error("Incorrect number of arguments. Expecting 4")
	}

	// ==== Input sanitation ====
	fmt.Println("- start init item")
	if len(args[0]) <= 0 {
		return shim.Error("1st argument must be a non-empty string")
	}
	if len(args[1]) <= 0 {
		return shim.Error("2nd argument must be a non-empty string")
	}
	if len(args[2]) <= 0 {
		return shim.Error("3rd argument must be a non-empty string")
	}
	if len(args[3]) <= 0 {
		return shim.Error("4rd argument must be a non-empty string")
	}
	itemName := args[0]
	property := args[1]
	price, err := strconv.Atoi(args[2])
	if err != nil {
		return shim.Error("3rd argument must be a numeric string")
	}
	if price < 0 {
		return shim.Error("price must be greater than 0")
	}
	owner := args[3]
	
	keyName := owner+itemName+property
	
	// ==== Check if item already exists ====
	itemAsBytes, err := stub.GetState(keyName)
	if err != nil {
		return shim.Error("Failed to get item: " + err.Error())
	} else if itemAsBytes != nil {
		fmt.Println("This item already exists: " + keyName)
		return shim.Error("This item already exists: " + keyName)
	}

	// ==== Create item object and marshal to JSON ====
	objectType := "item"
	item := &item{objectType, itemName, property, price, owner}
	itemJSONasBytes, err := json.Marshal(item)
	if err != nil {
		return shim.Error(err.Error())
	}

	// === Save item to state ===
	err = stub.PutState(keyName, itemJSONasBytes)
	if err != nil {
		return shim.Error(err.Error())
	}

	indexName := "Name~Property~Owner"
	NamePropertyOwnerIndexKey, err := stub.CreateCompositeKey(indexName, []string{item.Name, item.Property, item.Owner})
	if err != nil {
		return shim.Error(err.Error())
	}
	//  Save index entry to state. Only the key name is needed, no need to store a duplicate copy of the item.
	//  Note - passing a 'nil' value will effectively delete the key from state, therefore we pass null character as value
	value := []byte{0x00}
	stub.PutState(NamePropertyOwnerIndexKey, value)

	// ==== item saved and indexed. Return success ====
	fmt.Println("- end init item")
	return shim.Success(nil)
}

// ============================================================
// initUser - create a new user, store into chaincode state
// ============================================================
func (t *SimpleChaincode) initUser(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	var A string    // Entities
	var Aval int // Asset holdings
	var err error

	//   0      1
	// "tom", "1000"
	if len(args) != 1 {
		return shim.Error("Incorrect number of arguments. Expecting 1")
	}

	// ==== Input sanitation ====
	fmt.Println("- start create a user")
	if len(args[0]) <= 0 {
		return shim.Error("1st argument must be a non-empty string")
	}

	A = args[0]
	Aval = 1000
    	
	// ==== Check if user already exists ====
	userAsBytes, err := stub.GetState(A)
	if err != nil {
		return shim.Error("Failed to get user: " + err.Error())
	} else if userAsBytes != nil {
		fmt.Println("This user already exists: " + A)
		return shim.Error("This user already exists: " + A)
	}
	
	// Write the state to the ledger
	err = stub.PutState(A, []byte(strconv.Itoa(Aval)))
	if err != nil {
		return shim.Error(err.Error())
	}

	return shim.Success(nil)
}

// ==================================================
// deleteItem - remove a item key/value pair from state
// ==================================================
func (t *SimpleChaincode) deleteItem(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	var jsonResp string
	var itemJSON item
	if len(args) != 3 {
		return shim.Error("Incorrect number of arguments. Expecting 3")
	}
	itemName := args[0]
	property := args[1]
	owner := args[2]
	keyName := owner+itemName+property

	valAsbytes, err := stub.GetState(keyName) //get the log from chaincode state
	if err != nil {
		jsonResp = "{\"Error\":\"Failed to get state for " + itemName + "\"}"
		return shim.Error(jsonResp)
	} else if valAsbytes == nil {
		jsonResp = "{\"Error\":\"item does not exist: " + itemName + "\"}"
		return shim.Error(jsonResp)
	}

	err = json.Unmarshal([]byte(valAsbytes), &itemJSON)
	if err != nil {
		jsonResp = "{\"Error\":\"Failed to decode JSON of: " + itemName + "\"}"
		return shim.Error(jsonResp)
	}

	err = stub.DelState(keyName) //remove the log from chaincode state
	if err != nil {
		return shim.Error("Failed to delete state:" + err.Error())
	}

	// maintain the index
	indexName := "Name~Property~Owner"
	NamePropertyOwnerIndexKey, err := stub.CreateCompositeKey(indexName, []string{itemJSON.Name, itemJSON.Property, itemJSON.Owner})
	if err != nil {
		return shim.Error(err.Error())
	}

	//  Delete index entry to state.
	err = stub.DelState(NamePropertyOwnerIndexKey)
	if err != nil {
		return shim.Error("Failed to delete state:" + err.Error())
	}
	return shim.Success(nil)
}

// ===========================================================
// transfer a item by setting a new owner name on the item
// ===========================================================
func (t *SimpleChaincode) transferItem(stub shim.ChaincodeStubInterface, args []string) pb.Response {

	//   0                 1                  2      3      4
	// "IPR", "Intellectual property right", "35", "bob", "tom"
	if len(args) < 5 {
		return shim.Error("Incorrect number of arguments. Expecting 5")
	}

	itemName := args[0]
	property := args[1]
	oldOwner := args[3]
	newOwner := args[4]
	itemPrice, err := strconv.Atoi(args[2])
	if err != nil {
		return shim.Error("3rd argument must be a numeric string")
	}
	oldKeyName := oldOwner+itemName+property
	fmt.Println("- start transferItem ", itemName, newOwner)

	// Get the newOwner's balance from the ledger
	newOwnervalbytes, err := stub.GetState(newOwner)
	if err != nil {
		return shim.Error("Failed to get state")
	}
	if newOwnervalbytes == nil {
		return shim.Error("Entity not found")
	}
	newOwnerval, _ := strconv.Atoi(string(newOwnervalbytes))
	
	itemAsBytes, err := stub.GetState(oldKeyName)
	if err != nil {
		return shim.Error("Failed to get item:" + err.Error())
	} else if itemAsBytes == nil {
		return shim.Error("Item does not exist")
	}
	itemToTransfer := item{}
	err = json.Unmarshal(itemAsBytes, &itemToTransfer) //unmarshal it aka JSON.parse()
	if err != nil {
		return shim.Error(err.Error())
	}
	
	// Get the oldOwner's balance from the ledger	
	oldOwnervalbytes, err := stub.GetState(oldOwner)
	if err != nil {
		return shim.Error("Failed to get state")
	}
	if oldOwnervalbytes == nil {
		return shim.Error("Entity not found")
	}
	oldOwnerval, _ := strconv.Atoi(string(oldOwnervalbytes))
	if newOwnerval < itemPrice {
		return shim.Error("Don't have enough balance!")
	}
	
	oldOwnerval = oldOwnerval + itemPrice
	newOwnerval = newOwnerval - itemPrice
	fmt.Printf("%s = %d, %s = %d\n", []byte(oldOwner), oldOwnerval, []byte(newOwner), newOwnerval)
	
	itemToTransfer.Owner = newOwner //change the owner
	

	newKeyName := newOwner+itemName+property
	
	// Write the state back to the ledger
	err = stub.PutState(oldOwner, []byte(strconv.Itoa(oldOwnerval)))
	if err != nil {
		return shim.Error(err.Error())
	}

	err = stub.PutState(newOwner, []byte(strconv.Itoa(newOwnerval)))
	if err != nil {
		return shim.Error(err.Error())
	}

	itemJSONasBytes, _ := json.Marshal(itemToTransfer)
	err = stub.PutState(newKeyName, itemJSONasBytes) //recreate the item
	if err != nil {
		return shim.Error(err.Error())
	}
	err = stub.DelState(oldKeyName) //remove the oldOwnerKey from chaincode state
	if err != nil {
		return shim.Error("Failed to delete state:" + err.Error())
	}

	fmt.Println("- end transferItem (success)")
	return shim.Success(nil)
}

// query user's balance
func (t *SimpleChaincode) queryBalance(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	var A string // Entities
	var err error

	if len(args) != 1 {
		return shim.Error("Incorrect number of arguments. Expecting name of the person to query")
	}

	A = args[0]

	// Get the state from the ledger
	Avalbytes, err := stub.GetState(A)
	if err != nil {
		jsonResp := "{\"Error\":\"Failed to get state for " + A + "\"}"
		return shim.Error(jsonResp)
	}

	if Avalbytes == nil {
		jsonResp := "{\"Error\":\"Nil amount for " + A + "\"}"
		return shim.Error(jsonResp)
	}

	jsonResp := "{\"Name\":\"" + A + "\",\"Amount\":\"" + string(Avalbytes) + "\"}"
	fmt.Printf("Query Response:%s\n", jsonResp)
	return shim.Success(Avalbytes)
}

// =========================================================================================
// queryItemsByItemPropertyOwner queries for items based on a passed in item, property and owner.
// This is an example of a parameterized query where the query logic is baked into the chaincode,
// and accepting three query parameters (item, property, owner).
// Only available on state databases that support rich query (e.g. CouchDB)
// =========================================================================================
func (t *SimpleChaincode) queryItemsByItemPropertyOwner(stub shim.ChaincodeStubInterface, args []string) pb.Response {

	//   0                1                    2
	// "IPR", "Intellectual property right", "tom"

	if len(args[2]) <= 0 {
		return shim.Error("3rd argument must be a non-empty string")
	} else if len(args[0]) <= 0 {
		if len(args[1]) <= 0 {
			return t.queryItemsByOwner(stub, args)
		} else {
			return t.queryItemsByPropertyOwner(stub, args)
		}
	} else if len(args[1]) <= 0 {
			return t.queryItemsByItemOwner(stub, args)
		}
		
	if len(args) < 3 {
		return shim.Error("Incorrect number of arguments. Expecting 3")
	}
	item := args[0]
	property := args[1]
	owner := args[2]

	queryString := fmt.Sprintf("{\"selector\":{\"docType\":\"item\",\"item\":\"%s\",\"property\":\"%s\",\"owner\":\"%s\"}}", item, property, owner)

	queryResults, err := getQueryResultForQueryString(stub, queryString)
	if err != nil {
		return shim.Error(err.Error())
	}
	return shim.Success(queryResults)
}

// =========================================================================================
// queryItemsByOwner queries for items based on a passed in owner.
// This is an example of a parameterized query where the query logic is baked into the chaincode,
// and accepting a single query parameter (owner).
// Only available on state databases that support rich query (e.g. CouchDB)
// =========================================================================================
func (t *SimpleChaincode) queryItemsByOwner(stub shim.ChaincodeStubInterface, args []string) pb.Response {

	//   0
	// "bob"
	if len(args) < 1 {
		return shim.Error("Incorrect number of arguments. Expecting 1")
	}

	owner := args[0]

	queryString := fmt.Sprintf("{\"selector\":{\"docType\":\"item\",\"owner\":\"%s\"}}", owner)

	queryResults, err := getQueryResultForQueryString(stub, queryString)
	if err != nil {
		return shim.Error(err.Error())
	}
	return shim.Success(queryResults)
}

// =========================================================================================
// queryItemsByPropertyOwner queries for items based on a passed in property and owner.
// This is an example of a parameterized query where the query logic is baked into the chaincode,
// and accepting two query parameters (property, owner).
// Only available on state databases that support rich query (e.g. CouchDB)
// =========================================================================================
func (t *SimpleChaincode) queryItemsByPropertyOwner(stub shim.ChaincodeStubInterface, args []string) pb.Response {

	//              0                   1 
	// "Intellectual property right", "tom"
	if len(args) < 2 {
		return shim.Error("Incorrect number of arguments. Expecting 2")
	}

	property := args[0]
	owner := args[1]

	queryString := fmt.Sprintf("{\"selector\":{\"docType\":\"item\",\"property\":\"%s\",\"owner\":\"%s\"}}", property, owner)

	queryResults, err := getQueryResultForQueryString(stub, queryString)
	if err != nil {
		return shim.Error(err.Error())
	}
	return shim.Success(queryResults)
}

// =========================================================================================
// queryItemsByItemOwner queries for items based on a passed in item and owner.
// This is an example of a parameterized query where the query logic is baked into the chaincode,
// and accepting two query parameters (item, owner).
// Only available on state databases that support rich query (e.g. CouchDB)
// =========================================================================================
func (t *SimpleChaincode) queryItemsByItemOwner(stub shim.ChaincodeStubInterface, args []string) pb.Response {

	//   0      1
	// "IPR", "tom"
	if len(args) < 2 {
		return shim.Error("Incorrect number of arguments. Expecting 2")
	}

	item := args[0]
	owner := args[1]

	queryString := fmt.Sprintf("{\"selector\":{\"docType\":\"item\",\"item\":\"%s\",\"owner\":\"%s\"}}", item, owner)

	queryResults, err := getQueryResultForQueryString(stub, queryString)
	if err != nil {
		return shim.Error(err.Error())
	}
	return shim.Success(queryResults)
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
