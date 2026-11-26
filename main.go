package main

import (
	"context"
	"encoding/json"
	"errors"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// Une prévision pour une heure.
type WeatherHour struct {
	Time        string
	Temperature float64
	RainChance  int
	WindSpeed   float64
}

type FavorableSlot struct {
	Start string
	End   string
}

type Beach struct {
	Name           string
	City           string
	Latitude       float64
	Longitude      float64
	WeatherStatus  string
	Temperature    float64
	Time           string
	RainChance     int
	WindSpeed      float64
	WeatherMessage string
	Department     string
}

var beaches = []Beach{
	// GIRONDE
	{
		Name:       "Plage du Petit Nice",
		City:       "La Teste-de-Buch",
		Department: "Gironde",
		Latitude:   44.54,
		Longitude:  -1.25,
	},
	{
		Name:       "Plage Pereire",
		City:       "Arcachon",
		Department: "Gironde",
		Latitude:   44.65,
		Longitude:  -1.19,
	},
	{
		Name:       "Plage de la Salie",
		City:       "La Teste-de-Buch",
		Department: "Gironde",
		Latitude:   44.51,
		Longitude:  -1.24,
	},
	{
		Name:       "Lacanau Océan",
		City:       "Lacanau",
		Department: "Gironde",
		Latitude:   44.99,
		Longitude:  -1.20,
	},
	{
		Name:       "Plage du Gressier",
		City:       "Le Porge",
		Department: "Gironde",
		Latitude:   44.89,
		Longitude:  -1.24,
	},

	// LANDES
	{
		Name:       "Biscarrosse-Plage",
		City:       "Biscarrosse",
		Department: "Landes",
		Latitude:   44.45,
		Longitude:  -1.25,
	},
	{
		Name:       "Mimizan-Plage",
		City:       "Mimizan",
		Department: "Landes",
		Latitude:   44.21,
		Longitude:  -1.30,
	},
	{
		Name:       "Contis-Plage",
		City:       "Saint-Julien-en-Born",
		Latitude:   44.09,
		Department: "Landes",
		Longitude:  -1.33,
	},
	{
		Name:       "Plage Centrale",
		City:       "Soorts-Hossegor",
		Department: "Landes",
		Latitude:   43.66,
		Longitude:  -1.44,
	},

	// CHARENTE-MARITIME
	{
		Name:       "Grande Plage de Royan",
		City:       "Royan",
		Department: "Charente-Maritime",
		Latitude:   45.62,
		Longitude:  -1.03,
	},
	{
		Name:       "Plage de la Grande Côte",
		City:       "Saint-Palais-sur-Mer",
		Department: "Charente-Maritime",
		Latitude:   45.65,
		Longitude:  -1.13,
	},
	{
		Name:       "Plage des Minimes",
		City:       "La Rochelle",
		Department: "Charente-Maritime",
		Latitude:   46.14,
		Longitude:  -1.17,
	},

	// PYRÉNÉES-ATLANTIQUES
	{
		Name:       "Grande Plage",
		City:       "Biarritz",
		Department: "Pyrénées-Atlantiques",
		Latitude:   43.48,
		Longitude:  -1.56,
	},
	{
		Name:       "Plage d'Ilbarritz",
		City:       "Bidart",
		Department: "Pyrénées-Atlantiques",
		Latitude:   43.44,
		Longitude:  -1.58,
	},
	{
		Name:       "Grande Plage",
		City:       "Hendaye",
		Department: "Pyrénées-Atlantiques",
		Latitude:   43.37,
		Longitude:  -1.78,
	},
}

// Les données utilisées par notre page HTML.
type HomeData struct {
	Weather        []WeatherHour
	WeatherError   string
	FavorableSlots []FavorableSlot
	Beaches        []Beach
	BeachesJSON    template.JS
}

// La structure du JSON renvoyé par Open-Meteo.
type WeatherResponse struct {
	Hourly struct {
		Time        []string  `json:"time"`
		Temperature []float64 `json:"temperature_2m"`
		RainChance  []int     `json:"precipitation_probability"`
		WindSpeed   []float64 `json:"wind_speed_10m"`
	} `json:"hourly"`
}

// Récupérer la météo auprès de l'API.
func getWeather(latitude, longitude float64) ([]WeatherHour, error) {
	params := url.Values{}

	params.Set("latitude", strconv.FormatFloat(latitude, 'f', -1, 64))
	params.Set("longitude", strconv.FormatFloat(longitude, 'f', -1, 64))
	params.Set("hourly", "temperature_2m,precipitation_probability,wind_speed_10m")
	params.Set("forecast_days", "1")
	params.Set("timezone", "Europe/Paris")

	apiURL := "https://api.open-meteo.com/v1/forecast?" + params.Encode()

	// Limiter la durée de la requête.
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, err
	}

	client := &http.Client{
		Timeout: 8 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("réponse météo invalide")
	}

	var data WeatherResponse

	err = json.NewDecoder(resp.Body).Decode(&data)
	if err != nil {
		return nil, err
	}

	// Vérifier que les tableaux contiennent des données.
	if len(data.Hourly.Time) == 0 ||
		len(data.Hourly.Temperature) != len(data.Hourly.Time) ||
		len(data.Hourly.RainChance) != len(data.Hourly.Time) ||
		len(data.Hourly.WindSpeed) != len(data.Hourly.Time) {
		return nil, errors.New("réponse météo invalide")
	}

	var hours []WeatherHour

	// Ne garder que les heures de 9 h à 18 h.
	for i, timeString := range data.Hourly.Time {
		parsedTime, err := time.Parse("2006-01-02T15:04", timeString)
		if err != nil {
			continue
		}
		hours = append(hours, WeatherHour{
			Time:        parsedTime.Format("2006-01-02T15:04"),
			Temperature: data.Hourly.Temperature[i],
			RainChance:  data.Hourly.RainChance[i],
			WindSpeed:   data.Hourly.WindSpeed[i],
		})
	}

	return hours, nil
}

func findFavorableSlots(weather []WeatherHour) []FavorableSlot {
	var slots []FavorableSlot

	var slotStart string
	var slotEnd string
	inSlot := false

	for _, hour := range weather {
		favorable := hour.Temperature >= 22 &&
			hour.RainChance <= 20 &&
			hour.WindSpeed <= 25

		if favorable {
			if !inSlot {
				slotStart = hour.Time
				inSlot = true
			}
			// Chaque observation météo représente une heure.
			parsedTime, err := time.Parse("15h", hour.Time)
			if err == nil {
				slotEnd = parsedTime.Add(time.Hour).Format("15h")
			}
		} else if inSlot {
			slots = append(slots, FavorableSlot{
				Start: slotStart,
				End:   slotEnd,
			})
			inSlot = false
		}
	}

	// Si la dernière heure est favorable, on ferme le créneau ici.
	if inSlot {
		slots = append(slots, FavorableSlot{
			Start: slotStart,
			End:   slotEnd,
		})
	}

	return slots
}

func getWeatherStatus(hour WeatherHour) string {
	if hour.Temperature >= 22 &&
		hour.RainChance < 10 &&
		hour.WindSpeed <= 25 {
		return "green"
	}

	if hour.Temperature >= 18 &&
		hour.RainChance <= 50 &&
		hour.WindSpeed <= 40 {
		return "yellow"
	}

	return "red"
}

func updateBeaches(beaches []Beach) []Beach {
	location, err := time.LoadLocation("Europe/Paris")
	if err != nil {
		log.Println("Erreur de fuseau horaire :", err)
		return beaches
	}

	now := time.Now().In(location)

	for i := range beaches {
		weather, err := getWeather(
			beaches[i].Latitude,
			beaches[i].Longitude,
		)

		if err != nil || len(weather) == 0 {
			beaches[i].WeatherStatus = "red"
			beaches[i].Time = "Indisponible"
			continue
		}

		found := false

		for _, hour := range weather {
			// Convertir l'heure de la prévision en date Go
			parsedTime, err := time.ParseInLocation(
				"2006-01-02T15:04",
				hour.Time,
				location,
			)

			if err != nil {
				continue
			}

			// Vérifier la date et l'heure actuelles
			if parsedTime.Year() == now.Year() &&
				parsedTime.YearDay() == now.YearDay() &&
				parsedTime.Hour() == now.Hour() {

				beaches[i].Temperature = hour.Temperature
				beaches[i].RainChance = hour.RainChance
				beaches[i].WindSpeed = hour.WindSpeed
				beaches[i].Time = parsedTime.Format("15h")
				beaches[i].WeatherStatus = getWeatherStatus(hour)
				switch beaches[i].WeatherStatus {
				case "green":
					beaches[i].WeatherMessage = "Conditions favorables : température agréable, peu de pluie et vent modéré."

				case "yellow":
					beaches[i].WeatherMessage = "Conditions moyennes : " +
						strconv.Itoa(hour.RainChance) + "% de risque de pluie, " +
						strconv.Itoa(int(hour.Temperature)) + "°C et " +
						strconv.Itoa(int(hour.WindSpeed)) + " km/h de vent."

				case "red":
					beaches[i].WeatherMessage = "Conditions défavorables : " +
						strconv.Itoa(hour.RainChance) + "% de risque de pluie, " +
						strconv.Itoa(int(hour.Temperature)) + "°C et " +
						strconv.Itoa(int(hour.WindSpeed)) + " km/h de vent."
				}

				found = true
				break
			}
		}

		if !found {
			beaches[i].WeatherStatus = "red"
			beaches[i].Time = "Indisponible"
		}
	}

	return beaches
}

func main() {
	tmpl, err := template.ParseFiles("templates/home.html")
	if err != nil {
		log.Fatal(err)
	}

	// Page d'accueil.
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}

		data := HomeData{}
		data.Beaches = updateBeaches(beaches)

		// Transformer les plages en JSON pour JavaScript.
		beachesJSON, err := json.Marshal(data.Beaches)
		if err != nil {
			log.Println("Erreur JSON :", err)
			http.Error(w, "Erreur lors de la préparation des plages", http.StatusInternalServerError)
			return
		}

		data.BeachesJSON = template.JS(beachesJSON)
		log.Println("Plages JSON :", string(beachesJSON))

		weather, err := getWeather(44.54, -1.25)
		if err != nil {
			log.Println("Erreur météo :", err)
			data.WeatherError = "Les prévisions sont momentanément indisponibles."
		} else {
			data.Weather = weather
		}

		data.FavorableSlots = findFavorableSlots(data.Weather)

		err = tmpl.Execute(w, data)
		if err != nil {
			log.Println("Erreur template :", err)
		}
	})

	// Fichiers CSS et autres fichiers statiques.
	http.Handle("/static/", http.StripPrefix(
		"/static/",
		http.FileServer(http.Dir("./static")),
	))

	log.Println("BeachTime démarre sur http://localhost:8080")

	err = http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatal(err)
	}
}
