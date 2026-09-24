package zapis

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/bookly-kbtu/backend/internal/domain"
)

// Client implements domain.CatalogSource.
var _ domain.CatalogSource = (*Client)(nil)

func (c *Client) Code() string    { return Code }
func (c *Client) Name() string    { return "Zapis.kz" }
func (c *Client) BaseURL() string { return c.cfg.BaseURL }

func (c *Client) Cities(ctx context.Context) ([]domain.ExternalCity, domain.RawPayload, error) {
	const path = "/screen/home/cities"

	var resp citiesResponse
	raw, err := c.get(ctx, path, "", &resp)
	if err != nil {
		return nil, domain.RawPayload{}, err
	}

	cities := make([]domain.ExternalCity, 0, len(resp.Data.Cities))
	for _, ct := range resp.Data.Cities {
		cities = append(cities, domain.ExternalCity{
			ExternalID: strconv.Itoa(ct.ID),
			Name:       ct.Name,
			Slug:       ct.URLName,
			Latitude:   coordinate(ct.Latitude),
			Longitude:  coordinate(ct.Longitude),
		})
	}

	return cities, domain.RawPayload{Kind: "cities", RequestPath: path, Body: raw}, nil
}

// FirmIDs merges firmIds and firms[].id from search: the source splits results between them.
func (c *Client) FirmIDs(ctx context.Context, cityID string) ([]string, domain.RawPayload, error) {
	const path = "/firms/search"

	var resp searchResponse
	raw, err := c.get(ctx, path, cityID, &resp)
	if err != nil {
		return nil, domain.RawPayload{}, err
	}

	seen := make(map[int64]struct{}, len(resp.Data.FirmIDs)+len(resp.Data.Firms))
	ids := make([]string, 0, len(seen))
	add := func(id int64) {
		if _, ok := seen[id]; ok || id <= 0 {
			return
		}
		seen[id] = struct{}{}
		ids = append(ids, strconv.FormatInt(id, 10))
	}

	for _, id := range resp.Data.FirmIDs {
		add(id)
	}
	for _, f := range resp.Data.Firms {
		add(f.ID)
	}

	return ids, domain.RawPayload{Kind: "firm_search", ExternalID: cityID, RequestPath: path, Body: raw}, nil
}

func (c *Client) Firm(ctx context.Context, cityID, firmID string) (*domain.ExternalFirm, domain.RawPayload, error) {
	path := "/firms/" + firmID

	var resp firmResponse
	raw, err := c.get(ctx, path, cityID, &resp)
	if err != nil {
		return nil, domain.RawPayload{}, err
	}

	firm, err := c.mapFirm(firmID, &resp)
	if err != nil {
		return nil, domain.RawPayload{}, err
	}

	return firm, domain.RawPayload{Kind: "firm", ExternalID: firmID, RequestPath: path, Body: raw}, nil
}

func (c *Client) FirmMasters(ctx context.Context, cityID, firmID string) ([]domain.ExternalMaster, domain.RawPayload, error) {
	path := "/firms/" + firmID + "/masters"

	var resp mastersResponse
	raw, err := c.get(ctx, path, cityID, &resp)
	if err != nil {
		return nil, domain.RawPayload{}, err
	}

	masters := make([]domain.ExternalMaster, 0, len(resp.Data.Masters))
	for _, m := range resp.Data.Masters {
		name := strings.TrimSpace(m.Name + " " + m.Surname)
		if m.ID <= 0 || name == "" {
			continue
		}
		masters = append(masters, domain.ExternalMaster{
			ExternalID:    strconv.FormatInt(m.ID, 10),
			DisplayName:   name,
			Profession:    strings.TrimSpace(m.Profession),
			Experience:    strings.TrimSpace(m.Experience),
			AvatarURL:     c.assetURL(m.AvatarURL),
			AverageRating: m.Rating,
			RatingsCount:  m.RatingsCount,
			IsOnline:      optionalBool(m.IsOnline),
		})
	}

	return masters, domain.RawPayload{Kind: "firm_masters", ExternalID: firmID, RequestPath: path, Body: raw}, nil
}

func (c *Client) mapFirm(firmID string, resp *firmResponse) (*domain.ExternalFirm, error) {
	f := resp.Data.Firm
	name := strings.TrimSpace(f.Name)
	if name == "" {
		return nil, fmt.Errorf("firm %s: empty name", firmID)
	}

	firm := &domain.ExternalFirm{
		// Requested ID wins: detail payloads sometimes carry id 0.
		ExternalID:    firmID,
		Name:          name,
		Category:      f.Category,
		EntityType:    strings.TrimSpace(f.Type),
		URLKey:        f.URLKey,
		Address:       strings.TrimSpace(f.Address),
		Description:   strings.TrimSpace(f.Description),
		AvatarURL:     c.assetURL(f.AvatarURL),
		AverageRating: f.AverageRating,
		RatingsCount:  f.RatingsCount,
		ReviewsCount:  f.ReviewCount,
		WorkStartTime: clockTime(f.WorkStartTime),
		WorkEndTime:   clockTime(f.WorkEndTime),
		IsOnline:      optionalBool(f.IsOnline),
		IsPromoted:    optionalBool(f.IsPromoted),
	}

	if loc := resp.Data.Location; loc != nil {
		firm.Latitude = coordinate(loc.MarkerY)
		firm.Longitude = coordinate(loc.MarkerX)
		firm.MapProvider = loc.Type
	}

	seenPhotos := make(map[string]struct{}, len(f.Pictures))
	for _, p := range f.Pictures {
		u := c.assetURL(p)
		if _, ok := seenPhotos[u]; ok || u == "" {
			continue
		}
		seenPhotos[u] = struct{}{}
		firm.PhotoURLs = append(firm.PhotoURLs, u)
	}

	// Negative IDs are UI pseudo-categories ("Популярные").
	for _, cat := range resp.Data.Categories {
		if cat.ID <= 0 {
			continue
		}
		firm.Categories = append(firm.Categories, domain.ExternalCategory{
			Kind:       domain.SourceCategory,
			ExternalID: strconv.FormatInt(cat.ID, 10),
			Name:       cat.Name,
			IconURL:    c.assetURL(cat.IconURL),
		})
	}

	// Subcategory parent is known only through services.
	parents := make(map[int64]int64, len(resp.Data.Services))
	for _, s := range resp.Data.Services {
		if s.SubCategoryID > 0 && s.CategoryID > 0 {
			parents[s.SubCategoryID] = s.CategoryID
		}
	}
	for _, sub := range resp.Data.SubCategories {
		if sub.ID <= 0 {
			continue
		}
		category := domain.ExternalCategory{
			Kind:       domain.SourceSubcategory,
			ExternalID: strconv.FormatInt(sub.ID, 10),
			Name:       sub.Name,
		}
		if parent, ok := parents[sub.ID]; ok {
			category.ParentExternalID = strconv.FormatInt(parent, 10)
		}
		firm.Categories = append(firm.Categories, category)
	}

	// The same service may be listed under several categories.
	seenServices := make(map[int64]struct{}, len(resp.Data.Services))
	for _, s := range resp.Data.Services {
		if _, ok := seenServices[s.ID]; ok || s.ID <= 0 || strings.TrimSpace(s.Name) == "" {
			continue
		}
		seenServices[s.ID] = struct{}{}
		firm.Services = append(firm.Services, mapService(s))
	}

	return firm, nil
}

func mapService(s service) domain.ExternalService {
	out := domain.ExternalService{
		ExternalID:            strconv.FormatInt(s.ID, 10),
		CategoryExternalID:    optionalID(s.CategoryID),
		SubcategoryExternalID: optionalID(s.SubCategoryID),
		Name:                  strings.TrimSpace(s.Name),
		Description:           strings.TrimSpace(s.Description),
		Currency:              "KZT",
		IsExpress:             optionalBool(s.Express),
	}

	if s.Price > 0 {
		minAmount := tengeToMinor(s.Price)
		out.PriceMinAmount = &minAmount

		maxAmount := minAmount
		if s.PriceMax > s.Price {
			maxAmount = tengeToMinor(s.PriceMax)
		}
		out.PriceMaxAmount = &maxAmount
	}

	if s.Duration > 0 {
		duration := s.Duration
		out.DurationMinutes = &duration
	}

	return out
}

// clockTime extracts "HH:MM" from "22-09-2026 10:00" or "10:00".
func clockTime(raw string) *string {
	raw = strings.TrimSpace(raw)
	if i := strings.LastIndexByte(raw, ' '); i >= 0 {
		raw = raw[i+1:]
	}

	parts := strings.Split(raw, ":")
	if len(parts) < 2 {
		return nil
	}
	hour, err1 := strconv.Atoi(parts[0])
	minute, err2 := strconv.Atoi(parts[1])
	if err1 != nil || err2 != nil || hour < 0 || hour > 23 || minute < 0 || minute > 59 {
		return nil
	}

	out := fmt.Sprintf("%02d:%02d", hour, minute)
	return &out
}

func tengeToMinor(amount float64) int64 {
	return int64(math.Round(amount * 100))
}

// coordinate treats 0 as missing: the source sends 0 instead of null.
func coordinate(v float64) *float64 {
	if v == 0 {
		return nil
	}
	return &v
}

func optionalBool(b flexibleBool) *bool {
	if !b.Set {
		return nil
	}
	v := b.Value
	return &v
}

func optionalID(id int64) string {
	if id <= 0 {
		return ""
	}
	return strconv.FormatInt(id, 10)
}
