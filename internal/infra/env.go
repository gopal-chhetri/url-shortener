package infra

import (
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Env struct {
	DBName                   string
	DBHost                   string
	DBPort                   string
	DBUser                   string
	DBPass                   string
	AppEnv                   string
	BaseURL                  string
	Port                     string
	AccessTokenExpiryMinute  int
	RefreshTokenExpiryMinute int
	AccessTokenSecret        string
	RefreshTokenSecret       string
	RedisHost                string
	RedisPort                string
	RedisPassword            string
	RedisDB                  int
}

func NewEnv() *Env {
	err := godotenv.Load()
	if err != nil {
		log.Println("Error loading .env file, using environment variables from OS")
	}
	env := &Env{}
	env.loadFromEnvironment()

	return env
}

func (e *Env) loadFromEnvironment() {
	e.DBName = getEnv("DB_NAME")
	e.DBHost = getEnv("DB_HOST")
	e.DBPort = getEnv("DB_PORT")
	e.DBUser = getEnv("DB_USER")
	e.DBPass = getEnv("DB_PASS")
	e.AppEnv = getEnv("APP_ENV")
	e.BaseURL = getEnv("BASE_URL")
	e.Port = getEnv("PORT")
	e.AccessTokenExpiryMinute = getIntEnv("ACCESS_TOKEN_EXPIRY_MINUTE")
	e.RefreshTokenExpiryMinute = getIntEnv("REFRESH_TOKEN_EXPIRY_MINUTE")
	e.AccessTokenSecret = getEnv("ACCESS_TOKEN_SECRET")
	e.RefreshTokenSecret = getEnv("REFRESH_TOKEN_SECRET")
	e.RedisHost = getEnv("REDIS_HOST")
	e.RedisPort = getEnv("REDIS_PORT")
	e.RedisPassword = getEnv("REDIS_PASSWORD")
	e.RedisDB = getIntEnv("REDIS_DB")
}

func getEnv(envName string) string {
	envValue, exists := os.LookupEnv(envName)
	if !exists || envValue == "" {
		log.Printf("Env variable %s not found", envName)
	}
	return envValue
}

func getIntEnv(envName string) int {
	envValue, exists := os.LookupEnv(envName)
	fmt.Println(envName, envValue)
	if !exists || envValue == "" {
		log.Printf("Env variable %s not found", envName)
	}
	intEnv, err := strconv.Atoi(envValue)
	if err != nil {
		log.Printf("Env variable %s cannot be converted to int", envName)
	}
	return intEnv
}
