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
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	operatorv1alpha1 "github.com/DaviAraujoCC/k8s-operator-kubebuilder/api/v1alpha1"
	githubtype "github.com/DaviAraujoCC/k8s-operator-kubebuilder/pkg/utils/github/types"
)

// GithubRunnerAutoscalerReconciler reconciles a GithubRunnerAutoscaler object
type GithubRunnerAutoscalerReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

var (
	token   []byte
	orgname string
	cctx    context.Context
	cancel  context.CancelFunc
)

//+kubebuilder:rbac:groups=operator.hurb.com,resources=githubrunnerautoscalers,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=operator.hurb.com,resources=githubrunnerautoscalers/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=operator.hurb.com,resources=githubrunnerautoscalers/finalizers,verbs=update
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;update;patch
//+kubebuilder:rbac:groups=core,resources=secrets,verbs=get

func (r *GithubRunnerAutoscalerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	// Fetch the GithubRunnerAutoscaler instance
	githubrunner := &operatorv1alpha1.GithubRunnerAutoscaler{}
	err := r.Get(ctx, req.NamespacedName, githubrunner)
	if err != nil {
		if errors.IsNotFound(err) {
			log.Info("Unable to find GithubRunnerAutoscaler object")
			if cctx != nil {
				cancel()
			}
			return ctrl.Result{}, nil
		}
		log.Error(err, "Unable to read GithubRunnerAutoscaler")
		return ctrl.Result{}, err
	}

	// Check if the deployment already exists
	deployment := &appsv1.Deployment{}
	err = r.Get(ctx, client.ObjectKey{Name: githubrunner.Spec.DeploymentName, Namespace: githubrunner.Spec.Namespace}, deployment)
	if err != nil {
		if errors.IsNotFound(err) {
			log.Info("Unable to found Deployment object")
			return ctrl.Result{}, err
		}
		log.Error(err, "Unable to read Deployment name")
		return ctrl.Result{}, err
	}

	token, err = r.getToken(githubrunner)
	if err != nil {
		log.Error(err, "Unable to decode token")
		return ctrl.Result{}, err
	}

	orgname = githubrunner.Spec.OrgName

	if cctx != nil {
		log.Info("Recreating goroutine")
		cancel()
		cctx, cancel = context.WithCancel(context.Background())
	} else {
		cctx, cancel = context.WithCancel(context.Background())
	}

	log.Info("Created GithubRunnerAutoscaler for ", "GithubRunnerAutoscaler.Namespace", githubrunner.Namespace, "GithubRunnerAutoscaler.Name", githubrunner.Name)

	go r.autoscaleReplicas(ctx, cctx, deployment, githubrunner)

	githubrunner.Status.LastUpdateTime = metav1.Time{Time: time.Now()}
	r.Update(ctx, githubrunner)

	return ctrl.Result{}, nil
}

func (r *GithubRunnerAutoscalerReconciler) autoscaleReplicas(ctx context.Context, cctx context.Context, deploy *appsv1.Deployment, githubrunner *operatorv1alpha1.GithubRunnerAutoscaler) {
	log := log.FromContext(ctx)
	midIdle := 0.5
	for {
		data, err := requestGithubInfo(githubrunner)
		if err != nil {
			log.Error(err, "Unable to request github info, shutting down...")
			break
		}

		totalRunners := data.TotalCount
		qntRunnersBusy := 0
		for _, runner := range data.Runners {
			if runner.Busy {
				qntRunnersBusy++
			}
		}
		idleRunners := totalRunners - qntRunnersBusy
		percentIdle := float64(totalRunners-qntRunnersBusy) / float64(totalRunners)

		log.Info(fmt.Sprintf("Total runners: %d, busy runners: %d, idle runners: %d, percent idle: %f", totalRunners, qntRunnersBusy, idleRunners, percentIdle))
		midIdle = (midIdle + percentIdle) / 2

		oldReplicas := *deploy.Spec.Replicas

		switch {
		case *deploy.Spec.Replicas <= githubrunner.Spec.MinWorkers:
			deploy.Spec.Replicas = &githubrunner.Spec.MinWorkers
		case midIdle < 0.4 && *deploy.Spec.Replicas < githubrunner.Spec.MaxWorkers:
			replicas := *deploy.Spec.Replicas + int32(1)
			deploy.Spec.Replicas = &replicas
		case midIdle > 0.8 && *deploy.Spec.Replicas > githubrunner.Spec.MinWorkers:
			replicas := *deploy.Spec.Replicas - int32(1)
			deploy.Spec.Replicas = &replicas
		}

		if *deploy.Spec.Replicas != oldReplicas {
			log.Info(fmt.Sprintf("Changing replicas from %d to %d", oldReplicas, *deploy.Spec.Replicas))
			err = r.Update(ctx, deploy)
			if err != nil {
				log.Error(err, "Unable to update Deployment")
			}
		}
		select {
		case <-cctx.Done():
			return
		default:
			time.Sleep(15 * time.Second)
			continue
		}
	}
}

func requestGithubInfo(githubrunner *operatorv1alpha1.GithubRunnerAutoscaler) (githubtype.PayloadRunners, error) {

	req, err := http.NewRequest("GET", fmt.Sprintf("https://api.github.com/orgs/%s/actions/runners", orgname), nil)
	if err != nil {
		return githubtype.PayloadRunners{}, err
	}
	req.Header.Add("Authorization", fmt.Sprintf("token %s", string(token)))
	req.Header.Add("Accept", "application/vnd.github.v3+json")

	client := &http.Client{Timeout: time.Second * 10}
	resp, err := client.Do(req)
	if err != nil {
		return githubtype.PayloadRunners{}, err
	}
	defer resp.Body.Close()
	var data githubtype.PayloadRunners
	json.NewDecoder(resp.Body).Decode(&data)

	return data, nil
}

func (r *GithubRunnerAutoscalerReconciler) getToken(githubrunner *operatorv1alpha1.GithubRunnerAutoscaler) ([]byte, error) {
	secret := &corev1.Secret{}
	err := r.Get(context.Background(), client.ObjectKey{Name: githubrunner.Spec.GithubToken.SecretName, Namespace: githubrunner.Spec.Namespace}, secret)
	if err != nil && errors.IsNotFound(err) {
		return nil, err
	}
	tokenBase := secret.Data[githubrunner.Spec.GithubToken.KeyRef]

	return tokenBase, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *GithubRunnerAutoscalerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&operatorv1alpha1.GithubRunnerAutoscaler{}).
		Complete(r)
}
