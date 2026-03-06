package models

import "time"

type UserID int64

type DataEntryID int64

type User struct {
	ID           UserID
	Username     string
	PasswordHash string
	CreatedAt    time.Time
}

type DataEntry struct {
	ID            DataEntryID
	UserID        UserID
	DataType      string
	Title         string
	EncryptedData []byte
	Salt          []byte
	Metadata      string
	CreatedAt     time.Time
}

type DataEntrySummary struct {
	ID        DataEntryID
	DataType  string
	Title     string
	Metadata  string
	CreatedAt time.Time
}

type CredsData struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

type CardData struct {
	Number   string `json:"number"`
	Holder   string `json:"holder"`
	ExpMonth string `json:"exp_month"`
	ExpYear  string `json:"exp_year"`
	CVV      string `json:"cvv"`
}
