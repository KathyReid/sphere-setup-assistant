package main

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"
)

type OneTimeAuthHandler struct {
	username string
	password string
}

func (a *OneTimeAuthHandler) Init(username string) {
	a.username = username
}

func (a *OneTimeAuthHandler) AuthenticationInvalidated() {
	b := make([]byte, 4)
	_, err := rand.Read(b)
	if err != nil {
		fmt.Println("error:", err)
		panic(err)
	}
	// even though 2^32-1 doesn't divide evenly here, the probabilities
	// are small enough that all 10,000 numbers are equally likely.
	a.password = fmt.Sprintf("%04d", binary.LittleEndian.Uint32(b)%10000)
	logger.Infof("Generated new passcode: %s", a.password)
}

func (a *OneTimeAuthHandler) GetUsername() string {
	return a.username
}

func (a *OneTimeAuthHandler) GetPassword() string {
	return a.password
}
