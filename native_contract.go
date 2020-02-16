// SPDX-License-Identifier: LGPL-3.0-or-later
// Copyright 2019 the DNA Dev team
// Copyright 2018 the Ontology Dev team
//
package DNA_go_sdk

import (
	"bytes"
	"encoding/hex"
	"fmt"

	sdkcom "github.com/DNAProject/DNA-go-sdk/common"
	"github.com/DNAProject/DNA-go-sdk/utils"
	"github.com/DNAProject/DNA/common"
	"github.com/DNAProject/DNA/common/serialization"
	"github.com/DNAProject/DNA/core/payload"
	"github.com/DNAProject/DNA/core/types"
	cutils "github.com/DNAProject/DNA/core/utils"
	common2 "github.com/DNAProject/DNA/smartcontract/service/native/common"
	"github.com/DNAProject/DNA/smartcontract/service/native/gas"
	"github.com/DNAProject/DNA/smartcontract/service/native/global_params"
	"github.com/ontio/ontology-crypto/keypair"
)

var (
	GAS_CONTRACT_ADDRESS, _           = utils.AddressFromHexString("0200000000000000000000000000000000000000")
	GLOABL_PARAMS_CONTRACT_ADDRESS, _ = utils.AddressFromHexString("0400000000000000000000000000000000000000")
	GOVERNANCE_CONTRACT_ADDRESS, _    = utils.AddressFromHexString("0700000000000000000000000000000000000000")
)

var (
	GAS_CONTRACT_VERSION           = byte(0)
	GLOBAL_PARAMS_CONTRACT_VERSION = byte(0)
	GOVERNANCE_CONTRACT_VERSION    = byte(0)
)

var OPCODE_IN_PAYLOAD = map[byte]bool{0xc6: true, 0x6b: true, 0x6a: true, 0xc8: true, 0x6c: true, 0x68: true, 0x67: true,
	0x7c: true, 0xc1: true}

type NativeContract struct {
	dnaSdk       *DNASdk
	Gas          *Gas
	DID          *DID
	GlobalParams *GlobalParam
	Auth         *Auth

	// deployable native contracts
	DIDContractAddr     common.Address
	AuthContractAddr    common.Address
	DIDContractVersion  byte
	AuthContractVersion byte
}

func newNativeContract(dnaSkd *DNASdk) *NativeContract {
	native := &NativeContract{
		dnaSdk:              dnaSkd,
		DIDContractAddr:     common2.DIDContractAddress,
		AuthContractAddr:    common2.AuthContractAddress,
		DIDContractVersion:  byte(0),
		AuthContractVersion: byte(0),
	}
	native.Gas = &Gas{native: native, dnaSkd: dnaSkd}
	native.DID = &DID{native: native, dnaSdk: dnaSkd}
	native.GlobalParams = &GlobalParam{native: native, dnaSkd: dnaSkd}
	native.Auth = &Auth{native: native, dnaSkd: dnaSkd}
	return native
}

func (this *NativeContract) NewNativeInvokeTransaction(
	gasPrice,
	gasLimit uint64,
	version byte,
	contractAddress common.Address,
	method string,
	params []interface{},
) (*types.MutableTransaction, error) {
	if params == nil {
		params = make([]interface{}, 0, 1)
	}
	//Params cannot empty, if params is empty, fulfil with empty string
	if len(params) == 0 {
		params = append(params, "")
	}
	invokeCode, err := cutils.BuildNativeInvokeCode(contractAddress, version, method, params)
	if err != nil {
		return nil, fmt.Errorf("BuildNativeInvokeCode error:%s", err)
	}
	return this.dnaSdk.NewInvokeTransaction(gasPrice, gasLimit, invokeCode), nil
}

func (this *NativeContract) DeployNativeContract(
	gasPrice, gasLimit uint64,
	signer *Account,
	baseNativeContract common.Address,
	initParam []byte,
	name, version, author, email, desp string,
) (common.Uint256, error) {
	ndc := &payload.NativeDeployCode{
		BaseContractAddress: baseNativeContract,
		InitParam:           initParam,
	}
	sink := common.NewZeroCopySink(nil)
	ndc.Serialization(sink)

	tx, err := cutils.NewDeployTransaction(sink.Bytes(), name, version, author, email, desp, payload.NATIVE_TYPE)
	if err != nil {
		return common.UINT256_EMPTY, err
	}
	if err := this.dnaSdk.SignToTransaction(tx, signer); err != nil {
		return common.UINT256_EMPTY, err
	}

	return this.dnaSdk.SendTransaction(tx)
}

func (this *NativeContract) InvokeNativeContract(
	gasPrice,
	gasLimit uint64,
	signer *Account,
	version byte,
	contractAddress common.Address,
	method string,
	params []interface{},
) (common.Uint256, error) {
	tx, err := this.NewNativeInvokeTransaction(gasPrice, gasLimit, version, contractAddress, method, params)
	if err != nil {
		return common.UINT256_EMPTY, err
	}
	err = this.dnaSdk.SignToTransaction(tx, signer)
	if err != nil {
		return common.UINT256_EMPTY, err
	}
	return this.dnaSdk.SendTransaction(tx)
}

func (this *NativeContract) PreExecInvokeNativeContract(
	contractAddress common.Address,
	version byte,
	method string,
	params []interface{},
) (*sdkcom.PreExecResult, error) {
	tx, err := this.NewNativeInvokeTransaction(0, 0, version, contractAddress, method, params)
	if err != nil {
		return nil, err
	}
	return this.dnaSdk.PreExecTransaction(tx)
}

func (this *NativeContract) SetDIDContract(addr common.Address, version byte) {
	this.DIDContractAddr = addr
	this.DIDContractVersion = version
}

func (this *NativeContract) SetAuthContract(addr common.Address, version byte) {
	this.AuthContractAddr = addr
	this.AuthContractVersion = version
}

type Gas struct {
	dnaSkd *DNASdk
	native *NativeContract
}

func (this *Gas) NewTransferTransaction(gasPrice, gasLimit uint64, from, to common.Address, amount uint64) (*types.MutableTransaction, error) {
	state := &gas.State{
		From:  from,
		To:    to,
		Value: amount,
	}
	return this.NewMultiTransferTransaction(gasPrice, gasLimit, []*gas.State{state})
}

func (this *Gas) Transfer(gasPrice, gasLimit uint64, from *Account, to common.Address, amount uint64) (common.Uint256, error) {
	tx, err := this.NewTransferTransaction(gasPrice, gasLimit, from.Address, to, amount)
	if err != nil {
		return common.UINT256_EMPTY, err
	}
	err = this.dnaSkd.SignToTransaction(tx, from)
	if err != nil {
		return common.UINT256_EMPTY, err
	}
	return this.dnaSkd.SendTransaction(tx)
}

func (this *Gas) NewMultiTransferTransaction(gasPrice, gasLimit uint64, states []*gas.State) (*types.MutableTransaction, error) {
	return this.native.NewNativeInvokeTransaction(
		gasPrice,
		gasLimit,
		GAS_CONTRACT_VERSION,
		GAS_CONTRACT_ADDRESS,
		gas.TRANSFER_NAME,
		[]interface{}{states})
}

func (this *Gas) MultiTransfer(gasPrice, gasLimit uint64, states []*gas.State, signer *Account) (common.Uint256, error) {
	tx, err := this.NewMultiTransferTransaction(gasPrice, gasLimit, states)
	if err != nil {
		return common.UINT256_EMPTY, err
	}
	err = this.dnaSkd.SignToTransaction(tx, signer)
	if err != nil {
		return common.UINT256_EMPTY, err
	}
	return this.dnaSkd.SendTransaction(tx)
}

func (this *Gas) NewTransferFromTransaction(gasPrice, gasLimit uint64, sender, from, to common.Address, amount uint64) (*types.MutableTransaction, error) {
	state := &gas.TransferFrom{
		Sender: sender,
		From:   from,
		To:     to,
		Value:  amount,
	}
	return this.native.NewNativeInvokeTransaction(
		gasPrice,
		gasLimit,
		GAS_CONTRACT_VERSION,
		GAS_CONTRACT_ADDRESS,
		gas.TRANSFERFROM_NAME,
		[]interface{}{state},
	)
}

func (this *Gas) TransferFrom(gasPrice, gasLimit uint64, sender *Account, from, to common.Address, amount uint64) (common.Uint256, error) {
	tx, err := this.NewTransferFromTransaction(gasPrice, gasLimit, sender.Address, from, to, amount)
	if err != nil {
		return common.UINT256_EMPTY, err
	}
	err = this.dnaSkd.SignToTransaction(tx, sender)
	if err != nil {
		return common.UINT256_EMPTY, err
	}
	return this.dnaSkd.SendTransaction(tx)
}

func (this *Gas) NewApproveTransaction(gasPrice, gasLimit uint64, from, to common.Address, amount uint64) (*types.MutableTransaction, error) {
	state := &gas.State{
		From:  from,
		To:    to,
		Value: amount,
	}
	return this.native.NewNativeInvokeTransaction(
		gasPrice,
		gasLimit,
		GAS_CONTRACT_VERSION,
		GAS_CONTRACT_ADDRESS,
		gas.APPROVE_NAME,
		[]interface{}{state},
	)
}

func (this *Gas) Approve(gasPrice, gasLimit uint64, from *Account, to common.Address, amount uint64) (common.Uint256, error) {
	tx, err := this.NewApproveTransaction(gasPrice, gasLimit, from.Address, to, amount)
	if err != nil {
		return common.UINT256_EMPTY, err
	}
	err = this.dnaSkd.SignToTransaction(tx, from)
	if err != nil {
		return common.UINT256_EMPTY, err
	}
	return this.dnaSkd.SendTransaction(tx)
}

func (this *Gas) Allowance(from, to common.Address) (uint64, error) {
	type allowanceStruct struct {
		From common.Address
		To   common.Address
	}
	preResult, err := this.native.PreExecInvokeNativeContract(
		GAS_CONTRACT_ADDRESS,
		GAS_CONTRACT_VERSION,
		gas.ALLOWANCE_NAME,
		[]interface{}{&allowanceStruct{From: from, To: to}},
	)
	if err != nil {
		return 0, err
	}
	balance, err := preResult.Result.ToInteger()
	if err != nil {
		return 0, err
	}
	return balance.Uint64(), nil
}

func (this *Gas) Symbol() (string, error) {
	preResult, err := this.native.PreExecInvokeNativeContract(
		GAS_CONTRACT_ADDRESS,
		GAS_CONTRACT_VERSION,
		gas.SYMBOL_NAME,
		[]interface{}{},
	)
	if err != nil {
		return "", err
	}
	return preResult.Result.ToString()
}

func (this *Gas) BalanceOf(address common.Address) (uint64, error) {
	preResult, err := this.native.PreExecInvokeNativeContract(
		GAS_CONTRACT_ADDRESS,
		GAS_CONTRACT_VERSION,
		gas.BALANCEOF_NAME,
		[]interface{}{address[:]},
	)
	if err != nil {
		return 0, err
	}
	balance, err := preResult.Result.ToInteger()
	if err != nil {
		return 0, err
	}
	return balance.Uint64(), nil
}

func (this *Gas) Name() (string, error) {
	preResult, err := this.native.PreExecInvokeNativeContract(
		GAS_CONTRACT_ADDRESS,
		GAS_CONTRACT_VERSION,
		gas.NAME_NAME,
		[]interface{}{},
	)
	if err != nil {
		return "", err
	}
	return preResult.Result.ToString()
}

func (this *Gas) Decimals() (byte, error) {
	preResult, err := this.native.PreExecInvokeNativeContract(
		GAS_CONTRACT_ADDRESS,
		GAS_CONTRACT_VERSION,
		gas.DECIMALS_NAME,
		[]interface{}{},
	)
	if err != nil {
		return 0, err
	}
	decimals, err := preResult.Result.ToInteger()
	if err != nil {
		return 0, err
	}
	return byte(decimals.Uint64()), nil
}

func (this *Gas) TotalSupply() (uint64, error) {
	preResult, err := this.native.PreExecInvokeNativeContract(
		GAS_CONTRACT_ADDRESS,
		GAS_CONTRACT_VERSION,
		gas.TOTAL_SUPPLY_NAME,
		[]interface{}{},
	)
	if err != nil {
		return 0, err
	}
	balance, err := preResult.Result.ToInteger()
	if err != nil {
		return 0, err
	}
	return balance.Uint64(), nil
}

type DID struct {
	dnaSdk *DNASdk
	native *NativeContract
}

func (this *DID) NewInitDIDTransaction(gasPrice, gasLimit uint64, didMethod string) (*types.MutableTransaction, error) {
	return this.native.NewNativeInvokeTransaction(
		gasPrice,
		gasLimit,
		this.native.DIDContractVersion,
		this.native.DIDContractAddr,
		"initDID",
		[]interface{}{
			[]byte(didMethod),
		},
	)
}

func (this *DID) InitDID(gasPrice, gasLimit uint64, signer *Account, didMethod string) (common.Uint256, error) {
	tx, err := this.NewInitDIDTransaction(gasPrice, gasLimit, didMethod)
	if err != nil {
		return common.UINT256_EMPTY, err
	}
	if err := this.dnaSdk.SignToTransaction(tx, signer); err != nil {
		return common.UINT256_EMPTY, err
	}
	return this.dnaSdk.SendTransaction(tx)
}

func (this *DID) NewRegIDWithPublicKeyTransaction(gasPrice, gasLimit uint64, ontId string, pubKey keypair.PublicKey) (*types.MutableTransaction, error) {
	type regIDWithPublicKey struct {
		OntId  string
		PubKey []byte
	}
	return this.native.NewNativeInvokeTransaction(
		gasPrice,
		gasLimit,
		this.native.DIDContractVersion,
		this.native.DIDContractAddr,
		"regIDWithPublicKey",
		[]interface{}{
			&regIDWithPublicKey{
				OntId:  ontId,
				PubKey: keypair.SerializePublicKey(pubKey),
			},
		},
	)
}

func (this *DID) RegIDWithPublicKey(gasPrice, gasLimit uint64, signer *Account, ontId string, controller *Controller) (common.Uint256, error) {
	tx, err := this.NewRegIDWithPublicKeyTransaction(gasPrice, gasLimit, ontId, controller.PublicKey)
	if err != nil {
		return common.UINT256_EMPTY, err
	}
	err = this.dnaSdk.SignToTransaction(tx, signer)
	if err != nil {
		return common.UINT256_EMPTY, err
	}
	err = this.dnaSdk.SignToTransaction(tx, controller)
	if err != nil {
		return common.UINT256_EMPTY, err
	}
	return this.dnaSdk.SendTransaction(tx)
}

func (this *DID) NewRegIDWithAttributesTransaction(gasPrice, gasLimit uint64, ontId string, pubKey keypair.PublicKey, attributes []*DDOAttribute) (*types.MutableTransaction, error) {
	type regIDWithAttribute struct {
		OntId      string
		PubKey     []byte
		Attributes []*DDOAttribute
	}
	return this.native.NewNativeInvokeTransaction(
		gasPrice,
		gasLimit,
		this.native.DIDContractVersion,
		this.native.DIDContractAddr,
		"regIDWithAttributes",
		[]interface{}{
			&regIDWithAttribute{
				OntId:      ontId,
				PubKey:     keypair.SerializePublicKey(pubKey),
				Attributes: attributes,
			},
		},
	)
}

func (this *DID) RegIDWithAttributes(gasPrice, gasLimit uint64, signer *Account, ontId string, controller *Controller, attributes []*DDOAttribute) (common.Uint256, error) {
	tx, err := this.NewRegIDWithAttributesTransaction(gasPrice, gasLimit, ontId, controller.PublicKey, attributes)
	if err != nil {
		return common.UINT256_EMPTY, err
	}
	err = this.dnaSdk.SignToTransaction(tx, signer)
	if err != nil {
		return common.UINT256_EMPTY, err
	}
	err = this.dnaSdk.SignToTransaction(tx, controller)
	if err != nil {
		return common.UINT256_EMPTY, err
	}
	return this.dnaSdk.SendTransaction(tx)
}

func (this *DID) GetDDO(ontId string) (*DDO, error) {
	result, err := this.native.PreExecInvokeNativeContract(
		this.native.DIDContractAddr,
		this.native.DIDContractVersion,
		"getDDO",
		[]interface{}{ontId},
	)
	if err != nil {
		return nil, err
	}
	data, err := result.Result.ToByteArray()
	if err != nil {
		return nil, err
	}
	buf := bytes.NewBuffer(data)
	keyData, err := serialization.ReadVarBytes(buf)
	if err != nil {
		return nil, fmt.Errorf("key ReadVarBytes error:%s", err)
	}
	owners, err := this.getPublicKeys(ontId, keyData)
	if err != nil {
		return nil, fmt.Errorf("getPublicKeys error:%s", err)
	}
	attrData, err := serialization.ReadVarBytes(buf)
	attrs, err := this.getAttributes(ontId, attrData)
	if err != nil {
		return nil, fmt.Errorf("getAttributes error:%s", err)
	}
	recoveryData, err := serialization.ReadVarBytes(buf)
	if err != nil {
		return nil, fmt.Errorf("recovery ReadVarBytes error:%s", err)
	}
	var addr string
	if len(recoveryData) != 0 {
		address, err := common.AddressParseFromBytes(recoveryData)
		if err != nil {
			return nil, fmt.Errorf("AddressParseFromBytes error:%s", err)
		}
		addr = address.ToBase58()
	}

	ddo := &DDO{
		OntId:      ontId,
		Owners:     owners,
		Attributes: attrs,
		Recovery:   addr,
	}
	return ddo, nil
}

func (this *DID) NewAddKeyTransaction(gasPrice, gasLimit uint64, ontId string, newPubKey, pubKey keypair.PublicKey) (*types.MutableTransaction, error) {
	type addKey struct {
		OntId     string
		NewPubKey []byte
		PubKey    []byte
	}
	return this.native.NewNativeInvokeTransaction(
		gasPrice,
		gasLimit,
		this.native.DIDContractVersion,
		this.native.DIDContractAddr,
		"addKey",
		[]interface{}{
			&addKey{
				OntId:     ontId,
				NewPubKey: keypair.SerializePublicKey(newPubKey),
				PubKey:    keypair.SerializePublicKey(pubKey),
			},
		})
}

func (this *DID) AddKey(gasPrice, gasLimit uint64, ontId string, signer *Account, newPubKey keypair.PublicKey, controller *Controller) (common.Uint256, error) {
	tx, err := this.NewAddKeyTransaction(gasPrice, gasLimit, ontId, newPubKey, controller.PublicKey)
	if err != nil {
		return common.UINT256_EMPTY, err
	}
	err = this.dnaSdk.SignToTransaction(tx, signer)
	if err != nil {
		return common.UINT256_EMPTY, err
	}
	err = this.dnaSdk.SignToTransaction(tx, controller)
	if err != nil {
		return common.UINT256_EMPTY, err
	}
	return this.dnaSdk.SendTransaction(tx)
}

func (this *DID) NewRevokeKeyTransaction(gasPrice, gasLimit uint64, ontId string, removedPubKey, pubKey keypair.PublicKey) (*types.MutableTransaction, error) {
	type removeKey struct {
		OntId      string
		RemovedKey []byte
		PubKey     []byte
	}
	return this.native.NewNativeInvokeTransaction(
		gasPrice,
		gasLimit,
		this.native.DIDContractVersion,
		this.native.DIDContractAddr,
		"removeKey",
		[]interface{}{
			&removeKey{
				OntId:      ontId,
				RemovedKey: keypair.SerializePublicKey(removedPubKey),
				PubKey:     keypair.SerializePublicKey(pubKey),
			},
		},
	)
}

func (this *DID) RevokeKey(gasPrice, gasLimit uint64, ontId string, signer *Account, removedPubKey keypair.PublicKey, controller *Controller) (common.Uint256, error) {
	tx, err := this.NewRevokeKeyTransaction(gasPrice, gasLimit, ontId, removedPubKey, controller.PublicKey)
	if err != nil {
		return common.UINT256_EMPTY, err
	}
	err = this.dnaSdk.SignToTransaction(tx, signer)
	if err != nil {
		return common.UINT256_EMPTY, err
	}
	err = this.dnaSdk.SignToTransaction(tx, controller)
	if err != nil {
		return common.UINT256_EMPTY, err
	}
	return this.dnaSdk.SendTransaction(tx)
}

func (this *DID) NewSetRecoveryTransaction(gasPrice, gasLimit uint64, ontId string, recovery common.Address, pubKey keypair.PublicKey) (*types.MutableTransaction, error) {
	type addRecovery struct {
		OntId    string
		Recovery common.Address
		Pubkey   []byte
	}
	return this.native.NewNativeInvokeTransaction(
		gasPrice,
		gasLimit,
		this.native.DIDContractVersion,
		this.native.DIDContractAddr,
		"addRecovery",
		[]interface{}{
			&addRecovery{
				OntId:    ontId,
				Recovery: recovery,
				Pubkey:   keypair.SerializePublicKey(pubKey),
			},
		})
}

func (this *DID) SetRecovery(gasPrice, gasLimit uint64, signer *Account, ontId string, recovery common.Address, controller *Controller) (common.Uint256, error) {
	tx, err := this.NewSetRecoveryTransaction(gasPrice, gasLimit, ontId, recovery, controller.PublicKey)
	if err != nil {
		return common.UINT256_EMPTY, err
	}
	err = this.dnaSdk.SignToTransaction(tx, signer)
	if err != nil {
		return common.UINT256_EMPTY, err
	}
	err = this.dnaSdk.SignToTransaction(tx, controller)
	if err != nil {
		return common.UINT256_EMPTY, err
	}
	return this.dnaSdk.SendTransaction(tx)
}

func (this *DID) NewChangeRecoveryTransaction(gasPrice, gasLimit uint64, ontId string, newRecovery, oldRecovery common.Address) (*types.MutableTransaction, error) {
	type changeRecovery struct {
		OntId       string
		NewRecovery common.Address
		OldRecovery common.Address
	}
	return this.native.NewNativeInvokeTransaction(
		gasPrice,
		gasLimit,
		this.native.DIDContractVersion,
		this.native.DIDContractAddr,
		"changeRecovery",
		[]interface{}{
			&changeRecovery{
				OntId:       ontId,
				NewRecovery: newRecovery,
				OldRecovery: oldRecovery,
			},
		})
}

func (this *DID) ChangeRecovery(gasPrice, gasLimit uint64, signer *Account, ontId string, newRecovery, oldRecovery common.Address, controller *Controller) (common.Uint256, error) {
	tx, err := this.NewChangeRecoveryTransaction(gasPrice, gasLimit, ontId, newRecovery, oldRecovery)
	if err != nil {
		return common.UINT256_EMPTY, err
	}
	err = this.dnaSdk.SignToTransaction(tx, signer)
	if err != nil {
		return common.UINT256_EMPTY, err
	}
	err = this.dnaSdk.SignToTransaction(tx, controller)
	if err != nil {
		return common.UINT256_EMPTY, err
	}
	return this.dnaSdk.SendTransaction(tx)
}

func (this *DID) NewAddAttributesTransaction(gasPrice, gasLimit uint64, ontId string, attributes []*DDOAttribute, pubKey keypair.PublicKey) (*types.MutableTransaction, error) {
	type addAttributes struct {
		OntId      string
		Attributes []*DDOAttribute
		PubKey     []byte
	}
	return this.native.NewNativeInvokeTransaction(
		gasPrice,
		gasLimit,
		this.native.DIDContractVersion,
		this.native.DIDContractAddr,
		"addAttributes",
		[]interface{}{
			&addAttributes{
				OntId:      ontId,
				Attributes: attributes,
				PubKey:     keypair.SerializePublicKey(pubKey),
			},
		})
}

func (this *DID) AddAttributes(gasPrice, gasLimit uint64, signer *Account, ontId string, attributes []*DDOAttribute, controller *Controller) (common.Uint256, error) {
	tx, err := this.NewAddAttributesTransaction(gasPrice, gasLimit, ontId, attributes, controller.PublicKey)
	if err != nil {
		return common.UINT256_EMPTY, err
	}
	err = this.dnaSdk.SignToTransaction(tx, signer)
	if err != nil {
		return common.UINT256_EMPTY, err
	}
	err = this.dnaSdk.SignToTransaction(tx, controller)
	if err != nil {
		return common.UINT256_EMPTY, err
	}

	return this.dnaSdk.SendTransaction(tx)
}

func (this *DID) NewRemoveAttributeTransaction(gasPrice, gasLimit uint64, ontId string, key []byte, pubKey keypair.PublicKey) (*types.MutableTransaction, error) {
	type removeAttribute struct {
		OntId  string
		Key    []byte
		PubKey []byte
	}
	return this.native.NewNativeInvokeTransaction(
		gasPrice,
		gasLimit,
		this.native.DIDContractVersion,
		this.native.DIDContractAddr,
		"removeAttribute",
		[]interface{}{
			&removeAttribute{
				OntId:  ontId,
				Key:    key,
				PubKey: keypair.SerializePublicKey(pubKey),
			},
		})
}

func (this *DID) RemoveAttribute(gasPrice, gasLimit uint64, signer *Account, ontId string, removeKey []byte, controller *Controller) (common.Uint256, error) {
	tx, err := this.NewRemoveAttributeTransaction(gasPrice, gasLimit, ontId, removeKey, controller.PublicKey)
	if err != nil {
		return common.UINT256_EMPTY, err
	}
	err = this.dnaSdk.SignToTransaction(tx, signer)
	if err != nil {
		return common.UINT256_EMPTY, err
	}
	err = this.dnaSdk.SignToTransaction(tx, controller)
	if err != nil {
		return common.UINT256_EMPTY, err
	}

	return this.dnaSdk.SendTransaction(tx)
}

func (this *DID) GetAttributes(ontId string) ([]*DDOAttribute, error) {
	preResult, err := this.native.PreExecInvokeNativeContract(
		this.native.DIDContractAddr,
		this.native.DIDContractVersion,
		"getAttributes",
		[]interface{}{ontId})
	if err != nil {
		return nil, err
	}
	data, err := preResult.Result.ToByteArray()
	if err != nil {
		return nil, fmt.Errorf("ToByteArray error:%s", err)
	}
	return this.getAttributes(ontId, data)
}

func (this *DID) getAttributes(ontId string, data []byte) ([]*DDOAttribute, error) {
	buf := bytes.NewBuffer(data)
	attributes := make([]*DDOAttribute, 0)
	for {
		if buf.Len() == 0 {
			break
		}
		key, err := serialization.ReadVarBytes(buf)
		if err != nil {
			return nil, fmt.Errorf("key ReadVarBytes error:%s", err)
		}
		valueType, err := serialization.ReadVarBytes(buf)
		if err != nil {
			return nil, fmt.Errorf("value type ReadVarBytes error:%s", err)
		}
		value, err := serialization.ReadVarBytes(buf)
		if err != nil {
			return nil, fmt.Errorf("value ReadVarBytes error:%s", err)
		}
		attributes = append(attributes, &DDOAttribute{
			Key:       key,
			Value:     value,
			ValueType: valueType,
		})
	}
	//reverse
	for i, j := 0, len(attributes)-1; i < j; i, j = i+1, j-1 {
		attributes[i], attributes[j] = attributes[j], attributes[i]
	}
	return attributes, nil
}

func (this *DID) VerifySignature(ontId string, keyIndex int, controller *Controller) (bool, error) {
	tx, err := this.native.NewNativeInvokeTransaction(
		0, 0,
		this.native.DIDContractVersion,
		this.native.DIDContractAddr,
		"verifySignature",
		[]interface{}{ontId, keyIndex})
	if err != nil {
		return false, err
	}
	err = this.dnaSdk.SignToTransaction(tx, controller)
	if err != nil {
		return false, err
	}
	preResult, err := this.dnaSdk.PreExecTransaction(tx)
	if err != nil {
		return false, err
	}
	return preResult.Result.ToBool()
}

func (this *DID) GetPublicKeys(ontId string) ([]*DDOOwner, error) {
	preResult, err := this.native.PreExecInvokeNativeContract(
		this.native.DIDContractAddr,
		this.native.DIDContractVersion,
		"getPublicKeys",
		[]interface{}{
			ontId,
		})
	if err != nil {
		return nil, err
	}
	data, err := preResult.Result.ToByteArray()
	if err != nil {
		return nil, err
	}
	return this.getPublicKeys(ontId, data)
}

func (this *DID) getPublicKeys(ontId string, data []byte) ([]*DDOOwner, error) {
	buf := bytes.NewBuffer(data)
	owners := make([]*DDOOwner, 0)
	for {
		if buf.Len() == 0 {
			break
		}
		index, err := serialization.ReadUint32(buf)
		if err != nil {
			return nil, fmt.Errorf("index ReadUint32 error:%s", err)
		}
		pubKeyId := fmt.Sprintf("%s#keys-%d", ontId, index)
		pkData, err := serialization.ReadVarBytes(buf)
		if err != nil {
			return nil, fmt.Errorf("PubKey Idenx:%d ReadVarBytes error:%s", index, err)
		}
		pubKey, err := keypair.DeserializePublicKey(pkData)
		if err != nil {
			return nil, fmt.Errorf("DeserializePublicKey Index:%d error:%s", index, err)
		}
		keyType := keypair.GetKeyType(pubKey)
		owner := &DDOOwner{
			pubKeyIndex: index,
			PubKeyId:    pubKeyId,
			Type:        GetKeyTypeString(keyType),
			Curve:       GetCurveName(pkData),
			Value:       hex.EncodeToString(pkData),
		}
		owners = append(owners, owner)
	}
	return owners, nil
}

func (this *DID) GetKeyState(ontId string, keyIndex int) (string, error) {
	type keyState struct {
		OntId    string
		KeyIndex int
	}
	preResult, err := this.native.PreExecInvokeNativeContract(
		this.native.DIDContractAddr,
		this.native.DIDContractVersion,
		"getKeyState",
		[]interface{}{
			&keyState{
				OntId:    ontId,
				KeyIndex: keyIndex,
			},
		})
	if err != nil {
		return "", err
	}
	return preResult.Result.ToString()
}

type GlobalParam struct {
	dnaSkd *DNASdk
	native *NativeContract
}

func (this *GlobalParam) GetGlobalParams(params []string) (map[string]string, error) {
	preResult, err := this.native.PreExecInvokeNativeContract(
		GLOABL_PARAMS_CONTRACT_ADDRESS,
		GLOBAL_PARAMS_CONTRACT_VERSION,
		global_params.GET_GLOBAL_PARAM_NAME,
		[]interface{}{params})
	if err != nil {
		return nil, err
	}
	results, err := preResult.Result.ToByteArray()
	if err != nil {
		return nil, err
	}
	queryParams := new(global_params.Params)
	err = queryParams.Deserialization(common.NewZeroCopySource(results))
	if err != nil {
		return nil, err
	}
	globalParams := make(map[string]string, len(params))
	for _, param := range params {
		index, values := queryParams.GetParam(param)
		if index < 0 {
			continue
		}
		globalParams[param] = values.Value
	}
	return globalParams, nil
}

func (this *GlobalParam) NewSetGlobalParamsTransaction(gasPrice, gasLimit uint64, params map[string]string) (*types.MutableTransaction, error) {
	var globalParams global_params.Params
	for k, v := range params {
		globalParams.SetParam(global_params.Param{Key: k, Value: v})
	}
	return this.native.NewNativeInvokeTransaction(
		gasPrice,
		gasLimit,
		GLOBAL_PARAMS_CONTRACT_VERSION,
		GLOABL_PARAMS_CONTRACT_ADDRESS,
		global_params.SET_GLOBAL_PARAM_NAME,
		[]interface{}{globalParams})
}

func (this *GlobalParam) SetGlobalParams(gasPrice, gasLimit uint64, signer *Account, params map[string]string) (common.Uint256, error) {
	tx, err := this.NewSetGlobalParamsTransaction(gasPrice, gasLimit, params)
	if err != nil {
		return common.UINT256_EMPTY, err
	}
	err = this.dnaSkd.SignToTransaction(tx, signer)
	if err != nil {
		return common.UINT256_EMPTY, err
	}
	return this.dnaSkd.SendTransaction(tx)
}

func (this *GlobalParam) NewTransferAdminTransaction(gasPrice, gasLimit uint64, newAdmin common.Address) (*types.MutableTransaction, error) {
	return this.native.NewNativeInvokeTransaction(
		gasPrice,
		gasLimit,
		GLOBAL_PARAMS_CONTRACT_VERSION,
		GLOABL_PARAMS_CONTRACT_ADDRESS,
		global_params.TRANSFER_ADMIN_NAME,
		[]interface{}{newAdmin})
}

func (this *GlobalParam) TransferAdmin(gasPrice, gasLimit uint64, signer *Account, newAdmin common.Address) (common.Uint256, error) {
	tx, err := this.NewTransferAdminTransaction(gasPrice, gasLimit, newAdmin)
	if err != nil {
		return common.UINT256_EMPTY, err
	}
	err = this.dnaSkd.SignToTransaction(tx, signer)
	if err != nil {
		return common.UINT256_EMPTY, err
	}
	return this.dnaSkd.SendTransaction(tx)
}

func (this *GlobalParam) NewAcceptAdminTransaction(gasPrice, gasLimit uint64, admin common.Address) (*types.MutableTransaction, error) {
	return this.native.NewNativeInvokeTransaction(
		gasPrice,
		gasLimit,
		GLOBAL_PARAMS_CONTRACT_VERSION,
		GLOABL_PARAMS_CONTRACT_ADDRESS,
		global_params.ACCEPT_ADMIN_NAME,
		[]interface{}{admin})
}

func (this *GlobalParam) AcceptAdmin(gasPrice, gasLimit uint64, signer *Account) (common.Uint256, error) {
	tx, err := this.NewAcceptAdminTransaction(gasPrice, gasLimit, signer.Address)
	if err != nil {
		return common.UINT256_EMPTY, err
	}
	err = this.dnaSkd.SignToTransaction(tx, signer)
	if err != nil {
		return common.UINT256_EMPTY, err
	}
	return this.dnaSkd.SendTransaction(tx)
}

func (this *GlobalParam) NewSetOperatorTransaction(gasPrice, gasLimit uint64, operator common.Address) (*types.MutableTransaction, error) {
	return this.native.NewNativeInvokeTransaction(
		gasPrice,
		gasLimit,
		GLOBAL_PARAMS_CONTRACT_VERSION,
		GLOABL_PARAMS_CONTRACT_ADDRESS,
		global_params.SET_OPERATOR,
		[]interface{}{operator},
	)
}

func (this *GlobalParam) SetOperator(gasPrice, gasLimit uint64, signer *Account, operator common.Address) (common.Uint256, error) {
	tx, err := this.NewSetOperatorTransaction(gasPrice, gasLimit, operator)
	if err != nil {
		return common.UINT256_EMPTY, err
	}
	err = this.dnaSkd.SignToTransaction(tx, signer)
	if err != nil {
		return common.UINT256_EMPTY, err
	}
	return this.dnaSkd.SendTransaction(tx)
}

func (this *GlobalParam) NewCreateSnapshotTransaction(gasPrice, gasLimit uint64) (*types.MutableTransaction, error) {
	return this.native.NewNativeInvokeTransaction(
		gasPrice,
		gasLimit,
		GLOBAL_PARAMS_CONTRACT_VERSION,
		GLOABL_PARAMS_CONTRACT_ADDRESS,
		global_params.CREATE_SNAPSHOT_NAME,
		[]interface{}{},
	)
}

func (this *GlobalParam) CreateSnapshot(gasPrice, gasLimit uint64, signer *Account) (common.Uint256, error) {
	tx, err := this.NewCreateSnapshotTransaction(gasPrice, gasLimit)
	if err != nil {
		return common.UINT256_EMPTY, err
	}
	err = this.dnaSkd.SignToTransaction(tx, signer)
	if err != nil {
		return common.UINT256_EMPTY, err
	}
	return this.dnaSkd.SendTransaction(tx)
}

type Auth struct {
	dnaSkd *DNASdk
	native *NativeContract
}

func (this *Auth) NewInitAuthTransaction(gasPrice, gasLimit uint64) (*types.MutableTransaction, error) {
	return this.native.NewNativeInvokeTransaction(
		gasPrice,
		gasLimit,
		this.native.AuthContractVersion,
		this.native.AuthContractAddr,
		"initAuth",
		[]interface{} {
			this.native.DIDContractAddr[:],
		},
	)
}

func (this *Auth) InitAuth(gasPrice, gasLimit uint64, signer *Account) (common.Uint256, error) {
	tx, err := this.NewInitAuthTransaction(gasPrice, gasLimit)
	if err != nil {
		return common.UINT256_EMPTY, err
	}
	if err := this.dnaSkd.SignToTransaction(tx, signer); err != nil {
		return common.UINT256_EMPTY, err
	}
	return this.dnaSkd.SendTransaction(tx)
}

func (this *Auth) NewAssignFuncsToRoleTransaction(gasPrice, gasLimit uint64, contractAddress common.Address, adminId, role []byte, funcNames []string, keyIndex int) (*types.MutableTransaction, error) {
	return this.native.NewNativeInvokeTransaction(
		gasPrice,
		gasLimit,
		this.native.AuthContractVersion,
		this.native.AuthContractAddr,
		"assignFuncsToRole",
		[]interface{}{
			contractAddress,
			adminId,
			role,
			funcNames,
			keyIndex,
		})
}

func (this *Auth) AssignFuncsToRole(gasPrice, gasLimit uint64, contractAddress common.Address, signer *Account, adminId, role []byte, funcNames []string, keyIndex int) (common.Uint256, error) {
	tx, err := this.NewAssignFuncsToRoleTransaction(gasPrice, gasLimit, contractAddress, adminId, role, funcNames, keyIndex)
	if err != nil {
		return common.UINT256_EMPTY, err
	}
	err = this.dnaSkd.SignToTransaction(tx, signer)
	if err != nil {
		return common.UINT256_EMPTY, err
	}
	return this.dnaSkd.SendTransaction(tx)
}

func (this *Auth) NewDelegateTransaction(gasPrice, gasLimit uint64, contractAddress common.Address, from, to, role []byte, period, level, keyIndex int) (*types.MutableTransaction, error) {
	return this.native.NewNativeInvokeTransaction(
		gasPrice,
		gasLimit,
		this.native.AuthContractVersion,
		this.native.AuthContractAddr,
		"delegate",
		[]interface{}{
			contractAddress,
			from,
			to,
			role,
			period,
			level,
			keyIndex,
		})
}

func (this *Auth) Delegate(gasPrice, gasLimit uint64, signer *Account, contractAddress common.Address, from, to, role []byte, period, level, keyIndex int) (common.Uint256, error) {
	tx, err := this.NewDelegateTransaction(gasPrice, gasLimit, contractAddress, from, to, role, period, level, keyIndex)
	if err != nil {
		return common.UINT256_EMPTY, err
	}
	err = this.dnaSkd.SignToTransaction(tx, signer)
	if err != nil {
		return common.UINT256_EMPTY, err
	}
	return this.dnaSkd.SendTransaction(tx)
}

func (this *Auth) NewWithdrawTransaction(gasPrice, gasLimit uint64, contractAddress common.Address, initiator, delegate, role []byte, keyIndex int) (*types.MutableTransaction, error) {
	return this.native.NewNativeInvokeTransaction(
		gasPrice,
		gasLimit,
		this.native.AuthContractVersion,
		this.native.AuthContractAddr,
		"withdraw",
		[]interface{}{
			contractAddress,
			initiator,
			delegate,
			role,
			keyIndex,
		})
}

func (this *Auth) Withdraw(gasPrice, gasLimit uint64, signer *Account, contractAddress common.Address, initiator, delegate, role []byte, keyIndex int) (common.Uint256, error) {
	tx, err := this.NewWithdrawTransaction(gasPrice, gasLimit, contractAddress, initiator, delegate, role, keyIndex)
	if err != nil {
		return common.UINT256_EMPTY, err
	}
	err = this.dnaSkd.SignToTransaction(tx, signer)
	if err != nil {
		return common.UINT256_EMPTY, err
	}
	return this.dnaSkd.SendTransaction(tx)
}

func (this *Auth) NewAssignOntIDsToRoleTransaction(gasPrice, gasLimit uint64, contractAddress common.Address, admontId, role []byte, persons [][]byte, keyIndex int) (*types.MutableTransaction, error) {
	return this.native.NewNativeInvokeTransaction(
		gasPrice,
		gasLimit,
		this.native.AuthContractVersion,
		this.native.AuthContractAddr,
		"assignOntIDsToRole",
		[]interface{}{
			contractAddress,
			admontId,
			role,
			persons,
			keyIndex,
		})
}

func (this *Auth) AssignOntIDsToRole(gasPrice, gasLimit uint64, signer *Account, contractAddress common.Address, admontId, role []byte, persons [][]byte, keyIndex int) (common.Uint256, error) {
	tx, err := this.NewAssignOntIDsToRoleTransaction(gasPrice, gasLimit, contractAddress, admontId, role, persons, keyIndex)
	if err != nil {
		return common.UINT256_EMPTY, err
	}
	err = this.dnaSkd.SignToTransaction(tx, signer)
	if err != nil {
		return common.UINT256_EMPTY, err
	}
	return this.dnaSkd.SendTransaction(tx)
}

func (this *Auth) NewTransferTransaction(gasPrice, gasLimit uint64, contractAddress common.Address, newAdminId []byte, keyIndex int) (*types.MutableTransaction, error) {
	return this.native.NewNativeInvokeTransaction(
		gasPrice,
		gasLimit,
		this.native.AuthContractVersion,
		this.native.AuthContractAddr,
		"transfer",
		[]interface{}{
			contractAddress,
			newAdminId,
			keyIndex,
		})
}

func (this *Auth) Transfer(gasPrice, gasLimit uint64, signer *Account, contractAddress common.Address, newAdminId []byte, keyIndex int) (common.Uint256, error) {
	tx, err := this.NewTransferTransaction(gasPrice, gasLimit, contractAddress, newAdminId, keyIndex)
	if err != nil {
		return common.UINT256_EMPTY, err
	}
	err = this.dnaSkd.SignToTransaction(tx, signer)
	if err != nil {
		return common.UINT256_EMPTY, err
	}
	return this.dnaSkd.SendTransaction(tx)
}

func (this *Auth) NewVerifyTokenTransaction(gasPrice, gasLimit uint64, contractAddress common.Address, caller []byte, funcName string, keyIndex int) (*types.MutableTransaction, error) {
	return this.native.NewNativeInvokeTransaction(
		gasPrice,
		gasLimit,
		this.native.AuthContractVersion,
		this.native.AuthContractAddr,
		"verifyToken",
		[]interface{}{
			contractAddress,
			caller,
			funcName,
			keyIndex,
		})
}

func (this *Auth) VerifyToken(gasPrice, gasLimit uint64, signer *Account, contractAddress common.Address, caller []byte, funcName string, keyIndex int) (common.Uint256, error) {
	tx, err := this.NewVerifyTokenTransaction(gasPrice, gasLimit, contractAddress, caller, funcName, keyIndex)
	if err != nil {
		return common.UINT256_EMPTY, err
	}
	err = this.dnaSkd.SignToTransaction(tx, signer)
	if err != nil {
		return common.UINT256_EMPTY, err
	}
	return this.dnaSkd.SendTransaction(tx)
}
