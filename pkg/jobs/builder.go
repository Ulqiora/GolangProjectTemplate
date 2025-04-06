package jobs

import (
	"context"
	"fmt"
	"time"

	"company-doc/internal/adapters/secondary/metrics/prometheuses"
	"github.com/go-co-op/gocron/v2"
	"github.com/google/uuid"
	"gitlab.wildberries.ru/wbbank/go-dpkg/dlog/v1"
)

type JobDefinitionWrap func() gocron.JobDefinition

func DurationJob(duration time.Duration) func() gocron.JobDefinition {
	return func() gocron.JobDefinition {
		return gocron.DurationJob(duration)
	}
}

func CronJob(crontab string, withSeconds bool) func() gocron.JobDefinition {
	return func() gocron.JobDefinition {
		return gocron.CronJob(crontab, withSeconds)
	}
}

type JobBuilder interface {
	SetTask(fn func(ctx context.Context) error, ctx context.Context) JobBuilder
	SetOptions(options ...gocron.JobOption) JobBuilder
	build(scheduler gocron.Scheduler, logger dlog.Logger, metrics prometheuses.Adapter) gocron.Job
}

type builder struct {
	definition JobDefinitionWrap
	task       gocron.Task
	options    []gocron.JobOption
}

func (b *builder) SetTask(fn func(ctx context.Context) error, ctx context.Context) JobBuilder {
	b.task = gocron.NewTask(fn, ctx)
	return b
}

func (b *builder) SetOptions(options ...gocron.JobOption) JobBuilder {
	b.options = append(b.options, options...)
	return b
}

func NewJobBuilder(kind JobDefinitionWrap) JobBuilder {
	return &builder{
		definition: kind,
		options:    []gocron.JobOption{gocron.WithIdentifier(uuid.New())},
	}
}

func (b *builder) build(scheduler gocron.Scheduler, logger dlog.Logger, metrics prometheuses.Adapter) gocron.Job {
	jobOption := gocron.AfterJobRunsWithError(func(jobID uuid.UUID, jobName string, err error) {
		if err != nil {
			logger.E().Writef("job(name: %s) completed with error: %v)", jobName, err)
			if metrics.Enable {
				metrics.Jobs.WithEntity(jobName).Gauge().Set(0.0)
			}
		} else {
			logger.I().Writef("job(name: %s) completed without error", jobName)
			if metrics.Enable {
				metrics.Jobs.WithEntity(jobName).Gauge().Set(1.0)
			}
		}
	})
	b.options = append(b.options, gocron.WithEventListeners(jobOption))

	job, err := scheduler.NewJob(b.definition(), b.task, b.options...)
	if err != nil {
		panic(fmt.Sprintf("failed to construct job: %v", err))
	}
	return job
}
