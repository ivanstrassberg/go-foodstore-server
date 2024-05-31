package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	jwt "github.com/golang-jwt/jwt"
	"github.com/stripe/stripe-go/v78"
	"github.com/stripe/stripe-go/v78/client"
	"github.com/stripe/stripe-go/webhook"
	bcrypt "golang.org/x/crypto/bcrypt"
)

type APIServer struct {
	listenAddr string
	store      Storage
	staticDir  string
}

func NewAPIServer(listenAddr string, store Storage, staticDir string) *APIServer {
	return &APIServer{
		listenAddr: listenAddr,
		store:      store,
		staticDir:  staticDir,
	}
}

func enableCors(w *http.ResponseWriter, req *http.Request) {
	(*w).Header().Set("Access-Control-Allow-Origin", "*")
	(*w).Header().Set("Access-Control-Allow-Methods", "DELETE, POST, GET, OPTIONS")
	(*w).Header().Set("Access-Control-Allow-Headers", "Content-Type, X-Authorization, X-Requested-With, email, Authorization")
	(*w).Header().Set("Access-Control-Allow-Credentials", "true")

	// if req.Method == "OPTIONS" {
	// 	(*w).WriteHeader(http.StatusOK)
	// 	return
	// }
}

func (s *APIServer) Run() {
	stripe.Key = "sk_test_51PGBY6RsvEv5vPVlSr7KscWnARE1JSwq2Yuz6EqrYxs0Ksx6d8l1Uum5O5HUXj1rK8Hb2btsUvljijPxxAZQjTbk00bx8sBvRo"
	// params := &stripe.ChargeParams{}
	sc := &client.API{}
	sc.Init("sk_test_51PGBY6RsvEv5vPVlSr7KscWnARE1JSwq2Yuz6EqrYxs0Ksx6d8l1Uum5O5HUXj1rK8Hb2btsUvljijPxxAZQjTbk00bx8sBvRo", nil)
	mux := http.NewServeMux()
	// mux = corsMiddleware(mx)
	mux.HandleFunc("/admin/{action}/{type}", withJWTauthAdmin(makeHTTPHandleFunc(s.handleAdmin))) // post: add/delete/update-category/product; get:
	mux.HandleFunc("/admin", withJWTauthAdmin(makeHTTPHandleFunc(s.handleAdmin)))
	mux.HandleFunc("/products", (makeHTTPHandleFunc(s.handleProducts)))
	mux.HandleFunc("/cart/{action}/{id}", withJWTauth(makeHTTPHandleFunc(s.handleCartActions))) // handle cart as a collection of methods,
	// used to operate it
	// mux.HandleFunc("/create-payment-intent", withJWTauth(makeHTTPHandleFunc(handleCreatePaymentIntent)))
	// handle the user cart as a collection of products
	mux.HandleFunc("/cart", withJWTauth(makeHTTPHandleFunc(s.handleCart)))
	mux.HandleFunc("/payment", (makeHTTPHandleFunc(s.handlePayment)))
	mux.HandleFunc("/config", (makeHTTPHandleFunc(s.handleConfig)))
	mux.HandleFunc("/orders", (makeHTTPHandleFunc(s.handleOrder)))
	mux.HandleFunc("/product/{id}", (makeHTTPHandleFunc(s.handleProductByID)))
	mux.HandleFunc("/index", withJWTauth(makeHTTPHandleFunc(s.handleMain)))
	mux.HandleFunc("/account/{id}", withJWTauth(makeHTTPHandleFunc(s.handleAccount)))
	mux.HandleFunc("/login", (makeHTTPHandleFunc(s.handleLogin)))
	mux.HandleFunc("/register", (makeHTTPHandleFunc(s.handleRegister)))
	mux.HandleFunc("/search/{key}", (makeHTTPHandleFunc(s.handleSearch)))

	log.Println("JSON API server running on port", s.listenAddr)
	if err := http.ListenAndServe(s.listenAddr, mux); err != nil {
		log.Fatalf("Error starting server: %s\n", err)
	}
}

func (s *APIServer) handleWebhook(w http.ResponseWriter, r *http.Request) error {
	if r.Method != "POST" {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return nil
	}
	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		log.Printf("ioutil.ReadAll: %v", err)
		return err
	}

	event, err := webhook.ConstructEvent(b, r.Header.Get("Stripe-Signature"), "sk_test_51PGBY6RsvEv5vPVlSr7KscWnARE1JSwq2Yuz6EqrYxs0Ksx6d8l1Uum5O5HUXj1rK8Hb2btsUvljijPxxAZQjTbk00bx8sBvRo")
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		log.Printf("webhook.ConstructEvent: %v", err)
		return err
	}

	if event.Type == "checkout.session.completed" {
		fmt.Println("Checkout Session completed!")
	}

	writeJSON(w, r, nil)
	return nil
}

func (s *APIServer) handleConfig(w http.ResponseWriter, r *http.Request) error {
	enableCors(&w, r)
	if r.Method != "GET" {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return nil
	}
	writeJSON(w, r, struct {
		PublishableKey string `json:"publishableKey"`
	}{
		PublishableKey: "pk_test_51PGBY6RsvEv5vPVlHUbe5pB27TSwBnFGH7t93QSkoef6FEy1hobnSCmWSJDk3cnQgj1Wrf9TybhyEyu79ZEtuNST00aSiTI6Vg",
	})
	return nil
}

func (s *APIServer) handleOrder(w http.ResponseWriter, r *http.Request) error {
	s.store.copyCartToOrders("ias114@tpu.ru", "ff")
	return nil
}

func (s *APIServer) handlePayment(w http.ResponseWriter, r *http.Request) error {
	enableCors(&w, r)
	// if r.Method == "POST" {
	// fmt.Println("fuck")
	email := r.Header.Get("email")
	var req ([]CheckoutReq)
	// fmt.Println("fuck2")
	err := json.NewDecoder(r.Body).Decode(&req)
	// fmt.Println(req)
	if err != nil {
		// log.Fatal(err)
		return err
	}
	// fmt.Println("before getting prodList")
	productList, err := s.store.GetCartProducts(email)
	if err != nil {
		return err
	}

	// total, err := calculateTotal(productList)
	total, err := calculateTotal(req, productList, email)
	if err != nil {
		// fmt.Println("fucied")
		WriteJSON(w, http.StatusBadRequest, ApiError{Error: "cart handle failure"})
	}
	fmt.Println(total)
	if err := handleCreatePaymentIntent(w, r, total); err == nil {

	}

	// }

	return nil
}

func (s *APIServer) handleSearch(w http.ResponseWriter, r *http.Request) error {
	enableCors(&w, r)
	key := r.PathValue("key")
	products, err := s.store.SearchProducts(key)
	if err != nil {
		WriteJSON(w, http.StatusInternalServerError, "something went wrong")
		return err
	}
	WriteJSON(w, http.StatusOK, products)
	return nil
}

func (s *APIServer) handleCart(w http.ResponseWriter, r *http.Request) error {
	email := r.Header.Get("email")

	productList, err := s.store.GetCartProducts(email)
	// if r.Method == "POST" {
	// 	// WriteJSON(w, http.StatusOK, sum)
	// 	// fmt.Println(sum, "change to int prices")
	// 	// total, err := calculateTotal(productList)
	if err != nil {
		WriteJSON(w, http.StatusBadRequest, ApiError{Error: "cart handle failure"})
	}
	// 	handleCreatePaymentIntent(w, r, total)
	// 	return nil
	// }
	// // fmt.Println(sum, "not counting")
	// // handleCreatePaymentIntent(w, r, sum)
	// if err != nil {
	// 	WriteJSON(w, http.StatusInternalServerError, ApiError{Error: "something went wrong during cart handling"})
	// 	return err
	// }

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(productList)
	// WriteJSON(w, http.StatusOK, "products listed")
	return nil
}

func (s *APIServer) handleCartActions(w http.ResponseWriter, r *http.Request) error {
	action := r.PathValue("action")
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		WriteJSON(w, http.StatusBadRequest, "id is not a numeric value")
	}
	email := r.Header.Get("email")
	switch action {
	case "add":
		answ, err := s.store.AddProductToCustomerCart(id, email)
		if err != nil {
			return err
		}
		if !answ {
			WriteJSON(w, http.StatusBadRequest, "something went wrong")
		}
		WriteJSON(w, http.StatusOK, "product added")
	case "delete":
		answ, err := s.store.RemoveProductFromCustomerCart(id, email)
		if err != nil {
			return err
		}
		if !answ {
			WriteJSON(w, http.StatusBadRequest, "something went wrong")
		}
		WriteJSON(w, http.StatusOK, "product removed")
	default:
		WriteJSON(w, http.StatusBadRequest, ApiError{Error: "action not supported"})
	}

	return nil
}

func (s *APIServer) handleAdmin(w http.ResponseWriter, r *http.Request) error {
	action := r.PathValue("action")
	typeOf := r.PathValue("type")
	if r.Method == "GET" {
		if action == "get" {
			// switch typeOf {
			// case "customer":
			resp, _ := s.store.GetFromDB(typeOf)
			WriteJSON(w, http.StatusOK, resp)
			// }
		}
	}
	if r.Method == "POST" {
		switch action {
		case "add":
			switch typeOf {
			case "product":
				req := new(Product)
				err := json.NewDecoder(r.Body).Decode(req)
				if err != nil {
					return err
				}
				check, _ := s.store.IfExists("product", "name", req.Name)
				if !check {
					err := s.store.AddProduct(req.Name, req.Description, req.Price, req.Stock, req.Category_ID)
					if err != nil {
						return err
					}
					check, _ = s.store.IfExists("product", "name", req.Name)
					if check {
						WriteJSON(w, http.StatusAccepted, "product created")
						return nil
					}
				}
				WriteJSON(w, http.StatusBadRequest, ApiError{Error: "product already exists"})
				return nil
			case "category":
				req := new(ReqCategory)
				err := json.NewDecoder(r.Body).Decode(req)
				if err != nil {
					return err
				}
				check, _ := s.store.IfExists("category", "name", req.Name)
				if !check {
					err := s.store.AddCategory(req.Name, req.Description)
					if err != nil {
						return err
					}
					// fmt.Println(resp)
					check, _ = s.store.IfExists("category", "name", req.Name)
					if check {
						WriteJSON(w, http.StatusAccepted, "category created")
						return nil
					}
					WriteJSON(w, http.StatusBadRequest, "something went wrong")
					return nil
				}
				WriteJSON(w, http.StatusBadRequest, ApiError{Error: "category already exists"})
				return nil
			}
		case "delete":
			fmt.Println("deleting a ...")
			switch typeOf {
			case "product":
				req := new(Product)
				err := json.NewDecoder(r.Body).Decode(req)
				if err != nil {
					return err
				}
				check, _ := s.store.IfExists("product", "name", req.Name)
				if check {
					err := s.store.DeleteProduct(req.Name)
					if err != nil {
						return err
					}
					check, _ = s.store.IfExists("product", "name", req.Name)
					if !check {

						return WriteJSON(w, http.StatusAccepted, "product deleted")
					}
				}
				return WriteJSON(w, http.StatusBadRequest, ApiError{Error: "product does not exist"})
			case "category":
				fmt.Println(typeOf)
			}
			// default:
			// 	WriteJSON(w, http.StatusBadRequest, "the action not supported")
		}

		// WriteJSON(w, http.StatusOK, nil)
		// fmt.Println(r.Header)
		// token := r.Header.Get("X-Authorization")
		// fmt.Println(token)
		// // }

	}
	return nil
}

func (s *APIServer) handleProducts(w http.ResponseWriter, r *http.Request) error {
	products, err := s.store.GetFromDB("product")
	if err != nil {
		return WriteJSON(w, http.StatusBadRequest, "failed to fetch products")
	}

	// Write products as JSON
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(products)
	return nil
}

func (s *APIServer) handleProductByID(w http.ResponseWriter, r *http.Request) error {
	idStr := r.PathValue("id")
	// fmt.Println(idStr)
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return err
	}
	if check, _ := s.store.IfExists("product", "id", id); !check {
		WriteJSON(w, http.StatusBadRequest, "does not exist")
	}
	product, err := s.store.GetFromDBByID("product", id)
	if err != nil {
		return WriteJSON(w, http.StatusBadRequest, "failed to fetch product")
	}

	// Write products as JSON
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(product)
	return nil
}

func (s *APIServer) handleAccount(w http.ResponseWriter, r *http.Request) error {
	return nil
}

func (s *APIServer) handleMain(w http.ResponseWriter, r *http.Request) error {
	// enableCors(&w)
	// http.ServeFile(w, r, "/Users/ivansilin/Documents/coding/golang/foodShop/initHandle/static/index.html")
	// s.serveFileByURL(w, r)
	return nil
}

func (s *APIServer) handleLogin(w http.ResponseWriter, r *http.Request) error {

	if r.Method == "POST" {

		regReq := new(CustomerLR)
		if err := json.NewDecoder(r.Body).Decode(regReq); err != nil {
			return err
		}
		passDB, err := s.store.GetPassword(regReq.Email)
		if err != nil {
			return err
		}
		// shit, _ := HashPassword(regReq.PasswordHash)
		fmt.Println(regReq.Email, regReq.PasswordHash, passDB)
		check := HashToPassword(passDB, regReq.PasswordHash)
		// check := CheckPasswordHash(regReq.PasswordHash, passDB)
		fmt.Println(check)
		// check := true
		if check {
			resp, err := s.store.LoginCustomer(regReq.Email, passDB)

			if err != nil {
				WriteJSON(w, http.StatusBadRequest, ApiError{Error: "does not exist, or wrong password"})
				return err
			}
			// fmt.Println(resp)
			if resp {
				token, err := generateJWT(regReq.Email)
				if err != nil {
					return err
				}
				// fmt.Println(token, "<- this mf dont wanna stick to header")
				w.Header().Set("X-Authorization", token)
				w.Header().Add("Authorization", "sk_test_51PGBY6RsvEv5vPVlSr7KscWnARE1JSwq2Yuz6EqrYxs0Ksx6d8l1Uum5O5HUXj1rK8Hb2btsUvljijPxxAZQjTbk00bx8sBvRo")
				fmt.Println(w.Header().Get("Authorization"))
				WriteJSON(w, http.StatusOK, respData{
					XAuth: token,
					// Auth:  "Bearer " + "sk_test_51PGBY6RsvEv5vPVlSr7KscWnARE1JSwq2Yuz6EqrYxs0Ksx6d8l1Uum5O5HUXj1rK8Hb2btsUvljijPxxAZQjTbk00bx8sBvRo",
					Auth: "sk_test_51PGBY6RsvEv5vPVlSr7KscWnARE1JSwq2Yuz6EqrYxs0Ksx6d8l1Uum5O5HUXj1rK8Hb2btsUvljijPxxAZQjTbk00bx8sBvRo",
				})
				return nil
			}
		}
		WriteJSON(w, http.StatusForbidden, ApiError{Error: "forbidden"})
		return nil
	}
	return WriteJSON(w, http.StatusNotFound, "http method not supported")
}

func (s *APIServer) handleRegister(w http.ResponseWriter, r *http.Request) error {
	// if r.Method == "GET" {
	// 	// s.serveFileByURL(w, r)
	// 	// fmt.Println(s.store.IfExists("customer", "email", "ias115@tpu.ru"))
	// 	// err := s.store.RegisterCustomer(regReq.Email, regReq.PasswordHash)
	// }
	if r.Method == "POST" {
		regReq := new(CustomerLR)
		if err := json.NewDecoder(r.Body).Decode(regReq); err != nil {
			return err
		}

		// ph, err := HashPassword(regReq.PasswordHash)
		ph, err := HashPassword(regReq.PasswordHash)

		if err != nil {
			return err
		}
		check := isCommonMailDomain(regReq.Email)
		// fmt.Println(check, "check for common domain")
		if regReq.Email == " " || !check {
			WriteJSON(w, http.StatusBadRequest, 400)
			return nil
		}
		v, err := s.store.RegisterCustomer(regReq.Email, ph)
		// fmt.Println(v, "check whether user exists")
		if err != nil {
			return err
		}
		if v {
			return WriteJSON(w, http.StatusOK, CustomerLR{
				regReq.Email, "",
			})
		}
		WriteJSON(w, http.StatusBadRequest, 400)
		fmt.Println("end")
		return nil
	}
	// if r.Method == "DELETE" {
	// 	users, err := s.store.GetFromDB("customer")
	// 	if err != nil {
	// 		return err
	// 	}
	// 	for i := 0; i < len(users); i++ {
	// 		fmt.Println(users[i])
	// 	}
	// }
	WriteJSON(w, http.StatusBadRequest, 400)
	return nil
}

func makeHTTPHandleFunc(f APIfunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		enableCors(&w, r)
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}
		if err := f(w, r); err != nil {
			WriteJSON(w, http.StatusBadRequest, ApiError{Error: err.Error()})
		}

	}
}

type respData struct {
	XAuth string `json:"X-Authorization"`
	Auth  string `json:"Authorization"`
}

func WriteJSON(w http.ResponseWriter, status int, v any) error {
	w.Header().Add("Content-Type", "application/json")
	// w.WriteHeader(status)
	return json.NewEncoder(w).Encode(v)
}
func WriteJSONResponseless(w http.ResponseWriter, status int) error {
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(status)
	return nil
}

type ApiError struct {
	Error string `json:"error"`
}

func (s *APIServer) serveFileByURL(w http.ResponseWriter, r *http.Request) error {
	path := r.URL.Path
	parts := strings.Split(path, "/")
	endpoint := parts[len(parts)-1]
	file := s.staticDir + endpoint + ".html"
	_, err := os.Stat(file)
	// fmt.Println(file)
	if err != nil {
		return err
	}

	http.ServeFile(w, r, file)
	return nil
}

func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	return string(bytes), err
}
func HashToPassword(hash, password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

var jwtKey = []byte("testKey") // later get that to the ENV var

func generateJWT(email string) (string, error) {
	// expirationTime := time.Now().Add(5 * time.Minute)
	claims := &Claims{
		Email: email,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: time.Now().Add(time.Minute * 60).Unix(), // * 60
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	return token.SignedString([]byte(jwtKey))

}

func validateJWT(tokenString string) (*jwt.Token, error) {
	// secret := os.Getenv("JWT_SECRET")
	return jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("wrong")
		}
		return []byte(jwtKey), nil
	})
}

func withJWTauth(handleFunc http.HandlerFunc) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {
		enableCors(&w, r)
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}
		// fmt.Println("calling jwt middleware")
		tokenString := r.Header.Get("X-Authorization")
		// fmt.Println(tokenString)
		token, err := validateJWT(tokenString)
		if err != nil {
			// fmt.Println("yeah not authorized")
			WriteJSON(w, http.StatusUnauthorized, ApiError{Error: "forbidden"})
			return
		}
		emailString := r.Header.Get("email")
		checkEmail, err := ParseJWT(tokenString)
		if err != nil {
			WriteJSON(w, http.StatusUnauthorized, ApiError{Error: "forbidden"})
			return
		}
		// fmt.Println(emailString, checkEmail, "check if email's the same")

		if emailString == checkEmail && token.Valid {
			WriteJSONResponseless(w, http.StatusOK)
			handleFunc(w, r)
			return
		}

		WriteJSON(w, http.StatusUnauthorized, ApiError{Error: "forbidden"})
	}
}

func withJWTauthAdmin(handleFunc http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		enableCors(&w, r)

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}
		tokenString := r.Header.Get("X-Authorization")
		token, err := validateJWT(tokenString)
		if err != nil {
			WriteJSON(w, http.StatusUnauthorized, ApiError{Error: "forbidden"})
			return
		}
		emailString := r.Header.Get("email")
		checkEmail, err := ParseJWT(tokenString)
		if err != nil {
			WriteJSON(w, http.StatusUnauthorized, ApiError{Error: "forbidden"})
			return
		}
		if emailString == checkEmail && checkEmail == "admin@gmail.com" && token.Valid {
			// WriteJSON(w, http.StatusOK, "Welcome!")
			handleFunc(w, r)
			return
		}

		WriteJSON(w, http.StatusUnauthorized, ApiError{Error: "forbidden"})
	}

}

func ParseJWT(tokenString string) (string, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Method)
		}
		return []byte(jwtKey), nil
	})
	if err != nil {
		return "", err
	}
	if !token.Valid {
		return "", fmt.Errorf("invalid token")
	}
	return claims.Email, nil
}

type Claims struct {
	Email string `json:"email"`
	jwt.StandardClaims
}

type APIfunc func(http.ResponseWriter, *http.Request) error

var commonEmailDomains = []string{
	"gmail.com",
	"yahoo.com",
	"outlook.com",
	"hotmail.com",
	"live.com",
	"aol.com",
	"icloud.com",
	"mail.ru",
	"tpu.ru",
	"inbox.ru",
}

func isCommonMailDomain(email string) bool {
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return false
	}

	domain := parts[1]
	for _, commonDomain := range commonEmailDomains {
		if strings.EqualFold(domain, commonDomain) {
			return true
		}
	}

	return false
}

func calculateTotal(req []CheckoutReq, productList []Product, email string) (int64, error) {
	var total int64
	// fmt.Println("before calcTotal")
	fmt.Println(req, productList)
	for i := 0; i < len(productList); i++ {
		total += int64(productList[i].Price) * 100 * int64(req[i].Quantity)
	}

	return total, nil
}

func writeJSON(w http.ResponseWriter, r *http.Request, v interface{}) {
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(v); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Printf("json.NewEncoder.Encode: %v", err)
		return
	}
	// enableCors(&w,r)
	w.Header().Set("Content-Type", "application/json")
	if _, err := io.Copy(w, &buf); err != nil {
		log.Printf("io.Copy: %v", err)
		return
	}
}

// func getProductValues(row Product) (Product, error) {
// 	fmt.Println(row.)
// 	reader := strings.NewReader(row)
// 	var partDescription string
// 	_, err := fmt.Fscanf(reader, "%d %s %s", &product.ID, &product.Name, &partDescription)
// 	if err != nil {
// 		return Product{}, err
// 	}

// 	var remaining string
// 	_, err = fmt.Fscanln(reader, &remaining)
// 	if err != nil {
// 		return Product{}, err
// 	}

// 	product.Description = partDescription + " " + remaining
// 	product.Description = strings.TrimSuffix(product.Description, "}")

// 	_, err = fmt.Sscanf(product.Description, "%s %f %d %d %d",
// 		&product.Description, &product.Price, &product.Stock, &product.Rating, &product.Category_ID)

// 	if err != nil {
// 		return Product{}, err
// 	}

// 	return product, nil

// }
