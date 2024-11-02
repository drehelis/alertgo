package types

type ThreatAlert struct {
	ID    string   `json:"id"`
	Cat   string   `json:"cat"`
	Title string   `json:"title"`
	Data  []string `json:"data"`
	Desc  string   `json:"desc"`
}

type MessageState struct {
	ID        string
	MessageID string
	Locations []string
	Content   string
	HasPhoto  bool
	MapURL    string
}

type GeocodingResult struct {
	Results []struct {
		Geometry struct {
			Location struct {
				Lat float64 `json:"lat"`
				Lng float64 `json:"lng"`
			} `json:"location"`
		} `json:"geometry"`
	} `json:"results"`
}
