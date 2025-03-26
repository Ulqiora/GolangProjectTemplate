package jobs

import (
	"github.com/go-co-op/gocron/v2"
	"github.com/go-openapi/runtime/logger"
)

type builderObject interface {
	build(gocron.Scheduler) gocron.Job
}

type Scheduler struct {
	logger logger.Logger
	impl   gocron.Scheduler
}

func (s *Scheduler) Start() {
	s.impl.Start()
}

func (s *Scheduler) Stop() error {
	return s.impl.StopJobs()
}

func (s *Scheduler) AddJob(jobBuilder builderObject) {
	job := jobBuilder.build(s.impl)
	s.logger.Printf("job successful complete: ID(%v), name(%v)", job.ID(), job.Name())
}

func NewJobScheduler(logger logger.Logger, options ...gocron.SchedulerOption) (Scheduler, error) {
	scheduler, err := gocron.NewScheduler(options...)
	if err != nil {
		return Scheduler{}, err
	}
	return Scheduler{
		impl:   scheduler,
		logger: logger,
	}, nil
}
