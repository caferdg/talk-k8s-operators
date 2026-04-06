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

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// GitlabGroupSpec defines the desired state of GitlabGroup.
type GitlabGroupSpec struct {
	// GitLab group ID
	GroupID int `json:"groupId"`
	// Reference to a Secret containing the GitLab API token (key: "token")
	TokenSecretRef string `json:"tokenSecretRef"`
}

// GitlabGroupStatus defines the observed state of GitlabGroup.
type GitlabGroupStatus struct {
	// Full path of the group on GitLab
	FullPath string `json:"fullPath,omitempty"`
	// Web URL of the group
	WebURL string `json:"webUrl,omitempty"`
	// Description of the group
	Description string `json:"description,omitempty"`
	// Visibility level of the group (private, internal, public)
	Visibility string `json:"visibility,omitempty"`
	// Names of projects in the group
	Projects []string `json:"projects,omitempty"`
	// Names of subgroups in the group
	Subgroups []string `json:"subgroups,omitempty"`
	// Number of members in the group
	MemberCount int `json:"memberCount,omitempty"`
	// Standard conditions
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=ggroup
// +kubebuilder:printcolumn:name="Full Path",type=string,JSONPath=`.status.fullPath`
// +kubebuilder:printcolumn:name="Visibility",type=string,JSONPath=`.status.visibility`
// +kubebuilder:printcolumn:name="Members",type=integer,JSONPath=`.status.memberCount`
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// GitlabGroup is the Schema for the gitlabgroups API.
type GitlabGroup struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   GitlabGroupSpec   `json:"spec,omitempty"`
	Status GitlabGroupStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// GitlabGroupList contains a list of GitlabGroup.
type GitlabGroupList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []GitlabGroup `json:"items"`
}

func init() {
	SchemeBuilder.Register(&GitlabGroup{}, &GitlabGroupList{})
}
