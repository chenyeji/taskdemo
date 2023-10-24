package common

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"hash"
	"math/big"
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcutil/base58"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/ripemd160"
)

const (
	HexPrefix = "0x"
)

func Recover() {
	if r := recover(); r != nil {
		err := fmt.Errorf("%v\nstacktrace from panic: %s", r, string(debug.Stack()))
		logrus.Error(err)
		// 以防 log 失效
		fmt.Printf("\n------------- avoid log failure -------------\n%s\n---------------------------------------------\n",
			err)
	}
}

// TimeConsume provides convenience function for time-consuming calculation
func TimeConsume(start time.Time) {
	pc, _, _, ok := runtime.Caller(1)
	if !ok {
		return
	}

	// get Fun object from pc
	funcName := runtime.FuncForPC(pc).Name()
	logrus.WithField("tags", "func_time_consume").
		WithField("cost", time.Since(start).String()).Debug(funcName)
}

func DecodeHex(input string) (dec uint64, err error) {
	input = RemoveHexPrefix(input)
	dec, err = strconv.ParseUint(input, 16, 64)
	err = errors.WithStack(err)
	return
}

func RemoveHexPrefix(content string) string {
	return strings.TrimPrefix(content, HexPrefix)
}

func GetHexNumber(content string) (value *big.Int, err error) {
	content = strings.ToLower(content)
	content = RemoveHexPrefix(content)
	value, ok := new(big.Int).SetString(content, 16)
	if !ok {
		err = errors.Errorf("can not parse hex(%s) as big.Int", content)
	}
	return
}

type NetID struct {
	P2pkhID []byte
	P2shID  []byte
}

func GetAddressFromScriptSig(asm string, isTest bool) (address string, err error) {
	netID := NetID{P2pkhID: []byte{0}, P2shID: []byte{5}}

	asms := strings.Split(asm, " ")

	if len(asms) == 0 || len(asms[0]) == 0 {
		err = errors.New("asm can't be null")
		return
	}

	var inputID []byte
	var pubKeyByte []byte

	pubKeyByte, err = hex.DecodeString(asms[len(asms)-1])
	if err != nil {
		return
	}

	if (len(pubKeyByte) == 65 && pubKeyByte[0] == 4) || // 未压缩公钥
		(len(pubKeyByte) == 33 && (pubKeyByte[0] == 2 || pubKeyByte[0] == 3)) { // 压缩公钥  //p2pkh
		inputID = netID.P2pkhID
	} else if pubKeyByte[len(pubKeyByte)-1] == txscript.OP_CHECKSIG { // p2sh
		inputID = netID.P2shID
	}

	hash160 := Hash160(pubKeyByte)

	// Check for a valid script hash length.
	if len(hash160) != 20 {
		err = errors.New("scriptHash must be 20 bytes")
		return
	}

	address = CheckEncode(hash160, inputID)
	//logrus.Infof("addr:%s org input len:%d byte len:%d", address, len(asms[len(asms)-1]), len(pubKeyByte))
	return
}

func CheckEncode(input []byte, version []byte) string {
	b := make([]byte, 0, 1+len(input)+4)
	b = append(b, version...)
	b = append(b, input...)
	cksum := checksum(b)
	b = append(b, cksum[:]...)
	return base58.Encode(b)
}

func checksum(input []byte) (cksum [4]byte) {
	h := sha256.Sum256(input)
	h2 := sha256.Sum256(h[:])
	copy(cksum[:], h2[:4])
	return
}

// Calculate the hash of hasher over buf.
func calcHash(buf []byte, hasher hash.Hash) []byte {
	hasher.Write(buf)
	return hasher.Sum(nil)
}

// Hash160 calculates the hash ripemd160(sha256(b)).
func Hash160(buf []byte) []byte {
	return calcHash(calcHash(buf, sha256.New()), ripemd160.New())
}
