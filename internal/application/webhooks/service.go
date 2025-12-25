package webhooks

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	"webhookd/internal/application/ports"
	"webhookd/internal/domain/webhook"
)

type Service struct {
	repo ports.WebhookRepository
	now  func() time.Time
}

func NewService(repo ports.WebhookRepository) *Service {
	return &Service{
		repo: repo,
		now:  func() time.Time { return time.Now().UTC() },
	}
}

type CreateParams struct {
	Method  string
	Body    string
	Headers map[string]string
}

func (s *Service) Create(ctx context.Context, p CreateParams) (*webhook.Hook, error) {
	if s.repo == nil {
		return nil, errors.New("repo is nil")
	}
	id := webhook.ID(uuid.NewString())
	h, err := webhook.New(id, p.Method, p.Body, p.Headers, s.now())
	if err != nil {
		return nil, err
	}
	if err := s.repo.Create(ctx, h); err != nil {
		return nil, err
	}
	return h, nil
}

func (s *Service) Deactivate(ctx context.Context, id webhook.ID) (*webhook.Hook, bool, error) {
	return s.repo.Deactivate(ctx, id)
}

func (s *Service) Get(ctx context.Context, id webhook.ID) (*webhook.Hook, bool, error) {
	return s.repo.Get(ctx, id)
}

func (s *Service) Touch(ctx context.Context, id webhook.ID) (*webhook.Hook, bool, error) {
	return s.repo.Touch(ctx, id, s.now())
}

func (s *Service) List(ctx context.Context) (map[webhook.ID]*webhook.Hook, error) {
	return s.repo.List(ctx)
}
