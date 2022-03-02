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

package controllers

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	gh "github.com/DaviAraujoCC/k8s-operator-kubebuilder/github"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	operatorv1alpha1 "github.com/DaviAraujoCC/k8s-operator-kubebuilder/api/v1alpha1"
)

// GithubRunnerAutoscalerReconciler reconciles a GithubRunnerAutoscaler object
type GithubRunnerAutoscalerReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

var (
	ghClient    *gh.Client
	timeRefresh time.Time
)

const (
	defaultScaleDownThreshold = "0.4"
	defaultScaleUpThreshold   = "0.8"
	defaultScaleUpFactor      = "1.2"
	defaultScaleDownFactor    = "0.5"
)

//+kubebuilder:rbac:groups=operator.hurb.com,resources=githubrunnerautoscalers,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=operator.hurb.com,resources=githubrunnerautoscalers/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=operator.hurb.com,resources=githubrunnerautoscalers/finalizers,verbs=update
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;update;patch
//+kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch

func (r *GithubRunnerAutoscalerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	// Fetch the GithubRunnerAutoscaler instance
	githubrunner := &operatorv1alpha1.GithubRunnerAutoscaler{}
	err := r.Get(ctx, req.NamespacedName, githubrunner)
	if err != nil {
		if errors.IsNotFound(err) {
			log.Info("Unable to find GithubRunnerAutoscaler object")
			return ctrl.Result{}, nil
		}
		log.Error(err, "Unable to read GithubRunnerAutoscaler")
		return ctrl.Result{}, err
	}

	// Check if the deployment already exists
	deployment := &appsv1.Deployment{}
	err = r.Get(ctx, client.ObjectKey{Name: githubrunner.Spec.TargetDeploymentName, Namespace: githubrunner.Spec.TargetNamespace}, deployment)
	if err != nil {
		if errors.IsNotFound(err) {
			log.Info("Unable to find Deployment object")
			return ctrl.Result{}, err
		}
		log.Error(err, "Unable to read Deployment name")
		return ctrl.Result{}, err
	}

	if ghClient == nil || timeRefresh.Add(1*time.Hour).Before(time.Now()) {
		timeRefresh = time.Now()
		token, err := r.getToken(githubrunner)
		if err != nil {
			log.Error(err, "Unable to decode token")
			return ctrl.Result{}, err
		}
		orgname := githubrunner.Spec.OrgName
		ghConf := &gh.Config{Token: string(token)}
		ghClient, err = ghConf.NewClient(orgname)
		if err != nil {
			log.Error(err, "Unable to create Github client")
		}
	}

	strategy := githubrunner.Spec.Strategy.Type

	switch strategy {
	case "PercentRunnersBusy":
		githubrunnerNew := setScaleValuesOrDefault(githubrunner)
		log.Info("Created GithubRunnerAutoscaler for ", "GithubRunnerAutoscaler.Namespace", githubrunner.Namespace, "GithubRunnerAutoscaler.Name", githubrunner.Name)
		return r.autoscale(ctx, ghClient, deployment, githubrunnerNew)
	}

	log.Info("Strategy not found in object, ignoring GithubRunnerAutoscaler...", "GithubRunnerAutoscaler.Namespace", githubrunner.Namespace, "GithubRunnerAutoscaler.Name", githubrunner.Name)
	return ctrl.Result{}, nil
}

func (r *GithubRunnerAutoscalerReconciler) autoscale(ctx context.Context, ghClient *gh.Client, deploy *appsv1.Deployment, githubrunner *operatorv1alpha1.GithubRunnerAutoscaler) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	runners, err := ghClient.ListOrganizationRunners()
	if err != nil {
		log.Error(err, "Unable to list Github runners")
		return ctrl.Result{}, err
	}
	totalRunners := 0
	qntRunnersBusy := 0
	for _, runner := range runners {
		if *runner.Busy {
			qntRunnersBusy++
		}
		totalRunners++
	}
	idleRunners := totalRunners - qntRunnersBusy
	percentBusy := float64(qntRunnersBusy) / float64(totalRunners)
	log.Info(fmt.Sprintf("Total runners: %d, busy runners: %d, idle runners: %d, percent busy: %f", totalRunners, qntRunnersBusy, idleRunners, percentBusy))

	replicas := *deploy.Spec.Replicas

	scaleUpThreshold, _ := strconv.ParseFloat(githubrunner.Spec.Strategy.ScaleUpFactor, 32)
	scaleDownThreshold, _ := strconv.ParseFloat(githubrunner.Spec.Strategy.ScaleDownFactor, 32)
	scaleUpFactor, _ := strconv.ParseFloat(githubrunner.Spec.Strategy.ScaleUpFactor, 32)
	scaleDownFactor, _ := strconv.ParseFloat(githubrunner.Spec.Strategy.ScaleDownFactor, 32)

	switch {
	case replicas < githubrunner.Spec.MinReplicas:
		deploy.Spec.Replicas = &githubrunner.Spec.MinReplicas
	case percentBusy >= scaleUpThreshold && *deploy.Spec.Replicas < githubrunner.Spec.MaxReplicas:
		replicasNew := math.Ceil(float64(replicas) * scaleUpFactor)
		replicasConv := int32(replicasNew)
		if replicasConv > githubrunner.Spec.MaxReplicas {
			log.Info("Desired deployment replicas is bigger than max workers, setting replicas to max workers.")
			deploy.Spec.Replicas = &githubrunner.Spec.MaxReplicas
		} else {
			deploy.Spec.Replicas = &replicasConv
		}
	case percentBusy <= scaleDownThreshold && replicas > githubrunner.Spec.MinReplicas:
		replicasNew := math.Ceil(float64(replicas) * scaleDownFactor)
		replicasConv := int32(replicasNew)
		if replicasConv < githubrunner.Spec.MinReplicas {
			log.Info("Desired deployment replicas is less than min workers, setting replicas to min workers.")
			deploy.Spec.Replicas = &githubrunner.Spec.MinReplicas
		} else {
			deploy.Spec.Replicas = &replicasConv
		}
	}

	if *deploy.Spec.Replicas != replicas {
		log.Info(fmt.Sprintf("Changing replicas from %d to %d", replicas, *deploy.Spec.Replicas))
		err := r.Update(ctx, deploy)
		if err != nil && !strings.Contains(err.Error(), "has been modified") {
			log.Error(err, "Unable to update Deployment")
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

func (r *GithubRunnerAutoscalerReconciler) getToken(githubrunner *operatorv1alpha1.GithubRunnerAutoscaler) ([]byte, error) {
	secret := &corev1.Secret{}
	err := r.Get(context.Background(), client.ObjectKey{Name: githubrunner.Spec.GithubToken.SecretName, Namespace: githubrunner.Spec.TargetNamespace}, secret)
	if err != nil && errors.IsNotFound(err) {
		return nil, err
	}
	tokenBase := secret.Data[githubrunner.Spec.GithubToken.KeyRef]

	return tokenBase, nil
}

func setScaleValuesOrDefault(githubrunner *operatorv1alpha1.GithubRunnerAutoscaler) *operatorv1alpha1.GithubRunnerAutoscaler {
	if _, err := strconv.ParseFloat(githubrunner.Spec.Strategy.ScaleUpThreshold, 32); err != nil || githubrunner.Spec.Strategy.ScaleUpThreshold == "" {
		githubrunner.Spec.Strategy.ScaleUpThreshold = defaultScaleUpThreshold
	}
	if _, err := strconv.ParseFloat(githubrunner.Spec.Strategy.ScaleDownThreshold, 32); err != nil || githubrunner.Spec.Strategy.ScaleDownThreshold == "" {
		githubrunner.Spec.Strategy.ScaleDownThreshold = defaultScaleDownThreshold
	}
	if _, err := strconv.ParseFloat(githubrunner.Spec.Strategy.ScaleUpFactor, 32); err != nil || githubrunner.Spec.Strategy.ScaleUpFactor == "" {
		githubrunner.Spec.Strategy.ScaleUpFactor = defaultScaleUpFactor
	}
	if _, err := strconv.ParseFloat(githubrunner.Spec.Strategy.ScaleDownFactor, 32); err != nil || githubrunner.Spec.Strategy.ScaleDownFactor == "" {
		githubrunner.Spec.Strategy.ScaleDownFactor = defaultScaleDownFactor
	}
	return githubrunner
}

// SetupWithManager sets up the controller with the Manager.
func (r *GithubRunnerAutoscalerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&operatorv1alpha1.GithubRunnerAutoscaler{}).
		Complete(r)
}
