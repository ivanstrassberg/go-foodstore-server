package main

import (
	"log"
	"net/http"

	"github.com/stripe/stripe-go/v78"
	"github.com/stripe/stripe-go/v78/client"
	"github.com/stripe/stripe-go/v78/paymentintent"
)

// func StripeInit() {
// 	stripe.Key = "sk_test_51PGBY6RsvEv5vPVlSr7KscWnARE1JSwq2Yuz6EqrYxs0Ksx6d8l1Uum5O5HUXj1rK8Hb2btsUvljijPxxAZQjTbk00bx8sBvRo"
// 	// params := &stripe.ChargeParams{}
// 	sc := &client.API{}
// 	sc.Init("sk_test_51PGBY6RsvEv5vPVlSr7KscWnARE1JSwq2Yuz6EqrYxs0Ksx6d8l1Uum5O5HUXj1rK8Hb2btsUvljijPxxAZQjTbk00bx8sBvRo", nil)
// 	// sc.Charges.Get("ch_3Ln3j02eZvKYlo2C0d5IZWuG", params)
// }

func handleCreatePaymentIntent(w http.ResponseWriter, r *http.Request, sum int64) error {
	// if r.Method != "POST" {
	// 	http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
	// 	return
	// }
	// Create a PaymentIntent with amount and currency
	stripe.Key = "sk_test_51PGBY6RsvEv5vPVlSr7KscWnARE1JSwq2Yuz6EqrYxs0Ksx6d8l1Uum5O5HUXj1rK8Hb2btsUvljijPxxAZQjTbk00bx8sBvRo"
	// params := &stripe.ChargeParams{}
	sc := &client.API{}
	sc.Init("sk_test_51PGBY6RsvEv5vPVlSr7KscWnARE1JSwq2Yuz6EqrYxs0Ksx6d8l1Uum5O5HUXj1rK8Hb2btsUvljijPxxAZQjTbk00bx8sBvRo", nil)
	params := &stripe.PaymentIntentParams{

		Amount:   stripe.Int64(sum),
		Currency: stripe.String(string(stripe.CurrencyRUB)),
		// In the latest version of the API, specifying the `automatic_payment_methods` parameter is optional because Stripe enables its functionality by default.
		AutomaticPaymentMethods: &stripe.PaymentIntentAutomaticPaymentMethodsParams{
			Enabled: stripe.Bool(true),
		},
	}

	pi, err := paymentintent.New(params)
	log.Printf("pi.New: %v", pi.ClientSecret)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Printf("pi.New: %v", err)
		return err
	}

	writeJSON(w, r, struct {
		ClientSecret string `json:"clientSecret"`
	}{
		ClientSecret: pi.ClientSecret,
	})
	return nil
}
