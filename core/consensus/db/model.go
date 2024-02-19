package db

import "time"

// Rate schema of the rate table
type Rate struct {
	ID              string
	Price           string
	First_Signer    string
	Sign_Data       string
	LastSigned_Time time.Time
	Created_Time    time.Time
}
