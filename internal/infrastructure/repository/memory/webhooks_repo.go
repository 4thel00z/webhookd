package memory

import (
	"context"
	"sync"
	"time"

	"webhookd/internal/domain/webhook"
)

type WebhooksRepo struct {
	mu    sync.RWMutex
	hooks map[webhook.ID]*webhook.Hook
}

func NewWebhooksRepo() *WebhooksRepo {
	return &WebhooksRepo{hooks: map[webhook.ID]*webhook.Hook{}}
}

func (r *WebhooksRepo) Create(_ context.Context, h *webhook.Hook) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.hooks[h.ID] = cloneHook(h)
	return nil
}

func (r *WebhooksRepo) Get(_ context.Context, id webhook.ID) (*webhook.Hook, bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	h, ok := r.hooks[id]
	if !ok {
		return nil, false, nil
	}
	return cloneHook(h), true, nil
}

func (r *WebhooksRepo) Deactivate(_ context.Context, id webhook.ID) (*webhook.Hook, bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	h, ok := r.hooks[id]
	if !ok {
		return nil, false, nil
	}
	h = cloneHook(h)
	h.Deactivate()
	r.hooks[id] = h
	return cloneHook(h), true, nil
}

func (r *WebhooksRepo) Touch(_ context.Context, id webhook.ID, now time.Time) (*webhook.Hook, bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	h, ok := r.hooks[id]
	if !ok {
		return nil, false, nil
	}
	h = cloneHook(h)
	h.Touch(now)
	r.hooks[id] = h
	return cloneHook(h), true, nil
}

func (r *WebhooksRepo) List(_ context.Context) (map[webhook.ID]*webhook.Hook, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make(map[webhook.ID]*webhook.Hook, len(r.hooks))
	for k, v := range r.hooks {
		out[k] = cloneHook(v)
	}
	return out, nil
}

func cloneHook(h *webhook.Hook) *webhook.Hook {
	if h == nil {
		return nil
	}
	c := *h
	if h.Headers != nil {
		c.Headers = make(map[string]string, len(h.Headers))
		for k, v := range h.Headers {
			c.Headers[k] = v
		}
	}
	return &c
}


