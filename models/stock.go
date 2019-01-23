package models

import (
	"fmt"
	"stock-microsrvc/utils"
	"time"
)

// Stock ...
type Stock struct {
	ID     int       `json:"id"`
	Symbol string    `json:"symbol"`
	Price  float64   `json:"price"`
	Date   time.Time `json:"date"`
}

// Stocks ...
type Stocks []Stock

// NewStock creates a new stock object. The symbol
// is required to create a new stock.
func NewStock(symb string) *Stock {
	stock := Stock{Symbol: symb}
	return &stock
}

// SetSymb sets the symbol of the stock.
func (s *Stock) SetSymb(symb string) {
	s.Symbol = symb
}

// SetPrice sets the price of the stock.
func (s *Stock) SetPrice(price float64) {
	s.Price = price
}

// SetDate sets the date of the stock.
func (s *Stock) SetDate(date time.Time) {
	s.Date = date
}

func GetAllStockDBSymbols() (Stocks, error) {
	db, err := createDB()
	utils.CheckErr(err)
	query := fmt.Sprintf(`
		select symb from stocks.stocks
	`)

	stmt, err := db.Prepare(query)
	if err != nil {
		return nil, err
	}
	res, err := stmt.Query()
	if err != nil {
		return nil, err
	}

	type t_struct struct { Symbol string `json:"symbol"` }
	type t_structs t_struct[]

	var (
		symbol string
		stocks t_structs
	)

	for res.Next() {
		if err := res.Scan(&symb); err != nil {
			panic(err)
		}

		stocks = append(stocks, t_struct{symb})
	}
	return stocks, nil
}

// GetStockDBHistory ...
func (s *Stock) GetStockDBHistory(days int) (Stocks, error) {
	now := time.Now()
	queryDate := now.AddDate(0, 0, -days)
	db, err := createDB()
	utils.CheckErr(err)
	query := fmt.Sprintf(`select id, symb, price, time from %s where
                          time > $1 order by time desc`, s.Symbol)
	stmt, err := db.Prepare(query)
	if err != nil {
		return nil, err
	}
	res, err := stmt.Query(queryDate)
	if err != nil {
		return nil, err
	}

	var (
		id     int
		symb   string
		price  float64
		date   time.Time
		stocks Stocks
	)
	for res.Next() {
		if err := res.Scan(&id, &symb, &price, &date); err != nil {
			panic(err)
		}

		stocks = append(stocks, Stock{id, symb, price, date})
	}
	return stocks, nil
}

// UpdateDBPrice updates the database for the stock
// with the given price.
func (s *Stock) UpdateDBPrice(price float64) {
	db, err := createDB()
	defer db.Close()
	utils.CheckErr(err)

	stmt, err := db.Prepare(`update stocks.stocks set price = $1 where symb = $2`)
	utils.CheckErr(err)

	_, err = stmt.Exec(price, s.Symbol)
	utils.CheckErr(err)
	fmt.Println("Stock updated...")
}

func InsertDBStock(s *Stock) error {
	db, err := createDB()
	defer db.Close()
	utils.CheckErr(err)

	stmt, err := db.Prepare(`insert into stocks.stocks (symb, price) values (upper($1), $2)`)
	utils.CheckErr(err)

	_, err = stmt.Exec(s.Symbol, s.Price)
	utils.CheckErr(err)

	return err
}

func (s *Stock) GetStockDBPrice() float64 {
	query := fmt.Sprintf("select id, price, last_update from stocks.stocks where upper(symb) = upper('%s')", s.Symbol)

	res, err := executeQuery(query)
	utils.CheckErr(err)

	var price float64
	var date time.Time
	var id int
	for res.Next() {
		if err := res.Scan(&id, &price, &date); err != nil {
			panic(err)
		}

		s.Price = price
		s.Date = date
		s.ID = id
		return price
	}

	return price

}

func (s *Stock) DeleteDBStock() error {
	db, err := createDB()
	defer db.Close()
	utils.CheckErr(err)

	// Begin transaction
	tx, err := db.Begin()
	utils.CheckErr(err)

	{

		stmt, err := tx.Prepare("delete from stocks.stocks where upper(symb) = upper($1);")
		if err != nil {
			tx.Rollback()
			return err
		}
		defer stmt.Close()

		if _, err := stmt.Exec(s.Symbol); err != nil {
			tx.Rollback()
			return err
		}

	}
	{
		stmt, err := tx.Prepare("drop table $1;")
		if err != nil {
			tx.Rollback()
			return err
		}
		defer stmt.Close()

		if _, err := stmt.Exec(s.Symbol); err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit()

}
