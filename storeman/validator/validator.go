package validator

import (
	"encoding/json"
	"github.com/wanchain/go-wanchain/common"
	"github.com/wanchain/go-wanchain/common/hexutil"
	"github.com/wanchain/go-wanchain/core/types"
	"github.com/wanchain/go-wanchain/crypto"
	"github.com/wanchain/go-wanchain/log"
	"github.com/wanchain/go-wanchain/rlp"
	mpccrypto "github.com/wanchain/go-wanchain/storeman/storemanmpc/crypto"
	mpcprotocol "github.com/wanchain/go-wanchain/storeman/storemanmpc/protocol"
	mpcsyslog "github.com/wanchain/go-wanchain/storeman/syslog"
	"time"
	"github.com/wanchain/go-wanchain/storeman/btc"
	"math/big"
)

type SendTxArgs struct {
	From      common.Address  `json:"from"`
	To        *common.Address `json:"to"`
	Gas       *hexutil.Big    `json:"gas"`
	GasPrice  *hexutil.Big    `json:"gasPrice"`
	Value     *hexutil.Big    `json:"value"`
	Data      hexutil.Bytes   `json:"data"`
	Nonce     *hexutil.Uint64 `json:"nonce"`
	ChainType string          `json:"chaintype"`
	ChainID   *hexutil.Big    `json:"chainID"`
	SignType  string          `json:"signType"` //input 'hash' for hash sign (r,s,v), else for full sign(rawTransaction)
}

func ValidateTx(signer mpccrypto.MPCTxSigner, leaderTxRawData []byte, leaderTxLeaderHashBytes []byte) bool {
	var leaderTx types.Transaction
	err := rlp.DecodeBytes(leaderTxRawData, &leaderTx)
	if err != nil {
		mpcsyslog.Err("ValidateTx leader tx data decode fail. err:%s", err.Error())
		log.Error("ValidateTx leader tx data decode fail", "error", err)
		return false
	}

	keysBytes := make([]byte, 0)
	keysBytes = append(keysBytes, leaderTx.Value().Bytes()...)
	keysBytes = append(keysBytes, leaderTx.Data()...)

	key := crypto.Keccak256(keysBytes)
	log.Info("ValidateTx", "key", common.ToHex(key))
	mpcsyslog.Info("mpc ValidateTx. key:%s", common.ToHex(key))

	followerDB, err := GetDB()
	if err != nil {
		mpcsyslog.Err("ValidateTx leader get database fail. err:%s", err.Error())
		log.Error("ValidateTx leader get database fail", "error", err)
		return false
	}

	txDatach := make(chan []byte)
	defer close(txDatach)

	go func() {
		timeCh := time.After(mpcprotocol.MPCTimeOut)

		for {
			select {
			case <-timeCh:
				mpcsyslog.Info("ValidateTx time out")
				log.Info("ValidateTx time out")
				txDatach <- nil
				return

			default:
				isExist, err := followerDB.Has(key)
				if err == nil {
					if isExist {
						followerTxRawData, err := followerDB.Get(key)
						if err == nil {
							txDatach <- followerTxRawData
							mpcsyslog.Info("ValidateTx, followerTxRawData is got")
							log.Info("ValidateTx, followerTxRawData is got")
						} else {
							txDatach <- nil
							mpcsyslog.Err("ValidateTx, getting followerTxRawData fail. err:%s", err.Error())
							log.Error("ValidateTx, getting followerTxRawData fail", "error", err)
						}

						return
					} else {
						time.Sleep(200 * time.Microsecond)
					}

				} else {
					txDatach <- nil
					mpcsyslog.Err("ValidateTx, followerTxRawData key check has fail. err:%s", err.Error())
					log.Error("ValidateTx, followerTxRawData key check has fail", "error", err)
					return
				}
			}
		}
	}()

	select {
	case followerTxRawData := <-txDatach:
		if followerTxRawData == nil {
			mpcsyslog.Err("ValidateTx, tx data from db is nil. ValidateTx key:%s, tx Hash:%s",
				common.ToHex(key), common.ToHex(leaderTxLeaderHashBytes))
			log.Error("ValidateTx, tx data from db is nil", "ValidateTx key",
				common.ToHex(key), "tx Hash", common.ToHex(leaderTxLeaderHashBytes))
			return false
		}

		var followerRawTx SendTxArgs
		err = json.Unmarshal(followerTxRawData, &followerRawTx)
		if err != nil {
			mpcsyslog.Err("ValidateTx, follower tx data decode fail. err:%s", err.Error())
			log.Error("ValidateTx, follower tx data decode fail", "error", err)
			return false
		}

		followerCreatedTx := types.NewTransaction(leaderTx.Nonce(), *followerRawTx.To, followerRawTx.Value.ToInt(),
			leaderTx.Gas(), leaderTx.GasPrice(), followerRawTx.Data)
		followerCreatedHash := signer.Hash(followerCreatedTx)
		leaderTxLeaderHash := common.BytesToHash(leaderTxLeaderHashBytes)

		if followerCreatedHash == leaderTxLeaderHash {
			mpcsyslog.Info("ValidateTx, validate success")
			log.Info("ValidateTx, validate success")
			return true
		} else {
			mpcsyslog.Err("ValidateTx, leader tx hash is not same with follower tx hash. leaderTxLeaderHash:%s, followerCreatedHash:%s",
				leaderTxLeaderHash.String(), followerCreatedHash.String())
			log.Error("ValidateTx, leader tx hash is not same with follower tx hash", "leaderTxLeaderHash",
				leaderTxLeaderHash, "followerCreatedHash", followerCreatedHash)
			return false
		}
	}
}


func ValidateBtcTx(args *btc.MsgTxArgs) bool {
	if args == nil {
		return false
	}

	keyWithoutTxin, keyWithTxin := GetKeyFromBtcTx(args)
	log.Info("-----------------GetKeyFromBtcTx", "keyWithoutTxin", common.ToHex(keyWithoutTxin))
	log.Info("-----------------GetKeyFromBtcTx", "keyWithTxin", common.ToHex(keyWithTxin))

	followerDB, err := GetDB()
	if err != nil {
		mpcsyslog.Err("ValidateBtcTx get database fail. err:%s", err.Error())
		log.Error("ValidateBtcTx get database fail", "error", err)
		return false
	}

	start := time.Now()
	for {
		isExist, err := followerDB.Has(keyWithTxin)
		if err != nil {
			mpcsyslog.Err("ValidateBtcTx, keyWithTxin check has fail. err:%s", err.Error())
			log.Error("ValidateBtcTx, keyWithTxin check has fail", "error", err)
			return false
		} else if isExist {
			mpcsyslog.Info("ValidateBtcTx, keyWithTxin is got")
			log.Info("ValidateBtcTx, keyWithTxin is got")
			return true
		}

		isExist, err = followerDB.Has(keyWithoutTxin)
		if err != nil {
			mpcsyslog.Err("ValidateBtcTx, keyWithoutTxin check has fail. err:%s", err.Error())
			log.Error("ValidateBtcTx, keyWithoutTxin check has fail", "error", err)
			return false
		} else if isExist {
			mpcsyslog.Info("ValidateBtcTx, keyWithoutTxin is got")
			log.Info("ValidateBtcTx, keyWithoutTxin is got")
			return true
		}

		if time.Now().Sub(start) >= mpcprotocol.MPCTimeOut {
			mpcsyslog.Info("ValidateBtcTx time out")
			log.Info("ValidateBtcTx time out")
			return false
		}

		time.Sleep(200 * time.Microsecond)
	}
}


func GetKeyFromBtcTx(args *btc.MsgTxArgs) (keyWithoutTxIn []byte, keyWithTxIn []byte) {
	keyWithoutTxIn = append(keyWithoutTxIn, big.NewInt(int64(args.Version)).Bytes()...)
	keyWithoutTxIn = append(keyWithoutTxIn, big.NewInt(int64(args.LockTime)).Bytes()...)

	for _, out := range args.TxOut {
		keyWithoutTxIn = append(keyWithoutTxIn, big.NewInt(out.Value).Bytes()...)
		keyWithoutTxIn = append(keyWithoutTxIn, []byte(out.PkScript)...)
	}

	keyWithTxIn = make([]byte, len(keyWithoutTxIn))
	copy(keyWithTxIn, keyWithoutTxIn)
	log.Info("-----------------GetKeyFromBtcTx", "keyWithTxin", common.ToHex(keyWithTxIn))
	log.Info("-----------------GetKeyFromBtcTx", "keyWithoutTxIn", common.ToHex(keyWithoutTxIn))
	for _, in := range args.TxIn {
		log.Warn("-----------------GetKeyFromBtcTx, add txIn info to key")
		keyWithTxIn = append(keyWithTxIn, in.PreviousOutPoint.Hash[:]...)
		keyWithTxIn = append(keyWithTxIn, big.NewInt(int64(in.PreviousOutPoint.Index)).Bytes()...)
		keyWithTxIn = append(keyWithTxIn, []byte(in.SignatureScript)...)
		keyWithTxIn = append(keyWithTxIn, big.NewInt(int64(in.Sequence)).Bytes()...)
	}

	keyWithoutTxIn = crypto.Keccak256(keyWithoutTxIn)
	keyWithTxIn = crypto.Keccak256(keyWithTxIn)

	return keyWithoutTxIn, keyWithTxIn
}
