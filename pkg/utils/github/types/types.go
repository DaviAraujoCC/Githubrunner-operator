package github

type PayloadRunners struct {
	TotalCount int `json:"total_count"`
	Runners    []struct {
		ID     int    `json:"id"`
		Name   string `json:"name"`
		Os     string `json:"os"`
		Status string `json:"status"`
		Busy   bool   `json:"busy"`
		Labels []struct {
			ID   int    `json:"id"`
			Name string `json:"name"`
			Type string `json:"type"`
		} `json:"labels"`
	} `json:"runners"`
}
