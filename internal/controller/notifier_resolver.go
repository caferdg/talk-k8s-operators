package controller

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	platformv1alpha1 "github.com/caferdg/talk-k8s-operators/api/v1alpha1"
	"github.com/caferdg/talk-k8s-operators/pkg/adapters/secondary/discord"
	"github.com/caferdg/talk-k8s-operators/pkg/ports/secondary"
)

// resolveNotifierAdapters resolves all configured provider adapters from a Notifier CR.
func resolveNotifierAdapters(ctx context.Context, c client.Client, notifier *platformv1alpha1.Notifier) ([]secondary.Notifier, error) {
	log := logf.FromContext(ctx)
	var notifiers []secondary.Notifier

	if notifier.Spec.Discord != nil {
		var secret corev1.Secret
		key := types.NamespacedName{
			Name:      notifier.Spec.Discord.WebhookSecretRef,
			Namespace: notifier.Namespace,
		}
		if err := c.Get(ctx, key, &secret); err != nil {
			return nil, fmt.Errorf("discord webhook secret %s not found: %w", key.Name, err)
		}
		webhookURL := string(secret.Data["webhookUrl"])
		if webhookURL == "" {
			return nil, fmt.Errorf("secret %s does not contain 'webhookUrl' key", key.Name)
		}
		notifiers = append(notifiers, discord.NewClient(webhookURL))
	}

	if notifier.Spec.Slack != nil {
		log.Info("slack provider configured but not implemented")
	}
	if notifier.Spec.Telegram != nil {
		log.Info("telegram provider configured but not implemented")
	}

	return notifiers, nil
}
