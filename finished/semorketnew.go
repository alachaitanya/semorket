package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/hyperledger/fabric/core/chaincode/shim"
)

var logger = shim.NewLogger("SemorketChaincode")

//Participant types
const AUTHORITY = "regulator"
const PRIMARY_LENDER = "pl"
const SECONDARY_LENDER = "sl"

//Status types - mortgage life cycle
const STATE_PL_OWNERSHIP = 0
const STATE_SL_OWNERSHIP = 1

type SimpleChaincode struct {
}

//Mortgage
type Mortgage struct {
	MortID string `json:"mortID"`
	Lendee string `json:"lendee"`
	Owner  string `json:"owner"`
}

//Mortgages holder
type MortID_Holder struct {
	MortIDs []string `json:"mortIDs"`
}

//User_and_eCert
type User_and_eCert struct {
	Identity string `json:"identity"`
	eCert    string `json:"ecert"`
}

//Init function
func (t *SimpleChaincode) Init(stub shim.ChaincodeStubInterface, function string, args []string) ([]byte, error) {

	//Args
	//				0
	//			peer_address

	var MortIDs MortID_Holder
	bytes, err := json.Marshal(MortIDs)

	if err != nil {
		return nil, errors.New("Error creating MortID_Holder record")
	}

	err = stub.PutState("MortIDs", bytes)

	for i := 0; i < len(args); i = i + 2 {
		t.add_ecert(stub, args[i], args[i+1])
	}

	return nil, nil
}

//GENERAL FUNCTIONS------------------------------------------------------------------------------------------------------------------------

//GET ECERT
func (t *SimpleChaincode) get_ecert(stub shim.ChaincodeStubInterface, name string) ([]byte, error) {

	ecert, err := stub.GetState(name)

	if err != nil {
		return nil, errors.New("Couldn't retrieve ecert for user " + name)
	}

	return ecert, nil
}

//ADD ECERT
func (t *SimpleChaincode) add_ecert(stub shim.ChaincodeStubInterface, name string, ecert string) ([]byte, error) {

	err := stub.PutState(name, []byte(ecert))

	if err == nil {
		return nil, errors.New("Error storing eCert for user " + name + " identity: " + ecert)
	}

	return nil, nil

}

//GET CALLER - Retrieves the username of the user who invoked the chaincode.
func (t *SimpleChaincode) get_username(stub shim.ChaincodeStubInterface) (string, error) {

	username, err := stub.ReadCertAttribute("username")
	if err != nil {
		return "", errors.New("Couldn't get attribute 'username'. Error: " + err.Error())
	}
	return string(username), nil
}

//CHECK AFFILIATION
func (t *SimpleChaincode) check_affiliation(stub shim.ChaincodeStubInterface) (string, error) {
	affiliation, err := stub.ReadCertAttribute("role")
	if err != nil {
		return "", errors.New("Couldn't get attribute 'role'. Error: " + err.Error())
	}
	return string(affiliation), nil

}

//GET CALLER DATA
func (t *SimpleChaincode) get_caller_data(stub shim.ChaincodeStubInterface) (string, string, error) {

	user, err := t.get_username(stub)

	// if err != nil { return "", "", err }

	// ecert, err := t.get_ecert(stub, user);

	// if err != nil { return "", "", err }

	affiliation := "pl"
	//affiliation, err := t.check_affiliation(stub)

	if err != nil {
		return "", "", err
	}

	return user, affiliation, nil
}

//RETRIEVE MORTGAGE
func (t *SimpleChaincode) retrieve_mortgage(stub shim.ChaincodeStubInterface, mortID string) (Mortgage, error) {

	var m Mortgage

	bytes, err := stub.GetState(mortID)

	if err != nil {
		fmt.Printf("RETRIEVE_MORTGAGE: Failed to invoke mortgage_code: %s", err)
		return m, errors.New("RETRIEVE_MORTGAGE: Error retrieving mortgage with mortID = " + mortID)
	}

	err = json.Unmarshal(bytes, &m)

	if err != nil {
		fmt.Printf("RETRIEVE_MORTGAGE: Corrupt mortgage record "+string(bytes)+": %s", err)
		return m, errors.New("RETRIEVE_MORTGAGE: Corrupt mortgage record" + string(bytes))
	}

	return m, nil
}

//SAVE CHANGES
func (t *SimpleChaincode) save_changes(stub shim.ChaincodeStubInterface, m Mortgage) (bool, error) {

	bytes, err := json.Marshal(m)

	if err != nil {
		fmt.Printf("SAVE_CHANGES: Error converting mortgage record: %s", err)
		return false, errors.New("Error converting mortgage record")
	}

	err = stub.PutState(m.MortID, bytes)

	if err != nil {
		fmt.Printf("SAVE_CHANGES: Error storing mortgage record: %s", err)
		return false, errors.New("Error storing mortgage record")
	}

	return true, nil
}

//Router Functions--------------------------------------------------------------------------------------------------------------------------

//INVOKE
func (t *SimpleChaincode) Invoke(stub shim.ChaincodeStubInterface, function string, args []string) ([]byte, error) {

	caller, caller_affiliation, err := t.get_caller_data(stub)

	if err != nil {
		return nil, errors.New("Error retrieving caller information")
	}

	if function == "create_mortgage" {
		return t.create_mortgage(stub, caller, caller_affiliation, args[0])
	} else if function == "ping" {
		return t.ping(stub)
	} else { // If the function is not a create then there must be a mortgage so we need to retrieve the mortgage.
		argPos := 1
		m, err := t.retrieve_mortgage(stub, args[argPos])
		if err != nil {
			fmt.Printf("INVOKE: Error retrieving mortgage: %s", err)
			return nil, errors.New("Error retrieving mortgage")
		}
		if function == "pl_to_sl" {
			return t.pl_to_sl(stub, m, caller, caller_affiliation, args[0], "sl")
		}
		return nil, errors.New("Function of the name " + function + " doesn't exist.")
	}
}

//QUERY
func (t *SimpleChaincode) Query(stub shim.ChaincodeStubInterface, function string, args []string) ([]byte, error) {

	caller, caller_affiliation, err := t.get_caller_data(stub)
	if err != nil {
		fmt.Printf("QUERY: Error retrieving caller details", err)
		return nil, errors.New("QUERY: Error retrieving caller details: " + err.Error())
	}

	logger.Debug("function: ", function)
	logger.Debug("caller: ", caller)
	logger.Debug("affiliation: ", caller_affiliation)

	if function == "get_mortgage_details" {
		if len(args) != 1 {
			fmt.Printf("Incorrect number of arguments passed")
			return nil, errors.New("QUERY: Incorrect number of arguments passed")
		}
		m, err := t.retrieve_mortgage(stub, args[0])
		if err != nil {
			fmt.Printf("QUERY: Error retrieving mortgage: %s", err)
			return nil, errors.New("QUERY: Error retrieving mortgage " + err.Error())
		}
		return t.get_mortgage_details(stub, m, caller, caller_affiliation)
	} else if function == "check_unique_mortgage" {
		return t.check_unique_mortgage(stub, args[0], caller, caller_affiliation)
	} else if function == "get_mortgages" {
		return t.get_mortgages(stub, caller, caller_affiliation)
	} else if function == "get_ecert" {
		return t.get_ecert(stub, args[0])
	} else if function == "ping" {
		return t.ping(stub)
	}

	return nil, errors.New("Received unknown function invocation " + function)

}

//PING FUNCTION
func (t *SimpleChaincode) ping(stub shim.ChaincodeStubInterface) ([]byte, error) {
	return []byte("Hello, world!"), nil
}

//CREATE MORTGAGE
func (t *SimpleChaincode) create_mortgage(stub shim.ChaincodeStubInterface, caller string, caller_affiliation string, mortID string) ([]byte, error) {
	var m Mortgage

	mort_ID := "\"MortID\":\"" + mortID + "\", " // Variables to define the JSON
	lendee := "\"Lendee\":\"UNDEFINED\", "
	owner := "\"Owner\":\"" + caller + "\""

	mortgage_json := "{" + mort_ID + lendee + owner + "}" // Concatenates the variables to create the total JSON object

	if mort_ID == "" {
		fmt.Printf("CREATE_MORTGAGE: Invalid mortID provided")
		return nil, errors.New("Invalid mortID provided")
	}

	err := json.Unmarshal([]byte(mortgage_json), &m) // Convert the JSON defined above into a mortgage object for go

	if err != nil {
		return nil, errors.New("Invalid JSON object")
	}

	record, err := stub.GetState(m.MortID) // If not an error then a record exists so we cant create a new mortgage with this MortID as it must be unique

	if record != nil {
		return nil, errors.New("Mortgage already exists")
	}

	if caller_affiliation != PRIMARY_LENDER { // Only a primary lender (pl) can create a new mortgage

		return nil, errors.New(fmt.Sprintf("Permission Denied. create_mortgage. %m === %m", caller_affiliation, PRIMARY_LENDER))

	}

	_, err = t.save_changes(stub, m)

	if err != nil {
		fmt.Printf("CREATE_MORTGAGE: Error saving changes: %s", err)
		return nil, errors.New("Error saving changes")
	}

	bytes, err := stub.GetState("mortIDs")

	if err != nil {
		return nil, errors.New("Unable to get mortIDs")
	}

	var mortIDs MortID_Holder

	err = json.Unmarshal(bytes, &mortIDs)

	if err != nil {
		return nil, errors.New("Corrupt MortID_Holder record")
	}

	mortIDs.MortIDs = append(mortIDs.MortIDs, mortID)

	bytes, err = json.Marshal(mortIDs)

	if err != nil {
		fmt.Print("Error creating MortID_Holder record")
	}

	err = stub.PutState("mortIDs", bytes)

	if err != nil {
		return nil, errors.New("Unable to put the state")
	}

	return nil, nil

}

//TRANSFER - PRIMARY LENDER TO SECONDARY LENDER
func (t *SimpleChaincode) pl_to_sl(stub shim.ChaincodeStubInterface, m Mortgage, caller string, caller_affiliation string, recipient_name string, recipient_affiliation string) ([]byte, error) {

	if m.Owner == caller &&
		caller_affiliation == PRIMARY_LENDER &&
		recipient_affiliation == SECONDARY_LENDER { // If the roles and users are ok

		m.Owner = recipient_name // then make the recipient the new owner

	} else { // Otherwise if there is an error
		fmt.Printf("PL_TO_SL: Permission Denied")
		return nil, errors.New(fmt.Sprintf("Permission Denied. authority_to_manufacturer. %m %m === %m, %m === %m, %m === %m, %m === %m, %m === %m", m, m.Owner, caller, caller_affiliation, PRIMARY_LENDER, recipient_affiliation, false))

	}

	_, err := t.save_changes(stub, m) // Write new state

	if err != nil {
		fmt.Printf("PL_TO_SL: Error saving changes: %s", err)
		return nil, errors.New("Error saving changes")
	}

	return nil, nil
}

//READ FUNCTIONS--------------------------------------------------------------------------------------------------------------------

//GET MORTGAGE DETAILS
func (t *SimpleChaincode) get_mortgage_details(stub shim.ChaincodeStubInterface, m Mortgage, caller string, caller_affiliation string) ([]byte, error) {

	bytes, err := json.Marshal(m)

	if err != nil {
		return nil, errors.New("GET_MORTGAGE_DETAILS: Invalid mortgage object")
	}

	if m.Owner == caller {
		return bytes, nil
	} else {
		return nil, errors.New("Permission Denied. get_mortgage_details")
	}
}

//GET MORTGAGES
func (t *SimpleChaincode) get_mortgages(stub shim.ChaincodeStubInterface, caller string, caller_affiliation string) ([]byte, error) {
	bytes, err := stub.GetState("mortIDs")

	if err != nil {
		return nil, errors.New("Unable to get mortIDs")
	}

	var mortIDs MortID_Holder

	err = json.Unmarshal(bytes, &mortIDs)

	if err != nil {
		return nil, errors.New("Corrupt MortID_Holder")
	}

	result := "["

	var temp []byte
	var m Mortgage

	for _, mortgage := range mortIDs.MortIDs {

		m, err = t.retrieve_mortgage(stub, mortgage)

		if err != nil {
			return nil, errors.New("Failed to retrieve Mortgage")
		}

		temp, err = t.get_mortgage_details(stub, m, caller, caller_affiliation)

		if err == nil {
			result += string(temp) + ","
		}
	}

	if len(result) == 1 {
		result = "[]"
	} else {
		result = result[:len(result)-1] + "]"
	}

	return []byte(result), nil
}

//CHECK UNIQUE MORTGAGE
func (t *SimpleChaincode) check_unique_mortgage(stub shim.ChaincodeStubInterface, mortgage string, caller string, caller_affiliation string) ([]byte, error) {
	_, err := t.retrieve_mortgage(stub, mortgage)
	if err == nil {
		return []byte("false"), errors.New("mortgage is not unique")
	} else {
		return []byte("true"), nil
	}
}

//MAIN
func main() {
	err := shim.Start(new(SimpleChaincode))
	if err != nil {
		fmt.Printf("Error starting Chaincode: %s", err)
	}
}
