package worker

import (
	"codepulse/internal/domain"
	"codepulse/internal/runner"
	"codepulse/internal/storage"
	"context"
	"encoding/json"
	"log/slog"

	"github.com/segmentio/kafka-go"
)

type Processor struct {
	taskStorage *storage.TaskStorage
	subStorage  *storage.SubmissionStorage
	runner      *runner.DockerRunner
}

func NewProcessor(taskStorage *storage.TaskStorage, subStorage *storage.SubmissionStorage, runner *runner.DockerRunner) *Processor {
	return &Processor{
		taskStorage: taskStorage,
		subStorage:  subStorage,
		runner:      runner}
}

func (p *Processor) Process(ctx context.Context, msg kafka.Message) error {
	var event domain.SubmissionEvent

	if err := json.Unmarshal(msg.Value, &event); err != nil {
		return err
	}

	if err := p.subStorage.UpdateStatus(ctx, event.SubmissionID, domain.StatusRunning); err != nil {
		return err
	}

	sub, err := p.subStorage.FindSubmissionByID(ctx, event.SubmissionID)
	if err != nil {
		return err
	}
	task, err := p.taskStorage.FindTaskByID(ctx, event.TaskID)
	if err != nil {
		return err
	}

	tests, err := p.taskStorage.FindAllTestsByTaskID(ctx, event.TaskID)
	if err != nil {
		return err
	}

	res, err := p.runner.Run(ctx, sub.SourceCode, tests, task.TimeLimitMs, task.MemoryLimitMb)
	if err != nil {
		return err
	}

	if err := p.subStorage.UpdateResult(ctx, event.SubmissionID, *res); err != nil {
		return err
	}

	slog.Info("submission processed", "id", event.SubmissionID, "verdict", res.Verdict)

	return nil
}
