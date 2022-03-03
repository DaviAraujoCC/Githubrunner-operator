package v1alpha1

import "strconv"

const (
	defaultScaleDownThreshold = "0.4"
	defaultScaleUpThreshold   = "0.8"
	defaultScaleUpFactor      = "1.2"
	defaultScaleDownFactor    = "0.5"
)

func (gr *GithubRunnerAutoscaler) SetScaleValuesOrDefault() {
	if _, err := strconv.ParseFloat(gr.Spec.Strategy.ScaleUpThreshold, 32); err != nil || gr.Spec.Strategy.ScaleUpThreshold == "" {
		gr.Spec.Strategy.ScaleUpThreshold = defaultScaleUpThreshold
	}
	if _, err := strconv.ParseFloat(gr.Spec.Strategy.ScaleDownThreshold, 32); err != nil || gr.Spec.Strategy.ScaleDownThreshold == "" {
		gr.Spec.Strategy.ScaleDownThreshold = defaultScaleDownThreshold
	}
	if _, err := strconv.ParseFloat(gr.Spec.Strategy.ScaleUpFactor, 32); err != nil || gr.Spec.Strategy.ScaleUpFactor == "" {
		gr.Spec.Strategy.ScaleUpFactor = defaultScaleUpFactor
	}
	if _, err := strconv.ParseFloat(gr.Spec.Strategy.ScaleDownFactor, 32); err != nil || gr.Spec.Strategy.ScaleDownFactor == "" {
		gr.Spec.Strategy.ScaleDownFactor = defaultScaleDownFactor
	}
}
