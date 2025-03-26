package jobs

import (
	"context"
	"fmt"
	"time"

	"github.com/go-co-op/gocron/v2"
	"github.com/google/uuid"
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
	build(scheduler gocron.Scheduler) gocron.Job
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

func (b *builder) build(scheduler gocron.Scheduler) gocron.Job {
	job, err := scheduler.NewJob(b.definition(), b.task, b.options...)
	if err != nil {
		panic(fmt.Sprintf("failed to construct job: %v", err))
	}
	return job
}
