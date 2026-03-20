package auth

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math/big"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"elv-mcp-experiment/elv-mcp-experiment/types"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/eluv-io/common-go/format/eat"
	"github.com/eluv-io/common-go/format/id"
	types2 "github.com/eluv-io/common-go/format/types"
	"github.com/eluv-io/utc-go"
)

func generateStateChannel() string {

	ethUrl := os.Getenv("ETH_URL")
	pkStr := os.Getenv("PRIVATE_KEY")
	qIdStr := os.Getenv("QID_INDEX")
	usrCtx := make([]map[string]interface{}, 0)

	ec, err := ethclient.Dial(ethUrl)
	if err != nil {
		log.Fatal("Error connecting to the node", err)
	}
	defer ec.Close()

	if strings.HasPrefix(pkStr, "0x") {
		pkStr = pkStr[2:]
	}
	pk, err := crypto.HexToECDSA(pkStr)
	if err != nil {
		log.Fatal("Error generating private key", err)
	}
	userAddr := crypto.PubkeyToAddress(pk.PublicKey)

	qId, err := id.FromString(qIdStr)
	if err != nil {
		log.Fatal(err)
	}
	contentAddress := IDToAddress(qId)

	tsMillis := time.Now().UnixNano() / (1000 * 1000)

	sigHash, err := hashBatchTransaction(
		userAddr,
		contentAddress,
		big.NewInt(0),
		tsMillis)
	if err != nil {
		log.Fatal(err)
	}
	sig, err := crypto.Sign(sigHash, pk)
	if err != nil {
		log.Fatal(err)
	}

	usrCtxJson := ""
	if len(usrCtx) > 0 && usrCtx[0] != nil {
		var b []byte
		b, err = json.Marshal(usrCtx[0])
		if err != nil {
			log.Fatal("failed to marshal usr ctx to JSON")
		}
		usrCtxJson = string(b)
	}

	res, err := CallRpcUrl(
		ethUrl,
		"elv_channelContentRequestContext",
		[]interface{}{
			userAddr,
			contentAddress,
			big.NewInt(0),
			tsMillis,
			hexutil.Encode(sig),
			usrCtxJson,
		})

	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(" Code generated statechannel Token:", res.(string))
	return res.(string)
}

// To implement abi.encodePacked(address, address, uint256, uint256) solidity method
// so it matches ecrecover output
func hashBatchTransaction(fromAddress common.Address, toAddress common.Address, balance *big.Int, tsMillis int64) ([]byte, error) {

	packBatchTransaction := func(fromAddress common.Address, toAddress common.Address, balance *big.Int, tsMillis int64) ([]byte, error) {
		addressTy, _ := abi.NewType("address", "", []abi.ArgumentMarshaling{})
		uint256Ty, _ := abi.NewType("uint256", "", []abi.ArgumentMarshaling{})

		arguments := abi.Arguments{
			{Type: addressTy},
			{Type: addressTy},
			{Type: uint256Ty},
			{Type: uint256Ty},
		}

		packNonce := big.NewInt(tsMillis)

		packedBytes, _ := arguments.Pack(
			fromAddress,
			toAddress,
			balance,
			packNonce,
		)

		if len(packedBytes) != 4*32 {
			return nil, fmt.Errorf("behavior of Pack seems to have changed; expected 128 bytes, got %v", len(packedBytes))
		}

		packedPackedBytes := append(packedBytes[12:32], packedBytes[44:64]...)
		packedPackedBytes = append(packedPackedBytes, packedBytes[64:]...)
		return packedPackedBytes, nil
	}

	packedTrans, err := packBatchTransaction(fromAddress, toAddress, balance, tsMillis)
	if err != nil {
		return nil, err
	}
	return crypto.Keccak256(packedTrans), nil
}

func IDToAddress(id id.ID) common.Address {
	return common.BytesToAddress(id.Bytes())
}

func CallRpcUrl(rawUrl string, method string, params []interface{}) (interface{}, error) {
	u, err := url.Parse(rawUrl)
	if err != nil {
		return nil, err
	}

	switch u.Scheme {
	case "":
		rpcClient, err := rpc.Dial(rawUrl)
		if err != nil {
			return nil, err
		}

		var res interface{}
		err = rpcClient.Call(&res, method, params...)
		if err != nil {
			return nil, err
		}
		return res, nil

	case "http", "https":

		parsedParams, err := json.Marshal(params)
		if err != nil {
			return nil, err
		}

		fullBody := fmt.Sprintf(
			`{ "id": 1, "jsonrpc": "2.0", "method": "%s", "params": %s }`,
			method, string(parsedParams),
		)

		res, err := http.Post(rawUrl, "application/json", strings.NewReader(fullBody))
		if err != nil {
			return nil, err
		}

		data, err := io.ReadAll(res.Body)
		if err != nil {
			return nil, err
		}

		var out map[string]interface{}
		err = json.Unmarshal(data, &out)
		if err != nil {
			return nil, err
		}

		if out["error"] != nil {
			return "", errors.New(fmt.Sprintf("RPC method failure: %v", out["error"]))
		}

		var result interface{}
		if out["result"] != nil {
			result = out["result"].(interface{})
		}

		return result, nil
	}
	return nil, fmt.Errorf("Unsupported scheme: %s\n", u.Scheme)
}

func generateEditorSigned(cfg *types.Config, Qlib, QID string) string {
	QLibID, err := id.FromString(Qlib)
	if err != nil {
		log.Fatalf("Invalid QLib format: %v", err)
	}

	parsedQID, err := id.FromString(QID)
	if err != nil {
		log.Fatalf("Invalid QID format: %v", err)
	}
	parsedSpaceID, err := id.FromString(cfg.QSpaceID)
	if err != nil {
		log.Fatalf("Invalid QSpaceID format: %v", err)
	}
	finalLibID := types2.QLibID(QLibID)
	finalQID := types2.QID(parsedQID)
	finalSpaceID := types2.QSpaceID(parsedSpaceID)

	rawKey := cfg.PkStr

	if strings.HasPrefix(rawKey, "0x") {
		rawKey = rawKey[2:]
	}

	// Parse Hex to Struct
	privateKey, err := crypto.HexToECDSA(rawKey)
	if err != nil {
		log.Fatalf("Failed to parse private key: %v", err)
	}

	log.Printf("generate QlibID: %s, QID: %s", finalLibID.String(), finalQID.String())

	now := utc.Now()

	// Note: We use finalLibID and finalQID here
	builder := eat.NewEditorSigned(finalSpaceID, finalLibID, finalQID).
		WithIssuedAt(now).
		WithExpires(now.Add(24 * time.Hour))

	// Note: We pass the parsed 'privateKey' struct here
	bearer, err := builder.Sign(privateKey).Encode()
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Generated code editor signed token:", bearer)
	return bearer
}
