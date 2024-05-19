package main

// Importaciones necesarias para el proyecto.
import (
	"fmt" // Paquete para formatear texto y datos.
	"log" // Paquete para registrar mensajes de registro.
	"regexp" // Paquete para manejar expresiones regulares.
	"strconv" // Paquete para convertir cadenas a números.
	"strings" // Paquete para manipular cadenas de texto.
	
	"github.com/PuerkitoBio/goquery" // Paquete para hacer scraping en documentos HTML.
	"github.com/gocolly/colly" // Paquete para scraping web.
	"github.com/gofiber/fiber/v2" // Paquete para construir aplicaciones web rápidas.
)

// Date representa una fecha con día y mes.
type Date struct {
	Day   int `json:"day"`   // Día de la fecha.
	Month int `json:"month"` // Mes de la fecha.
}

// Event representa un evento histórico.
type Event struct {
	IsBeforeJesus bool   `json:"isBeforeJesus"` // Indica si el evento ocurrió antes de Jesús.
	Year          int    `json:"year"`          // Año del evento.
	Content       string `json:"content"`       // Contenido del evento.
}

func main() {
	app := fiber.New()

	// Middleware para manejar CORS (Cross-Origin Resource Sharing).
	app.Use(func(c *fiber.Ctx) error {
		c.Set("Access-Control-Allow-Origin", "*")
		c.Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE")
		c.Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if c.Method() == "OPTIONS" {
			return c.SendStatus(fiber.StatusNoContent)
		}
		return c.Next()
	})

	// Ruta para recibir la fecha y devolver datos.
	app.Post("/endpoint", func(c *fiber.Ctx) error {
		var date Date

		// Parsear el cuerpo de la solicitud a una estructura Date.
		if err := c.BodyParser(&date); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Cannot parse JSON",
			})
		}

		// Formatear la fecha.
		formattedDate := fmt.Sprintf("%d_de_%s", date.Day, getMonthName(date.Month))

		// Obtener datos a partir de la fecha.
		scrapedData := scrapeData(date)

		// Devolver los datos obtenidos.
		return c.JSON(fiber.Map{
			"message":     "Fecha recibida",
			"date":        formattedDate,
			"dateData":    date,
			"scrapedData": scrapedData,
		})
	})

	// Iniciar el servidor en el puerto 3000.
	log.Println("Servidor escuchando en http://localhost:3000")
	app.Listen(":3000")
}

// Función para obtener el nombre del mes.
func getMonthName(month int) string {
	months := [...]string{"", "enero", "febrero", "marzo", "abril", "mayo", "junio", "julio", "agosto", "septiembre", "octubre", "noviembre", "diciembre"}
	if month >= 1 && month <= 12 {
		return months[month]
	}
	return ""
}

// Función para extraer datos de Wikipedia basados en una fecha.
func scrapeData(date Date) map[string][]Event {
	col := colly.NewCollector()
	var flag bool
	var events []Event
	var births []Event
	var deaths []Event

	// Configurar eventos para el colector.
	col.OnRequest(func(r *colly.Request) {
		fmt.Println("Visiting: ", r.URL)
	})

	col.OnError(func(_ *colly.Response, err error) {
		log.Println("Something went wrong: ", err)
	})

	col.OnResponse(func(r *colly.Response) {
		fmt.Println("Page visited: ", r.Request.URL)
	})

	// Buscar elementos HTML específicos y extraer datos.
	col.OnHTML("span.mw-headline", func(e *colly.HTMLElement) {
		switch e.Attr("id") {
		case "Acontecimientos":
			flag = true
			e.DOM.Parent().NextAll().Each(func(_ int, s *goquery.Selection) {
				// Verificar si el siguiente elemento es un encabezado.
				if s.Is("h2") {
					flag = false
				}
				// Buscar todos los elementos <li> dentro de <ul>.
				if flag {
					s.Find("li").Each(func(_ int, li *goquery.Selection) {
						event := parseEvent(li.Text())
						if event.Content != "" {
							events = append(events, event)
							fmt.Println("Acontecimiento encontrado:", event)
						}
					})
				}

			})
		case "Nacimientos":
			flag = true
			e.DOM.Parent().NextAll().Each(func(_ int, s *goquery.Selection) {
				// Verificar si el siguiente elemento es un encabezado.
				if s.Is("h2") {
					flag = false
				}
				// Buscar todos los elementos <li> dentro de <ul>.
				if flag {
					s.Find("li").Each(func(_ int, li *goquery.Selection) {
						birth := parseEvent(li.Text())
						if birth.Content != "" {
							births = append(births, birth)
							fmt.Println("Nacimiento encontrado:", birth)
						}
					})
				}

			})

		case "Fallecimientos":
			flag = true
			e.DOM.Parent().NextAll().Each(func(_ int, s *goquery.Selection) {
				// Verificar si el siguiente elemento es un encabezado.
				if s.Is("h2") {
					flag = false
				}
				// Buscar todos los elementos <li> dentro de <ul>.
				if flag {
					s.Find("li").Each(func(_ int, li *goquery.Selection) {

						death := parseEvent(li.Text())
						if death.Content != "" {
							deaths = append(deaths, death)
							fmt.Println("Fallecimiento encontrado:", death)
						}
					})
				}

			})

		}
	})

	col.OnScraped(func(r *colly.Response) {
		fmt.Println(r.Request.URL, "scraped!")
	})

	// Construir la URL con la fecha.
	formattedDate := fmt.Sprintf("https://es.wikipedia.org/wiki/%d_de_%s", date.Day, getMonthName(date.Month))
	fmt.Println("Formatted URL:", formattedDate)

	// Visitar la página con la URL formateada.
	col.Visit(formattedDate)

	return map[string][]Event{
		"events": events,
		"births": births,
		"deaths": deaths,
	}
}

// Función para analizar un texto y extraer información sobre un evento.
func parseEvent(text string) Event {
	var year int
	var isBeforeJesus bool

	acRegex := regexp.MustCompile(`\ba\.?\s*C\.?:`)
	hasAC := acRegex.FindString(text)

	// Verificar si el texto contiene "a.C." o "a.C:" para determinar si ocurrió antes de Jesús.
	if hasAC != "" {
		isBeforeJesus = true
		text = acRegex.ReplaceAllString(text, "")
	} else {
		isBeforeJesus = false
	}

	yearRegex := regexp.MustCompile(`^(\d+):`)
	// Extraer el año del texto.
	matches := yearRegex.FindStringSubmatch(text)
	if len(matches) > 0 {
		year, _ = strconv.Atoi(matches[1])
	} else {
		isBeforeJesus = true
	}

	// Eliminar el año del texto y obtener el contenido del evento.
	content := strings.TrimSpace(yearRegex.ReplaceAllString(text, ""))

	return Event{
		IsBeforeJesus: isBeforeJesus,
		Year:          year,
		Content:       content,
	}
}
