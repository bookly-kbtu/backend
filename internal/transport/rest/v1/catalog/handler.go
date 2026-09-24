package catalog

import (
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"github.com/bookly-kbtu/backend/internal/domain"
	"github.com/bookly-kbtu/backend/internal/transport/rest/v1/response"
	cataloguc "github.com/bookly-kbtu/backend/internal/usecase/catalog"
)

type Handler struct {
	uc *cataloguc.Service
}

func New(uc *cataloguc.Service) *Handler {
	return &Handler{uc: uc}
}

type CityResponse struct {
	ID         uuid.UUID `json:"id"`
	Source     string    `json:"source"`
	ExternalID string    `json:"external_id"`
	Name       string    `json:"name"`
	Slug       *string   `json:"slug"`
	Latitude   *float64  `json:"latitude"`
	Longitude  *float64  `json:"longitude"`
	LastSeenAt time.Time `json:"last_seen_at"`
}

type CategoryResponse struct {
	ID               uuid.UUID `json:"id"`
	Source           string    `json:"source"`
	Kind             string    `json:"kind"`
	ExternalID       string    `json:"external_id"`
	ParentExternalID *string   `json:"parent_external_id"`
	Name             string    `json:"name"`
	IconURL          *string   `json:"icon_url"`
	LastSeenAt       time.Time `json:"last_seen_at"`
}

type FirmResponse struct {
	ID             uuid.UUID  `json:"id"`
	Source         string     `json:"source"`
	CityID         *uuid.UUID `json:"city_id"`
	CityName       *string    `json:"city_name"`
	ExternalID     string     `json:"external_id"`
	Name           string     `json:"name"`
	Category       *string    `json:"category"`
	EntityType     *string    `json:"entity_type"`
	URLKey         *string    `json:"url_key"`
	AddressText    *string    `json:"address_text"`
	AvatarURL      *string    `json:"avatar_url"`
	AverageRating  *float64   `json:"average_rating"`
	RatingsCount   *int       `json:"ratings_count"`
	ReviewsCount   *int       `json:"reviews_count"`
	WorkStartTime  *string    `json:"work_start_time"`
	WorkEndTime    *string    `json:"work_end_time"`
	IsOnline       *bool      `json:"is_online"`
	IsPromoted     *bool      `json:"is_promoted"`
	Latitude       *float64   `json:"latitude"`
	Longitude      *float64   `json:"longitude"`
	DistanceMeters *float64   `json:"distance_meters"`
	ServicesCount  int        `json:"services_count"`
	MastersCount   int        `json:"masters_count"`
	LastSeenAt     time.Time  `json:"last_seen_at"`
}

type FirmDetailResponse struct {
	FirmResponse
	Description *string `json:"description"`
	MapProvider *string `json:"map_provider"`
}

type FirmPhotoResponse struct {
	PhotoURL   string    `json:"photo_url"`
	LastSeenAt time.Time `json:"last_seen_at"`
}

type ServiceResponse struct {
	ID                    uuid.UUID `json:"id"`
	ExternalID            string    `json:"external_id"`
	CategoryExternalID    *string   `json:"category_external_id"`
	SubcategoryExternalID *string   `json:"subcategory_external_id"`
	Name                  string    `json:"name"`
	Description           *string   `json:"description"`
	PriceMinAmount        *int64    `json:"price_min_amount"`
	PriceMaxAmount        *int64    `json:"price_max_amount"`
	Currency              *string   `json:"currency"`
	DurationMinutes       *int      `json:"duration_minutes"`
	IsExpress             *bool     `json:"is_express"`
	LastSeenAt            time.Time `json:"last_seen_at"`
}

type MasterResponse struct {
	ID             uuid.UUID `json:"id"`
	Source         string    `json:"source"`
	ExternalID     string    `json:"external_id"`
	DisplayName    string    `json:"display_name"`
	Profession     *string   `json:"profession"`
	ExperienceText *string   `json:"experience_text"`
	AvatarURL      *string   `json:"avatar_url"`
	AverageRating  *float64  `json:"average_rating"`
	RatingsCount   *int      `json:"ratings_count"`
	IsOnline       *bool     `json:"is_online"`
	LastSeenAt     time.Time `json:"last_seen_at"`
}

func (h *Handler) Register(api fiber.Router) {
	api.Get("/cities", h.listCities)
	api.Get("/categories", h.listCategories)
	api.Get("/firms", h.listFirms)
	api.Get("/firms/:id", h.getFirm)
	api.Get("/firms/:id/photos", h.listFirmPhotos)
	api.Get("/firms/:id/services", h.listFirmServices)
	api.Get("/firms/:id/masters", h.listFirmMasters)
}

// listCities godoc
//
//	@Summary	List imported market cities
//	@Tags		market
//	@Produce	json
//	@Success	200	{array}		CityResponse
//	@Failure	500	{object}	response.ErrorBody
//	@Router		/market/cities [get]
func (h *Handler) listCities(c *fiber.Ctx) error {
	cities, err := h.uc.ListCities(c.UserContext())
	if err != nil {
		return err
	}

	out := make([]CityResponse, 0, len(cities))
	for _, city := range cities {
		out = append(out, cityResponse(city))
	}
	return response.OK(c, out)
}

// listCategories godoc
//
//	@Summary	List imported market categories
//	@Tags		market
//	@Produce	json
//	@Param		kind	query		string	false	"category kind"	Enums(category, subcategory)
//	@Param		q		query		string	false	"case-insensitive name search"
//	@Param		limit	query		int		false	"max rows, default 20, max 100"
//	@Success	200		{array}		CategoryResponse
//	@Failure	400		{object}	response.ErrorBody
//	@Failure	500		{object}	response.ErrorBody
//	@Router		/market/categories [get]
func (h *Handler) listCategories(c *fiber.Ctx) error {
	limit, err := queryInt(c, "limit")
	if err != nil {
		return err
	}

	categories, err := h.uc.ListCategories(c.UserContext(), c.Query("kind"), c.Query("q"), limit)
	if err != nil {
		return err
	}

	out := make([]CategoryResponse, 0, len(categories))
	for _, category := range categories {
		out = append(out, categoryResponse(category))
	}
	return response.OK(c, out)
}

// listFirms godoc
//
//	@Summary		List imported market firms
//	@Description	Returns read-only imported firms from external catalogues such as zapis.kz. These rows are market data, not Bookly booking entities.
//	@Tags			market
//	@Produce		json
//	@Param			city_id		query		string	false	"source city UUID"
//	@Param			category_id	query		string	false	"source category UUID"
//	@Param			q			query		string	false	"case-insensitive search by name, address or description"
//	@Param			lat			query		number	false	"latitude for distance filtering/sorting"
//	@Param			lng			query		number	false	"longitude for distance filtering/sorting"
//	@Param			radius_m	query		int		false	"radius in meters, requires lat and lng"
//	@Param			limit		query		int		false	"max rows, default 20, max 100"
//	@Param			offset		query		int		false	"pagination offset"
//	@Success		200			{array}		FirmResponse
//	@Failure		400			{object}	response.ErrorBody
//	@Failure		500			{object}	response.ErrorBody
//	@Router			/market/firms [get]
func (h *Handler) listFirms(c *fiber.Ctx) error {
	filter, err := firmFilter(c)
	if err != nil {
		return err
	}

	firms, err := h.uc.ListFirms(c.UserContext(), filter)
	if err != nil {
		return err
	}

	out := make([]FirmResponse, 0, len(firms))
	for _, firm := range firms {
		out = append(out, firmResponse(firm))
	}
	return response.OK(c, out)
}

// getFirm godoc
//
//	@Summary	Get imported market firm
//	@Tags		market
//	@Produce	json
//	@Param		id	path		string	true	"firm UUID"
//	@Param		lat	query		number	false	"latitude for distance"
//	@Param		lng	query		number	false	"longitude for distance"
//	@Success	200	{object}	FirmDetailResponse
//	@Failure	400	{object}	response.ErrorBody
//	@Failure	404	{object}	response.ErrorBody
//	@Failure	500	{object}	response.ErrorBody
//	@Router		/market/firms/{id} [get]
func (h *Handler) getFirm(c *fiber.Ctx) error {
	id, err := paramUUID(c, "id")
	if err != nil {
		return err
	}
	lat, lng, err := geoQuery(c)
	if err != nil {
		return err
	}

	firm, err := h.uc.GetFirm(c.UserContext(), id, lat, lng)
	if err != nil {
		return err
	}
	return response.OK(c, firmDetailResponse(*firm))
}

// listFirmPhotos godoc
//
//	@Summary	List imported firm photos
//	@Tags		market
//	@Produce	json
//	@Param		id	path		string	true	"firm UUID"
//	@Success	200	{array}		FirmPhotoResponse
//	@Failure	400	{object}	response.ErrorBody
//	@Failure	500	{object}	response.ErrorBody
//	@Router		/market/firms/{id}/photos [get]
func (h *Handler) listFirmPhotos(c *fiber.Ctx) error {
	id, err := paramUUID(c, "id")
	if err != nil {
		return err
	}

	photos, err := h.uc.ListFirmPhotos(c.UserContext(), id)
	if err != nil {
		return err
	}

	out := make([]FirmPhotoResponse, 0, len(photos))
	for _, photo := range photos {
		out = append(out, FirmPhotoResponse{PhotoURL: photo.PhotoURL, LastSeenAt: photo.LastSeenAt})
	}
	return response.OK(c, out)
}

// listFirmServices godoc
//
//	@Summary	List imported firm services
//	@Tags		market
//	@Produce	json
//	@Param		id	path		string	true	"firm UUID"
//	@Success	200	{array}		ServiceResponse
//	@Failure	400	{object}	response.ErrorBody
//	@Failure	500	{object}	response.ErrorBody
//	@Router		/market/firms/{id}/services [get]
func (h *Handler) listFirmServices(c *fiber.Ctx) error {
	id, err := paramUUID(c, "id")
	if err != nil {
		return err
	}

	services, err := h.uc.ListFirmServices(c.UserContext(), id)
	if err != nil {
		return err
	}

	out := make([]ServiceResponse, 0, len(services))
	for _, service := range services {
		out = append(out, serviceResponse(service))
	}
	return response.OK(c, out)
}

// listFirmMasters godoc
//
//	@Summary	List imported firm masters
//	@Tags		market
//	@Produce	json
//	@Param		id	path		string	true	"firm UUID"
//	@Success	200	{array}		MasterResponse
//	@Failure	400	{object}	response.ErrorBody
//	@Failure	500	{object}	response.ErrorBody
//	@Router		/market/firms/{id}/masters [get]
func (h *Handler) listFirmMasters(c *fiber.Ctx) error {
	id, err := paramUUID(c, "id")
	if err != nil {
		return err
	}

	masters, err := h.uc.ListFirmMasters(c.UserContext(), id)
	if err != nil {
		return err
	}

	out := make([]MasterResponse, 0, len(masters))
	for _, master := range masters {
		out = append(out, masterResponse(master))
	}
	return response.OK(c, out)
}

func firmFilter(c *fiber.Ctx) (domain.SourceFirmFilter, error) {
	limit, err := queryInt(c, "limit")
	if err != nil {
		return domain.SourceFirmFilter{}, err
	}
	offset, err := queryInt(c, "offset")
	if err != nil {
		return domain.SourceFirmFilter{}, err
	}
	radius, err := queryIntPtr(c, "radius_m")
	if err != nil {
		return domain.SourceFirmFilter{}, err
	}
	lat, lng, err := geoQuery(c)
	if err != nil {
		return domain.SourceFirmFilter{}, err
	}
	cityID, err := queryUUIDPtr(c, "city_id")
	if err != nil {
		return domain.SourceFirmFilter{}, err
	}
	categoryID, err := queryUUIDPtr(c, "category_id")
	if err != nil {
		return domain.SourceFirmFilter{}, err
	}

	return domain.SourceFirmFilter{
		CityID:     cityID,
		CategoryID: categoryID,
		Query:      c.Query("q"),
		Latitude:   lat,
		Longitude:  lng,
		RadiusM:    radius,
		Limit:      limit,
		Offset:     offset,
	}, nil
}

func geoQuery(c *fiber.Ctx) (*float64, *float64, error) {
	lat, err := queryFloatPtr(c, "lat")
	if err != nil {
		return nil, nil, err
	}
	lng, err := queryFloatPtr(c, "lng")
	if err != nil {
		return nil, nil, err
	}
	return lat, lng, nil
}

func paramUUID(c *fiber.Ctx, name string) (uuid.UUID, error) {
	id, err := uuid.Parse(c.Params(name))
	if err != nil {
		return uuid.Nil, fiber.NewError(fiber.StatusBadRequest, "invalid "+name)
	}
	return id, nil
}

func queryUUIDPtr(c *fiber.Ctx, name string) (*uuid.UUID, error) {
	raw := strings.TrimSpace(c.Query(name))
	if raw == "" {
		return nil, nil
	}
	id, err := uuid.Parse(raw)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusBadRequest, "invalid "+name)
	}
	return &id, nil
}

func queryInt(c *fiber.Ctx, name string) (int, error) {
	raw := strings.TrimSpace(c.Query(name))
	if raw == "" {
		return 0, nil
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fiber.NewError(fiber.StatusBadRequest, "invalid "+name)
	}
	return value, nil
}

func queryIntPtr(c *fiber.Ctx, name string) (*int, error) {
	value, err := queryInt(c, name)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(c.Query(name)) == "" {
		return nil, nil
	}
	return &value, nil
}

func queryFloatPtr(c *fiber.Ctx, name string) (*float64, error) {
	raw := strings.TrimSpace(c.Query(name))
	if raw == "" {
		return nil, nil
	}
	value, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusBadRequest, "invalid "+name)
	}
	return &value, nil
}

func cityResponse(city domain.SourceCity) CityResponse {
	return CityResponse{
		ID:         city.ID,
		Source:     city.SourceCode,
		ExternalID: city.ExternalID,
		Name:       city.Name,
		Slug:       city.Slug,
		Latitude:   city.Latitude,
		Longitude:  city.Longitude,
		LastSeenAt: city.LastSeenAt,
	}
}

func categoryResponse(category domain.SourceCatalogCategory) CategoryResponse {
	return CategoryResponse{
		ID:               category.ID,
		Source:           category.SourceCode,
		Kind:             string(category.Kind),
		ExternalID:       category.ExternalID,
		ParentExternalID: category.ParentExternalID,
		Name:             category.Name,
		IconURL:          category.IconURL,
		LastSeenAt:       category.LastSeenAt,
	}
}

func firmResponse(firm domain.SourceFirmSummary) FirmResponse {
	return FirmResponse{
		ID:             firm.ID,
		Source:         firm.SourceCode,
		CityID:         firm.CityID,
		CityName:       firm.CityName,
		ExternalID:     firm.ExternalID,
		Name:           firm.Name,
		Category:       firm.Category,
		EntityType:     firm.EntityType,
		URLKey:         firm.URLKey,
		AddressText:    firm.AddressText,
		AvatarURL:      firm.AvatarURL,
		AverageRating:  firm.AverageRating,
		RatingsCount:   firm.RatingsCount,
		ReviewsCount:   firm.ReviewsCount,
		WorkStartTime:  firm.WorkStartTime,
		WorkEndTime:    firm.WorkEndTime,
		IsOnline:       firm.IsOnline,
		IsPromoted:     firm.IsPromoted,
		Latitude:       firm.Latitude,
		Longitude:      firm.Longitude,
		DistanceMeters: firm.DistanceMeters,
		ServicesCount:  firm.ServicesCount,
		MastersCount:   firm.MastersCount,
		LastSeenAt:     firm.LastSeenAt,
	}
}

func firmDetailResponse(firm domain.SourceFirm) FirmDetailResponse {
	return FirmDetailResponse{
		FirmResponse: firmResponse(firm.SourceFirmSummary),
		Description:  firm.Description,
		MapProvider:  firm.MapProvider,
	}
}

func serviceResponse(service domain.SourceService) ServiceResponse {
	return ServiceResponse{
		ID:                    service.ID,
		ExternalID:            service.ExternalID,
		CategoryExternalID:    service.CategoryExternalID,
		SubcategoryExternalID: service.SubcategoryExternalID,
		Name:                  service.Name,
		Description:           service.Description,
		PriceMinAmount:        service.PriceMinAmount,
		PriceMaxAmount:        service.PriceMaxAmount,
		Currency:              service.Currency,
		DurationMinutes:       service.DurationMinutes,
		IsExpress:             service.IsExpress,
		LastSeenAt:            service.LastSeenAt,
	}
}

func masterResponse(master domain.SourceMaster) MasterResponse {
	return MasterResponse{
		ID:             master.ID,
		Source:         master.SourceCode,
		ExternalID:     master.ExternalID,
		DisplayName:    master.DisplayName,
		Profession:     master.Profession,
		ExperienceText: master.ExperienceText,
		AvatarURL:      master.AvatarURL,
		AverageRating:  master.AverageRating,
		RatingsCount:   master.RatingsCount,
		IsOnline:       master.IsOnline,
		LastSeenAt:     master.LastSeenAt,
	}
}
