package platform

import (
	"strconv"
	"time"

	"github.com/bookly-kbtu/backend/internal/domain"
	"github.com/bookly-kbtu/backend/internal/transport/rest/v1/response"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

func (h *Handler) registerRemaining(api fiber.Router, auth, admin fiber.Handler) {
	api.Get("/masters", h.publicMasters)
	api.Get("/masters/:id", h.publicMaster)
	api.Get("/masters/:id/services", h.publicServices)
	api.Get("/masters/:id/locations", h.publicLocations)
	api.Get("/masters/:id/slots", h.publicSlots)
	a := api.Group("/admin", auth, admin)
	a.Get("/categories", h.adminCategories)
	a.Post("/categories", h.createCategory)
	a.Put("/categories/:id", h.updateCategory)
	a.Delete("/categories/:id", h.deleteCategory)
	a.Get("/masters", h.adminMasters)
	a.Patch("/masters/:id/status", h.masterStatus)
	a.Get("/bookings", h.adminBookings)
	a.Get("/import/sources", h.importSources)
	a.Get("/import/sources/:code/cities", h.importCities)
	a.Post("/import/runs", h.runImport)
	a.Get("/import/runs/:id", h.importRun)
	a.Post("/market/firms/:id/promote", h.promoteFirm)
	n := api.Group("/notifications", auth)
	n.Get("/", h.notifications)
	n.Get("/unread-count", h.unreadCount)
	n.Post("/read-all", h.readAllNotifications)
	n.Post("/:id/read", h.readNotification)
	assistant := api.Group("/assistant/requests", auth)
	assistant.Post("/", h.createAssistant)
	assistant.Get("/", h.listAssistant)
	assistant.Get("/:id", h.getAssistant)
	assistant.Get("/:id/candidates", h.assistantCandidates)
	assistant.Put("/:id/selection", h.selectAssistant)
	assistant.Post("/:id/book", h.bookAssistant)
}

func pagination(c *fiber.Ctx) (int, int, error) {
	limit, err := strconv.Atoi(c.Query("limit", "20"))
	if err != nil || limit < 1 || limit > 100 {
		return 0, 0, fiber.NewError(400, "limit must be 1..100")
	}
	offset, err := strconv.Atoi(c.Query("offset", "0"))
	if err != nil || offset < 0 {
		return 0, 0, fiber.NewError(400, "offset must be nonnegative")
	}
	return limit, offset, nil
}
func queryUUID(c *fiber.Ctx, key string) (*uuid.UUID, error) {
	if c.Query(key) == "" {
		return nil, nil
	}
	id, err := uuid.Parse(c.Query(key))
	if err != nil || id == uuid.Nil {
		return nil, fiber.NewError(400, "invalid "+key)
	}
	return &id, nil
}
func queryBool(c *fiber.Ctx, key string) (*bool, error) {
	if c.Query(key) == "" {
		return nil, nil
	}
	v, err := strconv.ParseBool(c.Query(key))
	if err != nil {
		return nil, fiber.NewError(400, "invalid "+key)
	}
	return &v, nil
}
func queryTime(c *fiber.Ctx, key string) (*time.Time, error) {
	if c.Query(key) == "" {
		return nil, nil
	}
	v, err := time.Parse(time.RFC3339, c.Query(key))
	if err != nil {
		return nil, fiber.NewError(400, key+" must be RFC3339")
	}
	return &v, nil
}

// publicMasters godoc
// @Summary Search active Bookly masters
// @Tags masters
// @Produce json
// @Param q query string false "Display name"
// @Param category_id query string false "Category UUID"
// @Param limit query int false "Page size (1..100)" default(20)
// @Param offset query int false "Offset" default(0)
// @Success 200 {array} MasterProfileResponse
// @Failure 400 {object} response.ErrorBody
// @Router /masters [get]
func (h *Handler) publicMasters(c *fiber.Ctx) error { return h.masters(c, true) }

// adminMasters godoc
// @Summary Admin search masters including inactive profiles
// @Tags admin
// @Security BearerAuth
// @Produce json
// @Param q query string false "Display name"
// @Param category_id query string false "Category UUID"
// @Param is_active query bool false "Active state"
// @Param limit query int false "Page size (1..100)" default(20)
// @Param offset query int false "Offset" default(0)
// @Success 200 {array} MasterProfileResponse
// @Failure 400 {object} response.ErrorBody
// @Failure 401 {object} response.ErrorBody
// @Failure 403 {object} response.ErrorBody
// @Router /admin/masters [get]
func (h *Handler) adminMasters(c *fiber.Ctx) error { return h.masters(c, false) }

func (h *Handler) masters(c *fiber.Ctx, public bool) error {
	limit, offset, err := pagination(c)
	if err != nil {
		return err
	}
	category, err := queryUUID(c, "category_id")
	if err != nil {
		return err
	}
	var active *bool
	if !public {
		active, err = queryBool(c, "is_active")
		if err != nil {
			return err
		}
	}
	items, err := h.uc.ListMasters(c.UserContext(), domain.MasterFilter{Query: c.Query("q"), CategoryID: category, Active: active, Limit: limit, Offset: offset}, public)
	if err != nil {
		return err
	}
	out := make([]MasterProfileResponse, 0, len(items))
	for _, item := range items {
		out = append(out, MasterProfileResponse(item))
	}
	return response.OK(c, out)
}

// publicMaster godoc
// @Summary Get active Bookly master
// @Tags masters
// @Produce json
// @Param id path string true "Master UUID"
// @Success 200 {object} MasterProfileResponse
// @Failure 400 {object} response.ErrorBody
// @Failure 404 {object} response.ErrorBody
// @Router /masters/{id} [get]
func (h *Handler) publicMaster(c *fiber.Ctx) error {
	id, err := paramUUID(c, "id")
	if err != nil {
		return err
	}
	item, err := h.uc.GetPublicMaster(c.UserContext(), id)
	if err != nil {
		return err
	}
	return response.OK(c, MasterProfileResponse(*item))
}

// publicServices godoc
// @Summary List active services of a Bookly master
// @Tags masters
// @Produce json
// @Param id path string true "Master UUID"
// @Success 200 {array} ServiceResponse
// @Failure 400 {object} response.ErrorBody
// @Failure 404 {object} response.ErrorBody
// @Router /masters/{id}/services [get]
func (h *Handler) publicServices(c *fiber.Ctx) error {
	id, err := paramUUID(c, "id")
	if err != nil {
		return err
	}
	if _, err = h.uc.GetPublicMaster(c.UserContext(), id); err != nil {
		return err
	}
	items, err := h.uc.ListMasterServices(c.UserContext(), id, true)
	if err != nil {
		return err
	}
	out := make([]ServiceResponse, 0, len(items))
	for _, item := range items {
		out = append(out, ServiceResponse(item))
	}
	return response.OK(c, out)
}

// publicLocations godoc
// @Summary List active locations of a Bookly master
// @Tags masters
// @Produce json
// @Param id path string true "Master UUID"
// @Success 200 {array} LocationResponse
// @Failure 400 {object} response.ErrorBody
// @Failure 404 {object} response.ErrorBody
// @Router /masters/{id}/locations [get]
func (h *Handler) publicLocations(c *fiber.Ctx) error {
	id, err := paramUUID(c, "id")
	if err != nil {
		return err
	}
	if _, err = h.uc.GetPublicMaster(c.UserContext(), id); err != nil {
		return err
	}
	items, err := h.uc.ListMasterLocations(c.UserContext(), id)
	if err != nil {
		return err
	}
	out := make([]LocationResponse, 0, len(items))
	for _, item := range items {
		out = append(out, LocationResponse(item))
	}
	return response.OK(c, out)
}

type SlotResponse struct {
	StartsAt time.Time `json:"starts_at"`
	EndsAt   time.Time `json:"ends_at"`
}

// publicSlots godoc
// @Summary Available slots for one local calendar day
// @Description Uses the location time zone and a 15-minute grid. Excludes past slots, exceptions and bookings at every location. Dates are limited to the next year.
// @Tags masters
// @Produce json
// @Param id path string true "Master UUID"
// @Param service_id query string true "Master service UUID"
// @Param location_id query string true "Master location UUID"
// @Param date query string true "Local date YYYY-MM-DD"
// @Success 200 {array} SlotResponse
// @Failure 400 {object} response.ErrorBody
// @Failure 404 {object} response.ErrorBody
// @Router /masters/{id}/slots [get]
func (h *Handler) publicSlots(c *fiber.Ctx) error {
	id, err := paramUUID(c, "id")
	if err != nil {
		return err
	}
	service, err := queryUUID(c, "service_id")
	if err != nil {
		return err
	}
	location, err := queryUUID(c, "location_id")
	if err != nil {
		return err
	}
	if service == nil || location == nil {
		return fiber.NewError(400, "service_id and location_id are required")
	}
	items, err := h.uc.Slots(c.UserContext(), id, *service, *location, c.Query("date"))
	if err != nil {
		return err
	}
	return response.OK(c, items)
}

type CategoryRequest struct {
	Name     string `json:"name"`
	Slug     string `json:"slug"`
	IsActive *bool  `json:"is_active"`
}
type MasterStatusRequest struct {
	IsActive *bool `json:"is_active"`
}

// adminCategories godoc
// @Summary List all categories including inactive
// @Tags admin
// @Security BearerAuth
// @Produce json
// @Param limit query int false "Page size (1..100)" default(20)
// @Param offset query int false "Offset" default(0)
// @Success 200 {array} CategoryResponse
// @Failure 400 {object} response.ErrorBody
// @Failure 401 {object} response.ErrorBody
// @Failure 403 {object} response.ErrorBody
// @Router /admin/categories [get]
func (h *Handler) adminCategories(c *fiber.Ctx) error {
	limit, offset, err := pagination(c)
	if err != nil {
		return err
	}
	items, err := h.uc.AdminCategories(c.UserContext(), limit, offset)
	if err != nil {
		return err
	}
	out := make([]CategoryResponse, 0, len(items))
	for _, item := range items {
		out = append(out, CategoryResponse(item))
	}
	return response.OK(c, out)
}

// createCategory godoc
// @Summary Create service category
// @Tags admin
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param body body CategoryRequest true "Category (is_active defaults to true)"
// @Success 201 {object} CategoryResponse
// @Failure 400 {object} response.ErrorBody
// @Failure 401 {object} response.ErrorBody
// @Failure 403 {object} response.ErrorBody
// @Failure 409 {object} response.ErrorBody
// @Router /admin/categories [post]
func (h *Handler) createCategory(c *fiber.Ctx) error { return h.saveCategory(c, uuid.Nil) }

// updateCategory godoc
// @Summary Update service category
// @Tags admin
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "Category UUID"
// @Param body body CategoryRequest true "Category (name and slug required)"
// @Success 200 {object} CategoryResponse
// @Failure 400 {object} response.ErrorBody
// @Failure 401 {object} response.ErrorBody
// @Failure 403 {object} response.ErrorBody
// @Failure 404 {object} response.ErrorBody
// @Failure 409 {object} response.ErrorBody
// @Router /admin/categories/{id} [put]
func (h *Handler) updateCategory(c *fiber.Ctx) error {
	id, err := paramUUID(c, "id")
	if err != nil {
		return err
	}
	return h.saveCategory(c, id)
}
func (h *Handler) saveCategory(c *fiber.Ctx, id uuid.UUID) error {
	var req CategoryRequest
	if err := parseBody(c, &req); err != nil {
		return err
	}
	item, err := h.uc.SaveCategory(c.UserContext(), id, domain.CategoryInput(req))
	if err != nil {
		return err
	}
	if id == uuid.Nil {
		return response.Created(c, CategoryResponse(*item))
	}
	return response.OK(c, CategoryResponse(*item))
}

// deleteCategory godoc
// @Summary Deactivate category (preserves existing bookings)
// @Tags admin
// @Security BearerAuth
// @Param id path string true "Category UUID"
// @Success 204
// @Failure 400 {object} response.ErrorBody
// @Failure 401 {object} response.ErrorBody
// @Failure 403 {object} response.ErrorBody
// @Failure 404 {object} response.ErrorBody
// @Router /admin/categories/{id} [delete]
func (h *Handler) deleteCategory(c *fiber.Ctx) error {
	id, err := paramUUID(c, "id")
	if err != nil {
		return err
	}
	if err = h.uc.DisableCategory(c.UserContext(), id); err != nil {
		return err
	}
	return c.SendStatus(204)
}

// masterStatus godoc
// @Summary Activate or deactivate master profile
// @Tags admin
// @Security BearerAuth
// @Accept json
// @Param id path string true "Master UUID"
// @Param body body MasterStatusRequest true "Active state"
// @Success 204
// @Failure 400 {object} response.ErrorBody
// @Failure 401 {object} response.ErrorBody
// @Failure 403 {object} response.ErrorBody
// @Failure 404 {object} response.ErrorBody
// @Router /admin/masters/{id}/status [patch]
func (h *Handler) masterStatus(c *fiber.Ctx) error {
	id, err := paramUUID(c, "id")
	if err != nil {
		return err
	}
	var req MasterStatusRequest
	if err = parseBody(c, &req); err != nil {
		return err
	}
	if req.IsActive == nil {
		return fiber.NewError(400, "is_active required")
	}
	if err = h.uc.SetMasterActive(c.UserContext(), id, *req.IsActive); err != nil {
		return err
	}
	return c.SendStatus(204)
}

// adminBookings godoc
// @Summary Search all bookings
// @Tags admin
// @Security BearerAuth
// @Produce json
// @Param master_id query string false "Master UUID"
// @Param client_id query string false "Client UUID"
// @Param status query string false "Booking status" Enums(pending,confirmed,completed,cancelled_by_client,cancelled_by_master,no_show)
// @Param from query string false "Inclusive starts_at RFC3339"
// @Param to query string false "Exclusive starts_at RFC3339"
// @Param limit query int false "Page size (1..100)" default(20)
// @Param offset query int false "Offset" default(0)
// @Success 200 {array} BookingResponse
// @Failure 400 {object} response.ErrorBody
// @Failure 401 {object} response.ErrorBody
// @Failure 403 {object} response.ErrorBody
// @Router /admin/bookings [get]
func (h *Handler) adminBookings(c *fiber.Ctx) error {
	limit, offset, err := pagination(c)
	if err != nil {
		return err
	}
	master, err := queryUUID(c, "master_id")
	if err != nil {
		return err
	}
	client, err := queryUUID(c, "client_id")
	if err != nil {
		return err
	}
	from, err := queryTime(c, "from")
	if err != nil {
		return err
	}
	to, err := queryTime(c, "to")
	if err != nil {
		return err
	}
	items, err := h.uc.AdminBookings(c.UserContext(), domain.BookingFilter{MasterID: master, ClientID: client, Status: c.Query("status"), From: from, To: to, Limit: limit, Offset: offset})
	if err != nil {
		return err
	}
	return response.OK(c, bookingResponses(items))
}
