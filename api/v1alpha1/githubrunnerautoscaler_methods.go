package v1alpha1

import (
	"context"
	"strconv"

	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	defaultScaleDownThreshold  = "0.4"
	defaultScaleUpThreshold    = "0.8"
	defaultScaleUpMultiplier   = "1.2"
	defaultScaleDownMultiplier = "0.5"
)

func (gr *GithubRunnerAutoscaler) ValidateValues() {
	log := log.FromContext(context.Background())
	if _, err := strconv.ParseFloat(gr.Spec.Strategy.ScaleUpThreshold, 32); err != nil || gr.Spec.Strategy.ScaleUpThreshold == "" {
		log.Info("ScaleUpThreshold is not a valid float, using default value")
		gr.Spec.Strategy.ScaleUpThreshold = defaultScaleUpThreshold
	}
	if _, err := strconv.ParseFloat(gr.Spec.Strategy.ScaleDownThreshold, 32); err != nil || gr.Spec.Strategy.ScaleDownThreshold == "" {
		log.Info("ScaleDownThreshold is not a valid float, using default value")
		gr.Spec.Strategy.ScaleDownThreshold = defaultScaleDownThreshold
	}
	if _, err := strconv.ParseFloat(gr.Spec.Strategy.ScaleUpMultiplier, 32); err != nil || gr.Spec.Strategy.ScaleUpMultiplier == "" {
		log.Info("ScaleUpMultiplier is not a valid float, using default value")
		gr.Spec.Strategy.ScaleUpMultiplier = defaultScaleUpMultiplier
	}
	if _, err := strconv.ParseFloat(gr.Spec.Strategy.ScaleDownMultiplier, 32); err != nil || gr.Spec.Strategy.ScaleDownMultiplier == "" {
		log.Info("ScaleDownMultiplier is not a valid float, using default value")
		gr.Spec.Strategy.ScaleDownMultiplier = defaultScaleDownMultiplier
	}
}
