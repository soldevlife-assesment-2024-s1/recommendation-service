package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"recommendation-service/config"
	"recommendation-service/internal/module/recommendation/models/entity"
	"recommendation-service/internal/module/recommendation/models/response"
	"recommendation-service/internal/pkg/errors"

	"github.com/goccy/go-json"
	"github.com/jmoiron/sqlx"
	"github.com/redis/go-redis/v9"
	circuit "github.com/rubyist/circuitbreaker"
	"github.com/uptrace/opentelemetry-go-extra/otelzap"
)

type repositories struct {
	db               *sqlx.DB
	log              *otelzap.Logger
	httpClient       *circuit.HTTPClient
	cfgUserService   *config.UserServiceConfig
	cfgTicketService *config.TicketServiceConfig
	redisClient      *redis.Client
}

// FindVenues implements Repositories.
func (r *repositories) FindVenues(ctx context.Context) ([]entity.Venues, error) {
	var venues []entity.Venues
	err := r.db.SelectContext(ctx, &venues, "SELECT * FROM venues")
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return venues, nil
}

// FindTicketByRegionName implements Repositories.
func (r *repositories) FindTicketByRegionName(ctx context.Context, regionName string) ([]response.Ticket, error) {
	// call http to ticket service
	url := fmt.Sprintf("http://%s:%s/api/private/ticket?region_name=%s", r.cfgTicketService.Host, r.cfgTicketService.Port, regionName)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := r.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		r.log.Ctx(ctx).Error(fmt.Sprintf("Failed to get ticket", resp.StatusCode))
		return nil, errors.BadRequest("Failed to get ticket")
	}

	// parse response
	// var respData []response.Ticket

	// dec := json.NewDecoder(resp.Body)
	// if err := dec.Decode(&respData); err != nil {
	// 	return nil, err
	// }

	var respBase response.BaseResponse

	dec := json.NewDecoder(resp.Body)
	if err := dec.Decode(&respBase); err != nil {
		return nil, err
	}

	jsonData, err := json.Marshal(respBase.Data)
	if err != nil {
		return nil, err
	}

	var dataTickets []response.Ticket

	err = json.Unmarshal(jsonData, &dataTickets)
	if err != nil {
		return nil, err
	}

	return dataTickets, nil
}

// FindUserProfile implements Repositories.
func (r *repositories) FindUserProfile(ctx context.Context, userID int64) (response.UserProfile, error) {
	// http call to user service
	url := fmt.Sprintf("http://%s:%s/api/private/user/profile?user_id=%d", r.cfgUserService.Host, r.cfgUserService.Port, userID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return response.UserProfile{}, err
	}
	resp, err := r.httpClient.Do(req)
	if err != nil {
		return response.UserProfile{}, err
	}

	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		r.log.Ctx(ctx).Error(fmt.Sprintf("Failed to get user profile", resp.StatusCode))
		return response.UserProfile{}, errors.BadRequest("Failed to get user profile")
	}

	// parse response
	// var respData response.UserProfile

	// dec := json.NewDecoder(resp.Body)
	// if err := dec.Decode(&respData); err != nil {
	// 	return response.UserProfile{}, err
	// }

	var respBase response.BaseResponse

	dec := json.NewDecoder(resp.Body)
	if err := dec.Decode(&respBase); err != nil {
		return response.UserProfile{}, err
	}

	var dataProfile response.GetProfileResponse

	byteJson, err := json.Marshal(respBase.Data)
	if err != nil {
		return response.UserProfile{}, err
	}

	err = json.Unmarshal(byteJson, &dataProfile)
	if err != nil {
		return response.UserProfile{}, err
	}

	respData := response.UserProfile{
		UserID:   dataProfile.UserID,
		Username: dataProfile.FirstName + " " + dataProfile.LastName,
		Region:   dataProfile.Region,
	}

	return respData, nil
}

// FindVenueByName implements Repositories.
func (r *repositories) FindVenueByName(ctx context.Context, name string) (entity.Venues, error) {
	var venue entity.Venues
	err := r.db.GetContext(ctx, &venue, "SELECT * FROM venues WHERE name = $1", name)

	if err == sql.ErrNoRows {
		return entity.Venues{}, nil
	}
	if err != nil {
		return entity.Venues{}, err
	}
	return venue, nil
}

// UpsertVenue implements Repositories.
func (r *repositories) UpsertVenue(ctx context.Context, payload entity.Venues) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			err = tx.Rollback()
			if err != nil {
				r.log.Ctx(ctx).Error(fmt.Sprintf("Failed to rollback transaction: %v", err))
			}
			return
		}
		err = tx.Commit()
		if err != nil {
			r.log.Ctx(ctx).Error(fmt.Sprintf("Failed to commit transaction: %v", err))
			return
		}
	}()

	// Check if the venue already exists
	var existingVenueID int64
	err = tx.QueryRowContext(ctx, "SELECT id FROM venues WHERE name = $1 FOR UPDATE", payload.Name).Scan(&existingVenueID)
	if err != nil && err != sql.ErrNoRows {
		return err
	}

	if err == sql.ErrNoRows {
		// Venue does not exist, insert a new record
		_, err = tx.ExecContext(ctx, "INSERT INTO venues (name, is_sold_out, is_first_sold_out) VALUES ($1, $2, $3)", payload.Name, payload.IsSoldOut, payload.IsFirstSoldOut)
		if err != nil {
			return err
		}
	} else {
		// Venue already exists, update the existing record
		_, err = tx.ExecContext(ctx, "UPDATE venues SET is_sold_out = $1, SET is_sold_out_first = $2 WHERE id = $3", payload.IsSoldOut, payload.IsFirstSoldOut, existingVenueID)
		if err != nil {
			return err
		}
	}

	return nil
}

type Repositories interface {
	// http
	ValidateToken(ctx context.Context, token string) (response.UserServiceValidate, error)
	FindUserProfile(ctx context.Context, userID int64) (response.UserProfile, error)
	FindTicketByRegionName(ctx context.Context, regionName string) ([]response.Ticket, error)
	// db
	UpsertVenue(ctx context.Context, payload entity.Venues) error
	FindVenueByName(ctx context.Context, name string) (entity.Venues, error)
	FindVenues(ctx context.Context) ([]entity.Venues, error)
}

func New(db *sqlx.DB, log *otelzap.Logger, httpClient *circuit.HTTPClient, redisClient *redis.Client, userService *config.UserServiceConfig, ticketService *config.TicketServiceConfig) Repositories {
	return &repositories{
		db:               db,
		log:              log,
		httpClient:       httpClient,
		redisClient:      redisClient,
		cfgUserService:   userService,
		cfgTicketService: ticketService,
	}
}

func (r *repositories) ValidateToken(ctx context.Context, token string) (response.UserServiceValidate, error) {
	// http call to user service
	url := fmt.Sprintf("http://%s:%s/api/private/user/validate?token=%s", r.cfgUserService.Host, r.cfgUserService.Port, token)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return response.UserServiceValidate{}, err
	}
	resp, err := r.httpClient.Do(req)
	if err != nil {
		return response.UserServiceValidate{}, err
	}

	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		r.log.Ctx(ctx).Error(fmt.Sprintf("Invalid token", resp.StatusCode))
		return response.UserServiceValidate{}, errors.BadRequest("Invalid token")
	}

	// parse response
	var respBase response.BaseResponse

	dec := json.NewDecoder(resp.Body)
	if err := dec.Decode(&respBase); err != nil {
		return response.UserServiceValidate{
			IsValid: false,
			UserID:  0,
		}, err
	}

	respBase.Data = respBase.Data.(map[string]interface{})
	respData := response.UserServiceValidate{
		IsValid:   respBase.Data.(map[string]interface{})["is_valid"].(bool),
		UserID:    int64(respBase.Data.(map[string]interface{})["user_id"].(float64)),
		EmailUser: respBase.Data.(map[string]interface{})["email_user"].(string),
	}

	if !respData.IsValid {
		r.log.Ctx(ctx).Error("Invalid token")
		return response.UserServiceValidate{
			IsValid: false,
			UserID:  0,
		}, errors.BadRequest("Invalid token")
	}

	// validate token
	return response.UserServiceValidate{
		IsValid:   respData.IsValid,
		UserID:    respData.UserID,
		EmailUser: respData.EmailUser,
	}, nil
}
