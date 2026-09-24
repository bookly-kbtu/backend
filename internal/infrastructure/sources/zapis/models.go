package zapis

import (
	"encoding/json"
	"fmt"
)

// Response shapes of the public zapis.kz catalogue (observed, not an official contract).

// flexibleBool accepts true/false, 0/1 and null: the catalogue mixes them.
type flexibleBool struct {
	Value bool
	Set   bool
}

func (b *flexibleBool) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		*b = flexibleBool{}
		return nil
	}

	var boolean bool
	if err := json.Unmarshal(data, &boolean); err == nil {
		*b = flexibleBool{Value: boolean, Set: true}
		return nil
	}

	var number float64
	if err := json.Unmarshal(data, &number); err == nil {
		*b = flexibleBool{Value: number != 0, Set: true}
		return nil
	}

	return fmt.Errorf("expected boolean, number or null, got %s", data)
}

type city struct {
	ID        int     `json:"id"`
	Name      string  `json:"name"`
	URLName   string  `json:"urlName"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

type citiesResponse struct {
	Data struct {
		Cities []city `json:"cities"`
	} `json:"data"`
}

type searchResponse struct {
	Data struct {
		FirmIDs []int64 `json:"firmIds"`
		Firms   []struct {
			ID int64 `json:"id"`
		} `json:"firms"`
	} `json:"data"`
}

type firmResponse struct {
	Data struct {
		Firm struct {
			ID            int64    `json:"id"`
			Name          string   `json:"name"`
			Type          string   `json:"type"`
			Category      string   `json:"category"`
			URLKey        string   `json:"urlKey"`
			Address       string   `json:"address"`
			Description   string   `json:"description"`
			AvatarURL     string   `json:"avatarUrl"`
			Pictures      []string `json:"pictures"`
			AverageRating *float64 `json:"averageRating"`
			RatingsCount  *int     `json:"ratingsCount"`
			ReviewCount   *int     `json:"reviewCount"`
			// "22-09-2026 10:00": today's date + time, not a plain time.
			WorkStartTime string       `json:"workStartTime"`
			WorkEndTime   string       `json:"workEndTime"`
			IsOnline      flexibleBool `json:"isOnline"`
			IsPromoted    flexibleBool `json:"isPromoted"`
		} `json:"firm"`
		Location *struct {
			Type    string  `json:"type"`
			MarkerX float64 `json:"markerX"` // longitude
			MarkerY float64 `json:"markerY"` // latitude
		} `json:"location"`
		Services      []service `json:"services"`
		Categories    []idName  `json:"categories"`
		SubCategories []idName  `json:"subCategories"`
	} `json:"data"`
}

type idName struct {
	ID      int64  `json:"id"`
	Name    string `json:"name"`
	IconURL string `json:"iconUrl"`
}

type service struct {
	ID            int64        `json:"id"`
	CategoryID    int64        `json:"categoryId"`
	SubCategoryID int64        `json:"subCategoryId"`
	Name          string       `json:"name"`
	Description   string       `json:"description"`
	Price         float64      `json:"price"`    // tenge
	PriceMax      float64      `json:"priceMax"` // tenge
	Duration      int          `json:"duration"` // minutes
	Express       flexibleBool `json:"express"`
}

type mastersResponse struct {
	Data struct {
		Masters []master `json:"masters"`
	} `json:"data"`
}

type master struct {
	ID           int64        `json:"id"`
	Name         string       `json:"name"`
	Surname      string       `json:"surname"`
	Profession   string       `json:"profession"`
	Experience   string       `json:"experience"`
	AvatarURL    string       `json:"avatarUrl"`
	Rating       *float64     `json:"rating"`
	RatingsCount *int         `json:"ratingsCount"`
	IsOnline     flexibleBool `json:"isOnline"`
}
