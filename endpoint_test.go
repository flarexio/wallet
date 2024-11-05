package wallet

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	_ "github.com/gagliardetto/solana-go/programs/token"
)

func TestSignTransaction(t *testing.T) {
	assert := assert.New(t)

	jsonStr := `{
    	"tx": "AUxmDVEjHdhE/OZBPho1Fdd88czxuR6HWLwPImTXHDde9HflsFWvoNkBwQKEEglLYVEMRHQpt6ZBQgRN+S+kngyAAQABBIWoLkhOe3hh6oAqa6hiLEMT00sl0kyvr3MDCIr8uPQmpN7WQKXTyLBX3+cRuRRTT4YAzIew2h2FkQhd3T3vw97M7AC0UN8mhMbaYQECJb9KqJ+8ub9oQQa0lCBE2Y4Mzwbd9uHXZaGT2cvhRs7reawctIXtX1s3kTqM9YV+/wCpAzT6gw+Tyy0CM1xciNYzBYeMsiuhDk+wLJu2f5uxGrABAwMBAgAJAwCAxqR+jQMAAA=="
	}`

	var req SignTransactionRequest
	err := json.Unmarshal([]byte(jsonStr), &req)
	if err != nil {
		assert.Fail(err.Error())
		return
	}

	fmt.Println(req.Transaction.String())

	resp := &SignTransactionResponse{
		Transaction: req.Transaction,
	}

	_, err = json.Marshal(resp)
	assert.NoError(err)
}
