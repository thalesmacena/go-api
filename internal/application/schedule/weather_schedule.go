package schedule

import (
	"context"
	"go-api/internal/domain/usecase/weather"
	"go-api/pkg/log"
	"go-api/pkg/redis"
	"time"

	"github.com/google/uuid"
	"github.com/robfig/cron/v3"
	"go.uber.org/zap"
)

// WeatherSchedulerConfig holds configuration for the weather scheduler
type WeatherSchedulerConfig struct {
	CronExpression  string
	LockTTL         time.Duration
	RefreshInterval time.Duration
}

// WeatherScheduler handles scheduled weather monitoring updates with distributed locking
type WeatherScheduler struct {
	cron        *cron.Cron
	useCase     weather.UseCase
	redisClient *redis.Client
	config      *WeatherSchedulerConfig
}

// NewWeatherScheduler creates a new weather scheduler with distributed locking support
func NewWeatherScheduler(useCase weather.UseCase, redisClient *redis.Client, cronExpression string, lockTTL int, refreshInterval int) *WeatherScheduler {
	return &WeatherScheduler{
		cron:        cron.New(),
		useCase:     useCase,
		redisClient: redisClient,
		config: &WeatherSchedulerConfig{
			CronExpression:  cronExpression,
			LockTTL:         time.Duration(lockTTL) * time.Second,
			RefreshInterval: time.Duration(refreshInterval) * time.Second,
		},
	}
}

// InitWeatherScheduleTasks initializes weather schedule tasks with distributed locking
func (s *WeatherScheduler) InitWeatherScheduleTasks(ctx context.Context) {
	go func() {
		// Create a scheduled task lock with persistent refresh
		lock := redis.NewScheduledTaskLock(
			s.redisClient,
			"weather_monitoring_scheduler",
			s.getLockTTL(),
			s.getRefreshInterval(),
			"weather_schedules",
		)

		err := lock.Lock(ctx)
		if err != nil {
			log.Errorf("Failed to acquire distributed lock, weather scheduler will not be initialized: %v", err)
			return
		}

		// Start auto-refresh to maintain the lock indefinitely
		refreshErrChan := lock.AutoRefresh(ctx)

		// Get cron expression from config
		cronExpression := s.config.CronExpression

		// Schedule task to run at configured times (default: 02:00, 10:00, and 18:00 daily)
		_, err = s.cron.AddFunc(cronExpression, s.ExecuteScheduledTask)

		if err != nil {
			log.Errorf("Failed to initialize weather scheduler, cron will not be started: %v", err)
			return
		}

		// Start the scheduler
		s.cron.Start()
		log.Infof("Weather monitoring scheduler started successfully with cron expression: %s", cronExpression)

		// Monitor auto-refresh errors and stop scheduler if refresh fails
		err = <-refreshErrChan

		// Stop the scheduler due to refresh failure or context cancellation
		if s.cron != nil {
			cronCtx := s.cron.Stop()
			<-cronCtx.Done()
		}

		if err != nil {
			log.Errorf("Weather monitoring scheduler stopped due to auto-refresh failure: %v", err)
		} else {
			log.Info("Weather monitoring scheduler stopped gracefully")
		}
	}()
}

// ExecuteScheduledTask executes the city monitoring update
func (s *WeatherScheduler) ExecuteScheduledTask() {
	// Generate request ID for tracking
	requestID := uuid.New().String()

	log.Info("Weather monitoring scheduled task triggered", zap.String("request_id", requestID))

	// Execute the scheduled task
	log.Info("Executing scheduled weather monitoring update for all cities", zap.String("request_id", requestID))
	if err := s.useCase.UpdateAllCitiesMonitoringScheduled(requestID); err != nil {
		log.Error("Failed to execute scheduled weather monitoring update", zap.String("request_id", requestID), zap.Error(err))
		return
	}

	log.Info("Scheduled weather monitoring update completed successfully", zap.String("request_id", requestID))
}

// Stop gracefully stops the scheduler
func (s *WeatherScheduler) Stop() {
	if s.cron != nil {
		ctx := s.cron.Stop()
		<-ctx.Done()
	}
}

// Helper methods to get duration values from config
func (s *WeatherScheduler) getLockTTL() time.Duration {
	if s.config.LockTTL > 0 {
		return s.config.LockTTL
	}
	return 10 * time.Minute // Default: 10 minutes
}

func (s *WeatherScheduler) getRefreshInterval() time.Duration {
	if s.config.RefreshInterval > 0 {
		return s.config.RefreshInterval
	}
	return 1 * time.Minute // Default: 1 minute
}
