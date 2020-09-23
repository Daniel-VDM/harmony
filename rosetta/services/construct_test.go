package services

import (
	"bytes"
	"encoding/json"
	"math/big"
	"reflect"
	"testing"

	"github.com/ethereum/go-ethereum/common/hexutil"

	"github.com/coinbase/rosetta-sdk-go/types"
	"github.com/ethereum/go-ethereum/crypto"
	hmytypes "github.com/harmony-one/harmony/core/types"
	stakingTypes "github.com/harmony-one/harmony/staking/types"
)

func TestGetAddressFromPublicKey(t *testing.T) {
	key, err := crypto.GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	refAddr := crypto.PubkeyToAddress(key.PublicKey)
	compressedPublicKey := crypto.CompressPubkey(&key.PublicKey)
	addr, rosettaError := getAddressFromPublicKey(&types.PublicKey{
		Bytes:     compressedPublicKey,
		CurveType: types.Secp256k1,
	})
	if rosettaError != nil {
		t.Fatal(rosettaError)
	}
	if !reflect.DeepEqual(refAddr, *addr) {
		t.Errorf("expected adder %v, got %v", refAddr, *addr)
	}
}

func TestUnpackWrappedTransactionFromHexString(t *testing.T) {
	key, err := crypto.GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	refAddr := crypto.PubkeyToAddress(key.PublicKey)
	refAddrID, rosettaError := newAccountIdentifier(refAddr)
	if rosettaError != nil {
		t.Fatal(rosettaError)
	}
	refEstGasUsed := big.NewInt(100000)
	signer := hmytypes.NewEIP155Signer(big.NewInt(0))

	// Test plain transactions
	tx, err := createTestTransaction(
		signer, 0, 1, 2, refEstGasUsed.Uint64(), big.NewInt(1e10), []byte{0x01, 0x02},
	)
	if err != nil {
		t.Fatal(err)
	}
	buf := &bytes.Buffer{}
	if err := tx.EncodeRLP(buf); err != nil {
		t.Fatal(err)
	}
	wrappedTransaction := WrappedTransaction{
		RLPBytes:         buf.Bytes(),
		From:             refAddrID,
		EstimatedGasUsed: refEstGasUsed.Uint64(),
		IsStaking:        false,
	}
	marshalledBytes, err := json.Marshal(wrappedTransaction)
	if err != nil {
		t.Fatal(err)
	}
	testWrappedTx, testTx, rosettaError := unpackWrappedTransactionFromHexString(hexutil.Encode(marshalledBytes))
	if rosettaError != nil {
		t.Fatal(rosettaError)
	}
	if types.Hash(tx) != types.Hash(testTx) {
		t.Error("unwrapped tx does not match reference tx")
	}
	if types.Hash(testWrappedTx) != types.Hash(wrappedTransaction) {
		t.Error("unwrapped tx struct does not matched reference tx struct")
	}

	// Test staking transactions
	receiverKey, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf(err.Error())
	}
	stx, err := createTestStakingTransaction(func() (stakingTypes.Directive, interface{}) {
		return stakingTypes.DirectiveDelegate, stakingTypes.Delegate{
			DelegatorAddress: refAddr,
			ValidatorAddress: crypto.PubkeyToAddress(receiverKey.PublicKey),
			Amount:           tenOnes,
		}
	}, key, 10, refEstGasUsed.Uint64())
	if err != nil {
		t.Fatal(err)
	}
	buf = &bytes.Buffer{}
	if err := stx.EncodeRLP(buf); err != nil {
		t.Fatal(err)
	}
	wrappedTransaction.RLPBytes = buf.Bytes()
	wrappedTransaction.IsStaking = true
	marshalledBytes, err = json.Marshal(wrappedTransaction)
	if err != nil {
		t.Fatal(err)
	}
	testWrappedTx, testStx, rosettaError := unpackWrappedTransactionFromHexString(hexutil.Encode(marshalledBytes))
	if rosettaError != nil {
		t.Fatal(rosettaError)
	}
	if types.Hash(testStx) != types.Hash(stx) {
		t.Error("unwrapped tx does not match reference tx")
	}
	if types.Hash(testWrappedTx) != types.Hash(wrappedTransaction) {
		t.Error("unwrapped tx struct does not matched reference tx struct")
	}

	// Test Invalid hex string
	_, _, rosettaError = unpackWrappedTransactionFromHexString("asdasd")
	if rosettaError == nil {
		t.Fatal("expected error")
	}

	// Test invalid marshall
	marshalledBytesFail := marshalledBytes[:]
	marshalledBytesFail[0] = 0x0
	_, _, rosettaError = unpackWrappedTransactionFromHexString(hexutil.Encode(marshalledBytesFail))
	if rosettaError == nil {
		t.Fatal("expected error")
	}

	// test invalid RLP encoding for staking
	wrappedTransaction.RLPBytes = []byte{0x0}
	marshalledBytesFail, err = json.Marshal(wrappedTransaction)
	if err != nil {
		t.Fatal(err)
	}
	_, _, rosettaError = unpackWrappedTransactionFromHexString(hexutil.Encode(marshalledBytesFail))
	if rosettaError == nil {
		t.Fatal("expected error")
	}

	// test invalid RLP encoding for plain
	wrappedTransaction.IsStaking = false
	marshalledBytesFail, err = json.Marshal(wrappedTransaction)
	if err != nil {
		t.Fatal(err)
	}
	_, _, rosettaError = unpackWrappedTransactionFromHexString(hexutil.Encode(marshalledBytesFail))
	if rosettaError == nil {
		t.Fatal("expected error")
	}

}
