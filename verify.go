//nolint:unparam
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/meshplus/bitxhub-kit/crypto/asym/ecdsa"
	"github.com/meshplus/bitxhub-kit/types"
)

type response struct {
	IsPass bool   `json:"is_pass"`
	Data   []byte `json:"data"`
}

type ValidatorServer struct {
	router *gin.Engine
	port   string

	ctx    context.Context
	cancel context.CancelFunc
}

func NewValidatorServer(port string) (*ValidatorServer, error) {
	ctx, cancel := context.WithCancel(context.Background())
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	return &ValidatorServer{
		router: router,
		port:   port,
		ctx:    ctx,
		cancel: cancel,
	}, nil
}

func (g *ValidatorServer) Start() error {
	g.router.Use(gin.Recovery())
	v1 := g.router.Group("/v1")
	{
		v1.POST("verify", g.verifyMultiSign)
	}

	go func() {
		go func() {
			err := g.router.Run(fmt.Sprintf(":%s", g.port))
			if err != nil {
				panic(err)
			}
		}()
		<-g.ctx.Done()
	}()
	return nil
}

func (g *ValidatorServer) verifyMultiSign(c *gin.Context) {
	res := &response{}
	signatures := c.Query("signatures")
	hash := c.Query("hash")
	threshold := c.Query("threshold")
	validators := c.Query("validators")

	var multiSignatures [][]byte
	if err := json.Unmarshal([]byte(signatures), &multiSignatures); err != nil {
		res.IsPass = false
		res.Data = []byte("multi signatures json unmarshal error")
		c.JSON(http.StatusOK, res)
		return
	}
	var vList []string
	if err := json.Unmarshal([]byte(validators), &vList); err != nil {
		res.IsPass = false
		res.Data = []byte("validators json unmarshal error")
		c.JSON(http.StatusOK, res)
		return
	}
	t, err := strconv.ParseUint(threshold, 10, 64)
	if err != nil {
		res.IsPass = false
		res.Data = []byte("threshold parse error")
		c.JSON(http.StatusOK, res)
		return
	}

	var bxhSigners []string
	for _, sig := range multiSignatures {
		if len(sig) != 65 {
			continue
		}

		v, r, s := getRawSignature(sig)

		addr, err := ecdsa.RecoverPlain([]byte(hash), r, s, v, true)
		if err != nil {
			res.IsPass = false
			res.Data = []byte("recover plain error")
			c.JSON(http.StatusOK, res)
			return
		}

		if addressArrayContains(vList, addr) {
			if addressArrayContains(bxhSigners, addr) {
				continue
			}
			bxhSigners = append(bxhSigners, types.NewAddress(addr).String())
			if uint64(len(bxhSigners)) == t {
				res.IsPass = true
				c.JSON(http.StatusOK, res)
				return
			}
		}
	}

	res.IsPass = false
	c.JSON(http.StatusOK, res)
}

func getRawSignature(sig []byte) (v, r, s *big.Int) {
	if len(sig) != 65 {
		return nil, nil, nil
	}

	r = &big.Int{}
	r.SetBytes(sig[:32])
	s = &big.Int{}
	s.SetBytes(sig[32:64])
	v = &big.Int{}
	v.SetBytes(sig[64:])

	return v, r, s
}

func addressArrayContains(addrs []string, address []byte) bool {
	for _, addr := range addrs {
		if addr == types.NewAddress(address).String() {
			return true
		}
	}

	return false
}

// func uint64ToBytesInBigEndian(i uint64) []byte {
// 	bytes := make([]byte, 8)

// 	binary.BigEndian.PutUint64(bytes, i)

// 	return bytes
// }
