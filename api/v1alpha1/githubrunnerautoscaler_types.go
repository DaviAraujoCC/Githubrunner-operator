/*
Copyright 2022.

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

//Ability to create secret automatically
type GithubRunnerAutoscalerSpec struct {
	TargetSpec  TargetSpec      `json:"targetSpec"`
	OrgName     string          `json:"orgName"`
	GithubToken GithubTokenSpec `json:"githubToken"`
	Strategy    StrategySpec    `json:"strategy"`
}

type TargetSpec struct {
	TargetDeploymentName string `json:"targetDeploymentName"`
	TargetNamespace      string `json:"targetNamespace"`
	MinReplicas          int32  `json:"minReplicas"`
	MaxReplicas          int32  `json:"maxReplicas"`
}

type StrategySpec struct {
	Type string `json:"type"`
	//+optional
	ScaleUpThreshold string `json:"scaleUpThreshold,omitempty"`
	//+optional
	ScaleDownThreshold string `json:"scaleDownThreshold,omitempty"`
	//+optional
	ScaleUpMultiplier string `json:"scaleUpMultiplier,omitempty"`
	//+optional
	ScaleDownMultiplier string `json:"scaleDownMultiplier,omitempty"`
}

type GithubTokenSpec struct {
	SecretName string `json:"secretName"`
	KeyRef     string `json:"keyRef"`
}

// GithubRunnerAutoscalerStatus defines the observed state of GithubRunnerAutoscaler
type GithubRunnerAutoscalerStatus struct {
}

//+kubebuilder:object:root=true

// GithubRunnerAutoscaler is the Schema for the githubrunnerautoscalers API]
//+kubebuilder:subresource:status
type GithubRunnerAutoscaler struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   GithubRunnerAutoscalerSpec   `json:"spec,omitempty"`
	Status GithubRunnerAutoscalerStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// GithubRunnerAutoscalerList contains a list of GithubRunnerAutoscaler
type GithubRunnerAutoscalerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []GithubRunnerAutoscaler `json:"items"`
}

func init() {
	SchemeBuilder.Register(&GithubRunnerAutoscaler{}, &GithubRunnerAutoscalerList{})
}
