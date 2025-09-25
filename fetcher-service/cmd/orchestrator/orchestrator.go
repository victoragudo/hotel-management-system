package main

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"github.com/victoragudo/hotel-management-system/fetcher-service/internal/infrastructure/queue"
	"github.com/victoragudo/hotel-management-system/fetcher-service/pkg/constants"
	"github.com/victoragudo/hotel-management-system/fetcher-service/proto/orchestrator"
	constants2 "github.com/victoragudo/hotel-management-system/pkg/constants"
	"github.com/victoragudo/hotel-management-system/pkg/database"
	"gorm.io/gorm"
)

type OrchestratorGRPCServer struct {
	orchestrator.UnimplementedOrchestratorServiceServer
	config            Config
	logger            *slog.Logger
	rabbitMQPublisher *queue.RabbitMQPublisher
	db                *gorm.DB
}

func (s *OrchestratorGRPCServer) ProcessFetchRequest(ctx context.Context, fetchRequest *orchestrator.FetchRequest) (*orchestrator.FetchResponse, error) {
	ft := fetchRequest.MessageType
	if ft == orchestrator.MessageType_UNSPECIFIED {
		return &orchestrator.FetchResponse{
			Success:     false,
			Message:     "invalid message type",
			RequestId:   fetchRequest.RequestId,
			JobsCreated: 0,
			Jobs:        nil,
		}, nil
	}

	jobsCreated, jobInfos, err := s.enqueueJobs(ctx, ft)
	if err != nil {
		s.logger.Error("ProcessFetchRequest failed", "error", err, "request_id", fetchRequest.RequestId)
		return &orchestrator.FetchResponse{
			Success:     false,
			Message:     fmt.Sprintf("enqueue failed: %v", err),
			RequestId:   fetchRequest.RequestId,
			JobsCreated: 0,
			Jobs:        nil,
		}, nil
	}
	if jobsCreated > 0 {
		s.logger.Info("jobs enqueued", "request_id", fetchRequest.RequestId, "jobs_created", jobsCreated, "jobs", jobInfos)
	}

	return &orchestrator.FetchResponse{
		Success:     true,
		Message:     "jobs enqueued",
		RequestId:   fetchRequest.RequestId,
		JobsCreated: int32(jobsCreated),
		Jobs:        jobInfos,
	}, nil
}

func (s *OrchestratorGRPCServer) GetHealthStatus(_ context.Context, _ *orchestrator.HealthRequest) (*orchestrator.HealthResponse, error) {
	return &orchestrator.HealthResponse{
		Status: "healthy",
	}, nil
}

// enqueueJobs enqueues jobs for processing based on the specified fetch type and hotel ID, using batching for database queries.
// It publishes job information to RabbitMQ and handles retries in case of failures. Returns the count of jobs enqueued,
// details of the jobs enqueued, and any error encountered during the operation.
func (s *OrchestratorGRPCServer) enqueueJobs(ctx context.Context, messageType orchestrator.MessageType) (int, []*orchestrator.JobInfo, error) {
	messageTypeStr := "hotel"
	switch messageType {
	case orchestrator.MessageType_UPDATE_HOTEL:
		messageTypeStr = constants.MessageTypeUpdateHotel
	case orchestrator.MessageType_UPDATE_REVIEW:
		messageTypeStr = constants.MessageTypeUpdateReview
	case orchestrator.MessageType_UPDATE_TRANSLATION:
		messageTypeStr = constants.MessageTypeUpdateTranslation
	case orchestrator.MessageType_FETCH_MISSING_TRANSLATIONS:
		messageTypeStr = constants.MessageTypeFetchTranslation
	case orchestrator.MessageType_FETCH_MISSING_REVIEWS:
		messageTypeStr = constants.MessageTypeFetchReview
	case orchestrator.MessageType_UNSPECIFIED:
		return 0, nil, nil
	}

	jobsTotal := 0
	jobInfos := make([]*orchestrator.JobInfo, 0)
	batchJobsTotal, batchJobInfos, err := s.processBatch(ctx, messageTypeStr, true)
	if err != nil {
		return jobsTotal, jobInfos, err
	}

	jobsTotal += batchJobsTotal
	jobInfos = append(jobInfos, batchJobInfos...)
	return jobsTotal, jobInfos, nil
}

// processBatch handles the common batch processing logic for querying hotel ID and publishing jobs.
// It returns the total number of jobs processed and any error encountered.
func (s *OrchestratorGRPCServer) processBatch(ctx context.Context, messageTypeStr string, collectJobInfos bool) (int, []*orchestrator.JobInfo, error) {
	batchSize := s.config.BatchSize
	if batchSize <= 0 {
		batchSize = 1000
	}
	batchDelay := time.Duration(s.config.BatchDelayMs) * time.Millisecond

	var lastHotelID int64 = 0
	jobsTotal := 0
	var jobInfos []*orchestrator.JobInfo
	if collectJobInfos {
		jobInfos = make([]*orchestrator.JobInfo, 0)
	}

	for {
		select {
		case <-ctx.Done():
			return jobsTotal, jobInfos, ctx.Err()
		default:
		}

		var (
			records             []database.IDWithHotelID
			missingTranslations []database.HotelMissingLang
			missingReviews      []database.IDWithHotelID
			err                 error
		)

		switch messageTypeStr {
		case constants.MessageTypeUpdateHotel:
			records, err = database.QueryHotelIDsByID(ctx, s.db, lastHotelID, batchSize)
		case constants.MessageTypeUpdateReview:
			records, err = database.QueryReviewIDsByID(ctx, s.db, lastHotelID, batchSize)
		case constants.MessageTypeUpdateTranslation:
			records, err = database.QueryTranslationIDsByID(ctx, s.db, lastHotelID, batchSize)
		case constants.MessageTypeFetchTranslation:
			missingTranslations, err = database.GetHotelsWithMissingTranslationsRaw(ctx, s.db, lastHotelID, batchSize)
		case constants.MessageTypeFetchReview:
			missingReviews, err = database.GetMissingReviewsFromHotelID(ctx, s.db, lastHotelID, batchSize)
		default:
			records, err = database.QueryHotelIDsByID(ctx, s.db, lastHotelID, batchSize)
		}

		if err != nil {
			return jobsTotal, jobInfos, err
		}

		var jobs []queue.Message

		if messageTypeStr == constants.MessageTypeFetchTranslation {
			if len(missingTranslations) == 0 {
				break
			}

			lastHotelID = missingTranslations[len(missingTranslations)-1].HotelID
			jobs = make([]queue.Message, 0, len(missingTranslations))

			for _, missingTranslation := range missingTranslations {
				messageID := fmt.Sprintf("%d_%s", missingTranslation.HotelID, missingTranslation.MissingLang)
				jobs = append(jobs, queue.Message{ID: messageID, Type: messageTypeStr, Data: map[string]any{
					constants2.HotelId: strconv.FormatInt(missingTranslation.HotelID, 10),
					constants2.Lang:    missingTranslation.MissingLang,
				}})

				if collectJobInfos {
					jobInfos = append(jobInfos, &orchestrator.JobInfo{
						HotelId:     int32(missingTranslation.HotelID),
						MessageType: orchestrator.MessageType_FETCH_MISSING_TRANSLATIONS,
						Status:      orchestrator.JobStatus_JOB_STATUS_PENDING,
					})
				}
			}
		} else if messageTypeStr == constants.MessageTypeFetchReview {
			if len(missingReviews) == 0 {
				break
			}

			lastHotelID = missingReviews[len(missingReviews)-1].HotelID
			jobs = make([]queue.Message, 0, len(missingReviews))

			for _, missingReview := range missingReviews {
				jobs = append(jobs, queue.Message{ID: missingReview.ID, Type: messageTypeStr, Data: map[string]any{
					constants2.HotelId: strconv.FormatInt(missingReview.HotelID, 10),
				}})

				if collectJobInfos {
					jobInfos = append(jobInfos, &orchestrator.JobInfo{
						HotelId:     int32(missingReview.HotelID),
						MessageType: orchestrator.MessageType_FETCH_MISSING_REVIEWS,
						Status:      orchestrator.JobStatus_JOB_STATUS_PENDING,
					})
				}
			}
		} else {
			if len(records) == 0 {
				break
			}

			lastHotelID = records[len(records)-1].HotelID
			jobs = make([]queue.Message, 0, len(records))

			for _, record := range records {
				jobs = append(jobs, queue.Message{ID: record.ID, Type: messageTypeStr, Data: map[string]any{
					constants2.HotelId: strconv.FormatInt(record.HotelID, 10),
				}})

				if collectJobInfos {
					var messageType orchestrator.MessageType
					switch messageTypeStr {
					case constants.MessageTypeUpdateHotel:
						messageType = orchestrator.MessageType_UPDATE_HOTEL
					case constants.MessageTypeUpdateReview:
						messageType = orchestrator.MessageType_UPDATE_REVIEW
					case constants.MessageTypeUpdateTranslation:
						messageType = orchestrator.MessageType_UPDATE_TRANSLATION
					case constants.MessageTypeFetchReview:
						messageType = orchestrator.MessageType_FETCH_MISSING_REVIEWS
					}
					jobInfos = append(jobInfos, &orchestrator.JobInfo{HotelId: int32(record.HotelID), MessageType: messageType, Status: orchestrator.JobStatus_JOB_STATUS_PENDING})
				}
			}
		}

		if err := s.rabbitMQPublisher.PublishWithRetry(ctx, jobs, s.config.MaxRetryAttempts); err != nil {
			return jobsTotal, jobInfos, err
		}
		jobsTotal += len(jobs)
		time.Sleep(batchDelay)
	}
	return jobsTotal, jobInfos, nil
}

// runOnce orchestrates hotel update processing and missing translations processing in batch mode, querying the database and publishing jobs to RabbitMQ.
func (s *OrchestratorGRPCServer) runOnce(ctx context.Context) {
	hotelJobsTotal, _, err := s.processBatch(ctx, constants.MessageTypeUpdateHotel, false)
	if err != nil {
		s.logger.Error("hotel batch processing failed", "error", err)
		return
	}

	translationJobsTotal, _, err := s.processBatch(ctx, constants.MessageTypeFetchTranslation, false)
	if err != nil {
		s.logger.Error("missing translations batch processing failed", "error", err)
		return
	}

	reviewJobsTotal, _, err := s.processBatch(ctx, constants.MessageTypeFetchReview, false)
	if err != nil {
		s.logger.Error("missing reviews batch processing failed", "error", err)
		return
	}

	totalJobs := hotelJobsTotal + translationJobsTotal + reviewJobsTotal
	if totalJobs > 0 {
		s.logger.Info("jobs published", "hotel_jobs", hotelJobsTotal, "translation_jobs", translationJobsTotal, "jobs_total", totalJobs)
	} else {
		s.logger.Info("no jobs published yet")
	}
}
