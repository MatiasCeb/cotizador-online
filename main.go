package main

import (
	"fmt"
	"html/template"
	"log"
	"math"
	"net/http"
	"os"
	"strconv"

	"gopkg.in/gomail.v2"
)

type CalculationData struct {
	Cost int
	Plans []PaymentPlan
	Step int
}

type PaymentPlan struct {
	Name string
	Amount int
}

type EmailData struct {
	Cost int
	Plan string
}

type ResultData struct {
	Success bool
	Email   string
	Error   string
}

func main() {
	http.HandleFunc("/", indexHandler)
	http.HandleFunc("/health", healthHandler)
	http.HandleFunc("/calculate", calculateHandler)
	http.HandleFunc("/select-plan", selectPlanHandler)
	http.HandleFunc("/send-email", sendEmailHandler)
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	log.Println("Server starting on :3000")
	log.Fatal(http.ListenAndServe(":3000", nil))
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "OK")
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	tmpl := template.Must(template.ParseFiles("templates/base.html", "templates/index.html"))
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
	costRounded := int(math.Round(cost))

	// Payment plans
	plans := []PaymentPlan{
		{"Transferencia", costRounded},
		{"Pago único", costRounded},
		{"3 cuotas", int(math.Round(cost / 3))},
		{"6 cuotas", int(math.Round(cost / 6))},
		{"12 cuotas", int(math.Round(cost / 12))},
	}

	data := CalculationData{Cost: costRounded, Plans: plans, Step: 2}

	tmpl := template.Must(template.ParseFiles("templates/base.html", "templates/payment.html"))
	tmpl.Execute(w, data)
}

func selectPlanHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	costStr := r.FormValue("cost")
	plan := r.FormValue("plan-pago")

	cost, _ := strconv.Atoi(costStr)

	data := EmailData{Cost: cost, Plan: plan}

	tmpl := template.Must(template.ParseFiles("templates/base.html", "templates/email.html"))
	tmpl.Execute(w, data)
}

func sendEmailHandler(w http.ResponseWriter, r *http.Request) {
	email := r.FormValue("email")
	costStr := r.FormValue("cost")
	plan := r.FormValue("plan")

	cost, _ := strconv.Atoi(costStr)

	body := fmt.Sprintf("Cotización: Costo total $%d. Plan seleccionado: %s.", cost, plan)

	from := os.Getenv("EMAIL_FROM")
	password := os.Getenv("EMAIL_PASSWORD")

	data := ResultData{Email: email}

	if from == "" || password == "" {
		data.Success = false
		data.Error = "Configurar variables de entorno EMAIL_FROM y EMAIL_PASSWORD"
	} else {
		m := gomail.NewMessage()
		m.SetHeader("From", from)
		m.SetHeader("To", email, from)
		m.SetHeader("Subject", "Cotización de Garantía")
		m.SetBody("text/plain", body)

		d := gomail.NewDialer("castilloconsultores.com.ar", 465, from, password)

		if err := d.DialAndSend(m); err != nil {
			data.Success = false
			data.Error = err.Error()
		} else {
			data.Success = true
		}
	}

	tmpl := template.Must(template.ParseFiles("templates/base.html", "templates/result.html"))
	tmpl.Execute(w, data)
}