package model

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"
)

const DateFormat = "2006-01-02"

type Date struct {
	time.Time
}

func NewDate(year int, month time.Month, day int) *Date {
	return &Date{time.Date(year, month, day, 0, 0, 0, 0, time.UTC)}
}

// для работы с HTTP
func (d Date) MarshalJSON() ([]byte, error) {
	return json.Marshal(d.Time.Format(DateFormat))
}

func (d *Date) UnmarshalJSON(data []byte) error {
	var dateStr string
	err := json.Unmarshal(data, &dateStr)
	if err != nil {
		return err
	}
	parsed, err := time.Parse(DateFormat, dateStr)
	if err != nil {
		return err
	}
	d.Time = parsed
	return nil
}

func (d Date) String() string {
	return d.Time.Format(DateFormat)
}

func (d *Date) UnmarshalText(data []byte) error {
	parsed, err := time.Parse(DateFormat, string(data))
	if err != nil {
		return err
	}
	d.Time = parsed
	return nil
}

// для работы с БД
// Реализация интерфейса sql.Scanner для чтения значения из базы данных.
func (d *Date) Scan(value interface{}) error {
	v, ok := value.(time.Time)
	if !ok {
		return fmt.Errorf("unsupported Scan, storing driver.Value type %T into type Date", value)
	}
	d.Time = v
	return nil
}

// Реализация интерфейса driver.Valuer для сохранения значения в базе данных.
func (d Date) Value() (driver.Value, error) {
	return d.Time, nil
}
