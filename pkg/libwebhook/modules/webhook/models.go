package webhook

import (
	"errors"
	"fmt"
	"strings"
	"webhookd/pkg/libwebhook"
)

type PostGenerateWebhookRequest struct {
	Method  string            `json:"method"`
	Body    string            `json:"body"`
	Headers map[string]string `json:"headers"`
}

type PostGenerateWebhookResponse struct {
	Path  string  `json:"path"`
	Error *string `json:"error,omitempty"`
}

type PostUnregisterWebhookRequest struct {
	UUID   string `json:"uuid"`
	Method string `json:"method"`
}

func (r PostGenerateWebhookRequest) Validate() error {

	method := strings.ToLower(r.Method)

	if _, ok := libwebhook.ContainsString(method, []string{
		"get",
		"post",
		"delete",
		"put",
		"option",
		"patch",
	}); !ok {
		return errors.New(fmt.Sprintf("method: %s is not supported right now", method))
	}

	return nil
}

func (r PostUnregisterWebhookRequest) Validate() error {

	method := strings.ToLower(r.Method)

	if _, ok := libwebhook.ContainsString(method, []string{
		"get",
		"post",
		"delete",
		"put",
		"option",
		"patch",
	}); !ok {
		return errors.New(fmt.Sprintf("method: %s is not supported right now", method))
	}

	return nil
}
