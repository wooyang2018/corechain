package event

import (
	"fmt"

	"github.com/wooyang2018/corechain/engine/base"
	"github.com/wooyang2018/corechain/protos"
)

// Router distribute events according to the event type and filter
type Router struct {
	topics map[protos.SubscribeType]Topic
}

// NewRounterFromChainMG instance Router from ChainManager
func NewRounterFromChainMG(chainmg ChainManager) *Router {
	blockTopic := NewBlockTopic(chainmg)
	r := &Router{
		topics: make(map[protos.SubscribeType]Topic),
	}
	r.topics[protos.SubscribeType_BLOCK] = blockTopic

	return r
}

// NewRounterFromChainMG instance Router from base.Engine
func NewRouter(engine base.Engine) *Router {
	return NewRounterFromChainMG(NewChainManager(engine))
}

// EncodeFunc encodes event payload
type EncodeFunc func(x interface{}) ([]byte, error)

// Subscribe route events from protos.SubscribeType and filter buffer
func (r *Router) Subscribe(tp protos.SubscribeType, filterbuf []byte) (EncodeFunc, Iterator, error) {
	topic, ok := r.topics[tp]
	if !ok {
		return nil, nil, fmt.Errorf("subscribe type %s unsupported", tp)
	}
	filter, err := topic.ParseFilter(filterbuf)
	if err != nil {
		return nil, nil, fmt.Errorf("parse filter error: %s", err)
	}
	iter, err := topic.NewIterator(filter)
	return topic.MarshalEvent, iter, err
}

// RawSubscribe route events from protos.SubscribeType and filter struct
func (r *Router) RawSubscribe(tp protos.SubscribeType, filter interface{}) (Iterator, error) {
	topic, ok := r.topics[tp]
	if !ok {
		return nil, fmt.Errorf("subscribe type %s unsupported", tp)
	}
	iter, err := topic.NewIterator(filter)
	return iter, err
}
