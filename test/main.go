// SPDX-License-Identifier: LGPL-3.0-or-later
// Copyright 2019 the DNA Dev team
// Copyright 2018 the Ontology Dev team
//
package main

import (
	"fmt"
	sdk "github.com/DNAProject/DNA-go-sdk"
	"github.com/DNAProject/DNA/core/payload"
)

func main() {
	testDnaSdk := sdk.NewDNASdk()
	testDnaSdk.NewRpcClient().SetAddress("http://dappnode1.dna.io:20336")
	for i := uint32(4513925); i > 100000; i++ {
		block, err := testDnaSdk.GetBlockByHeight(i)
		if err != nil {
			fmt.Println("error: ", err)
			return
		}
		for _, tx := range block.Transactions {
			invokeCode, ok := tx.Payload.(*payload.InvokeCode)
			if ok {
				res, err := sdk.ParsePayload(invokeCode.Code)
				if err != nil {
					//fmt.Printf("error: %s, height:%d\n", err, i)
					continue
				}
				fmt.Println("res:", res)
				fmt.Printf("height: %d\n", i)
			}
		}
	}
}
