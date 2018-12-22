package bitcoin

import (
	"bitcoinclient/logger"
	"bitcoinclient/util"
	"encoding/json"
	"errors"
	"github.com/btcsuite/btcd/btcjson"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/rpcclient"
	"github.com/btcsuite/btcutil"
	"sync"
)

type BitcoinClient struct {
	cli       *rpcclient.Client
	testParam *TestParam
}

type Wallet struct {
	Address string `json:"address"`
	//PubKey  string `json:"pubKey"`
	PrivKey string `json:"privKey"`
}

type TestParam struct {
	Wallet1 Wallet `json:"wallet1"`
	Wallet2 Wallet `json:"wallet2"`
}

func (b *BitcoinClient) genWallets(num int) []*Wallet {
	ws := []*Wallet{}

	for i := 0; i < num; i++ {
		w := &Wallet{}
		w.PrivKey, _, w.Address = util.GetNewAddress()
		ws = append(ws, w)
	}

	return ws
}

func (b *BitcoinClient) GetBlockCount() (int64, error) {
	blockCount, err := b.cli.GetBlockCount()
	if err != nil {
		logger.Error(err)
		return 0, err
	}

	logger.Info("Highest block number is", blockCount)

	return blockCount, nil
}

func (b *BitcoinClient) GetBalance(account string) (float64, error) {
	asset, err := b.cli.GetBalance(account)
	if err != nil {
		logger.Error(err)
		return 0, err
	}

	logger.Infof("The balance of %s is %v\n", account, asset)

	return asset.ToBTC(), nil
}

func (b *BitcoinClient) QueryTransaction(txid string) error {
	txHash, err := chainhash.NewHashFromStr(txid)
	if err != nil {
		logger.Error(err)
		return err
	}

	tx, err := b.cli.GetTransaction(txHash)
	if err != nil {
		logger.Error(err)
		return err
	}

	data, _ := json.Marshal(tx)

	logger.Info(string(data))

	return nil
}

func (b *BitcoinClient) TransferTo(to string, num float64) (string, error) {
	toAddress, err := btcutil.DecodeAddress(to, &chaincfg.MainNetParams)
	if err != nil {
		logger.Error(err)
		return "", err
	}

	amount, err := btcutil.NewAmount(num)
	if err != nil {
		logger.Error(err)
		return "", err
	}

	txid, err := b.cli.SendToAddress(toAddress, amount)
	if err != nil {
		logger.Error(err)
		return "", err
	}

	return txid.String(), nil
}

func (b *BitcoinClient) GetInputs(num, fee float64) ([]btcjson.TransactionInput, float64, error) {
	var inputs []btcjson.TransactionInput

	uns, err := b.cli.ListUnspent()
	if err != nil {
		logger.Error(err)
		return nil, 0.0, err
	}

	data, _ := json.Marshal(uns)
	logger.Debug(string(data))

	var sum float64
	for i := 0; i < len(uns); i++ {
		if !uns[i].Spendable {
			continue
		}

		if uns[i].Vout != 0 {
			logger.Infof("%+v\n", uns[i])
		}

		input := btcjson.TransactionInput{
			Txid: uns[i].TxID,
			Vout: uns[i].Vout,
		}
		inputs = append(inputs, input)

		sum += uns[i].Amount
		if sum >= num+fee {
			break
		}
	}

	if sum < num+fee {
		return nil, 0.0, errors.New("not sufficient funds")
	}

	return inputs, sum, nil
}

func (b *BitcoinClient) Transfer(from, to string, num, fee float64) (string, error) {
	if num <= 0.0 || fee <= 0.0 {
		return "", errors.New("num or fee can not be less than 0")
	}

	fromAddress, err := btcutil.DecodeAddress(from, &chaincfg.MainNetParams)
	if err != nil {
		logger.Error(err)
		return "", err
	}

	toAddress, err := btcutil.DecodeAddress(to, &chaincfg.MainNetParams)
	if err != nil {
		logger.Error(err)
		return "", err
	}

	inputs, sum, err := b.GetInputs(num, fee)
	if err != nil {
		logger.Error(err)
		return "", err
	}

	amount, err := btcutil.NewAmount(num)
	if err != nil {
		logger.Error(err)
		return "", err
	}

	changeAmount, err := btcutil.NewAmount(sum - num - fee)
	if err != nil {
		logger.Error(err)
		return "", err
	}

	amounts := make(map[btcutil.Address]btcutil.Amount)
	amounts[toAddress] = amount
	amounts[fromAddress] = changeAmount

	lockTime := int64(0)

	tx, err := b.cli.CreateRawTransaction(inputs, amounts, &lockTime)
	if err != nil {
		logger.Error(err)
		return "", err
	}

	data, _ := json.Marshal(tx)
	logger.Debug(string(data))

	signedTx, complete, err := b.cli.SignRawTransaction(tx)
	if err != nil {
		logger.Error(err)
		return "", err
	}

	data, _ = json.Marshal(signedTx)
	logger.Debug(string(data))

	if !complete {
		logger.Error("sign not complete")
		return "", errors.New("sign not complete")
	}

	txHash, err := b.cli.SendRawTransaction(signedTx, true)
	if err != nil {
		logger.Error(err)
		return "", err
	}
	logger.Info(txHash.String())

	return txHash.String(), nil
}

func (b *BitcoinClient) testApi() error {
	var err error

	/*	_, err = b.GetBlockCount()
		if err != nil {
			logger.Error(err)
			return err
		}

		_, err = b.GetBalance("*")
		if err != nil {
			logger.Error(err)
			return err
		}

		txid, err := b.TransferTo("2MzxYfzCuVUwaKTaxB48cX4Noh7RqcWByGM", 14.0)
		if err != nil {
			logger.Error(err)
			return err
		}

		err = b.QueryTransaction(txid)
		if err != nil {
			logger.Error(err)
			return err
		}*/

	_, err = b.Transfer("2N1H1D5pWTJvWpeJoZYL1z4NpuW48LWvGhr", "2MzxYfzCuVUwaKTaxB48cX4Noh7RqcWByGM", 3199.0, 1.0)
	if err != nil {
		logger.Error(err)
		return err
	}

	return nil
}

func (b *BitcoinClient) testing(wg *sync.WaitGroup) {
	defer wg.Done()
	var err error

	err = b.testApi()
	if err != nil {
		logger.Error(err)
		return
	}
}

func NewBitcoinClient(ipport string, wg *sync.WaitGroup) (*BitcoinClient, error) {
	b := new(BitcoinClient)

	connCfg := &rpcclient.ConnConfig{
		Host:         ipport,
		User:         "admin1",
		Pass:         "123",
		HTTPPostMode: true, // Bitcoin core only supports HTTP POST mode
		DisableTLS:   true, // Bitcoin core does not provide TLS by default
	}

	var err error
	b.cli, err = rpcclient.New(connCfg, nil)
	if err != nil {
		logger.Error(err)
		return nil, err
	}
	//defer b.cli.Shutdown()

	go b.testing(wg)

	return b, nil
}
