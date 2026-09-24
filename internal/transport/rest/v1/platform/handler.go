package platform

import (
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/bookly-kbtu/backend/internal/domain"
	"github.com/bookly-kbtu/backend/internal/pkg/ctxuser"
	"github.com/bookly-kbtu/backend/internal/transport/rest/v1/response"
	importeruc "github.com/bookly-kbtu/backend/internal/usecase/importer"
	platformuc "github.com/bookly-kbtu/backend/internal/usecase/platform"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type Handler struct {
	uc       *platformuc.Service
	importer *importeruc.Service
	importMu sync.Mutex
}

func New(uc *platformuc.Service, importer *importeruc.Service) *Handler {
	return &Handler{uc: uc, importer: importer}
}

// PatchClientProfileRequest does not carry the avatar: it is uploaded via /users/me/avatar.
type PatchClientProfileRequest struct {
	FirstName *string `json:"first_name"`
	LastName  *string `json:"last_name"`
}

type ClientProfileResponse struct {
	UserID    uuid.UUID `json:"user_id"`
	Phone     *string   `json:"phone"`
	FirstName *string   `json:"first_name"`
	LastName  *string   `json:"last_name"`
	AvatarURL *string   `json:"avatar_url"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// UpsertMasterProfileRequest does not carry the avatar: it is uploaded via /masters/me/avatar.
type UpsertMasterProfileRequest struct {
	DisplayName string  `json:"display_name"`
	Description *string `json:"description"`
}

type MasterProfileResponse struct {
	UserID      uuid.UUID `json:"user_id"`
	DisplayName string    `json:"display_name"`
	Description *string   `json:"description"`
	AvatarURL   *string   `json:"avatar_url"`
	IsActive    bool      `json:"is_active"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type LocationRequest struct {
	Name        string   `json:"name"`
	AddressText string   `json:"address_text"`
	Latitude    *float64 `json:"latitude"`
	Longitude   *float64 `json:"longitude"`
	TimeZone    string   `json:"time_zone"`
	IsPrimary   bool     `json:"is_primary"`
}

type LocationResponse struct {
	ID          uuid.UUID `json:"id"`
	MasterID    uuid.UUID `json:"master_id"`
	Name        string    `json:"name"`
	AddressText string    `json:"address_text"`
	Latitude    *float64  `json:"latitude"`
	Longitude   *float64  `json:"longitude"`
	TimeZone    string    `json:"time_zone"`
	IsPrimary   bool      `json:"is_primary"`
	IsActive    bool      `json:"is_active"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type ServiceRequest struct {
	CategoryID      uuid.UUID `json:"category_id"`
	Name            string    `json:"name"`
	Description     *string   `json:"description"`
	PriceAmount     int64     `json:"price_amount"`
	Currency        string    `json:"currency"`
	DurationMinutes int       `json:"duration_minutes"`
	IsActive        bool      `json:"is_active"`
}

type CategoryResponse struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Slug      string    `json:"slug"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type ServiceResponse struct {
	ID              uuid.UUID `json:"id"`
	MasterID        uuid.UUID `json:"master_id"`
	CategoryID      uuid.UUID `json:"category_id"`
	CategoryName    string    `json:"category_name"`
	Name            string    `json:"name"`
	Description     *string   `json:"description"`
	PriceAmount     int64     `json:"price_amount"`
	Currency        string    `json:"currency"`
	DurationMinutes int       `json:"duration_minutes"`
	IsActive        bool      `json:"is_active"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type WorkingHourRequest struct {
	LocationID uuid.UUID `json:"location_id"`
	Weekday    int       `json:"weekday"`
	StartTime  string    `json:"start_time"`
	EndTime    string    `json:"end_time"`
}

type ReplaceWorkingHoursRequest struct {
	Items []WorkingHourRequest `json:"items"`
}

type WorkingHourResponse struct {
	ID         uuid.UUID `json:"id"`
	MasterID   uuid.UUID `json:"master_id"`
	LocationID uuid.UUID `json:"location_id"`
	Weekday    int       `json:"weekday"`
	StartTime  string    `json:"start_time"`
	EndTime    string    `json:"end_time"`
	IsActive   bool      `json:"is_active"`
}

type ScheduleExceptionRequest struct {
	LocationID     *uuid.UUID `json:"location_id"`
	ExceptionKind  string     `json:"exception_kind"`
	StartsOn       time.Time  `json:"starts_on"`
	EndsOn         time.Time  `json:"ends_on"`
	LocalStartTime *string    `json:"local_start_time"`
	LocalEndTime   *string    `json:"local_end_time"`
	Reason         *string    `json:"reason"`
}

type ScheduleExceptionResponse struct {
	ID             uuid.UUID  `json:"id"`
	MasterID       uuid.UUID  `json:"master_id"`
	LocationID     *uuid.UUID `json:"location_id"`
	ExceptionKind  string     `json:"exception_kind"`
	StartsOn       time.Time  `json:"starts_on"`
	EndsOn         time.Time  `json:"ends_on"`
	LocalStartTime *string    `json:"local_start_time"`
	LocalEndTime   *string    `json:"local_end_time"`
	Reason         *string    `json:"reason"`
}

type CreateBookingRequest struct {
	MasterID         uuid.UUID `json:"master_id"`
	MasterServiceID  uuid.UUID `json:"master_service_id"`
	MasterLocationID uuid.UUID `json:"master_location_id"`
	StartsAt         time.Time `json:"starts_at"`
	ClientComment    *string   `json:"client_comment"`
}

type BookingResponse struct {
	ID                  uuid.UUID `json:"id"`
	ClientID            uuid.UUID `json:"client_id"`
	MasterID            uuid.UUID `json:"master_id"`
	MasterServiceID     uuid.UUID `json:"master_service_id"`
	MasterLocationID    uuid.UUID `json:"master_location_id"`
	StartsAt            time.Time `json:"starts_at"`
	EndsAt              time.Time `json:"ends_at"`
	Status              string    `json:"status"`
	ServiceNameSnapshot string    `json:"service_name_snapshot"`
	PriceAmount         int64     `json:"price_amount"`
	Currency            string    `json:"currency"`
	DurationMinutes     int       `json:"duration_minutes"`
	ClientComment       *string   `json:"client_comment"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}

type AdminUserResponse struct {
	ID        uuid.UUID `json:"id"`
	Phone     *string   `json:"phone"`
	Status    string    `json:"status"`
	Roles     []string  `json:"roles"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type UpdateUserStatusRequest struct {
	Status string `json:"status" enums:"active,blocked,deleted"`
}

type ReplaceUserRolesRequest struct {
	Roles []domain.Role `json:"roles" swaggertype:"array,string" example:"client"`
}

type ImportRunResponse struct {
	ID                uuid.UUID  `json:"id"`
	SourceID          uuid.UUID  `json:"source_id"`
	SourceCode        string     `json:"source_code"`
	Status            string     `json:"status"`
	Params            any        `json:"params"`
	StartedAt         time.Time  `json:"started_at"`
	FinishedAt        *time.Time `json:"finished_at"`
	CitiesProcessed   int        `json:"cities_processed"`
	FirmsProcessed    int        `json:"firms_processed"`
	MastersProcessed  int        `json:"masters_processed"`
	ServicesProcessed int        `json:"services_processed"`
	ErrorsCount       int        `json:"errors_count"`
	ErrorSummary      *string    `json:"error_summary"`
}

type MarketStatsResponse struct {
	CitiesCount   int `json:"cities_count"`
	FirmsCount    int `json:"firms_count"`
	ServicesCount int `json:"services_count"`
	MastersCount  int `json:"masters_count"`
}

func (h *Handler) Register(api fiber.Router, requireAuth fiber.Handler, requireAdmin fiber.Handler) {
	api.Get("/categories", h.listCategories)
	api.Get("/users/me/profile", requireAuth, h.getClientProfile)
	api.Patch("/users/me/profile", requireAuth, h.patchClientProfile)

	// Before /masters/me: the group middleware would otherwise also match these routes.
	h.registerMedia(api, requireAuth)

	masters := api.Group("/masters/me", requireAuth)
	masters.Post("/profile", h.upsertMasterProfile)
	masters.Get("/profile", h.getMasterProfile)
	masters.Patch("/profile", h.upsertMasterProfile)
	masters.Get("/locations", h.listLocations)
	masters.Post("/locations", h.createLocation)
	masters.Patch("/locations/:id", h.updateLocation)
	masters.Delete("/locations/:id", h.deleteLocation)
	masters.Get("/services", h.listServices)
	masters.Post("/services", h.createService)
	masters.Patch("/services/:id", h.updateService)
	masters.Delete("/services/:id", h.deleteService)
	masters.Get("/working-hours", h.listWorkingHours)
	masters.Put("/working-hours", h.replaceWorkingHours)
	masters.Get("/schedule-exceptions", h.listScheduleExceptions)
	masters.Post("/schedule-exceptions", h.createScheduleException)
	masters.Delete("/schedule-exceptions/:id", h.deleteScheduleException)
	masters.Get("/bookings", h.listMasterBookings)
	masters.Post("/bookings/:id/confirm", h.confirmBooking)
	masters.Post("/bookings/:id/complete", h.completeBooking)
	masters.Post("/bookings/:id/cancel", h.cancelBookingByMaster)
	masters.Post("/bookings/:id/no-show", h.noShowBooking)

	api.Post("/bookings", requireAuth, h.createBooking)
	api.Get("/bookings/my", requireAuth, h.listClientBookings)
	api.Get("/bookings/:id", requireAuth, h.getBooking)
	api.Post("/bookings/:id/cancel", requireAuth, h.cancelBookingByClient)

	api.Get("/market/stats", h.marketStats)
	api.Get("/admin/users", requireAuth, requireAdmin, h.adminUsers)
	api.Patch("/admin/users/:id/status", requireAuth, requireAdmin, h.adminUpdateUserStatus)
	api.Put("/admin/users/:id/roles", requireAuth, requireAdmin, h.adminReplaceUserRoles)
	api.Get("/admin/import/runs", requireAuth, requireAdmin, h.adminImportRuns)
	h.registerRemaining(api, requireAuth, requireAdmin)
}

// getClientProfile godoc
//
//	@Summary	Current client profile
//	@Tags		users
//	@Security	BearerAuth
//	@Produce	json
//	@Success	200	{object}	ClientProfileResponse
//	@Failure	401	{object}	response.ErrorBody
//	@Router		/users/me/profile [get]
func (h *Handler) getClientProfile(c *fiber.Ctx) error {
	user := mustUser(c)
	profile, err := h.uc.GetClientProfile(c.UserContext(), user.ID)
	if err != nil {
		return err
	}
	return response.OK(c, clientProfileResponse(*profile))
}

// patchClientProfile godoc
//
//	@Summary	Update current client profile
//	@Tags		users
//	@Security	BearerAuth
//	@Accept		json
//	@Produce	json
//	@Param		body	body		PatchClientProfileRequest	true	"profile patch"
//	@Success	200		{object}	ClientProfileResponse
//	@Failure	400		{object}	response.ErrorBody
//	@Failure	401		{object}	response.ErrorBody
//	@Router		/users/me/profile [patch]
func (h *Handler) patchClientProfile(c *fiber.Ctx) error {
	user := mustUser(c)
	var req PatchClientProfileRequest
	if err := parseBody(c, &req); err != nil {
		return err
	}
	profile, err := h.uc.PatchClientProfile(c.UserContext(), user.ID, domain.PatchClientProfileInput(req))
	if err != nil {
		return err
	}
	return response.OK(c, clientProfileResponse(*profile))
}

// listCategories godoc
//
//	@Summary	List Bookly service categories
//	@Tags		categories
//	@Produce	json
//	@Success	200	{array}		CategoryResponse
//	@Failure	500	{object}	response.ErrorBody
//	@Router		/categories [get]
func (h *Handler) listCategories(c *fiber.Ctx) error {
	items, err := h.uc.ListCategories(c.UserContext())
	if err != nil {
		return err
	}
	out := make([]CategoryResponse, 0, len(items))
	for _, item := range items {
		out = append(out, categoryResponse(item))
	}
	return response.OK(c, out)
}

// upsertMasterProfile godoc
//
//	@Summary	Create or update current master profile
//	@Tags		master-cabinet
//	@Security	BearerAuth
//	@Accept		json
//	@Produce	json
//	@Param		body	body		UpsertMasterProfileRequest	true	"master profile"
//	@Success	200		{object}	MasterProfileResponse
//	@Failure	400		{object}	response.ErrorBody
//	@Failure	401		{object}	response.ErrorBody
//	@Router		/masters/me/profile [post]
//	@Router		/masters/me/profile [patch]
func (h *Handler) upsertMasterProfile(c *fiber.Ctx) error {
	user := mustUser(c)
	var req UpsertMasterProfileRequest
	if err := parseBody(c, &req); err != nil {
		return err
	}
	profile, err := h.uc.UpsertMasterProfile(c.UserContext(), user.ID, domain.UpsertMasterProfileInput(req))
	if err != nil {
		return err
	}
	return response.OK(c, masterProfileResponse(*profile))
}

// getMasterProfile godoc
//
//	@Summary	Current master profile
//	@Tags		master-cabinet
//	@Security	BearerAuth
//	@Produce	json
//	@Success	200	{object}	MasterProfileResponse
//	@Failure	401	{object}	response.ErrorBody
//	@Failure	404	{object}	response.ErrorBody
//	@Router		/masters/me/profile [get]
func (h *Handler) getMasterProfile(c *fiber.Ctx) error {
	user := mustUser(c)
	profile, err := h.uc.GetMasterProfile(c.UserContext(), user.ID)
	if err != nil {
		return err
	}
	return response.OK(c, masterProfileResponse(*profile))
}

// listLocations godoc
//
//	@Summary	List current master locations
//	@Tags		master-cabinet
//	@Security	BearerAuth
//	@Produce	json
//	@Success	200	{array}	LocationResponse
//	@Router		/masters/me/locations [get]
func (h *Handler) listLocations(c *fiber.Ctx) error {
	user := mustUser(c)
	items, err := h.uc.ListMasterLocations(c.UserContext(), user.ID)
	if err != nil {
		return err
	}
	out := make([]LocationResponse, 0, len(items))
	for _, item := range items {
		out = append(out, locationResponse(item))
	}
	return response.OK(c, out)
}

// createLocation godoc
//
//	@Summary	Create current master location
//	@Tags		master-cabinet
//	@Security	BearerAuth
//	@Accept		json
//	@Produce	json
//	@Param		body	body		LocationRequest	true	"location"
//	@Success	201		{object}	LocationResponse
//	@Router		/masters/me/locations [post]
func (h *Handler) createLocation(c *fiber.Ctx) error {
	user := mustUser(c)
	var req LocationRequest
	if err := parseBody(c, &req); err != nil {
		return err
	}
	item, err := h.uc.CreateMasterLocation(c.UserContext(), user.ID, locationInput(req))
	if err != nil {
		return err
	}
	return response.Created(c, locationResponse(*item))
}

// updateLocation godoc
//
//	@Summary	Update current master location
//	@Tags		master-cabinet
//	@Security	BearerAuth
//	@Accept		json
//	@Produce	json
//	@Param		id		path		string			true	"location UUID"
//	@Param		body	body		LocationRequest	true	"location"
//	@Success	200		{object}	LocationResponse
//	@Router		/masters/me/locations/{id} [patch]
func (h *Handler) updateLocation(c *fiber.Ctx) error {
	user := mustUser(c)
	id, err := paramUUID(c, "id")
	if err != nil {
		return err
	}
	var req LocationRequest
	if err := parseBody(c, &req); err != nil {
		return err
	}
	item, err := h.uc.UpdateMasterLocation(c.UserContext(), user.ID, id, locationInput(req))
	if err != nil {
		return err
	}
	return response.OK(c, locationResponse(*item))
}

// deleteLocation godoc
//
//	@Summary	Disable current master location
//	@Tags		master-cabinet
//	@Security	BearerAuth
//	@Param		id	path	string	true	"location UUID"
//	@Success	204
//	@Router		/masters/me/locations/{id} [delete]
func (h *Handler) deleteLocation(c *fiber.Ctx) error {
	user := mustUser(c)
	id, err := paramUUID(c, "id")
	if err != nil {
		return err
	}
	if err := h.uc.DisableMasterLocation(c.UserContext(), user.ID, id); err != nil {
		return err
	}
	return c.SendStatus(fiber.StatusNoContent)
}

// listServices godoc
//
//	@Summary	List current master services
//	@Tags		master-cabinet
//	@Security	BearerAuth
//	@Produce	json
//	@Success	200	{array}	ServiceResponse
//	@Router		/masters/me/services [get]
func (h *Handler) listServices(c *fiber.Ctx) error {
	user := mustUser(c)
	items, err := h.uc.ListMasterServices(c.UserContext(), user.ID, false)
	if err != nil {
		return err
	}
	out := make([]ServiceResponse, 0, len(items))
	for _, item := range items {
		out = append(out, serviceResponse(item))
	}
	return response.OK(c, out)
}

// createService godoc
//
//	@Summary	Create current master service
//	@Tags		master-cabinet
//	@Security	BearerAuth
//	@Accept		json
//	@Produce	json
//	@Param		body	body		ServiceRequest	true	"service"
//	@Success	201		{object}	ServiceResponse
//	@Router		/masters/me/services [post]
func (h *Handler) createService(c *fiber.Ctx) error {
	user := mustUser(c)
	var req ServiceRequest
	if err := parseBody(c, &req); err != nil {
		return err
	}
	item, err := h.uc.CreateMasterService(c.UserContext(), user.ID, serviceInput(req))
	if err != nil {
		return err
	}
	return response.Created(c, serviceResponse(*item))
}

// updateService godoc
//
//	@Summary	Update current master service
//	@Tags		master-cabinet
//	@Security	BearerAuth
//	@Accept		json
//	@Produce	json
//	@Param		id		path		string			true	"service UUID"
//	@Param		body	body		ServiceRequest	true	"service"
//	@Success	200		{object}	ServiceResponse
//	@Router		/masters/me/services/{id} [patch]
func (h *Handler) updateService(c *fiber.Ctx) error {
	user := mustUser(c)
	id, err := paramUUID(c, "id")
	if err != nil {
		return err
	}
	var req ServiceRequest
	if err := parseBody(c, &req); err != nil {
		return err
	}
	item, err := h.uc.UpdateMasterService(c.UserContext(), user.ID, id, serviceInput(req))
	if err != nil {
		return err
	}
	return response.OK(c, serviceResponse(*item))
}

// deleteService godoc
//
//	@Summary	Disable current master service
//	@Tags		master-cabinet
//	@Security	BearerAuth
//	@Param		id	path	string	true	"service UUID"
//	@Success	204
//	@Router		/masters/me/services/{id} [delete]
func (h *Handler) deleteService(c *fiber.Ctx) error {
	user := mustUser(c)
	id, err := paramUUID(c, "id")
	if err != nil {
		return err
	}
	if err := h.uc.DisableMasterService(c.UserContext(), user.ID, id); err != nil {
		return err
	}
	return c.SendStatus(fiber.StatusNoContent)
}

// listWorkingHours godoc
//
//	@Summary	List current master working hours
//	@Tags		master-cabinet
//	@Security	BearerAuth
//	@Produce	json
//	@Success	200	{array}	WorkingHourResponse
//	@Router		/masters/me/working-hours [get]
func (h *Handler) listWorkingHours(c *fiber.Ctx) error {
	user := mustUser(c)
	items, err := h.uc.ListWorkingHours(c.UserContext(), user.ID)
	if err != nil {
		return err
	}
	out := make([]WorkingHourResponse, 0, len(items))
	for _, item := range items {
		out = append(out, workingHourResponse(item))
	}
	return response.OK(c, out)
}

// replaceWorkingHours godoc
//
//	@Summary	Replace current master working hours
//	@Tags		master-cabinet
//	@Security	BearerAuth
//	@Accept		json
//	@Produce	json
//	@Param		body	body	ReplaceWorkingHoursRequest	true	"working hours"
//	@Success	200		{array}	WorkingHourResponse
//	@Router		/masters/me/working-hours [put]
func (h *Handler) replaceWorkingHours(c *fiber.Ctx) error {
	user := mustUser(c)
	var req ReplaceWorkingHoursRequest
	if err := parseBody(c, &req); err != nil {
		return err
	}
	items := make([]domain.WorkingHourInput, 0, len(req.Items))
	for _, item := range req.Items {
		items = append(items, domain.WorkingHourInput(item))
	}
	outItems, err := h.uc.ReplaceWorkingHours(c.UserContext(), user.ID, items)
	if err != nil {
		return err
	}
	out := make([]WorkingHourResponse, 0, len(outItems))
	for _, item := range outItems {
		out = append(out, workingHourResponse(item))
	}
	return response.OK(c, out)
}

// listScheduleExceptions godoc
//
//	@Summary	List current master schedule exceptions
//	@Tags		master-cabinet
//	@Security	BearerAuth
//	@Produce	json
//	@Success	200	{array}	ScheduleExceptionResponse
//	@Router		/masters/me/schedule-exceptions [get]
func (h *Handler) listScheduleExceptions(c *fiber.Ctx) error {
	user := mustUser(c)
	items, err := h.uc.ListScheduleExceptions(c.UserContext(), user.ID)
	if err != nil {
		return err
	}
	out := make([]ScheduleExceptionResponse, 0, len(items))
	for _, item := range items {
		out = append(out, scheduleExceptionResponse(item))
	}
	return response.OK(c, out)
}

// createScheduleException godoc
//
//	@Summary	Create current master schedule exception
//	@Tags		master-cabinet
//	@Security	BearerAuth
//	@Accept		json
//	@Produce	json
//	@Param		body	body		ScheduleExceptionRequest	true	"exception"
//	@Success	201		{object}	ScheduleExceptionResponse
//	@Router		/masters/me/schedule-exceptions [post]
func (h *Handler) createScheduleException(c *fiber.Ctx) error {
	user := mustUser(c)
	var req ScheduleExceptionRequest
	if err := parseBody(c, &req); err != nil {
		return err
	}
	item, err := h.uc.CreateScheduleException(c.UserContext(), user.ID, domain.ScheduleExceptionInput(req))
	if err != nil {
		return err
	}
	return response.Created(c, scheduleExceptionResponse(*item))
}

// deleteScheduleException godoc
//
//	@Summary	Delete current master schedule exception
//	@Tags		master-cabinet
//	@Security	BearerAuth
//	@Param		id	path	string	true	"exception UUID"
//	@Success	204
//	@Router		/masters/me/schedule-exceptions/{id} [delete]
func (h *Handler) deleteScheduleException(c *fiber.Ctx) error {
	user := mustUser(c)
	id, err := paramUUID(c, "id")
	if err != nil {
		return err
	}
	if err := h.uc.DeleteScheduleException(c.UserContext(), user.ID, id); err != nil {
		return err
	}
	return c.SendStatus(fiber.StatusNoContent)
}

// createBooking godoc
//
//	@Summary	Create booking
//	@Tags		bookings
//	@Security	BearerAuth
//	@Accept		json
//	@Produce	json
//	@Param		body	body		CreateBookingRequest	true	"booking"
//	@Success	201		{object}	BookingResponse
//	@Failure	409		{object}	response.ErrorBody
//	@Router		/bookings [post]
func (h *Handler) createBooking(c *fiber.Ctx) error {
	user := mustUser(c)
	var req CreateBookingRequest
	if err := parseBody(c, &req); err != nil {
		return err
	}
	item, err := h.uc.CreateBooking(c.UserContext(), user.ID, domain.CreateBookingInput(req))
	if err != nil {
		return err
	}
	return response.Created(c, bookingResponse(*item))
}

// listClientBookings godoc
//
//	@Summary	List my bookings
//	@Tags		bookings
//	@Security	BearerAuth
//	@Produce	json
//	@Success	200	{array}	BookingResponse
//	@Router		/bookings/my [get]
func (h *Handler) listClientBookings(c *fiber.Ctx) error {
	user := mustUser(c)
	items, err := h.uc.ListClientBookings(c.UserContext(), user.ID)
	if err != nil {
		return err
	}
	return response.OK(c, bookingResponses(items))
}

// getBooking godoc
//
//	@Summary	Get booking visible to current user
//	@Tags		bookings
//	@Security	BearerAuth
//	@Produce	json
//	@Param		id	path		string	true	"booking UUID"
//	@Success	200	{object}	BookingResponse
//	@Router		/bookings/{id} [get]
func (h *Handler) getBooking(c *fiber.Ctx) error {
	user := mustUser(c)
	id, err := paramUUID(c, "id")
	if err != nil {
		return err
	}
	item, err := h.uc.GetBookingForUser(c.UserContext(), id, user.ID)
	if err != nil {
		return err
	}
	return response.OK(c, bookingResponse(*item))
}

// cancelBookingByClient godoc
//
//	@Summary	Cancel my booking as client
//	@Tags		bookings
//	@Security	BearerAuth
//	@Param		id	path		string	true	"booking UUID"
//	@Success	200	{object}	BookingResponse
//	@Router		/bookings/{id}/cancel [post]
func (h *Handler) cancelBookingByClient(c *fiber.Ctx) error {
	return h.transition(c, domain.BookingCancelledByClient)
}

// listMasterBookings godoc
//
//	@Summary	List bookings for current master
//	@Tags		master-cabinet
//	@Security	BearerAuth
//	@Produce	json
//	@Success	200	{array}	BookingResponse
//	@Router		/masters/me/bookings [get]
func (h *Handler) listMasterBookings(c *fiber.Ctx) error {
	user := mustUser(c)
	items, err := h.uc.ListMasterBookings(c.UserContext(), user.ID)
	if err != nil {
		return err
	}
	return response.OK(c, bookingResponses(items))
}

// confirmBooking godoc
//
//	@Summary	Confirm booking as master
//	@Tags		master-cabinet
//	@Security	BearerAuth
//	@Param		id	path		string	true	"booking UUID"
//	@Success	200	{object}	BookingResponse
//	@Router		/masters/me/bookings/{id}/confirm [post]
func (h *Handler) confirmBooking(c *fiber.Ctx) error { return h.transition(c, domain.BookingConfirmed) }

// completeBooking godoc
//
//	@Summary	Complete booking as master
//	@Tags		master-cabinet
//	@Security	BearerAuth
//	@Param		id	path		string	true	"booking UUID"
//	@Success	200	{object}	BookingResponse
//	@Router		/masters/me/bookings/{id}/complete [post]
func (h *Handler) completeBooking(c *fiber.Ctx) error {
	return h.transition(c, domain.BookingCompleted)
}

// cancelBookingByMaster godoc
//
//	@Summary	Cancel booking as master
//	@Tags		master-cabinet
//	@Security	BearerAuth
//	@Param		id	path		string	true	"booking UUID"
//	@Success	200	{object}	BookingResponse
//	@Router		/masters/me/bookings/{id}/cancel [post]
func (h *Handler) cancelBookingByMaster(c *fiber.Ctx) error {
	return h.transition(c, domain.BookingCancelledByMaster)
}

// noShowBooking godoc
//
//	@Summary	Mark booking as no-show
//	@Tags		master-cabinet
//	@Security	BearerAuth
//	@Param		id	path		string	true	"booking UUID"
//	@Success	200	{object}	BookingResponse
//	@Router		/masters/me/bookings/{id}/no-show [post]
func (h *Handler) noShowBooking(c *fiber.Ctx) error { return h.transition(c, domain.BookingNoShow) }

func (h *Handler) transition(c *fiber.Ctx, status domain.BookingStatus) error {
	user := mustUser(c)
	id, err := paramUUID(c, "id")
	if err != nil {
		return err
	}
	item, err := h.uc.TransitionBooking(c.UserContext(), id, user.ID, status)
	if err != nil {
		return err
	}
	return response.OK(c, bookingResponse(*item))
}

// marketStats godoc
//
//	@Summary	Imported market stats
//	@Tags		market
//	@Produce	json
//	@Success	200	{object}	MarketStatsResponse
//	@Router		/market/stats [get]
func (h *Handler) marketStats(c *fiber.Ctx) error {
	stats, err := h.uc.MarketStats(c.UserContext())
	if err != nil {
		return err
	}
	return response.OK(c, MarketStatsResponse(*stats))
}

// adminUsers godoc
//
//	@Summary	Admin list users
//	@Tags		admin
//	@Security	BearerAuth
//	@Produce	json
//	@Param		limit	query	int	false	"limit"
//	@Param		offset	query	int	false	"offset"
//	@Success	200		{array}	AdminUserResponse
//	@Router		/admin/users [get]
func (h *Handler) adminUsers(c *fiber.Ctx) error {
	items, err := h.uc.ListAdminUsers(c.UserContext(), queryInt(c, "limit"), queryInt(c, "offset"))
	if err != nil {
		return err
	}
	out := make([]AdminUserResponse, 0, len(items))
	for _, item := range items {
		out = append(out, AdminUserResponse(item))
	}
	return response.OK(c, out)
}

// adminUpdateUserStatus godoc
//
//	@Summary	Admin update user status
//	@Tags		admin
//	@Security	BearerAuth
//	@Accept		json
//	@Param		id		path	string					true	"user UUID"
//	@Param		body	body	UpdateUserStatusRequest	true	"status"
//	@Success	204
//	@Router		/admin/users/{id}/status [patch]
func (h *Handler) adminUpdateUserStatus(c *fiber.Ctx) error {
	id, err := paramUUID(c, "id")
	if err != nil {
		return err
	}
	var req UpdateUserStatusRequest
	if err := parseBody(c, &req); err != nil {
		return err
	}
	if err := h.uc.UpdateUserStatus(c.UserContext(), id, req.Status); err != nil {
		return err
	}
	return c.SendStatus(fiber.StatusNoContent)
}

// adminReplaceUserRoles godoc
//
//	@Summary	Admin replace user roles
//	@Tags		admin
//	@Security	BearerAuth
//	@Accept		json
//	@Param		id		path	string					true	"user UUID"
//	@Param		body	body	ReplaceUserRolesRequest	true	"roles"
//	@Success	204
//	@Router		/admin/users/{id}/roles [put]
func (h *Handler) adminReplaceUserRoles(c *fiber.Ctx) error {
	id, err := paramUUID(c, "id")
	if err != nil {
		return err
	}
	var req ReplaceUserRolesRequest
	if err := parseBody(c, &req); err != nil {
		return err
	}
	if err := h.uc.ReplaceUserRoles(c.UserContext(), id, req.Roles); err != nil {
		return err
	}
	return c.SendStatus(fiber.StatusNoContent)
}

// adminImportRuns godoc
//
//	@Summary	Admin list import runs
//	@Tags		admin
//	@Security	BearerAuth
//	@Produce	json
//	@Param		limit	query	int	false	"limit"
//	@Success	200		{array}	ImportRunResponse
//	@Router		/admin/import/runs [get]
func (h *Handler) adminImportRuns(c *fiber.Ctx) error {
	runs, err := h.uc.ListImportRuns(c.UserContext(), queryInt(c, "limit"))
	if err != nil {
		return err
	}
	out := make([]ImportRunResponse, 0, len(runs))
	for _, run := range runs {
		out = append(out, ImportRunResponse{
			ID: run.ID, SourceID: run.SourceID, SourceCode: run.SourceCode, Status: string(run.Status),
			Params: run.Params, StartedAt: run.StartedAt, FinishedAt: run.FinishedAt,
			CitiesProcessed: run.CitiesProcessed, FirmsProcessed: run.FirmsProcessed,
			MastersProcessed: run.MastersProcessed, ServicesProcessed: run.ServicesProcessed,
			ErrorsCount: run.ErrorsCount, ErrorSummary: run.ErrorSummary,
		})
	}
	return response.OK(c, out)
}

func parseBody(c *fiber.Ctx, dest any) error {
	if err := c.BodyParser(dest); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	return nil
}

func mustUser(c *fiber.Ctx) ctxuser.User {
	user, _ := ctxuser.From(c.UserContext())
	return user
}

func paramUUID(c *fiber.Ctx, name string) (uuid.UUID, error) {
	id, err := uuid.Parse(c.Params(name))
	if err != nil {
		return uuid.Nil, fiber.NewError(fiber.StatusBadRequest, "invalid "+name)
	}
	return id, nil
}

func queryInt(c *fiber.Ctx, name string) int {
	value, _ := strconv.Atoi(strings.TrimSpace(c.Query(name)))
	return value
}

func locationInput(req LocationRequest) domain.LocationInput {
	return domain.LocationInput(req)
}

func serviceInput(req ServiceRequest) domain.ServiceInput {
	return domain.ServiceInput(req)
}

func clientProfileResponse(item domain.ClientProfile) ClientProfileResponse {
	return ClientProfileResponse(item)
}

func masterProfileResponse(item domain.MasterProfile) MasterProfileResponse {
	return MasterProfileResponse(item)
}

func locationResponse(item domain.MasterLocation) LocationResponse {
	return LocationResponse(item)
}

func categoryResponse(item domain.Category) CategoryResponse {
	return CategoryResponse(item)
}

func serviceResponse(item domain.ServiceItem) ServiceResponse {
	return ServiceResponse(item)
}

func workingHourResponse(item domain.WorkingHour) WorkingHourResponse {
	return WorkingHourResponse(item)
}

func scheduleExceptionResponse(item domain.ScheduleExceptionRecord) ScheduleExceptionResponse {
	return ScheduleExceptionResponse(item)
}

func bookingResponse(item domain.Booking) BookingResponse {
	return BookingResponse(item)
}

func bookingResponses(items []domain.Booking) []BookingResponse {
	out := make([]BookingResponse, 0, len(items))
	for _, item := range items {
		out = append(out, bookingResponse(item))
	}
	return out
}
