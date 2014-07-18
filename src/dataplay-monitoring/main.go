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

func main() {
	m := martini.Classic()

	m.Get("/api/info", GetInfo)

	// m.Run()
	port := "4000"
	if os.Getenv("monitorport") != "" {
		port = os.Getenv("monitorport")
	}
	log.Println("[martini] listening on :" + port)

	log.Fatal(http.ListenAndServe(":"+port, m))
}

func GetInfo(res http.ResponseWriter, req *http.Request) string {
	endPoint := "api"

	c, err := GetRedisConnection()
	if err != nil {
		http.Error(res, "Could not connect to Redis.", http.StatusInternalServerError)
		return ""
	}

	defer c.Close()

	r := c.Cmd("SELECT", 1) // DB 1
	if r.Err != nil {
		http.Error(res, "Could not select database from Redis.", http.StatusInternalServerError)
		return ""
	}

	sortedData, err := c.Cmd("SORT", endPoint, "LIMIT", 0, 100, "GET", endPoint+":*->duration", "BY", endPoint+":*->timestamp", "DESC").List()
	if err != nil {
		http.Error(res, "Could not select keys from Redis", http.StatusInternalServerError)
	}

	data := make([]float64, 0)
	for _, val := range sortedData {
		v, _ := strconv.ParseFloat(val, 10)
		data = append(data, v)
	}

	mean := Mean(data)
	variance := Variation(data)
	standev := StandDev(data)

	info := map[string]interface{}{
		"endpoint": endPoint,
		"time": map[string]interface{}{
			"mean":     math.Ceil(mean) / 1000,
			"variance": variance,
			"standev":  math.Ceil(standev) / 1000,
		},
	}
	b, _ := json.Marshal(info)

	return string(b)
}

func GetRedisConnection() (c *redis.Client, err error) {
	redishost := "10.0.0.2:6379"
	if os.Getenv("redishost") != "" {
		redishost = os.Getenv("redishost")
	}

	c, err = redis.DialTimeout("tcp", redishost, time.Duration(10)*time.Second)

	return c, err
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
