package main

import "time"

type Order struct {
	Id               string
	LocationNum      string
	CheckInTimestamp time.Time
	OrderSubTotal    float64
}
