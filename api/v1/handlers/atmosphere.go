// handlers/atmosphere.go
package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

type ForecastResponse struct {
	Name     string     `json:"name"`
	Forecast []Forecast `json:"forecast"`
}

type Forecast struct {
	Condition   Condition   `json:"condition"`
	Temperature Temperature `json:"temperature"`
	Sunrise     int64       `json:"sunrise"`
	Sunset      int64       `json:"sunset"`
	DayOfWeek   int         `json:"dayOfWeek"`
}

type Condition struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type Temperature struct {
	Current float64 `json:"current"`
	Min     float64 `json:"min"`
	Max     float64 `json:"max"`
}

type CurrentWeatherResponse struct {
	Name        string      `json:"name"`
	Condition   Condition   `json:"condition"`
	Temperature Temperature `json:"temperature"`
	Sunrise     int64       `json:"sunrise"`
	Sunset      int64       `json:"sunset"`
}

type AirPollutionResponse struct {
	Components map[string]float64 `json:"components"`
}

// Helper function to get the day of the week from a timestamp
func getDayOfWeek(timestamp int64) int {
	return int(time.Unix(timestamp, 0).Weekday())
}

/* Example response:
{
	"name": "Nowy Sącz",
	"forecast": [
		{
			"condition": {
				name: "Rain",
				description: "light rain",
			},
			"temperature": {
				current: 12,
				min: 10,
				max: 14,
			},
			"sunrise": 1726636384,
			"sunset": 1726680975,
			dayOfWeek: 0, // 0 - Sunday, 1 - Monday, ..., 6 - Saturday
		},
		{
			"condition": {
				name: "Rain",
				description: "light rain",
			},
			"temperature": {
				current: 12,
				min: 10,
				max: 14,
			},
			"sunrise": 1726636384,
			"sunset": 1726680975,
			dayOfWeek: 1,
		}
	]
}
*/
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

	resp, err := http.Get(url)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch forecast data"})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		c.JSON(resp.StatusCode, gin.H{"error": "Failed to fetch forecast data"})
		return
	}

	var openWeatherData map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&openWeatherData); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse forecast data"})
		return
	}

	forecastList := openWeatherData["list"].([]interface{})
	forecastResponse := ForecastResponse{Name: openWeatherData["city"].(map[string]interface{})["name"].(string)}

	for i, forecast := range forecastList {
		if i >= days {
			break
		}
		forecastMap := forecast.(map[string]interface{})
		condition := forecastMap["weather"].([]interface{})[0].(map[string]interface{})
		temperature := forecastMap["main"].(map[string]interface{})
		sunrise := int64(openWeatherData["city"].(map[string]interface{})["sunrise"].(float64))
		sunset := int64(openWeatherData["city"].(map[string]interface{})["sunset"].(float64))

		forecastResponse.Forecast = append(forecastResponse.Forecast, Forecast{
			Condition: Condition{
				Name:        condition["main"].(string),
				Description: condition["description"].(string),
			},
			Temperature: Temperature{
				Current: temperature["temp"].(float64),
				Min:     temperature["temp_min"].(float64),
				Max:     temperature["temp_max"].(float64),
			},
			Sunrise:   sunrise,
			Sunset:    sunset,
			DayOfWeek: getDayOfWeek(int64(forecastMap["dt"].(float64))),
		})
	}

	c.JSON(http.StatusOK, forecastResponse)
}

/* Example response:
	{
		"name": "Nowy Sącz",
		"condition": {
			name: "Rain",
			description: "light rain",
		},
		"temperature": {
			current: 12,
			min: 10,
			max: 14,
		},
		"sunrise": 1726636384,
		"sunset": 1726680975,
	}
*/
func CurrentWeatherHandler(c *gin.Context) {
	lang := c.DefaultQuery("lang", "en")
	units := c.DefaultQuery("units", "metric")
	apiKey := os.Getenv("OPENWEATHER_API_KEY")

	url := fmt.Sprintf("%s%s",
		Config.OpenWeather.BaseUrl,
		fmt.Sprintf(Config.OpenWeather.Endpoints.CurrentWeather, Config.OpenWeather.Lat, Config.OpenWeather.Lon, apiKey, lang, units),
	)

	resp, err := http.Get(url)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch current weather data"})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		c.JSON(resp.StatusCode, gin.H{"error": "Failed to fetch current weather data"})
		return
	}

	var openWeatherData map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&openWeatherData); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse current weather data"})
		return
	}

	condition := openWeatherData["weather"].([]interface{})[0].(map[string]interface{})
	temperature := openWeatherData["main"].(map[string]interface{})
	sunrise := int64(openWeatherData["sys"].(map[string]interface{})["sunrise"].(float64))
	sunset := int64(openWeatherData["sys"].(map[string]interface{})["sunset"].(float64))

	currentWeatherResponse := CurrentWeatherResponse{
		Name: openWeatherData["name"].(string),
		Condition: Condition{
			Name:        condition["main"].(string),
			Description: condition["description"].(string),
		},
		Temperature: Temperature{
			Current: temperature["temp"].(float64),
			Min:     temperature["temp_min"].(float64),
			Max:     temperature["temp_max"].(float64),
		},
		Sunrise: sunrise,
		Sunset:  sunset,
	}

	c.JSON(http.StatusOK, currentWeatherResponse)
}


/* Example response:
	{
		"components":{
			"co": 201.94053649902344,
			"no": 0.01877197064459324,
			"no2": 0.7711350917816162,
			"o3": 68.66455078125,
			"so2": 0.6407499313354492,
			"pm2_5": 0.5,
			"pm10": 0.540438711643219,
			"nh3": 0.12369127571582794
		}
	}
*/
func CurrentAirPollutionHandler(c *gin.Context) {
	apiKey := os.Getenv("OPENWEATHER_API_KEY")

	url := fmt.Sprintf("%s%s",
		Config.OpenWeather.BaseUrl,
		fmt.Sprintf(Config.OpenWeather.Endpoints.CurrentAirPollution, Config.OpenWeather.Lat, Config.OpenWeather.Lon, apiKey),
	)
	fmt.Println(url)
	resp, err := http.Get(url)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch air pollution data"})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		c.JSON(resp.StatusCode, gin.H{"error": "Failed to fetch air pollution data"})
		return
	}

	var openWeatherData map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&openWeatherData); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse air pollution data"})
		return
	}

	components := openWeatherData["list"].([]interface{})[0].(map[string]interface{})["components"].(map[string]interface{})

	airPollutionResponse := AirPollutionResponse{
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

	c.JSON(http.StatusOK, airPollutionResponse)
}