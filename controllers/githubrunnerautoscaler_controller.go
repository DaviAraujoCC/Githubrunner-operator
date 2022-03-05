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
	"time"

	gh "github.com/DaviAraujoCC/k8s-operator-kubebuilder/github"
	"github.com/google/go-github/v39/github"
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

	scaleUpThreshold    float64
	scaleDownThreshold  float64
	scaleUpMultiplier   float64
	scaleDownMultiplier float64
	minReplicas         int32
	maxReplicas         int32
	replicas            int32
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

	// Check if the deployment exists
	deployment := &appsv1.Deployment{}
	err = r.Get(ctx, client.ObjectKey{Name: githubrunner.Spec.TargetSpec.TargetDeploymentName, Namespace: githubrunner.Spec.TargetSpec.TargetNamespace}, deployment)
	if err != nil {
		if errors.IsNotFound(err) {
			log.Info("Unable to find Deployment object")
			return ctrl.Result{}, err
		}
		log.Error(err, "Unable to read Deployment object")
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
		ghClient, err = gh.NewClient(string(token), orgname)
		if err != nil {
			log.Error(err, "Unable to create Github client")
		}
	}

	// Set variables
	githubrunner.SetScaleValues()
	scaleUpThreshold, _ = strconv.ParseFloat(githubrunner.Spec.Strategy.ScaleUpThreshold, 32)
	scaleDownThreshold, _ = strconv.ParseFloat(githubrunner.Spec.Strategy.ScaleDownThreshold, 32)
	scaleUpMultiplier, _ = strconv.ParseFloat(githubrunner.Spec.Strategy.ScaleUpMultiplier, 32)
	scaleDownMultiplier, _ = strconv.ParseFloat(githubrunner.Spec.Strategy.ScaleDownMultiplier, 32)
	minReplicas = githubrunner.Spec.TargetSpec.MinReplicas
	maxReplicas = githubrunner.Spec.TargetSpec.MaxReplicas
	replicas = *deployment.Spec.Replicas

	strategy := githubrunner.Spec.Strategy.Type

	switch strategy {
	case "PercentRunnersBusy":

		return func(ghClient *gh.Client, deploy *appsv1.Deployment, githubrunner *operatorv1alpha1.GithubRunnerAutoscaler) (ctrl.Result, error) {

			runners, err := ghClient.ListOrganizationRunners()
			if err != nil {
				log.Error(err, "Unable to list Github runners")
				return ctrl.Result{}, err
			}

			calculate(runners, githubrunner, deploy, "busy", ctx)

			if *deploy.Spec.Replicas != replicas {
				log.Info(fmt.Sprintf("Changing replicas from %d to %d", replicas, *deploy.Spec.Replicas))
				err := r.Update(ctx, deploy)
				if err != nil {
					log.Error(err, "Unable to update Deployment")
					return ctrl.Result{}, err
				}
			}

			return ctrl.Result{}, nil
		}(ghClient, deployment, githubrunner)
	}

	log.Info("Strategy not found in object, ignoring GithubRunnerAutoscaler...", "GithubRunnerAutoscaler.Namespace", githubrunner.Namespace, "GithubRunnerAutoscaler.Name", githubrunner.Name)
	return ctrl.Result{}, nil
}

func (r *GithubRunnerAutoscalerReconciler) getToken(githubrunner *operatorv1alpha1.GithubRunnerAutoscaler) ([]byte, error) {
	secret := &corev1.Secret{}
	err := r.Get(context.Background(), client.ObjectKey{Name: githubrunner.Spec.GithubToken.SecretName, Namespace: githubrunner.Spec.TargetSpec.TargetNamespace}, secret)
	if err != nil && errors.IsNotFound(err) {
		return nil, err
	}
	tokenBase := secret.Data[githubrunner.Spec.GithubToken.KeyRef]

	return tokenBase, nil
}

func calculate(runners []*github.Runner, githubrunner *operatorv1alpha1.GithubRunnerAutoscaler, deploy *appsv1.Deployment, t string, ctx context.Context) {
	log := log.FromContext(ctx)

	switch t {
	case "busy":
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

		switch {
		case replicas < minReplicas:
			log.Info("Deployment have less replicas than min replicas, scaling up...", "Deployment.Namespace", deploy.Namespace, "Deployment.Name", deploy.Name, "minReplicas", minReplicas, "replicas", replicas)
			deploy.Spec.Replicas = &minReplicas
		case percentBusy >= scaleUpThreshold && *deploy.Spec.Replicas < maxReplicas:
			replicasNew := int32(math.Ceil(float64(replicas) * scaleUpMultiplier))
			if replicasNew > maxReplicas {
				log.Info("Desired deployment replicas (autoscale) is bigger than max workers, setting replicas to max workers.")
				deploy.Spec.Replicas = &maxReplicas
			} else {
				deploy.Spec.Replicas = &replicasNew
			}
		case percentBusy <= scaleDownThreshold && replicas > minReplicas:
			replicasNew := int32(math.Ceil(float64(replicas) * scaleDownMultiplier))
			if replicasNew < minReplicas {
				log.Info("Desired deployment replicas (autoscale) is less than min workers, setting replicas to min workers.")
				deploy.Spec.Replicas = &minReplicas
			} else {
				deploy.Spec.Replicas = &replicasNew
			}
		}
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *GithubRunnerAutoscalerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&operatorv1alpha1.GithubRunnerAutoscaler{}).
		Complete(r)
}
