package config

type Http struct {
	Port int `json:"port"`

	TLSEnable bool   `json:"tlsEnable"`
	TLSCRT    string `json:"tlsCRT"`
	TLSKey    string `json:"tlsKey"`
}
