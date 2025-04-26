package schedule

import (
	"github.com/robfig/cron/v3"
	"go-api/internal/domain/usecase/shorturl"
	"go-api/pkg/log"
	"go-api/pkg/msg"
	"go-api/pkg/resource"
)

type ShortUrlScheduler struct {
	cron    *cron.Cron
	useCase shorturl.UseCase
}

func NewShortUrlScheduler(useCase shorturl.UseCase) *ShortUrlScheduler {
	return &ShortUrlScheduler{cron: cron.New(), useCase: useCase}
}

// InitShortUrlScheduleTasks initializes short url schedule tasks
func (scheduler *ShortUrlScheduler) InitShortUrlScheduleTasks() {
	_, err := scheduler.cron.AddFunc(resource.GetString("app.short-url.clear.cron"), scheduler.ClearShortUrlByExpiration)

	if err != nil {
		panic(err)
	}

	scheduler.cron.Start()
}

func (scheduler *ShortUrlScheduler) ClearShortUrlByExpiration() {
	log.Info(msg.GetMessage("short-url.cron.start"))

	err := scheduler.useCase.ClearAllByExpiration()

	if err != nil {
		log.Error(msg.GetMessage("short-url.error.clear-failed"))
		return
	}

	log.Info(msg.GetMessage("short-url.cron.end"))
}
