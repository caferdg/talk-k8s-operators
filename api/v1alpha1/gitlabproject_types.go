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

// GitlabProjectSpec defines the desired state of GitlabProject.
type GitlabProjectSpec struct {
	// Reference to a GitlabGroup CR name in the same namespace
	ParentGroupRef string `json:"parentGroupRef"`
	// Branch to track
	TrackedBranch string `json:"trackedBranch"`
	// Reference to a Notifier CR name in the same namespace
	NotifierRef string `json:"notifierRef,omitempty"`
	// GitLab project properties
	Properties GitlabProjectProperties `json:"properties"`
}

// GitlabProjectProperties defines the desired GitLab project settings.
type GitlabProjectProperties struct {
	// Name of the project
	Name string `json:"name"`
	// Description of the project
	Description string `json:"description,omitempty"`
	// Visibility level (private, internal, public)
	Visibility string `json:"visibility,omitempty"`
}

// GitlabProjectStatus defines the observed state of GitlabProject.
type GitlabProjectStatus struct {
	// Full path of the project on GitLab
	FullPath string `json:"fullPath,omitempty"`
	// Description of the project
	Description string `json:"description,omitempty"`
	// Visibility level (private, internal, public)
	Visibility string `json:"visibility,omitempty"`
	// Last commit on the tracked branch
	LastCommit GitlabCommitStatus `json:"lastCommit,omitempty"`
	// Last pipeline on the tracked branch
	LastPipeline GitlabPipelineStatus `json:"lastPipeline,omitempty"`
	// Whether the project has been stamped on GitLab
	Stamped bool `json:"stamped,omitempty"`
	// Standard conditions
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// GitlabCommitStatus represents the last commit on a branch.
type GitlabCommitStatus struct {
	// Commit SHA
	SHA string `json:"sha,omitempty"`
	// Commit message
	Message string `json:"message,omitempty"`
	// Author name
	Author string `json:"author,omitempty"`
	// Commit timestamp
	CreatedAt string `json:"createdAt,omitempty"`
}

// GitlabPipelineStatus represents the last pipeline on a branch.
type GitlabPipelineStatus struct {
	// Pipeline ID
	ID int `json:"id,omitempty"`
	// Pipeline status (running, success, failed, etc.)
	Status string `json:"status,omitempty"`
	// Pipeline web URL
	WebURL string `json:"webUrl,omitempty"`
	// Pipeline creation timestamp
	CreatedAt string `json:"createdAt,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=gproject
// +kubebuilder:printcolumn:name="Full Path",type=string,JSONPath=`.status.fullPath`
// +kubebuilder:printcolumn:name="Branch",type=string,JSONPath=`.spec.trackedBranch`
// +kubebuilder:printcolumn:name="Last Commit",type=string,JSONPath=`.status.lastCommit.sha`
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// GitlabProject is the Schema for the gitlabprojects API.
type GitlabProject struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   GitlabProjectSpec   `json:"spec,omitempty"`
	Status GitlabProjectStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// GitlabProjectList contains a list of GitlabProject.
type GitlabProjectList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []GitlabProject `json:"items"`
}

func init() {
	SchemeBuilder.Register(&GitlabProject{}, &GitlabProjectList{})
}
