package station

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/nazazulfiqi/be-mrt-schedules/common/client"
)

type Service interface {
	GetAllStation() (response []StationResponse, err error)
	CheckSchedulesByStation(req RouteRequest) (response []ScheduleResponse, err error)
}

type service struct {
	client *http.Client
}

func NewService() Service {
	return &service{
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (s *service) GetAllStation() (response []StationResponse, err error) {
	url := "https://beweb-dev.jakartamrt.co.id/middleware/api/datum?fields[]=id&fields[]=slug&fields[]=name&filters[field][slug]=stasiun&locale=id"

	byteResponse, err := client.DoRequest(s.client, url)
	if err != nil {
		return
	}

	// API returns {"data": [{"id":..., "slug":..., "name":...}, ...], "meta":{...}}
	var parsed struct {
		Data []struct {
			ID   int    `json:"id"`
			Slug string `json:"slug"`
			Name string `json:"name"`
		} `json:"data"`
	}

	err = json.Unmarshal(byteResponse, &parsed)
	if err != nil {
		return
	}

	for _, item := range parsed.Data {
		name := strings.TrimSpace(item.Name)
		response = append(response, StationResponse{
			Id:   strconv.Itoa(item.ID),
			Name: name,
			Slug: item.Slug,
		})
	}

	return
}

func (s *service) CheckSchedulesByStation(req RouteRequest) (response []ScheduleResponse, err error) {
	url := "https://beweb-dev.jakartamrt.co.id/middleware/api/route"

	// build payload
	payload := map[string]string{
		"type": req.Type,
		"from": req.From,
		"to":   req.To,
	}
	if req.Datetime != "" {
		payload["datetime"] = req.Datetime
	}

	bodyReq, err := json.Marshal(payload)
	if err != nil {
		return
	}

	httpResp, err := s.client.Post(url, "application/json", strings.NewReader(string(bodyReq)))
	if err != nil {
		return
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode != http.StatusOK {
		err = errors.New("unexpected status code: " + httpResp.Status)
		return
	}

	bodyBytes, err := ioutil.ReadAll(httpResp.Body)
	if err != nil {
		return
	}
	if err != nil {
		return
	}

	// parse response
	var parsed struct {
		Success bool `json:"success"`
		Data    struct {
			From struct {
				ID     int    `json:"id"`
				Name   string `json:"name"`
				Object struct {
					Schedule struct {
						WeekdaysEnd   string `json:"weekdaysEnd"`
						WeekendsEnd   string `json:"weekendsEnd"`
						WeekdaysStart string `json:"weekdaysStart"`
						WeekendsStart string `json:"weekendsStart"`
						End           string `json:"end"`
						Start         string `json:"start"`
					} `json:"schedule"`
				} `json:"object"`
			} `json:"from"`
			To struct {
				ID     int    `json:"id"`
				Name   string `json:"name"`
				Object struct {
					Schedule struct {
						WeekdaysEnd   string `json:"weekdaysEnd"`
						WeekendsEnd   string `json:"weekendsEnd"`
						WeekdaysStart string `json:"weekdaysStart"`
						WeekendsStart string `json:"weekendsStart"`
						End           string `json:"end"`
						Start         string `json:"start"`
					} `json:"schedule"`
				} `json:"object"`
			} `json:"to"`
		} `json:"data"`
	}

	err = json.Unmarshal(bodyBytes, &parsed)
	if err != nil {
		return
	}

	// choose schedule strings
	fromSchedule := parsed.Data.From.Object.Schedule.WeekdaysEnd
	if fromSchedule == "" {
		fromSchedule = parsed.Data.From.Object.Schedule.End
	}
	toSchedule := parsed.Data.To.Object.Schedule.WeekdaysStart
	if toSchedule == "" {
		toSchedule = parsed.Data.To.Object.Schedule.Start
	}

	// parse schedule strings to times
	fromParsed, err := ConvertScheduleToTimeFormat(fromSchedule)
	if err != nil {
		return
	}
	toParsed, err := ConvertScheduleToTimeFormat(toSchedule)
	if err != nil {
		return
	}

	// determine base time (now or provided datetime)
	baseTime := time.Now()
	if req.Datetime != "" {
		// try parse with several layouts
		layouts := []string{"2006-01-02T15:04", time.RFC3339}
		var parsedTime time.Time
		var pErr error
		for _, l := range layouts {
			parsedTime, pErr = time.Parse(l, req.Datetime)
			if pErr == nil {
				baseTime = parsedTime
				break
			}
		}
		_ = pErr
	}

	// build response (times after baseTime)
	fromName := strings.TrimSpace(parsed.Data.From.Name)
	toName := strings.TrimSpace(parsed.Data.To.Name)

	for _, t := range fromParsed {
		if t.Format("15:04") > baseTime.Format("15:04") {
			response = append(response, ScheduleResponse{StationName: fromName, Time: t.Format("15:04")})
		}
	}
	for _, t := range toParsed {
		if t.Format("15:04") > baseTime.Format("15:04") {
			response = append(response, ScheduleResponse{StationName: toName, Time: t.Format("15:04")})
		}
	}

	return
}

func ConvertDataToResponse(schedule Schedule) (response []ScheduleResponse, err error) {
	var (
		LebakBulusTripName = "Stasiun Lebak Bulus Grab"
		BundaranHITripName = "Stasiun Bundaran HI Bank DKI"
	)

	scheduleLebakBulus := schedule.ScheduleLebakBulus
	scheduleBundaranHI := schedule.ScheduleBundaranHI

	scheduleLebakBulusParsed, err := ConvertScheduleToTimeFormat(scheduleLebakBulus)
	if err != nil {
		return
	}

	scheduleBundaranHIParsed, err := ConvertScheduleToTimeFormat(scheduleBundaranHI)
	if err != nil {
		return
	}

	// convert to response
	for _, item := range scheduleLebakBulusParsed {
		if item.Format("15:04") > time.Now().Format("15:04") {
			response = append(response, ScheduleResponse{
				StationName: LebakBulusTripName,
				Time:        item.Format("15:04"),
			})
		}
	}

	for _, item := range scheduleBundaranHIParsed {
		if item.Format("15:04") > time.Now().Format("15:04") {
			response = append(response, ScheduleResponse{
				StationName: BundaranHITripName,
				Time:        item.Format("15:04"),
			})
		}
	}

	return

}

func ConvertScheduleToTimeFormat(schedule string) (response []time.Time, err error) {
	// Regex untuk cari waktu dalam format HH:MM (misal 09:26)
	regex := regexp.MustCompile(`\b\d{2}:\d{2}\b`)
	matches := regex.FindAllString(schedule, -1)

	if len(matches) == 0 {
		err = errors.New("no valid time found in schedule")
		return
	}

	for _, match := range matches {
		parsedTime, err := time.Parse("15:04", match)
		if err != nil {
			err = errors.New("invalid time format: " + match)
			return nil, err
		}
		response = append(response, parsedTime)
	}

	return
}
