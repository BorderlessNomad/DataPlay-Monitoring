package main

import (
	"encoding/json"
	"github.com/codegangsta/martini"
	"github.com/fzzy/radix/redis"
	"log"
	"math"
	"net/http"
	"os"
	"strconv"
	"time"
)

var Logger *log.Logger = log.New(os.Stdout, "[API] ", log.Lshortfile)

func main() {
	port := "1938"
	if os.Getenv("DP_MONITORING_PORT") != "" {
		port = os.Getenv("DP_MONITORING_PORT")
	}

	m := martini.Classic()

	m.Get("/", GetInfo)
	m.Get("/info", GetInfo)

	Logger.Println("[martini] listening on :" + port)

	Logger.Fatal(http.ListenAndServe(":"+port, m))
}

func GetInfo(res http.ResponseWriter, req *http.Request) string {
	endPoint := "api"
	db := 1

	Logger.Println("GetRedisConnection", endPoint)
	c, err := GetRedisConnection()
	if err != nil {
		Logger.Println("Could not connect to Redis.", err)
		http.Error(res, "Could not connect to Redis.", http.StatusInternalServerError)
		return ""
	}

	defer c.Close()

	Logger.Println("SELECT DB", db)
	r := c.Cmd("SELECT", db)
	if r.Err != nil {
		Logger.Println("Could not select database from Redis.", r.Err)
		http.Error(res, "Could not select database from Redis.", http.StatusInternalServerError)
		return ""
	}

	Logger.Println("SORT", endPoint, "LIMIT", 0, 100, "GET", endPoint+":*->duration", "BY", endPoint+":*->timestamp", "DESC")
	sortedData, err := c.Cmd("SORT", endPoint, "LIMIT", 0, 100, "GET", endPoint+":*->duration", "BY", endPoint+":*->timestamp", "DESC").List()
	if err != nil {
		Logger.Println("Could not select keys from Redis.", err)
		http.Error(res, "Could not select keys from Redis.", http.StatusInternalServerError)
		return ""
	}

	data := make([]float64, 0)
	for _, val := range sortedData {
		v, _ := strconv.ParseFloat(val, 10)
		data = append(data, v)
	}

	mean := Mean(data)
	variation := Variation(data)
	standev := StandDev(data)

	info := map[string]interface{}{
		"endpoint":  endPoint,
		"mean":      math.Ceil(mean / 1000),
		"standev":   math.Ceil(standev / 1000),
		"variation": variation,
		"timestamp": time.Now(),
	}

	b, _ := json.Marshal(info)

	Logger.Println("Data:", string(b))

	return string(b)
}

func GetRedisConnection() (c *redis.Client, err error) {
	redisHost := "10.0.0.2"
	redisPort := "6379"

	if os.Getenv("DP_REDIS_HOST") != "" {
		redisHost = os.Getenv("DP_REDIS_HOST")
	}

	if os.Getenv("DP_REDIS_PORT") != "" {
		redisPort = os.Getenv("DP_REDIS_PORT")
	}

	Logger.Printf("Connecting to Redis on Host %s, Port %s...", redisHost, redisPort)
	c, err = redis.DialTimeout("tcp", redisHost+":"+redisPort, time.Duration(10)*time.Second)

	if err != nil {
		Logger.Println("Could not connect to the redis server.")
		return nil, err
	}

	Logger.Println("Connected!")

	return c, nil
}

/**
 * @brief calculates the coeficient of variation
 * @details calculates the relative variability (the ratio of the standard deviation to the mean)
 *
 * @param array of float values
 * @return variation value
 */
func Variation(x []float64) float64 {
	standDev := StandDev(x)
	mean := Mean(x)

	return standDev / mean
}

/**
 * @brief calculates the population standard deviation
 * @details (not the sample standard deviation as we are not interested in extrapolating)
 *
 * @param array of float values
 * @return standard deviation value
 */
func StandDev(x []float64) float64 {
	sumx := 0.0
	n := float64(len(x))
	mean := Mean(x)
	for _, v := range x {
		sumx += math.Pow((v - mean), 2)
	}

	return math.Sqrt(sumx / n)
}

/**
 * @brief calculates the mean average
 * @details
 *
 * @param float64 array of values
 * @return mean of values
 */
func Mean(x []float64) float64 {
	n := float64(len(x))
	sumx := 0.0
	for _, v := range x {
		sumx += v
	}

	return sumx / n
}

/**
 * @details Error Handler
 *
 * @param error
 * @return panic
 */
func check(e error) {
	if e != nil {
		panic(e)
	}
}
