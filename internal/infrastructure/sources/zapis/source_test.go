package zapis

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// Shapes follow real catalogue responses; values are synthetic.
const (
	citiesJSON = `{"data":{"cities":[{"id":1,"name":"Алматы","urlName":"almaty","latitude":43.24,"longitude":76.91}]}}`
	searchJSON = `{"data":{"firmIds":[692,15],"firms":[{"id":15},{"id":77}],"suggestions":[]}}`
	firmJSON   = `{"data":{
		"firm":{"id":0,"name":" Test Salon ","type":"Салон красоты","category":"SALON","urlKey":"test-692",
			"address":"Абая 10","avatarUrl":"/data/pics/salon/a.jpg?v=5","pictures":["/data/pics/1.jpg","/data/pics/1.jpg"],
			"averageRating":4.7,"ratingsCount":1609,"reviewCount":310,
			"workStartTime":"22-09-2026 10:00","workEndTime":"22-09-2026 21:00","isOnline":true,"isPromoted":1},
		"location":{"type":"TWO_GIS","markerX":76.941,"markerY":43.26},
		"categories":[{"id":-1,"name":"Популярные"},{"id":2,"name":"Ногти","iconUrl":"/static/ic_2.png"}],
		"subCategories":[{"id":81,"name":"Педикюр"}],
		"services":[
			{"id":52672,"name":"Педикюр","price":9500,"priceMax":12000,"duration":60,"categoryId":2,"subCategoryId":81,"express":null},
			{"id":52673,"name":"Экспресс","price":3000.5,"priceMax":0,"duration":0,"categoryId":2,"subCategoryId":0,"express":0},
			{"id":52672,"name":"Педикюр","price":9500,"duration":60,"categoryId":-1},
			{"id":0,"name":"broken"}
		]}}`
	mastersJSON = `{"data":{"masters":[
		{"id":86112,"name":"Айгерим","surname":"","profession":"Nail-Стилист","experience":"6 л.","avatarUrl":"/data/pics/m.jpg","rating":4.85,"ratingsCount":208,"isOnline":true},
		{"id":0,"name":"no id"}],"professions":[]}}`
)

func newTestClient(t *testing.T) *Client {
	t.Helper()

	routes := map[string]string{
		"/screen/home/cities": citiesJSON,
		"/firms/search":       searchJSON,
		"/firms/692":          firmJSON,
		"/firms/692/masters":  mastersJSON,
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/screen/home/cities" && r.Header.Get("city_id") != "1" {
			http.Error(w, "missing city_id", http.StatusBadRequest)
			return
		}
		body, ok := routes[r.URL.Path]
		if !ok {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)

	c, err := New(Config{BaseURL: srv.URL, AssetBaseURL: "https://zapis.kz"})
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestCitiesAndFirmIDs(t *testing.T) {
	c := newTestClient(t)
	ctx := context.Background()

	cities, raw, err := c.Cities(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(cities) != 1 || cities[0].ExternalID != "1" || cities[0].Slug != "almaty" || !json.Valid(raw.Body) {
		t.Fatalf("cities = %+v", cities)
	}

	ids, _, err := c.FirmIDs(ctx, "1")
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"692", "15", "77"}
	if len(ids) != len(want) {
		t.Fatalf("ids = %v; want %v", ids, want)
	}
	for i := range want {
		if ids[i] != want[i] {
			t.Fatalf("ids = %v; want %v", ids, want)
		}
	}
}

func TestFirmMapping(t *testing.T) {
	firm, raw, err := newTestClient(t).Firm(context.Background(), "1", "692")
	if err != nil {
		t.Fatal(err)
	}

	if firm.ExternalID != "692" || firm.Name != "Test Salon" || firm.Category != "SALON" {
		t.Errorf("identity = %q %q %q", firm.ExternalID, firm.Name, firm.Category)
	}
	if raw.Kind != "firm" || raw.ExternalID != "692" {
		t.Errorf("raw = %+v", raw)
	}
	if firm.WorkStartTime == nil || *firm.WorkStartTime != "10:00" || *firm.WorkEndTime != "21:00" {
		t.Errorf("work time = %v - %v", firm.WorkStartTime, firm.WorkEndTime)
	}
	if firm.IsPromoted == nil || !*firm.IsPromoted {
		t.Errorf("isPromoted 1 must map to true")
	}
	if firm.AvatarURL != "https://zapis.kz/data/pics/salon/a.jpg?v=5" {
		t.Errorf("avatar = %s", firm.AvatarURL)
	}
	if len(firm.PhotoURLs) != 1 {
		t.Errorf("photos must be deduplicated: %v", firm.PhotoURLs)
	}
	if firm.Latitude == nil || *firm.Latitude != 43.26 || *firm.Longitude != 76.941 {
		t.Errorf("location = %v %v", firm.Latitude, firm.Longitude)
	}

	if len(firm.Categories) != 2 {
		t.Fatalf("categories = %+v", firm.Categories)
	}
	if sub := firm.Categories[1]; sub.ExternalID != "81" || sub.ParentExternalID != "2" {
		t.Errorf("subcategory = %+v", sub)
	}

	if len(firm.Services) != 2 {
		t.Fatalf("services = %+v", firm.Services)
	}
	s := firm.Services[0]
	if *s.PriceMinAmount != 950000 || *s.PriceMaxAmount != 1200000 || *s.DurationMinutes != 60 || s.IsExpress != nil {
		t.Errorf("service 0 = %+v", s)
	}
	s = firm.Services[1]
	if *s.PriceMinAmount != 300050 || *s.PriceMaxAmount != 300050 || s.DurationMinutes != nil || s.IsExpress == nil || *s.IsExpress {
		t.Errorf("service 1 = %+v", s)
	}
}

func TestFirmMasters(t *testing.T) {
	masters, _, err := newTestClient(t).FirmMasters(context.Background(), "1", "692")
	if err != nil {
		t.Fatal(err)
	}
	if len(masters) != 1 {
		t.Fatalf("masters = %+v", masters)
	}
	m := masters[0]
	if m.DisplayName != "Айгерим" || m.Profession != "Nail-Стилист" || *m.AverageRating != 4.85 || m.AvatarURL != "https://zapis.kz/data/pics/m.jpg" {
		t.Errorf("master = %+v", m)
	}
}

func TestSourceErrorStatus(t *testing.T) {
	if _, _, err := newTestClient(t).Firm(context.Background(), "1", "404"); err == nil {
		t.Fatal("want error for 404")
	}
}

func TestClockTime(t *testing.T) {
	cases := map[string]string{"22-09-2026 10:00": "10:00", "9:05": "09:05", "": "", "25:00": "", "garbage": ""}
	for in, want := range cases {
		got := clockTime(in)
		if (got == nil && want != "") || (got != nil && *got != want) {
			t.Errorf("clockTime(%q) = %v; want %q", in, got, want)
		}
	}
}
