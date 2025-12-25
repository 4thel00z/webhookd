package ports

import (
	"context"
	"time"

	"webhookd/internal/domain/webhook"
)

type WebhookRepository interface {
	Create(ctx context.Context, h *webhook.Hook) error
	Get(ctx context.Context, id webhook.ID) (*webhook.Hook, bool, error)
	Deactivate(ctx context.Context, id webhook.ID) (*webhook.Hook, bool, error)
	Touch(ctx context.Context, id webhook.ID, now time.Time) (*webhook.Hook, bool, error)
	List(ctx context.Context) (map[webhook.ID]*webhook.Hook, error)
}
