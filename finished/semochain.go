package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/hyperledger/fabric/core/chaincode/shim"
)

// SimpleChaincode example simple Chaincode implementation
type SimpleChaincode struct {
}

func main() {
	err := shim.Start(new(SimpleChaincode))
	if err != nil {
		fmt.Printf("Error starting Simple chaincode: %s", err)
	}
}

// Init resets all the things
func (t *SimpleChaincode) Init(stub shim.ChaincodeStubInterface, function string, args []string) ([]byte, error) {
	if len(args) != 3 {
		return nil, errors.New("Incorrect number of arguments. Expecting 3 arguments.")
	}

	err := createTableOne(stub, args)

	if err != nil {
		return nil, err
	}

	return nil, nil
}

func createTableOne(stub shim.ChaincodeStubInterface, args []string) error {
	// Create table one
	var columnDefsTableOne []*shim.ColumnDefinition
	columnOneTableOneDef := shim.ColumnDefinition{Name: args[0],
		Type: shim.ColumnDefinition_STRING, Key: true}
	columnTwoTableOneDef := shim.ColumnDefinition{Name: args[1],
		Type: shim.ColumnDefinition_STRING, Key: false}
	columnThreeTableOneDef := shim.ColumnDefinition{Name: args[2],
		Type: shim.ColumnDefinition_STRING, Key: false}
	columnDefsTableOne = append(columnDefsTableOne, &columnOneTableOneDef)
	columnDefsTableOne = append(columnDefsTableOne, &columnTwoTableOneDef)
	columnDefsTableOne = append(columnDefsTableOne, &columnThreeTableOneDef)
	return stub.CreateTable("tableOne", columnDefsTableOne)
}

// Invoke isur entry point to invoke a chaincode function
func (t *SimpleChaincode) Invoke(stub shim.ChaincodeStubInterface, function string, args []string) ([]byte, error) {
	fmt.Println("invoke is running " + function)

	// Handle different functions
	if function == "init" {
		return t.Init(stub, "init", args)
	} else if function == "add" {
		return t.add(stub, args)
	}
	fmt.Println("invoke did not find func: " + function)

	return nil, errors.New("Received unknown function invocation: " + function)
}

// Query is our entry point for queries
func (t *SimpleChaincode) Query(stub shim.ChaincodeStubInterface, function string, args []string) ([]byte, error) {
	fmt.Println("query is running " + function)

	// Handle different functions
	if function == "read" { //read a variable
		return t.read(stub, args)
	}
	fmt.Println("query did not find func: " + function)

	return nil, errors.New("Received unknown function query: " + function)
}

// write - invoke function to write key/value pair
func (t *SimpleChaincode) add(stub shim.ChaincodeStubInterface, args []string) ([]byte, error) {
	if len(args) != 3 {
		return nil, errors.New("Incorrect number of arguments. Expecting 3 arguments (mortgage_id, lendee_name, servicer)")
	}

	var col1Val, col2Val, col3Val string

	col1Val = args[0]
	col2Val = args[1]
	col3Val = args[2]

	var columns []*shim.Column
	col1 := shim.Column{Value: &shim.Column_String_{String_: col1Val}}
	col2 := shim.Column{Value: &shim.Column_String_{String_: col2Val}}
	col3 := shim.Column{Value: &shim.Column_String_{String_: col3Val}}

	columns = append(columns, &col1)
	columns = append(columns, &col2)
	columns = append(columns, &col3)

	row := shim.Row{Columns: columns}
	ok, err := stub.InsertRow("tableOne", row)
	if err != nil {
		return nil, fmt.Errorf("insertRowTableOne operation failed. %s", err)
	}
	if !ok {
		return nil, errors.New("insertRowTableOne operation failed. Row with given key already exists")
	}

	return nil, nil
}

// read - query function to read key/value pair
func (t *SimpleChaincode) read(stub shim.ChaincodeStubInterface, args []string) ([]byte, error) {

	if len(args) < 1 {
		return nil, errors.New("getRowTableOne failed. Must include 1 key value")
	}

	col1Val := args[0]
	var columns []shim.Column
	col1 := shim.Column{Value: &shim.Column_String_{String_: col1Val}}
	columns = append(columns, col1)

	rowChannel, err := stub.GetRows("tableOne", columns)
	if err != nil {
		return nil, fmt.Errorf("getRowsTableOne operation failed. %s", err)
	}

	var rows []shim.Row
	for {
		select {
		case row, ok := <-rowChannel:
			if !ok {
				rowChannel = nil
			} else {
				rows = append(rows, row)
			}
		}
		if rowChannel == nil {
			break
		}
	}

	jsonRows, err := json.Marshal(rows)
	if err != nil {
		return nil, fmt.Errorf("getRowsTableOne operation failed. Error marshaling JSON: %s", err)
	}

	return jsonRows, nil
}
