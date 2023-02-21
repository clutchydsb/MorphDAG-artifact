package state

import (
	"Occam/core/types"
	"log"
	"strconv"
	"sync"
)

type StateDB struct {
	stateObjects      *sync.Map
	stateObjectsDirty *sync.Map
	trieDB            *TrieDB
}

// NewState creates a new state from a given state root
func NewState(blkFile string, rootHash []byte) (*StateDB, error) {
	trieDB, err := NewTrieDB(blkFile, rootHash)
	if err != nil {
		return nil, err
	}

	return &StateDB{
		stateObjects:      new(sync.Map),
		stateObjectsDirty: new(sync.Map),
		trieDB:            trieDB,
	}, nil
}

// Reset resets the statedb but keeps the underlying triedb (sweeps the memory)
func (s *StateDB) Reset() {
	s.stateObjects = new(sync.Map)
	s.stateObjectsDirty = new(sync.Map)
	s.trieDB.Clear()
}

// Exist reports whether the give account exists in the statedb
func (s *StateDB) Exist(addr []byte) bool {
	return s.getStateObject(addr) != nil
}

// GetBalance retrieves the balance of the given address
func (s *StateDB) GetBalance(addr []byte) int64 {
	obj := s.getStateObject(addr)
	if obj != nil {
		return obj.GetBalance()
	}
	return 0
}

// GetNonce retrieves the nonce of the given address
func (s *StateDB) GetNonce(addr []byte) uint32 {
	obj := s.getStateObject(addr)
	if obj != nil {
		return obj.GetNonce()
	}
	return 0
}

// GetState retrieves a value from the given account storage
func (s *StateDB) GetState(addr []byte, hash []byte) int64 {
	obj := s.getStateObject(addr)
	if obj != nil {
		return obj.GetState(hash)
	}
	return 0
}

// getStateObject retrieves state object
func (s *StateDB) getStateObject(addr []byte) *stateObject {
	// if state object is available in statedb
	if obj, ok := s.stateObjects.Load(string(addr)); ok {
		return obj.(*stateObject)
	}

	// load the object from the triedb
	acc, err := s.trieDB.FetchState(addr)
	if err != nil {
		log.Println("Failed to get state object")
		return nil
	}

	obj := newObject(addr, *acc)
	s.setStateObject(obj)
	return obj
}

// setStateObject sets state object
func (s *StateDB) setStateObject(object *stateObject) {
	s.stateObjects.Store(string(object.Address()), object)
}

// UpdateBalance updates the balance for the given account
func (s *StateDB) UpdateBalance(addr []byte, increment int64) {
	obj := s.GetOrNewStateObject(addr)
	if obj != nil {
		obj.SetBalance(increment)
		if _, ok := s.stateObjectsDirty.Load(string(addr)); !ok {
			// this account state has not been modified before
			s.stateObjectsDirty.Store(string(addr), struct{}{})
		}
	}
}

// UpdateState updates the value for the given account storage
func (s *StateDB) UpdateState(addr []byte, updates map[string]int64) {
	obj := s.GetOrNewStateObject(addr)
	if obj != nil {
		obj.SetState(updates)
		if _, ok := s.stateObjectsDirty.Load(string(addr)); !ok {
			// this account state has not been modified before
			s.stateObjectsDirty.Store(string(addr), struct{}{})
		}
	}
}

// UpdateStateForSerial updates the value for the given account storage (for serial execution)
func (s *StateDB) UpdateStateForSerial(addr []byte, key []byte, update int64) {
	obj := s.GetOrNewStateObject(addr)
	if obj != nil {
		obj.setState(key, update)
		if _, ok := s.stateObjectsDirty.Load(string(addr)); !ok {
			// this account state has not been modified before
			s.stateObjectsDirty.Store(string(addr), struct{}{})
		}
	}
}

// GetOrNewStateObject retrieves a state object or creates a new state object if nil
func (s *StateDB) GetOrNewStateObject(addr []byte) *stateObject {
	obj := s.getStateObject(addr)
	if obj == nil {
		obj = s.createObject(addr)
	}
	return obj
}

// createObject creates a new state object
func (s *StateDB) createObject(addr []byte) *stateObject {
	acc := Account{Balance: InitialBalance, Nonce: 0}
	newObj := newObject(addr, acc)
	s.setStateObject(newObj)
	return newObj
}

// createObject2 creates a new state object (does not store it in the statedb)
func (s *StateDB) createObject2(addr []byte) *stateObject {
	acc := Account{Balance: InitialBalance, Nonce: 0}
	newObj := newObject(addr, acc)
	return newObj
}

// Database retrieves the underlying triedb.
func (s *StateDB) Database() *TrieDB {
	return s.trieDB
}

// PreFetch pre-fetches the state of hot accounts into the statedb
func (s *StateDB) PreFetch(accounts map[string]struct{}) error {
	for addr := range accounts {
		if _, ok := s.stateObjects.Load(addr); !ok {
			acc, err := s.trieDB.FetchState([]byte(addr))
			if err != nil {
				return err
			}
			obj := newObject([]byte(addr), *acc)
			s.setStateObject(obj)
		}
	}
	return nil
}

// Commit writes the updated account state into the underlying triedb
func (s *StateDB) Commit() []byte {
	// retrieve the dirty state
	length := 0
	s.stateObjectsDirty.Range(func(key, value interface{}) bool {
		length++
		addr := key.(string)
		obj, _ := s.stateObjects.Load(addr)
		stateObj := obj.(*stateObject)
		// first commit to account storage
		stateObj.Commit()
		// write updates into the trie
		_ = s.trieDB.StoreState(stateObj)
		return true
	})

	if length > 0 {
		s.stateObjectsDirty = new(sync.Map)
	}

	// commit to the underlying leveldb
	root := s.trieDB.Commit()
	return root
}

// BatchCreateObjects creates a batch of state objects and stores them into the underlying triedb
func (s *StateDB) BatchCreateObjects(txs map[string][]*types.Transaction) {
	for blkNum := range txs {
		txSets := txs[blkNum]
		for _, tx := range txSets {
			payload := tx.Data()
			for addr := range payload {
				obj := s.createObject2([]byte(addr))
				updates := make(map[string]int64)
				for i := 0; i < 10; i++ {
					updates[strconv.Itoa(i)] = 10000
				}

				obj.SetState(updates)
				obj.Commit()

				if err := s.trieDB.StoreState(obj); err != nil {
					log.Println(err)
				}
			}
		}
	}
	s.trieDB.Commit()
}
