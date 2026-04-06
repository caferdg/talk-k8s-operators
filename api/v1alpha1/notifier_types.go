/*
Copyright 2026.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NotifierEvent represents a subscribable notification event.
type NotifierEvent string

const (
	EventPipelineRunning  NotifierEvent = "pipeline_running"
	EventPipelineSuccess  NotifierEvent = "pipeline_success"
	EventPipelineFailed   NotifierEvent = "pipeline_failed"
	EventPipelineCanceled NotifierEvent = "pipeline_canceled"
	EventNewCommit        NotifierEvent = "new_commit"
)

// NotifierSpec defines the desired state of Notifier.
type NotifierSpec struct {
	// Events to subscribe to.
	// Available: "pipeline_running", "pipeline_success", "pipeline_failed", "pipeline_canceled", "new_commit"
	Events []NotifierEvent `json:"events"`
	// Discord provider configuration
	Discord *DiscordProvider `json:"discord,omitempty"`
	// Slack provider configuration (not implemented)
	Slack *SlackProvider `json:"slack,omitempty"`
	// Telegram provider configuration (not implemented)
	Telegram *TelegramProvider `json:"telegram,omitempty"`
}

// DiscordProvider configures Discord webhook notifications.
type DiscordProvider struct {
	// Reference to a Secret containing the Discord webhook URL (key: "webhookUrl")
	WebhookSecretRef string `json:"webhookSecretRef"`
}

// SlackProvider configures Slack webhook notifications.
type SlackProvider struct {
	// Reference to a Secret containing the Slack webhook URL (key: "webhookUrl")
	WebhookSecretRef string `json:"webhookSecretRef"`
}

// TelegramProvider configures Telegram bot notifications.
type TelegramProvider struct {
	// Reference to a Secret containing bot token (key: "botToken") and chat ID (key: "chatId")
	SecretRef string `json:"secretRef"`
}

// NotifierStatus defines the observed state of Notifier.
type NotifierStatus struct {
	// Per-project tracking (key = GitlabProject CR name)
	ProjectStates map[string]ProjectNotificationState `json:"projectStates,omitempty"`
	// Standard conditions
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// ProjectNotificationState tracks what was last notified for a given project.
type ProjectNotificationState struct {
	// Last pipeline ID that was notified
	LastNotifiedPipelineID int `json:"lastNotifiedPipelineId,omitempty"`
	// Last pipeline status that was notified
	LastNotifiedPipelineStatus string `json:"lastNotifiedPipelineStatus,omitempty"`
	// Last commit SHA that was notified
	LastNotifiedCommitSHA string `json:"lastNotifiedCommitSha,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=notifier
// +kubebuilder:printcolumn:name="Events",type=string,JSONPath=`.spec.events`
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// Notifier is the Schema for the notifiers API.
type Notifier struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NotifierSpec   `json:"spec,omitempty"`
	Status NotifierStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// NotifierList contains a list of Notifier.
type NotifierList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Notifier `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Notifier{}, &NotifierList{})
}
