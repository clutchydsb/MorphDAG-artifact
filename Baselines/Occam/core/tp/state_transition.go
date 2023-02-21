package tp

import (
	"Occam/core/state"
	"Occam/core/types"
	"errors"
	"strings"
	"time"
)

type StateTransition struct {
	msg   types.Message
	value int64
	data  types.Payload
	state *state.StateDB
}

// NewStateTransition creates a new instance for state transition object
func NewStateTransition(msg types.Message, db *state.StateDB) *StateTransition {
	return &StateTransition{
		msg:   msg,
		value: msg.Value(),
		data:  msg.Data(),
		state: db,
	}
}

// ApplyMessage executes a given message
func ApplyMessage(db *state.StateDB, msg types.Message, cache *Cache) error {
	return NewStateTransition(msg, db).TransitionDb(cache)
}

// ApplyMessageForSerial executes a given message (for serial execution)
func ApplyMessageForSerial(db *state.StateDB, msg types.Message) error {
	return NewStateTransition(msg, db).SerialTransitionDb()
}

// TransitionDb conducts state transition
func (st *StateTransition) TransitionDb(cache *Cache) error {
	length := st.msg.Len()

	if length == 0 {
		err := st.commonTransfer()
		if err != nil {
			return err
		}
	} else {
		noWrites, err := st.stateTransfer(cache)
		if err != nil {
			// revert and clear cache
			cache.Clear()
			return err
		}
		if !noWrites {
			wSets := cache.GetState()
			st.commit(wSets)
			cache.Clear()
		}
	}

	return nil
}

// SerialTransitionDb conducts serial state transition
func (st *StateTransition) SerialTransitionDb() error {
	length := st.msg.Len()

	if length == 0 {
		err := st.commonTransfer()
		if err != nil {
			return err
		}
	} else {
		err := st.serialStateTransfer()
		if err != nil {
			return err
		}
	}

	return nil
}

// commonTransfer executes the common transfer transaction
func (st *StateTransition) commonTransfer() error {
	addrFrom := st.msg.From()
	addrTo := st.msg.To()

	bal := st.state.GetBalance(addrFrom)
	// the account does not have sufficient balance
	if bal < st.value {
		err := errors.New("tx execution failed: insufficient balance")
		return err
	}

	st.state.UpdateBalance(addrFrom, -st.value)
	st.state.UpdateBalance(addrTo, st.value)
	//log.Println("Tx execution succeed")
	return nil
}

// stateTransfer executes the state transfer transaction
func (st *StateTransition) stateTransfer(cache *Cache) (bool, error) {
	noWrites := true
	payload := st.data
	// simulate smart contract execution time
	mimicExecution(len(payload))
	for addr := range payload {
		rwSets := payload[addr]
		for _, rw := range rwSets {
			if strings.Compare(rw.Label, "r") == 0 {
				value := st.state.GetState([]byte(addr), rw.Addr)
				if value == 0 {
					err := errors.New("tx execution failed: fail to retrieve the account state")
					return noWrites, err
				}
			} else {
				// first writes to the cache
				noWrites = false
				cache.Expand()
				cache.SetState(addr, rw)
			}
		}
	}
	//log.Println("Tx execution succeeds")
	return noWrites, nil
}

// commit writes the content of cache into the statedb
func (st *StateTransition) commit(wSets map[string][]*types.RWSet) {
	for addr := range wSets {
		var updates = make(map[string]int64)
		writes := wSets[addr]

		for _, w := range writes {
			updates[string(w.Addr)] = w.Value
		}
		st.state.UpdateState([]byte(addr), updates)
	}
}

// serialStateTransfer serially executes the state transfer transaction
func (st *StateTransition) serialStateTransfer() error {
	payload := st.data
	// simulate smart contract execution time
	mimicExecution(len(payload))
	for addr := range payload {
		rwSets := payload[addr]
		for _, rw := range rwSets {
			if strings.Compare(rw.Label, "r") == 0 {
				value := st.state.GetState([]byte(addr), rw.Addr)
				if value == 0 {
					err := errors.New("tx execution failed: fail to retrieve the account state")
					return err
				}
			} else {
				// directly writes to the statedb
				st.state.UpdateStateForSerial([]byte(addr), rw.Addr, rw.Value)
			}
		}
	}
	//log.Println("Tx execution succeeds")
	return nil
}

func mimicExecution(ops int) {
	var a, b, c int
	a, b, c = 1, 1, 1

	for i := 0; i < ops*400; i++ {
		a += b
		b += a
		c += a
		time.Sleep(time.Nanosecond)
	}
}
