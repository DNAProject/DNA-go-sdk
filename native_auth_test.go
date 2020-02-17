// SPDX-License-Identifier: LGPL-3.0-or-later
// Copyright 2019 the DNA Dev team
// Copyright 2018 the Ontology Dev team
//
package DNA_go_sdk

import (
	"fmt"
	"testing"
	"time"

	"github.com/DNAProject/DNA-go-sdk/utils"
	"github.com/DNAProject/DNA/smartcontract/service/native/common"
)

func TestNativeContract_DeployAuthContract(t *testing.T) {
	// init, deploy did contract, update did method
	TestNativeContract_DeployDIDContract(t)

	// depoly auth contract
	initParamStr, err := GenerateID(testDidMethod)
	if err != nil {
		t.Errorf("generate ID in deploy auth native contract: %s", err)
		return
	}
	txHash, err := testDnaSdk.Native.DeployNativeContract(testGasPrice, testGasLimit, testDefAcc,
		common.AuthContractAddress, []byte(initParamStr),
		"testname", "1.0", "testAuthor", "test@example.com", "test deploying native Auth")
	testDnaSdk.WaitForGenerateBlock(30*time.Second, 1)
	event, err := testDnaSdk.GetSmartContractEvent(txHash.ToHexString())
	if err != nil {
		t.Errorf("test deploy auth native contract error: %s", err)
		return
	}
	fmt.Printf("test deploy native contract event: %+v\n", event)
	for _, ev := range event.Notify {
		fmt.Printf("contract: %s, state: %v\n", ev.ContractAddress, ev.States)
		authAddr, err := utils.AddressFromHexString(ev.ContractAddress)
		if err != nil {
			t.Errorf("failed to parse auth addr: %s", err)
		}
		testDnaSdk.Native.AuthContractAddr = authAddr
		t.Logf("update Auth contract to %s", ev.ContractAddress)
	}

	TestAuth_InitAuth(t)
	TestAuth_InitContractAdmin(t)
}

func TestAuth_InitAuth(t *testing.T) {
	txHash, err := testDnaSdk.Native.Auth.InitAuth(testGasPrice, testGasLimit, testDefAcc)
	if err != nil {
		t.Errorf("TestAuth_InitAuth, init auth err: %s", err)
		return
	}
	testDnaSdk.WaitForGenerateBlock(30*time.Second, 1)
	event, err := testDnaSdk.GetSmartContractEvent(txHash.ToHexString())
	if err != nil {
		t.Errorf("TestAuth_InitAuth get smart contract event: %s", err)
	}
	fmt.Printf("TestAuth_InitAuth event: %v\n", event)
}

func TestAuth_InitContractAdmin(t *testing.T) {
	// create auth admin
	testAuthAdmin, err := testWallet.NewDefaultSettingIdentity(testDidMethod, testPasswd)
	if err != nil {
		fmt.Printf("create auth admin identity err: %s \n", err)
		return
	}

	txHash, err := testDnaSdk.Native.Auth.InitContractAdmin(testGasPrice, testGasLimit, testDefAcc, []byte(testAuthAdmin.ID))
	if err != nil {
		t.Errorf("TestAuth_InitContractAdmin init contract admin %s: %s", testAuthAdmin.ID, err)
		return
	}
	testDnaSdk.WaitForGenerateBlock(30*time.Second, 1)
	event, err := testDnaSdk.GetSmartContractEvent(txHash.ToHexString())
	if err != nil {
		t.Errorf("TestAuth_InitContractAdmin get smart contract event: %s", err)
	}
	fmt.Printf("TestAuth_InitContractAdmin event: %v\n", event)
}
