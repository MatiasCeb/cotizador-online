package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"strconv"

	"gopkg.in/gomail.v2"
)

type CalculationData struct {
	Cost float64
	Plans []PaymentPlan
}

type PaymentPlan struct {
	Name string
	Amount float64
}

func main() {
	http.HandleFunc("/", indexHandler)
	http.HandleFunc("/calculate", calculateHandler)
	http.HandleFunc("/send-email", sendEmailHandler)
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	log.Println("Server starting on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	tmpl := template.Must(template.ParseFiles("templates/index.html"))
	tmpl.Execute(w, nil)
}

func calculateHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	tipoGarantia := r.FormValue("tipo-garantia")
	tipoAlquiler := r.FormValue("tipo-alquiler")
	duracionStr := r.FormValue("duracion")
	valorMesStr := r.FormValue("valor-mes")
	expensasStr := r.FormValue("expensas")

	duracion, _ := strconv.Atoi(duracionStr)
	valorMes, _ := strconv.ParseFloat(valorMesStr, 64)
	expensas, _ := strconv.ParseFloat(expensasStr, 64)

	// Simple calculation logic
	factor := 0.1 // 10% of annual rent + expenses
	if tipoGarantia == "renovacion" {
		factor = 0.08
	}
	if tipoAlquiler == "comercial" {
		factor += 0.02
	}

	cost := (valorMes + expensas) * float64(duracion) / 12 * factor

	// Payment plans
	plans := []PaymentPlan{
		{"Pago único", cost},
		{"3 cuotas", cost / 3},
		{"6 cuotas", cost / 6},
		{"12 cuotas", cost / 12},
	}

	data := CalculationData{Cost: cost, Plans: plans}

	tmpl := template.Must(template.ParseFiles("templates/payment.html"))
	tmpl.Execute(w, data)
}

func sendEmailHandler(w http.ResponseWriter, r *http.Request) {
	email := r.FormValue("email")
	// For demo, assume cost is stored or passed; here hardcode or from session
	// In real app, use sessions or pass data
	body := fmt.Sprintf("Cotización: Costo total $1000. Plan seleccionado: Pago único.")

	from := os.Getenv("EMAIL_FROM")
	password := os.Getenv("EMAIL_PASSWORD")

	if from == "" || password == "" {
		fmt.Fprintf(w, "<p>Error: Configurar variables de entorno EMAIL_FROM y EMAIL_PASSWORD</p>")
		return
	}

	m := gomail.NewMessage()
	m.SetHeader("From", from)
	m.SetHeader("To", email)
	m.SetHeader("Subject", "Cotización de Garantía")
	m.SetBody("text/plain", body)

	d := gomail.NewDialer("smtp.gmail.com", 587, from, password)

	if err := d.DialAndSend(m); err != nil {
		fmt.Fprintf(w, "<p>Error enviando email: %v</p>", err)
	} else {
		fmt.Fprintf(w, "<p>Cotización enviada a %s</p>", email)
	}
}