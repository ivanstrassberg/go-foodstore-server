package main

import (
	"database/sql"
	"errors"
	"fmt"
	"log"

	_ "github.com/lib/pq"
)

type Storage interface {
	RegisterCustomer(string, string) (bool, error)
	LoginCustomer(string, string) (bool, error)
	GetFromDB(string) ([]DBEntity, error)
	GetFromDBByID(string, int) ([]DBEntity, error)
	AddProduct(string, string, float64, int, int) error
	AddProductToCustomerCart(int, string) (bool, error)
	RemoveProductFromCustomerCart(int, string) (bool, error)
	GetCartProducts(string) ([]Product, error)
	// GetCustomers() ([]*Customer, error)
	GetPassword(string) (string, error)
	DeleteProduct(string) error
	DeleteCategory(string) error
	IfExists(string, string, any) (bool, error)
	AddCategory(string, string) error
	SearchProducts(string) ([]DBEntity, error)
	copyCartToOrders(string, string) error
}

type DBEntity interface{}

type PostgresStore struct {
	db *sql.DB
}

func (s *PostgresStore) AddProductToCustomerCart(id int, email string) (bool, error) {
	// query := `insert into cart_product (cart_id, product_id) values
	// ((select cart.id from cart where cart.customer_id =
	// (select customer.id from customer where customer.email = $1)),$2)
	// `
	query := `insert into cart_product (cart_id, product_id) 
	values 
	((select cart.id from cart where cart.customer_id = 
	(select customer.id from customer where customer.email = $1)),
	(select product.id from product where product.id = $2))`
	_, err := s.db.Exec(query, email, id)
	if err != nil {
		return false, err
	}
	// resp := s.IfExists("cart_product")
	return true, nil
}

func (s *PostgresStore) RemoveProductFromCustomerCart(id int, email string) (bool, error) {
	query := `DELETE FROM cart_product 
	where product_id = $1 and 
	cart_id = (
		SELECT id FROM cart 
		WHERE customer_id = (
			SELECT id FROM customer 
			WHERE email = $2
		)
	)`
	_, err := s.db.Exec(query, id, email)
	if err != nil {
		return false, err
	}
	return true, nil
}

func (s *PostgresStore) AddProduct(name string, desc string, price float64, stock int, cat_id int) error {
	query := `insert into product (name, description, price, stock, rating, category_id) values ($1, $2, $3, $4, 0, $5)`
	_, err := s.db.Query(query, name, desc, price, stock, cat_id)
	if err != nil {
		return err
	}
	return nil
}

func (s *PostgresStore) DeleteProduct(name string) error {
	query := `delete from product where name = $1`
	_, err := s.db.Query(query, name)
	if err != nil {
		return err
	}
	return nil
}

func (s *PostgresStore) AddCategory(name string, desc string) error {
	query := `insert into category (name, description) values ($1, $2)`
	resp, err := s.db.Query(query, name, desc)
	if err != nil {
		return err
	}
	fmt.Println(resp)
	return nil
}

func (s *PostgresStore) DeleteCategory(name string) error {
	query := `delete from category where name = $1`
	_, err := s.db.Query(query, name)
	if err != nil {
		return err
	}
	return nil
}

func NewPostgresStorage() (*PostgresStore, error) {
	// connStr := "user=postgres port=5433 dbname=foodMarket password=root sslmode=disable"
	connStr := "PGPASSWORD=RWpDOeGNNprpGOSnCitvbbKVgMWQYkVr psql -h monorail.proxy.rlwy.net -U postgres -p 26066 -d railway"
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		// log.Panic()
		return nil, err
	}
	return &PostgresStore{
		db: db,
	}, nil
}

func (s *PostgresStore) Init() error {
	fmt.Println("Initializing DB...")
	s.createProductTable()
	s.createCustomerTable()
	s.createCategoryTable()
	s.createCartTable()
	s.createCartProductJunctionTable()
	s.createOrderTable()
	s.createOrderFuncJSON()
	s.createOrderFunc()
	// s.createOrderProductJunctionTable()
	fmt.Println("DB Initialized.")
	// s.createConstraints()
	return nil
}

func (s *PostgresStore) createProductTable() error { // todo add constraints to the category
	query := `create table if not exists product (
		id serial primary key,
		name varchar(120),
		description varchar(1000),
		price decimal,
		stock integer,
		rating decimal,
		category_id integer,
		foreign key (category_id) references category(id)
	)`
	_, err := s.db.Exec(query)
	return err
}

// func (s *PostgresStore) createOrderTable() error {
// 	fd
// 	query := `create table if not exists customer (
// 		id serial primary key,
// 		email varchar(100) not null unique,
// 		password_hash varchar(120) not null,
// 		delivery_address varchar(500) default 'none',
// 		created_at timestamp default current_timestamp
// 	)`

// 	_, err := s.db.Exec(query)
// 	return err
// }

func (s *PostgresStore) createCustomerTable() error { // todo add constraints to the cart
	query := `create table if not exists customer (
		id serial primary key,
		email varchar(100) not null unique,
		password_hash varchar(120) not null,
		delivery_address varchar(500) default 'none',
		created_at timestamp default current_timestamp
	)`

	_, err := s.db.Exec(query)
	return err
}

func (s *PostgresStore) createCartTable() error {
	query := `
		create table if not exists cart (
			id serial primary key,
			customer_id int,
			foreign key (customer_id) references customer(id) on delete cascade
		)
	`
	_, err := s.db.Exec(query)
	if err != nil {
		return err
	}

	triggerQuery :=
		`
		CREATE OR REPLACE FUNCTION create_cart_for_customer() 
		RETURNS trigger AS $$
		BEGIN
			INSERT INTO cart (customer_id) VALUES (NEW.id);
			RETURN NEW;  
		END;
		$$ LANGUAGE plpgsql;  
		
		CREATE TRIGGER after_customer_insert
		AFTER INSERT ON customer 
		FOR EACH ROW  
		EXECUTE FUNCTION create_cart_for_customer();`
	_, err = s.db.Exec(triggerQuery)
	if err != nil {
		return err
	}
	return nil
}

func (s *PostgresStore) createCategoryTable() error {
	query := `create table if not exists category (
		id serial primary key,
		name varchar(120),
		description varchar(500)
	)`
	_, err := s.db.Exec(query)
	if err != nil {
		return err
	}
	return err
}

func (s *PostgresStore) createCartProductJunctionTable() error {
	query := `CREATE TABLE if not exists cart_product (
		cart_id int,  
		product_id int,  
		PRIMARY KEY (cart_id, product_id),  
		FOREIGN KEY (cart_id) REFERENCES cart(id),  
		FOREIGN KEY (product_id) REFERENCES product(id)  
	);`
	_, err := s.db.Exec(query)
	if err != nil {
		return err
	}
	return nil
}

func (s *PostgresStore) createOrderTable() error {
	query := `CREATE TABLE IF NOT EXISTS customer_orders (
		order_id SERIAL PRIMARY KEY,
		customer_id INT NOT NULL,
		cart_id INT NOT NULL,
		product_id INT NOT NULL,
		quantity INT NOT NULL,
		order_date TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (customer_id) REFERENCES customer(id),
		FOREIGN KEY (product_id) REFERENCES product(id)	
	);`
	_, err := s.db.Exec(query)
	if err != nil {
		return err
	}
	return nil
}

func (s *PostgresStore) createOrderFunc() error { // has quantity 1 by default
	query := `
	CREATE OR REPLACE FUNCTION copy_cart_to_orders(uemail VARCHAR(50)) RETURNS VOID AS $$
DECLARE
    v_cart_id INT;
    customer_id_user INT;
    product RECORD;
BEGIN
    -- Retrieve customer_id from email
    SELECT id INTO customer_id_user FROM customer WHERE email = uemail;
    IF NOT FOUND THEN
        RAISE EXCEPTION 'Customer not found for email: %', uemail;
    END IF;

    -- Retrieve cart_id using customer_id
    SELECT id INTO v_cart_id FROM cart WHERE customer_id = customer_id_user;
    IF NOT FOUND THEN
        RAISE EXCEPTION 'Cart not found for customer_id: %', customer_id_user;
    END IF;

    -- Loop through cart_product and insert into customer_orders
    FOR product IN 
        SELECT product_id 
        FROM cart_product 
        WHERE cart_id = v_cart_id 
    LOOP
        INSERT INTO customer_orders (customer_id, cart_id, product_id, quantity)
        VALUES (customer_id_user, v_cart_id, product.product_id, 1); -- Assuming quantity is 1 for simplicity
    END LOOP;

    -- Optionally, clear the cart after copying
    -- DELETE FROM cart_product WHERE cart_id = v_cart_id;
END;
$$ LANGUAGE plpgsql;
`

	_, err := s.db.Exec(query)
	if err != nil {
		return err
	}
	return nil
}

func (s *PostgresStore) createOrderFuncJSON() error { // has variable quantity
	query := `
	CREATE OR REPLACE FUNCTION copy_cart_to_orders_json(uemail VARCHAR(50), products jsonb) RETURNS VOID AS $$
DECLARE
    v_cart_id INT;
    customer_id_user INT;
    product_info jsonb;
BEGIN
    
    SELECT id INTO customer_id_user FROM customer WHERE email = uemail;
    IF NOT FOUND THEN
        RAISE EXCEPTION 'Customer not found for email: %', uemail;
    END IF;

    SELECT id INTO v_cart_id FROM cart WHERE customer_id = customer_id_user;
    IF NOT FOUND THEN
        RAISE EXCEPTION 'Cart not found for customer_id: %', customer_id_user;
    END IF;

    FOREACH product_info IN ARRAY products
    LOOP
        INSERT INTO customer_orders (customer_id, cart_id, product_id, quantity)
        VALUES (customer_id_user, v_cart_id, (product_info->>'product_id')::int, (product_info->>'quantity')::int);
    END LOOP;

    -- Optionally, clear the cart after copying
    -- DELETE FROM cart_product WHERE cart_id = v_cart_id;
END;
$$ LANGUAGE plpgsql;
`

	_, err := s.db.Exec(query)
	if err != nil {
		return err
	}
	return nil
}

func (s *PostgresStore) copyCartToOrders(email string, prosducts string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		} else if err != nil {
			tx.Rollback()
		} else {
			err = tx.Commit()
		}
	}()

	products := `[{ "product_id": 9, "quantity": 10 }, { "product_id": 6, "quantity": 12 }]`
	if err := s.copyCartToOrders(email, products); err != nil {
		log.Fatalf("Failed to copy cart to orders: %v\n", err)
	}
	_, err = tx.Exec("SELECT copy_cart_to_orders_json($1,$2)", email, products)
	if err != nil {
		return fmt.Errorf("failed to execute copy_cart_to_orders: %w", err)
	}

	return nil
}

// func (s *PostgresStore) createOrderProductJunctionTable() error {
// 	query := `CREATE TABLE if not exists order_product (
// 		order_id int,
// 		product_id int,
// 		PRIMARY KEY (order_id, product_id),
// 		FOREIGN KEY (order_id) REFERENCES customer_order(id),
// 		FOREIGN KEY (product_id) REFERENCES product(id)
// 	);`
// 	_, err := s.db.Exec(query)
// 	if err != nil {
// 		return err
// 	}
// 	return nil
// }

func (s *PostgresStore) GetCartProducts(email string) ([]Product, error) {
	var cartID int
	err := s.db.QueryRow(`
        SELECT cart.id
        FROM cart
        JOIN customer ON cart.customer_id = customer.id
        WHERE customer.email = $1
    `, email).Scan(&cartID)
	if err != nil {
		return nil, err
	}
	rows, err := s.db.Query(`
        SELECT product.id, product.name, product.description, product.price, product.stock, product.rating, product.category_id
        FROM cart_product
        JOIN product ON cart_product.product_id = product.id
        WHERE cart_product.cart_id = $1
    `, cartID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	products := []Product{}
	//
	for rows.Next() {
		product, err := scanIntoProduct(rows)
		if err != nil {
			return nil, err
		}
		products = append(products, product)
	}

	return products, nil
}

func (s *PostgresStore) RegisterCustomer(email string, password string) (bool, error) {
	check, err := s.IfExists("customer", "email", email)
	// fmt.Println(check, "storage check for if exists")
	if err != nil {
		return false, err
	}
	if !check {
		query := `insert into customer (email, password_hash) values ($1,$2)`
		_, err1 := s.db.Exec(query, email, password)
		if err1 != nil {
			return false, err1
		}
		return true, nil
	}

	return false, nil
}

func (s *PostgresStore) GetPassword(email string) (string, error) {
	rows, err := s.db.Query("select * from customer where email = $1", email)
	if err != nil {
		return " ", err
	}
	var ret string
	for rows.Next() {
		customer, err := scanIntoCustomer(rows)
		if err != nil {
			return " ", err
		}
		ret = customer.PasswordHash
	}
	return ret, nil
}

func (s *PostgresStore) LoginCustomer(email string, password string) (bool, error) {
	check, _ := s.IfExists("customer", "email", email)
	if !check {
		return false, nil
	}

	check2, err := s.checkCustomer(email, password)
	// fmt.Println("this is a check", check2)
	if err != nil {
		return false, err
	}

	return check2, nil
}

func (s *PostgresStore) SearchProducts(string) ([]DBEntity, error) {

	productDBEntity, err := s.GetFromDB("product")
	if err != nil {
		return nil, err
	}
	// fmt.Println(productDBEntity)
	return productDBEntity, nil
}

func (s *PostgresStore) GetFromDB(table string) ([]DBEntity, error) {
	validTables := map[string]bool{
		"customer": true,
		"product":  true,
		"category": true,
	}
	if !validTables[table] {
		return nil, fmt.Errorf("invalid table: %s", table)
	}
	query := fmt.Sprintf("SELECT * FROM %s", table)

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	results := []DBEntity{}

	switch table {
	case "product":
		for rows.Next() {
			product, err := scanIntoProduct(rows)
			if err != nil {
				return nil, err
			}
			results = append(results, product)
		}
	case "customer":
		for rows.Next() {
			customer, err := scanIntoCustomer(rows)
			if err != nil {
				return nil, err
			}
			results = append(results, customer)
		}
	case "category":
		for rows.Next() {
			category, err := scanIntoCategory(rows)
			if err != nil {
				return nil, err
			}
			results = append(results, category)
		}
	default:
		return nil, errors.New("unknown table: " + table)
	}

	return results, nil
}

func (s *PostgresStore) GetFromDBByID(table string, id int) ([]DBEntity, error) {
	validTables := map[string]bool{
		"customer": true,
		"product":  true,
		"category": true,
	}
	if !validTables[table] {
		return nil, fmt.Errorf("invalid table: %s", table)
	}
	query := fmt.Sprintf("SELECT * FROM %s where id = %v", table, id)

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	results := []DBEntity{}

	switch table {
	case "product":
		for rows.Next() {
			product, err := scanIntoProduct(rows)
			if err != nil {
				return nil, err
			}
			results = append(results, product)
		}
	case "customer":
		for rows.Next() {
			customer, err := scanIntoCustomer(rows)
			if err != nil {
				return nil, err
			}
			results = append(results, customer)
		}
	case "category":
		for rows.Next() {
			category, err := scanIntoCategory(rows)
			if err != nil {
				return nil, err
			}
			results = append(results, category)
		}
	default:
		return nil, errors.New("unknown table: " + table)
	}

	return results, nil
}

func scanIntoProduct(rows *sql.Rows) (Product, error) {
	var product Product
	err := rows.Scan(&product.ID, &product.Name, &product.Description, &product.Price, &product.Stock, &product.Rating, &product.Category_ID)
	if err != nil {
		return Product{}, err
	}
	return product, nil
}

func scanIntoCustomer(rows *sql.Rows) (*Customer, error) {
	var customer Customer
	if err := rows.Scan(&customer.ID, &customer.Email, &customer.PasswordHash, &customer.DeliveryAddress, &customer.CreatedAt); err != nil {
		return nil, err
	}
	return &customer, nil
}

func scanIntoCategory(rows *sql.Rows) (*Category, error) {
	var category Category
	if err := rows.Scan(&category.ID, &category.Name, &category.Description); err != nil {
		return nil, err
	}
	return &category, nil
}

func (s *PostgresStore) IfExists(table string, column string, target any) (bool, error) {
	validTables := map[string]bool{
		"customer": true,
		"product":  true,
		"category": true,
	}
	if !validTables[table] {
		return false, fmt.Errorf("invalid table: %s", table)
	}
	query := fmt.Sprintf(
		`SELECT
		  CASE
		  	WHEN EXISTS (SELECT 1 FROM %s WHERE %s = $1)
			THEN 1
			ELSE 0
		  END AS exists`,
		table,
		column,
	)

	var exists int
	err := s.db.QueryRow(query, target).Scan(&exists)
	if err != nil {
		return false, err
	}
	return exists == 1, nil
}

func (s *PostgresStore) checkCustomer(email, password string) (bool, error) {
	query := `select
	case
		when exists (select 1 from customer where email = $1 and password_hash = $2)
	  then 1
	  else 0
	end as user_exists
  `
	var r int
	err := s.db.QueryRow(query, email, password).Scan(&r)
	if err != nil {
		return false, err
	}
	if r == 0 {
		return false, nil
	}
	return true, nil
}

// func (s *PostgresStore) executeQueryViaDBQuery(query string) {} // todo later
