package utils

import (
	"fmt"
	"math/big"
	"path/filepath"
	"os"
	"strings"
	l4g "github.com/alecthomas/log4go"
	"time"
	"math/rand"
	"crypto/md5"
	"encoding/hex"
	"math"
)

func Faltal_error(err error) {
	if err==nil {
		return
	}
	l4g.Error(err)
	os.Exit(1)
}

func IsBytesEmpty(d []byte) bool {
	for _, b := range d {
		if b!=0 { return false }
	}
	return true
}

func string_has_prefix(s, prefix string) bool {
	return len(s) >= len(prefix) && s[0:len(prefix)] == prefix
}

func String_cat_prefix(s string, prefix string) (string) {
	if string_has_prefix(s, prefix) {
		return s[len(prefix):]
	}
	return s
}

func Hex_string_to_big_int(s string) (*big.Int, error) {
	var (
		err        error
		bigint     *big.Int
		string_hex string
	)
	string_hex = String_cat_prefix(s, "0x")

	bigint, isok := new(big.Int).SetString(string_hex, 16)
	if !isok {
		err = fmt.Errorf("can not convert hex string:%s, to big int", string_hex)
	}

	return bigint, err
}


func CurrentRuningFileDir() string {
	path, err := filepath.Abs(filepath.Dir(os.Args[0]))
	//path, err := filepath.Abs("./")
	if err != nil {
		Faltal_error(err)
	}
	return strings.Replace(path, "\\", "/", -1)
}

func PathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}

	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func RandString(l int) string{
	str := "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

	bytes := []byte(str)

	var result []byte
	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	for i := 0; i < int(l); i++ {
		result = append(result, bytes[r.Intn(int(len(bytes)))])
	}
	return string(result)
}

func MD5(text string) string{
	ctx := md5.New()
	ctx.Write([]byte(text))
	return hex.EncodeToString(ctx.Sum(nil))
}

func ToSpecialDecimal_asBigint(v uint64, from, to int) *big.Int {
	i := to - from
	ibig :=  big.NewInt(int64(v))
	if i>0 { return ibig.Div(ibig, big.NewInt(int64(math.Pow10( i))))
	} else { return ibig.Div(ibig, big.NewInt(int64(math.Pow10(-i)))) }
}

// decimal convert int to f
func DecimalCvt_i_f(v uint64, from, to int) *big.Float {
	fbig :=  big.NewFloat(float64(v))
	return fbig.Mul(fbig, big.NewFloat(math.Pow10(to - from)))
}

// this convert used for from < to
func DecimalCvt_f_i(v float64, from, to int) int64 {
	fbig :=  big.NewFloat(v)
	i, _ := fbig.Mul(fbig, big.NewFloat(math.Pow10(to - from))).Float64()
	return int64(i)
}

// decimal convert int to f
func DecimalCvt_f_f(v float64, from, to int) *big.Float {
	fbig :=  big.NewFloat(v)
	return fbig.Mul(fbig, big.NewFloat(math.Pow10(to - from)))
}

