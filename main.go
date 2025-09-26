package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"math"
	"net/http"
	"os"
	"strconv"
	"strings"

	"gopkg.in/gomail.v2"
)

type CalculationData struct {
	Cost int
	OriginalCost int
	DiscountApplied bool
	DiscountPercent int
	DiscountMessage string
	Plans []PaymentPlan
	Step int
}

type PaymentPlan struct {
	Name string
	Amount int
	Discount int
	PerCuota bool
}

type EmailData struct {
	Cost     int
	Plan     string
	Name     string
	Surname  string
	Phone    string
}

type ResultData struct {
	Success bool
	Email   string
	Error   string
}

type Coupon struct {
	Code      string `json:"code"`
	Percent   int    `json:"percent"`
	Remaining int    `json:"remaining"`
}

var coupons map[string]*Coupon

func loadCoupons() {
	coupons = make(map[string]*Coupon)
	data, err := os.ReadFile("coupons.json")
	if err != nil {
		// If file doesn't exist, initialize with default
		coupons["DESC10"] = &Coupon{Code: "DESC10", Percent: 10, Remaining: 5}
		coupons["RAICES"] = &Coupon{Code: "RAICES", Percent: 25, Remaining: 3}
		saveCoupons()
		return
	}
	var couponList []Coupon
	if err := json.Unmarshal(data, &couponList); err != nil {
		log.Println("Error loading coupons:", err)
		return
	}
	for _, c := range couponList {
		coupons[c.Code] = &c
	}
	// Ensure default coupons are present
	if _, ok := coupons["DESC10"]; !ok {
		coupons["DESC10"] = &Coupon{Code: "DESC10", Percent: 10, Remaining: 5}
	}
	if _, ok := coupons["RAICES"]; !ok {
		coupons["RAICES"] = &Coupon{Code: "RAICES", Percent: 25, Remaining: 3}
	}
	saveCoupons()
}

func saveCoupons() {
	var couponList []Coupon
	for _, c := range coupons {
		couponList = append(couponList, *c)
	}
	data, err := json.MarshalIndent(couponList, "", "  ")
	if err != nil {
		log.Println("Error marshaling coupons:", err)
		return
	}
	if err := os.WriteFile("coupons.json", data, 0644); err != nil {
		log.Println("Error saving coupons:", err)
	}
}

func main() {
	loadCoupons()
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

	duracionStr := r.FormValue("duracion")
	valorMesStr := r.FormValue("valor-mes")
	expensasStr := r.FormValue("expensas")
	cupon := r.FormValue("cupon")

	duracion, _ := strconv.Atoi(duracionStr)
	valorMes, _ := strconv.ParseFloat(valorMesStr, 64)
	expensas, _ := strconv.ParseFloat(expensasStr, 64)

	// Calculation based on duration
	var multiplier float64
	switch duracion {
	case 12:
		multiplier = 0.8
	case 24:
		multiplier = 1.5
	case 36:
		multiplier = 1.75
	default:
		multiplier = 0.8 // default to 12 months
	}

	cost := (valorMes + expensas) * multiplier
	costRounded := int(math.Round(cost))

	// Check coupon
	discountPercent := 0
	discountMessage := ""
	if cupon != "" {
		if c, ok := coupons[cupon]; ok && c.Remaining > 0 {
			discountPercent = c.Percent
			c.Remaining--
			saveCoupons()
			discountMessage = fmt.Sprintf("Cupón aplicado: %s (%d%% descuento)", cupon, discountPercent)
		} else {
			discountMessage = "Cupón inválido o agotado"
		}
	}

	discountedCost := costRounded
	if discountPercent > 0 {
		discountedCost = int(math.Round(float64(costRounded) * (1 - float64(discountPercent)/100)))
	}

	// Payment plans
	plansData := []struct {
		Name     string
		Discount int
	}{
		{"Transferencia", 15},
		{"Pago único", 0},
		{"3 cuotas", 0},
		{"6 cuotas", 0},
		{"12 cuotas", 0},
	}

	var plans []PaymentPlan
	for _, pd := range plansData {
		total := float64(discountedCost) * (1 - float64(pd.Discount)/100)
		var amount int
		if strings.Contains(pd.Name, "cuota") {
			parts := strings.Split(pd.Name, " ")
			n, _ := strconv.Atoi(parts[0])
			amount = int(math.Round(total / float64(n)))
		} else {
			amount = int(math.Round(total))
		}
		perCuota := strings.Contains(pd.Name, "cuota")
		log.Printf("Plan: %s, Amount: %d, Discount: %d, PerCuota: %t", pd.Name, amount, pd.Discount, perCuota)
		plans = append(plans, PaymentPlan{Name: pd.Name, Amount: amount, Discount: pd.Discount, PerCuota: perCuota})
	}
	log.Printf("Number of plans: %d", len(plans))

	data := CalculationData{
		Cost: discountedCost,
		OriginalCost: costRounded,
		DiscountApplied: discountPercent > 0,
		DiscountPercent: discountPercent,
		DiscountMessage: discountMessage,
		Plans: plans,
		Step: 2,
	}

	tmpl := template.Must(template.ParseFiles("templates/base.html", "templates/payment.html"))
	tmpl.Execute(w, data)
}

func selectPlanHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	planAndAmount := r.FormValue("plan-pago")
	parts := strings.Split(planAndAmount, "|")
	plan := parts[0]
	costStr := parts[1]
	cost, _ := strconv.Atoi(costStr)

	data := EmailData{Cost: cost, Plan: plan, Name: "", Surname: "", Phone: ""}

	tmpl := template.Must(template.ParseFiles("templates/base.html", "templates/email.html"))
	tmpl.Execute(w, data)
}

func sendEmailHandler(w http.ResponseWriter, r *http.Request) {
	email := r.FormValue("email")
	costStr := r.FormValue("cost")
	plan := r.FormValue("plan")
	name := r.FormValue("name")
	surname := r.FormValue("surname")
	phone := r.FormValue("phone")

	cost, _ := strconv.Atoi(costStr)

	body := fmt.Sprintf("Cliente: %s %s\nTeléfono: %s\nEmail: %s\n\nCotización: Costo total $%d. Plan seleccionado: %s.", name, surname, phone, email, cost, plan)

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