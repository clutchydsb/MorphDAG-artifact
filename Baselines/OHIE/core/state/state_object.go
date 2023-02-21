package state

import (
	"bytes"
	"encoding/gob"
	"log"
	"sync"
	"sync/atomic"
)

const InitialBalance = 1000

type stateObject struct {
	addr         []byte
	data         Account
	dirtyBalance int64
	dirtyStorage *sync.Map
}

// newObject creates a state object
func newObject(addr []byte, data Account) *stateObject {
	if data.Storage == nil {
		data.Storage = make(map[string]int64)
	}
	return &stateObject{
		addr:         addr,
		data:         data,
		dirtyBalance: 0,
		dirtyStorage: new(sync.Map),
	}
}

// GetState returns the latest value of a given key
func (s *stateObject) GetState(key []byte) int64 {
	return s.getState(key)
}

// getState the private version of fetching state
func (s *stateObject) getState(key []byte) int64 {
	// if we have a dirty value, return it
	value, ok := s.dirtyStorage.Load(string(key))
	if ok {
		return value.(int64)
	}
	// otherwise return the committed value
	value2 := s.data.Storage[string(key)]
	return value2
}

// setState updates a value in dirty storage
func (s *stateObject) setState(key []byte, increment int64) {
	// avoid update loss
	for {
		oldValue := s.getState(key)
		newValue := oldValue + increment
		cur := s.getState(key)
		if cur == oldValue {
			s.dirtyStorage.Store(string(key), newValue)
			break
		}
	}
}

// SetState inserts updated cache into dirty storage
func (s *stateObject) SetState(updates map[string]int64) {
	for key := range updates {
		increment := updates[key]
		s.setState([]byte(key), increment)
	}
}

// GetBalance obtains the latest account balance
func (s *stateObject) GetBalance() int64 {
	return s.getBalance()
}

// getBalance the private version of fetching balance
func (s *stateObject) getBalance() int64 {
	if atomic.LoadInt64(&s.dirtyBalance) == 0 {
		return s.data.Balance
	}
	return atomic.LoadInt64(&s.dirtyBalance)
}

// SetBalance updates the account balance
func (s *stateObject) SetBalance(increment int64) {
	// avoid update loss
	for {
		oldBalance := s.getBalance()
		newBalance := oldBalance + increment
		cur := s.getBalance()
		if cur == oldBalance {
			atomic.StoreInt64(&s.dirtyBalance, newBalance)
			break
		}
	}
}

// Commit moves all updated values and balance to account storage
func (s *stateObject) Commit() {
	length := 0
	s.dirtyStorage.Range(func(key, value interface{}) bool {
		length++
		addr := key.(string)
		vvalue := value.(int64)
		s.data.Storage[addr] = vvalue
		return true
	})

	if length > 0 {
		s.dirtyStorage = new(sync.Map)
	}

	if s.dirtyBalance != 0 {
		s.data.Balance = s.dirtyBalance
		s.dirtyBalance = 0
	}
}

func (s *stateObject) GetNonce() uint32 {
	return s.data.Nonce
}

func (s *stateObject) SetNonce(nonce uint32) {
	s.data.Nonce = nonce
}

func (s *stateObject) Address() []byte {
	return s.addr
}

func (s *stateObject) Account() Account {
	return s.data
}

type Account struct {
	Nonce   uint32
	Balance int64
	Storage map[string]int64
}

// Serialize returns a serialized account
// this method may lead to different contents of byte slice, thus we use json marshal to replace it
func (acc Account) Serialize() []byte {
	var encode bytes.Buffer

	enc := gob.NewEncoder(&encode)
	err := enc.Encode(acc)
	if err != nil {
		log.Panic("Account encode fail:", err)
	}

	return encode.Bytes()
}

// DeserializeAcc deserializes an account
func DeserializeAcc(data []byte) Account {
	var acc Account

	decode := gob.NewDecoder(bytes.NewReader(data))
	err := decode.Decode(&acc)
	if err != nil {
		log.Panic("Account decode fail:", err)
	}

	return acc
}
