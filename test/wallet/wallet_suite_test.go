package wallet_test

import (
	"context"
    "crypto/ecdsa"
    "errors"
	"fmt"
    "math/big"
	"os"
	"strings"
	"testing"

    "github.com/ethereum/go-ethereum"
    "github.com/ethereum/go-ethereum/accounts/abi"
    "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
    "github.com/ethereum/go-ethereum/crypto"
	"github.com/Masterminds/semver"
	"github.com/tokencard/contracts/v2/pkg/bindings"
	. "github.com/tokencard/contracts/v2/test/shared"
    . "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var Wallet *bindings.Wallet
var WalletAddress common.Address

func ethCall(tx *types.Transaction) ([]byte, error) {
	msg, _ := tx.AsMessage(types.HomesteadSigner{})

	calMsg := ethereum.CallMsg{
		From:     msg.From(),
		To:       msg.To(),
		Gas:      msg.Gas(),
		GasPrice: msg.GasPrice(),
		Value:    msg.Value(),
		Data:     msg.Data(),
	}

	return Backend.CallContract(context.Background(), calMsg, nil)
}

func SignData(nonce *big.Int, data []byte, prv *ecdsa.PrivateKey) ([]byte, error) {
	relayMessage := fmt.Sprintf("rlx:%s%s", abi.U256(nonce), data)
	hash := crypto.Keccak256([]byte(relayMessage))
	ethMessage := fmt.Sprintf("\x19Ethereum Signed Message:\n%d%s", len(hash), hash)
	hash = crypto.Keccak256([]byte(ethMessage))
	sig, err := crypto.Sign(hash, prv)
	if err != nil {
		return nil, err
	}
	if len(sig) != 65 {
		return nil, errors.New("invalid sig len")
	}
	sig[64] += 27
	return sig, nil
}

func SignMsg(msg []byte, prv *ecdsa.PrivateKey) ([]byte, error) {
	hash := crypto.Keccak256(msg)
	sig, err := crypto.Sign(hash, prv)
	if err != nil {
		return nil, err
	}
	if len(sig) != 65 {
		return nil, errors.New("invalid sig len")
	}
	sig[64] += 27
	return sig, nil
}

func isGasExhausted(tx *types.Transaction, gasLimit uint64) bool {
	r, err := Backend.TransactionReceipt(context.Background(), tx.Hash())
	Expect(err).ToNot(HaveOccurred())
	if r.Status == types.ReceiptStatusSuccessful {
		return false
	}
	return r.GasUsed == gasLimit
}

func init() {
	TestRig.AddCoverageForContracts("../../build/wallet/combined.json", "../../contracts")
}

func TestWalletSuite(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Contract Suite")
}

var _ = BeforeEach(func() {
	err := InitializeBackend()
	Expect(err).ToNot(HaveOccurred())
	// Deploy the Token wallet contract.
	var tx *types.Transaction
	WalletAddress, tx, Wallet, err = bindings.DeployWallet(BankAccount.TransactOpts(), Backend, Owner.Address(), true, ENSRegistryAddress, TokenWhitelistName, ControllerName, LicenceName, EthToWei(100))
	Expect(err).ToNot(HaveOccurred())
	Backend.Commit()
	Expect(isSuccessful(tx)).To(BeTrue())
})

var allPassed = true
var currentVersion = "2.3.0"

var _ = Describe("Wallet Version", func() {
	It("should return the current version", func() {
		v, err := Wallet.WALLETVERSION(nil)
		Expect(err).ToNot(HaveOccurred())
		Expect(v).To(Equal(currentVersion))
	})

	It("should be a Semver", func() {
		v, err := Wallet.WALLETVERSION(nil)
		Expect(err).ToNot(HaveOccurred())
		_, err = semver.NewVersion(v)
		Expect(err).ToNot(HaveOccurred())
	})

	It("should not start with a v prefix", func() {
		v, err := Wallet.WALLETVERSION(nil)
		Expect(err).ToNot(HaveOccurred())
		Expect(strings.HasPrefix(v, "v")).To(BeFalse())
	})
})

var _ = AfterEach(func() {
	td := CurrentGinkgoTestDescription()
	if td.Failed {
		allPassed = false
	}

})

var _ = AfterSuite(func() {
	if allPassed {
		TestRig.ExpectMinimumCoverage("wallet.sol", 98.00)
		TestRig.PrintGasUsage(os.Stdout)
	}
})

func isSuccessful(tx *types.Transaction) bool {
	r, err := Backend.TransactionReceipt(context.Background(), tx.Hash())
	Expect(err).ToNot(HaveOccurred())
	return r.Status == types.ReceiptStatusSuccessful
}

var _ = AfterEach(func() {
	td := CurrentGinkgoTestDescription()
	if td.Failed {
		fmt.Fprintf(GinkgoWriter, "\nLast Executed Smart Contract Line for %s:%d\n", td.FileName, td.LineNumber)
		fmt.Fprintln(GinkgoWriter, TestRig.LastExecuted())
	}
	err := Backend.Close()
	Expect(err).ToNot(HaveOccurred())
})
