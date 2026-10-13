package agoraservice

import "sync"

// callbackGate drains native callbacks before their Go state is changed.
// Lifecycle and observer registration calls must not run inside callbacks.
type callbackGate struct {
	mu     sync.Mutex
	cond   *sync.Cond
	active int
	paused int
}

func (g *callbackGate) begin() bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.paused != 0 {
		return false
	}
	g.active++
	return true
}

func (g *callbackGate) end() {
	g.mu.Lock()
	g.active--
	if g.active == 0 && g.cond != nil {
		g.cond.Broadcast()
	}
	g.mu.Unlock()
}

func (g *callbackGate) pause() {
	g.mu.Lock()
	g.paused++
	if g.cond == nil {
		g.cond = sync.NewCond(&g.mu)
	}
	for g.active != 0 {
		g.cond.Wait()
	}
	g.mu.Unlock()
}

func (g *callbackGate) resume() {
	g.mu.Lock()
	g.paused--
	g.mu.Unlock()
}
