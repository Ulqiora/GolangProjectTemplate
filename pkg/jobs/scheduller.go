package jobs

import (
	"company-doc/internal/adapters/secondary/metrics/prometheuses"
	"github.com/go-co-op/gocron/v2"
	"gitlab.wildberries.ru/wbbank/go-dpkg/dlog/v1"
)

type builderObject interface {
	build(scheduler gocron.Scheduler, logger dlog.Logger, gauge prometheuses.Adapter) gocron.Job
}

type Scheduler struct {
	logger  dlog.Logger
	impl    gocron.Scheduler
	metrics prometheuses.Adapter
}

func (s *Scheduler) Start() {
	s.impl.Start()
}

func (s *Scheduler) Stop() error {
	return s.impl.StopJobs()
}

func (s *Scheduler) AddJob(jobBuilders ...builderObject) {
	for i := range jobBuilders {
		job := jobBuilders[i].build(s.impl, s.logger, s.metrics)
		s.logger.I().Writef("job successful complete: ID(%v), name(%v)", job.ID(), job.Name())
	}
}

func NewJobScheduler(logger dlog.Logger, metrics prometheuses.Adapter, options ...gocron.SchedulerOption) (Scheduler, error) {
	scheduler, err := gocron.NewScheduler(options...)
	if err != nil {
		return Scheduler{}, err
	}
	return Scheduler{
		impl:    scheduler,
		metrics: metrics,
		logger:  logger,
	}, nil
}
