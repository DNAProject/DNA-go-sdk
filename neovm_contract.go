// SPDX-License-Identifier: LGPL-3.0-or-later
// Copyright 2019 the DNA Dev team
// Copyright 2018 the Ontology Dev team
//
package DNA_go_sdk

import (
	"encoding/hex"
	"fmt"
	"time"

	sdkcom "github.com/DNAProject/DNA-go-sdk/common"
	"github.com/DNAProject/DNA/common"
	"github.com/DNAProject/DNA/core/payload"
	"github.com/DNAProject/DNA/core/types"
	httpcom "github.com/DNAProject/DNA/http/base/common"
)

type NeoVMContract struct {
	dnaSkd *DNASdk
}

func newNeoVMContract(dnaSkd *DNASdk) *NeoVMContract {
	return &NeoVMContract{
		dnaSkd: dnaSkd,
	}
}

func (this *NeoVMContract) NewDeployNeoVMCodeTransaction(gasPrice, gasLimit uint64, contract *payload.DeployCode) *types.MutableTransaction {
	tx := &types.MutableTransaction{
		Version:  sdkcom.VERSION_TRANSACTION,
		TxType:   types.Deploy,
		Nonce:    uint32(time.Now().Unix()),
		Payload:  contract,
		GasPrice: gasPrice,
		GasLimit: gasLimit,
		Sigs:     make([]types.Sig, 0, 0),
	}
	return tx
}

//DeploySmartContract Deploy smart contract to dna
func (this *NeoVMContract) DeployNeoVMSmartContract(
	gasPrice,
	gasLimit uint64,
	singer *Account,
	needStorage bool,
	code,
	name,
	version,
	author,
	email,
	desc string) (common.Uint256, error) {

	invokeCode, err := hex.DecodeString(code)
	if err != nil {
		return common.UINT256_EMPTY, fmt.Errorf("code hex decode error:%s", err)
	}
	deployCode, err := payload.NewDeployCode(invokeCode, payload.NEOVM_TYPE, name, version, author, email, desc)
	if err != nil {
		return common.UINT256_EMPTY, fmt.Errorf("build deployCode err: %s", err)
	}
	tx := this.NewDeployNeoVMCodeTransaction(gasPrice, gasLimit, deployCode)
	err = this.dnaSkd.SignToTransaction(tx, singer)
	if err != nil {
		return common.Uint256{}, err
	}
	txHash, err := this.dnaSkd.SendTransaction(tx)
	if err != nil {
		return common.Uint256{}, fmt.Errorf("SendRawTransaction error:%s", err)
	}
	return txHash, nil
}

func (this *NeoVMContract) NewNeoVMInvokeTransaction(
	gasPrice,
	gasLimit uint64,
	contractAddress common.Address,
	params []interface{},
) (*types.MutableTransaction, error) {
	invokeCode, err := httpcom.BuildNeoVMInvokeCode(contractAddress, params)
	if err != nil {
		return nil, err
	}
	return this.dnaSkd.NewInvokeTransaction(gasPrice, gasLimit, invokeCode), nil
}

func (this *NeoVMContract) InvokeNeoVMContract(
	gasPrice,
	gasLimit uint64,
	signer *Account,
	contractAddress common.Address,
	params []interface{}) (common.Uint256, error) {
	tx, err := this.NewNeoVMInvokeTransaction(gasPrice, gasLimit, contractAddress, params)
	if err != nil {
		return common.UINT256_EMPTY, fmt.Errorf("NewNeoVMInvokeTransaction error:%s", err)
	}
	err = this.dnaSkd.SignToTransaction(tx, signer)
	if err != nil {
		return common.UINT256_EMPTY, err
	}
	return this.dnaSkd.SendTransaction(tx)
}

func (this *NeoVMContract) PreExecInvokeNeoVMContract(
	contractAddress common.Address,
	params []interface{}) (*sdkcom.PreExecResult, error) {
	tx, err := this.NewNeoVMInvokeTransaction(0, 0, contractAddress, params)
	if err != nil {
		return nil, err
	}
	return this.dnaSkd.PreExecTransaction(tx)
}
