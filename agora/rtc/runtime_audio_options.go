package agoraservice

import (
	"fmt"
	"sync"
)

// runtimeOption prevents one connection from silently changing an
// engine-wide option already selected by another connection in this runtime.
type runtimeOption[T comparable] struct {
	mu         sync.Mutex
	configured bool
	value      T
}

func (option *runtimeOption[T]) apply(value T, setter func(T) int) int {
	option.mu.Lock()
	defer option.mu.Unlock()
	if option.configured {
		if option.value != value {
			return -2002
		}
		return 0
	}
	if code := setter(value); code != 0 {
		return code
	}
	option.configured, option.value = true, value
	return 0
}

func (option *runtimeOption[T]) reset() {
	option.mu.Lock()
	defer option.mu.Unlock()
	var zero T
	option.configured, option.value = false, zero
}

func setDeliverMuteData(value bool) int {
	return agoraService.deliverMuteData.apply(value, func(value bool) int {
		return GetAgoraParameter().SetParameters(fmt.Sprintf(`{"che.audio.deliver_mute_data_for_fake_adm":%t}`, value))
	})
}
