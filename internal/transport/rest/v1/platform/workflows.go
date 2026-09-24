package platform

import (
	"context"
	"time"

	"github.com/bookly-kbtu/backend/internal/domain"
	"github.com/bookly-kbtu/backend/internal/transport/rest/v1/response"
	importeruc "github.com/bookly-kbtu/backend/internal/usecase/importer"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type NotificationResponse domain.Notification
type AssistantResponse domain.AssistantRequest
type AssistantCandidateResponse domain.AssistantCandidate
type AssistantIntentRequest domain.AssistantIntent
type PromoteFirmRequest domain.PromoteFirmInput
type PromotionResponse domain.Promotion
type UnreadCountResponse struct {
	Count int `json:"count"`
}
type CreateAssistantRequest struct {
	Transcript string                 `json:"transcript"`
	Intent     AssistantIntentRequest `json:"intent"`
}
type SelectAssistantRequest struct {
	MasterID  uuid.UUID `json:"master_id"`
	ServiceID uuid.UUID `json:"service_id"`
}
type BookAssistantRequest struct {
	LocationID    uuid.UUID `json:"location_id"`
	StartsAt      time.Time `json:"starts_at"`
	ClientComment *string   `json:"client_comment"`
}
type ImportSourceResponse struct {
	Code    string `json:"code"`
	Name    string `json:"name"`
	BaseURL string `json:"base_url"`
}
type ImportCityResponse struct {
	ExternalID string `json:"external_id"`
	Name       string `json:"name"`
	Slug       string `json:"slug"`
}
type RunImportRequest struct {
	SourceCode    string `json:"source_code"`
	CityID        string `json:"city_id"`
	MaxFirms      int    `json:"max_firms"`
	SaveSnapshots bool   `json:"save_snapshots"`
}

// notifications godoc
// @Summary List current user's in-app notifications
// @Tags notifications
// @Security BearerAuth
// @Produce json
// @Param unread_only query bool false "Only unread notifications"
// @Param limit query int false "Page size (1..100)" default(20)
// @Param offset query int false "Offset" default(0)
// @Success 200 {array} NotificationResponse
// @Failure 400 {object} response.ErrorBody
// @Failure 401 {object} response.ErrorBody
// @Router /notifications [get]
func (h *Handler) notifications(c *fiber.Ctx) error {
	limit, offset, err := pagination(c)
	if err != nil {
		return err
	}
	unread, err := queryBool(c, "unread_only")
	if err != nil {
		return err
	}
	items, err := h.uc.Notifications(c.UserContext(), mustUser(c).ID, unread != nil && *unread, limit, offset)
	if err != nil {
		return err
	}
	if items == nil {
		items = []domain.Notification{}
	}
	return response.OK(c, items)
}

// unreadCount godoc
// @Summary Count unread notifications
// @Tags notifications
// @Security BearerAuth
// @Produce json
// @Success 200 {object} UnreadCountResponse
// @Failure 401 {object} response.ErrorBody
// @Router /notifications/unread-count [get]
func (h *Handler) unreadCount(c *fiber.Ctx) error {
	count, err := h.uc.UnreadCount(c.UserContext(), mustUser(c).ID)
	if err != nil {
		return err
	}
	return response.OK(c, UnreadCountResponse{Count: count})
}

// readNotification godoc
// @Summary Mark own notification as read
// @Tags notifications
// @Security BearerAuth
// @Param id path string true "Notification UUID"
// @Success 204
// @Failure 400 {object} response.ErrorBody
// @Failure 401 {object} response.ErrorBody
// @Failure 404 {object} response.ErrorBody
// @Router /notifications/{id}/read [post]
func (h *Handler) readNotification(c *fiber.Ctx) error {
	id, err := paramUUID(c, "id")
	if err != nil {
		return err
	}
	if err = h.uc.ReadNotification(c.UserContext(), mustUser(c).ID, id); err != nil {
		return err
	}
	return c.SendStatus(204)
}

// readAllNotifications godoc
// @Summary Mark all own notifications as read
// @Tags notifications
// @Security BearerAuth
// @Success 204
// @Failure 401 {object} response.ErrorBody
// @Router /notifications/read-all [post]
func (h *Handler) readAllNotifications(c *fiber.Ctx) error {
	if err := h.uc.ReadAllNotifications(c.UserContext(), mustUser(c).ID); err != nil {
		return err
	}
	return c.SendStatus(204)
}

// createAssistant godoc
// @Summary Create assistant search request from structured intent
// @Description No LLM parsing is configured. Supply transcript for audit and intent.query or intent.category_id for deterministic search. max_price is in KZT minor units.
// @Tags assistant
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param body body CreateAssistantRequest true "Transcript and structured search intent"
// @Success 201 {object} AssistantResponse
// @Failure 400 {object} response.ErrorBody
// @Failure 401 {object} response.ErrorBody
// @Router /assistant/requests [post]
func (h *Handler) createAssistant(c *fiber.Ctx) error {
	var req CreateAssistantRequest
	if err := parseBody(c, &req); err != nil {
		return err
	}
	item, err := h.uc.CreateAssistantRequest(c.UserContext(), mustUser(c).ID, req.Transcript, domain.AssistantIntent(req.Intent))
	if err != nil {
		return err
	}
	return response.Created(c, item)
}

// listAssistant godoc
// @Summary List own assistant requests
// @Tags assistant
// @Security BearerAuth
// @Produce json
// @Param limit query int false "Page size (1..100)" default(20)
// @Param offset query int false "Offset" default(0)
// @Success 200 {array} AssistantResponse
// @Failure 400 {object} response.ErrorBody
// @Failure 401 {object} response.ErrorBody
// @Router /assistant/requests [get]
func (h *Handler) listAssistant(c *fiber.Ctx) error {
	limit, offset, err := pagination(c)
	if err != nil {
		return err
	}
	items, err := h.uc.AssistantRequests(c.UserContext(), mustUser(c).ID, limit, offset)
	if err != nil {
		return err
	}
	if items == nil {
		items = []domain.AssistantRequest{}
	}
	return response.OK(c, items)
}

// getAssistant godoc
// @Summary Get own assistant request and selection
// @Tags assistant
// @Security BearerAuth
// @Produce json
// @Param id path string true "Request UUID"
// @Success 200 {object} AssistantResponse
// @Failure 400 {object} response.ErrorBody
// @Failure 401 {object} response.ErrorBody
// @Failure 404 {object} response.ErrorBody
// @Router /assistant/requests/{id} [get]
func (h *Handler) getAssistant(c *fiber.Ctx) error {
	id, err := paramUUID(c, "id")
	if err != nil {
		return err
	}
	item, err := h.uc.AssistantRequest(c.UserContext(), mustUser(c).ID, id)
	if err != nil {
		return err
	}
	return response.OK(c, item)
}

// assistantCandidates godoc
// @Summary Search services using saved assistant intent
// @Tags assistant
// @Security BearerAuth
// @Produce json
// @Param id path string true "Request UUID"
// @Param limit query int false "Page size (1..100)" default(20)
// @Param offset query int false "Offset" default(0)
// @Success 200 {array} AssistantCandidateResponse
// @Failure 400 {object} response.ErrorBody
// @Failure 401 {object} response.ErrorBody
// @Failure 404 {object} response.ErrorBody
// @Router /assistant/requests/{id}/candidates [get]
func (h *Handler) assistantCandidates(c *fiber.Ctx) error {
	id, err := paramUUID(c, "id")
	if err != nil {
		return err
	}
	limit, offset, err := pagination(c)
	if err != nil {
		return err
	}
	items, err := h.uc.AssistantCandidates(c.UserContext(), mustUser(c).ID, id, limit, offset)
	if err != nil {
		return err
	}
	if items == nil {
		items = []domain.AssistantCandidate{}
	}
	return response.OK(c, items)
}

// selectAssistant godoc
// @Summary Select a service for an assistant request
// @Tags assistant
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "Request UUID"
// @Param body body SelectAssistantRequest true "Selected master and service"
// @Success 200 {object} AssistantResponse
// @Failure 400 {object} response.ErrorBody
// @Failure 401 {object} response.ErrorBody
// @Failure 404 {object} response.ErrorBody
// @Failure 409 {object} response.ErrorBody
// @Router /assistant/requests/{id}/selection [put]
func (h *Handler) selectAssistant(c *fiber.Ctx) error {
	id, err := paramUUID(c, "id")
	if err != nil {
		return err
	}
	var req SelectAssistantRequest
	if err = parseBody(c, &req); err != nil {
		return err
	}
	item, err := h.uc.SelectAssistantService(c.UserContext(), mustUser(c).ID, id, req.MasterID, req.ServiceID)
	if err != nil {
		return err
	}
	return response.OK(c, item)
}

// bookAssistant godoc
// @Summary Confirm booking of the selected service
// @Description Revalidates availability. Repeated calls return the existing booking for this request, without creating duplicates.
// @Tags assistant
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "Request UUID"
// @Param body body BookAssistantRequest true "Location and chosen available slot"
// @Success 200 {object} BookingResponse
// @Failure 400 {object} response.ErrorBody
// @Failure 401 {object} response.ErrorBody
// @Failure 404 {object} response.ErrorBody
// @Failure 409 {object} response.ErrorBody
// @Router /assistant/requests/{id}/book [post]
func (h *Handler) bookAssistant(c *fiber.Ctx) error {
	id, err := paramUUID(c, "id")
	if err != nil {
		return err
	}
	var req BookAssistantRequest
	if err = parseBody(c, &req); err != nil {
		return err
	}
	item, err := h.uc.BookAssistant(c.UserContext(), mustUser(c).ID, id, req.LocationID, req.StartsAt, req.ClientComment)
	if err != nil {
		return err
	}
	return response.OK(c, BookingResponse(*item))
}

// importSources godoc
// @Summary List registered import adapters
// @Tags admin-import
// @Security BearerAuth
// @Produce json
// @Success 200 {array} ImportSourceResponse
// @Failure 401 {object} response.ErrorBody
// @Failure 403 {object} response.ErrorBody
// @Router /admin/import/sources [get]
func (h *Handler) importSources(c *fiber.Ctx) error {
	out := make([]ImportSourceResponse, 0)
	for _, s := range h.importer.Sources() {
		out = append(out, ImportSourceResponse{s.Code, s.Name, s.BaseURL})
	}
	return response.OK(c, out)
}

// importCities godoc
// @Summary Fetch live cities supported by an import source
// @Tags admin-import
// @Security BearerAuth
// @Produce json
// @Param code path string true "Source code, e.g. zapis_kz"
// @Success 200 {array} ImportCityResponse
// @Failure 400 {object} response.ErrorBody
// @Failure 401 {object} response.ErrorBody
// @Failure 403 {object} response.ErrorBody
// @Failure 502 {object} response.ErrorBody
// @Router /admin/import/sources/{code}/cities [get]
func (h *Handler) importCities(c *fiber.Ctx) error {
	if !h.hasSource(c.Params("code")) {
		return fiber.NewError(400, "unknown source")
	}
	ctx, cancel := context.WithTimeout(c.UserContext(), 30*time.Second)
	defer cancel()
	items, err := h.importer.Cities(ctx, c.Params("code"))
	if err != nil {
		return fiber.NewError(502, "could not fetch source cities")
	}
	out := make([]ImportCityResponse, 0, len(items))
	for _, item := range items {
		out = append(out, ImportCityResponse{item.ExternalID, item.Name, item.Slug})
	}
	return response.OK(c, out)
}
func (h *Handler) hasSource(code string) bool {
	for _, s := range h.importer.Sources() {
		if s.Code == code {
			return true
		}
	}
	return false
}

// runImport godoc
// @Summary Run a bounded synchronous import batch
// @Description Imports one city with 1..50 firms, with a two-minute deadline. Returns the persisted run including failed status and partial progress. Inspect status and errors_count; HTTP 200 does not imply all firms succeeded. Full-city imports remain available through CLI.
// @Tags admin-import
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param body body RunImportRequest true "Import batch (max_firms defaults to 10)"
// @Success 200 {object} ImportRunResponse
// @Failure 400 {object} response.ErrorBody
// @Failure 401 {object} response.ErrorBody
// @Failure 403 {object} response.ErrorBody
// @Failure 409 {object} response.ErrorBody
// @Router /admin/import/runs [post]
func (h *Handler) runImport(c *fiber.Ctx) error {
	var req RunImportRequest
	if err := parseBody(c, &req); err != nil {
		return err
	}
	if req.MaxFirms == 0 {
		req.MaxFirms = 10
	}
	if !h.hasSource(req.SourceCode) || req.CityID == "" || req.MaxFirms < 1 || req.MaxFirms > 50 {
		return fiber.NewError(400, "valid source_code, city_id and max_firms 1..50 required")
	}
	if !h.importMu.TryLock() {
		return fiber.NewError(409, "an HTTP import is already running on this instance")
	}
	defer h.importMu.Unlock()
	ctx, cancel := context.WithTimeout(c.UserContext(), 2*time.Minute)
	defer cancel()
	run, err := h.importer.Run(ctx, importeruc.RunInput{SourceCode: req.SourceCode, CityIDs: []string{req.CityID}, MaxFirms: req.MaxFirms, SaveSnapshots: req.SaveSnapshots})
	if run == nil {
		return err
	}
	return response.OK(c, importRunResponse(*run))
}

// importRun godoc
// @Summary Get import run details
// @Tags admin-import
// @Security BearerAuth
// @Produce json
// @Param id path string true "Run UUID"
// @Success 200 {object} ImportRunResponse
// @Failure 400 {object} response.ErrorBody
// @Failure 401 {object} response.ErrorBody
// @Failure 403 {object} response.ErrorBody
// @Failure 404 {object} response.ErrorBody
// @Router /admin/import/runs/{id} [get]
func (h *Handler) importRun(c *fiber.Ctx) error {
	id, err := paramUUID(c, "id")
	if err != nil {
		return err
	}
	item, err := h.uc.GetImportRun(c.UserContext(), id)
	if err != nil {
		return err
	}
	return response.OK(c, importRunResponse(*item))
}
func importRunResponse(r domain.ImportRun) ImportRunResponse {
	return ImportRunResponse{ID: r.ID, SourceID: r.SourceID, SourceCode: r.SourceCode, Status: string(r.Status), Params: r.Params, StartedAt: r.StartedAt, FinishedAt: r.FinishedAt, CitiesProcessed: r.CitiesProcessed, FirmsProcessed: r.FirmsProcessed, MastersProcessed: r.MastersProcessed, ServicesProcessed: r.ServicesProcessed, ErrorsCount: r.ErrorsCount, ErrorSummary: r.ErrorSummary}
}

// promoteFirm godoc
// @Summary Copy imported firm into an existing Bookly account
// @Description Creates a location and selected services atomically. Existing master profile is preserved. Map each source service to a Bookly category and supply missing price/duration. Configure weekly hours after promotion. Repeating promotion returns 409.
// @Tags admin-import
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "Imported firm UUID"
// @Param body body PromoteFirmRequest true "Target user, timezone and selected services"
// @Success 201 {object} PromotionResponse
// @Failure 400 {object} response.ErrorBody
// @Failure 401 {object} response.ErrorBody
// @Failure 403 {object} response.ErrorBody
// @Failure 404 {object} response.ErrorBody
// @Failure 409 {object} response.ErrorBody
// @Router /admin/market/firms/{id}/promote [post]
func (h *Handler) promoteFirm(c *fiber.Ctx) error {
	id, err := paramUUID(c, "id")
	if err != nil {
		return err
	}
	var req PromoteFirmRequest
	if err = parseBody(c, &req); err != nil {
		return err
	}
	item, err := h.uc.PromoteFirm(c.UserContext(), mustUser(c).ID, id, domain.PromoteFirmInput(req))
	if err != nil {
		return err
	}
	return response.Created(c, item)
}
