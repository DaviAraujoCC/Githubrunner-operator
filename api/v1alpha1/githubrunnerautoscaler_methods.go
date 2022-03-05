package v1alpha1

import "strconv"

const (
	defaultScaleDownThreshold  = "0.4"
	defaultScaleUpThreshold    = "0.8"
	defaultScaleUpMultiplier   = "1.2"
	defaultScaleDownMultiplier = "0.5"
)

func (gr *GithubRunnerAutoscaler) SetScaleValues() {
	if _, err := strconv.ParseFloat(gr.Spec.Strategy.ScaleUpThreshold, 32); err != nil || gr.Spec.Strategy.ScaleUpThreshold == "" {
		gr.Spec.Strategy.ScaleUpThreshold = defaultScaleUpThreshold
	}
	if _, err := strconv.ParseFloat(gr.Spec.Strategy.ScaleDownThreshold, 32); err != nil || gr.Spec.Strategy.ScaleDownThreshold == "" {
		gr.Spec.Strategy.ScaleDownThreshold = defaultScaleDownThreshold
	}
	if _, err := strconv.ParseFloat(gr.Spec.Strategy.ScaleUpMultiplier, 32); err != nil || gr.Spec.Strategy.ScaleUpMultiplier == "" {
		gr.Spec.Strategy.ScaleUpMultiplier = defaultScaleUpMultiplier
	}
	if _, err := strconv.ParseFloat(gr.Spec.Strategy.ScaleDownMultiplier, 32); err != nil || gr.Spec.Strategy.ScaleDownMultiplier == "" {
		gr.Spec.Strategy.ScaleDownMultiplier = defaultScaleDownMultiplier
	}
}
