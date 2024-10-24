// handlers/atmosphere.go
package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"smuggr.xyz/optivum-bsf/common/models"

	"github.com/gin-gonic/gin"
)

// Helper function to get the day of the week from a timestamp
func getDayOfWeek(timestamp int64) int64 {
	return int64(time.Unix(timestamp, 0).Weekday())
}

func WeatherForecastHandler(c *gin.Context) {
	lang := c.DefaultQuery("lang", "en")
	units := c.DefaultQuery("units", "metric")
	apiKey := os.Getenv("OPENWEATHER_API_KEY")

	daysQuery := c.DefaultQuery("days", "3")
	var days int
	if d, err := strconv.Atoi(daysQuery); err == nil {
		days = d
	} else {
		days = 1
	}

	if days < 1 {
		days = 1
	} else if days > 5 {
		days = 5
	}

	url := fmt.Sprintf("%s%s",
		Config.OpenWeather.BaseUrl,
		fmt.Sprintf(Config.OpenWeather.Endpoints.ForecastWeather, Config.OpenWeather.Lat, Config.OpenWeather.Lon, apiKey, lang, units),
	)

	// #nosec G107
	resp, err := http.Get(url)
	if err != nil {
		Respond(c, http.StatusInternalServerError, models.APIResponse{
			Message: "failed to fetch forecast data",
			Success: false,
		})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		Respond(c, http.StatusInternalServerError, models.APIResponse{
			Message: "failed to fetch forecast data",
			Success: false,
		})
		return
	}

	var openWeatherData map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&openWeatherData); err != nil {
		Respond(c, http.StatusInternalServerError, models.APIResponse{
			Message: "failed to parse forecast data",
			Success: false,
		})
		return
	}

	forecastList := openWeatherData["list"].([]interface{})
	forecastResponse := &models.ForecastResponse{Name: openWeatherData["city"].(map[string]interface{})["name"].(string)}

	for i, forecast := range forecastList {
		if i >= days {
			break
		}
		forecastMap := forecast.(map[string]interface{})
		condition := forecastMap["weather"].([]interface{})[0].(map[string]interface{})
		temperature := forecastMap["main"].(map[string]interface{})
		sunrise := int64(openWeatherData["city"].(map[string]interface{})["sunrise"].(float64))
		sunset := int64(openWeatherData["city"].(map[string]interface{})["sunset"].(float64))

		forecastResponse.Forecast = append(forecastResponse.Forecast, &models.Forecast{
			Condition: &models.Condition{
				Name:        condition["main"].(string),
				Description: condition["description"].(string),
			},
			Temperature: &models.Temperature{
				Current: temperature["temp"].(float64),
				Min:     temperature["temp_min"].(float64),
				Max:     temperature["temp_max"].(float64),
			},
			Sunrise:   sunrise,
			Sunset:    sunset,
			DayOfWeek: getDayOfWeek(forecastMap["dt"].(int64)),
		})
	}

	Respond(c, http.StatusOK, forecastResponse)
}

func CurrentWeatherHandler(c *gin.Context) {
	lang := c.DefaultQuery("lang", "en")
	units := c.DefaultQuery("units", "metric")
	apiKey := os.Getenv("OPENWEATHER_API_KEY")

	url := fmt.Sprintf("%s%s",
		Config.OpenWeather.BaseUrl,
		fmt.Sprintf(Config.OpenWeather.Endpoints.CurrentWeather, Config.OpenWeather.Lat, Config.OpenWeather.Lon, apiKey, lang, units),
	)

	// #nosec G107 - URL is constructed from trusted configuration and validated
	resp, err := http.Get(url)
	if err != nil {
		Respond(c, http.StatusInternalServerError, models.APIResponse{
			Message: "failed to fetch current weather data",
			Success: false,
		})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		Respond(c, http.StatusInternalServerError, models.APIResponse{
			Message: "failed to fetch current weather data",
			Success: false,
		})
		return
	}

	var openWeatherData map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&openWeatherData); err != nil {
		Respond(c, http.StatusInternalServerError, models.APIResponse{
			Message: "failed to parse current weather data",
			Success: false,
		})
		return
	}

	condition := openWeatherData["weather"].([]interface{})[0].(map[string]interface{})
	temperature := openWeatherData["main"].(map[string]interface{})
	sunrise := int64(openWeatherData["sys"].(map[string]interface{})["sunrise"].(float64))
	sunset := int64(openWeatherData["sys"].(map[string]interface{})["sunset"].(float64))

	currentWeatherResponse := &models.CurrentWeatherResponse{
		Name: openWeatherData["name"].(string),
		Condition: &models.Condition{
			Name:        condition["main"].(string),
			Description: condition["description"].(string),
		},
		Temperature: &models.Temperature{
			Current: temperature["temp"].(float64),
			Min:     temperature["temp_min"].(float64),
			Max:     temperature["temp_max"].(float64),
		},
		Sunrise: sunrise,
		Sunset:  sunset,
	}

	Respond(c, http.StatusOK, currentWeatherResponse)
}

func CurrentAirPollutionHandler(c *gin.Context) {
	apiKey := os.Getenv("OPENWEATHER_API_KEY")

	url := fmt.Sprintf("%s%s",
		Config.OpenWeather.BaseUrl,
		fmt.Sprintf(Config.OpenWeather.Endpoints.CurrentAirPollution, Config.OpenWeather.Lat, Config.OpenWeather.Lon, apiKey),
	)

	// #nosec G107
	resp, err := http.Get(url)
	if err != nil {
		Respond(c, http.StatusInternalServerError, models.APIResponse{
			Message: "failed to fetch air pollution data",
			Success: false,
		})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		Respond(c, http.StatusInternalServerError, models.APIResponse{
			Message: "failed to fetch air pollution data",
			Success: false,
		})
		return
	}

	var openWeatherData map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&openWeatherData); err != nil {
		Respond(c, http.StatusInternalServerError, models.APIResponse{
			Message: "failed to parse air pollution data",
			Success: false,
		})
		return
	}

	components := openWeatherData["list"].([]interface{})[0].(map[string]interface{})["components"].(map[string]interface{})

	airPollutionResponse := &models.AirPollutionResponse{
		Components: map[string]float64{
			"co":    components["co"].(float64),
			"no":    components["no"].(float64),
			"no2":   components["no2"].(float64),
			"o3":    components["o3"].(float64),
			"so2":   components["so2"].(float64),
			"pm2_5": components["pm2_5"].(float64),
			"pm10":  components["pm10"].(float64),
			"nh3":   components["nh3"].(float64),
		},
	}

	Respond(c, http.StatusOK, airPollutionResponse)
}
