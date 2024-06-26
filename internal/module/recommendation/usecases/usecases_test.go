package usecases_test

import (
	"context"
	"database/sql"
	"recommendation-service/internal/module/recommendation/mocks"
	"recommendation-service/internal/module/recommendation/models/entity"
	"recommendation-service/internal/module/recommendation/models/request"
	"recommendation-service/internal/module/recommendation/models/response"
	"recommendation-service/internal/module/recommendation/usecases"
	"recommendation-service/internal/pkg/gorules"
	"testing"
	"time"

	"github.com/gorules/zen-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var (
	repoMock *mocks.Repositories
	uc       usecases.Usecases
	z        zen.Decision
)

func setup() {
	repoMock = new(mocks.Repositories)
	// init business rules engine
	pathTicketDiscounted := "../../../../assets/ticket-discounted.json"
	z, _ = gorules.Init(pathTicketDiscounted)
	uc = usecases.New(repoMock, z)
}

func teardown() {
	repoMock = nil
	uc = nil
	z = nil
}

func TestGetOnlineTicket(t *testing.T) {
	setup()
	defer teardown()

	t.Run("success", func(t *testing.T) {
		// mock
		repoMock.On("FindVenueByName", mock.Anything, mock.Anything).Return(entity.Venues{}, nil)

		_, err := uc.GetOnlineTicket(context.Background(), "Jakarta")
		assert.NoError(t, err)
	})
}

func TestUpdateTicketSoldOut(t *testing.T) {
	setup()
	defer teardown()

	t.Run("success", func(t *testing.T) {
		// mock
		payload := request.TicketSoldOut{
			VenueName: "Jakarta",
			IsSoldOut: true,
		}

		entityMock := entity.Venues{
			ID:             1,
			Name:           payload.VenueName,
			IsSoldOut:      payload.IsSoldOut,
			IsFirstSoldOut: false,
			CreatedAt:      time.Time{},
			UpdatedAt:      sql.NullTime{},
			DeletedAt:      sql.NullTime{},
		}
		entitiesMock := []entity.Venues{
			{
				ID:             1,
				Name:           payload.VenueName,
				IsSoldOut:      payload.IsSoldOut,
				IsFirstSoldOut: false,
				CreatedAt:      time.Time{},
				UpdatedAt:      sql.NullTime{},
				DeletedAt:      sql.NullTime{},
			},
		}

		repoMock.On("FindVenueByName", mock.Anything, mock.Anything).Return(entityMock, nil)
		repoMock.On("FindVenues", mock.Anything).Return(entitiesMock, nil)
		repoMock.On("UpsertVenue", mock.Anything, mock.Anything).Return(nil)

		err := uc.UpdateTicketSoldOut(context.Background(), &payload)
		assert.NoError(t, err)
	})
}

func TestGetRecommendation(t *testing.T) {
	setup()
	defer teardown()

	t.Run("success", func(t *testing.T) {
		// mock data
		profileMock := response.UserProfile{
			UserID:   1,
			Username: "user",
			Region:   "Jakarta",
		}
		venueMock := entity.Venues{
			ID:             1,
			Name:           "Jakarta",
			IsSoldOut:      false,
			IsFirstSoldOut: false,
			CreatedAt:      time.Time{},
			UpdatedAt:      sql.NullTime{},
			DeletedAt:      sql.NullTime{},
		}
		ticketsMock := []response.Ticket{
			{
				RegionName: "Jakarta",
				Level:      "Online",
				Price:      100000,
			},
		}

		// mock repo
		repoMock.On("FindUserProfile", mock.Anything, mock.Anything).Return(profileMock, nil)
		repoMock.On("FindVenueByName", mock.Anything, mock.Anything).Return(venueMock, nil)
		repoMock.On("FindTicketByRegionName", mock.Anything, mock.Anything).Return(ticketsMock, nil)

		result, err := uc.GetRecommendation(context.Background(), 1)
		assert.NoError(t, err)
		assert.NotNil(t, result)
	})
}

func TestUpdateVenueStatus(t *testing.T) {
	setup()
	defer teardown()

	t.Run("success", func(t *testing.T) {
		// mock
		payload := request.UpdateVenueStatus{
			VenueName: "Jakarta",
			IsSoldOut: true,
		}

		entityMock := entity.Venues{
			ID:             1,
			Name:           payload.VenueName,
			IsSoldOut:      payload.IsSoldOut,
			IsFirstSoldOut: false,
			CreatedAt:      time.Time{},
			UpdatedAt:      sql.NullTime{},
			DeletedAt:      sql.NullTime{},
		}
		entitiesMock := []entity.Venues{
			{
				ID:             1,
				Name:           payload.VenueName,
				IsSoldOut:      payload.IsSoldOut,
				IsFirstSoldOut: false,
				CreatedAt:      time.Time{},
				UpdatedAt:      sql.NullTime{},
				DeletedAt:      sql.NullTime{},
			},
		}

		repoMock.On("FindVenueByName", mock.Anything, mock.Anything).Return(entityMock, nil)
		repoMock.On("FindVenues", mock.Anything).Return(entitiesMock, nil)
		repoMock.On("UpsertVenue", mock.Anything, mock.Anything).Return(nil)

		err := uc.UpdateVenueStatus(context.Background(), &payload)
		assert.NoError(t, err)
	})
}
