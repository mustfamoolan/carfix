package jobs

import (
	"fmt"
)

type FetchCarfaxReportJobJob struct {
	Payload string
}

func NewFetchCarfaxReportJobJob(payload string) *FetchCarfaxReportJobJob {
	return &FetchCarfaxReportJobJob{
		Payload: payload,
	}
}

func (j *FetchCarfaxReportJobJob) Handle() error {
	// Job logic here
	fmt.Printf("Processing FetchCarfaxReportJobJob with payload: %s\n", j.Payload)
	return nil
}
