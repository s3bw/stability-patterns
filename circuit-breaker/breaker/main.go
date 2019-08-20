package main

import (
	"errors"
	"io/ioutil"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type Circuit struct {
	mutex    sync.Mutex
	isLocked bool
	health   string
	tick     time.Duration
}

func createCircuit(url string, checkTick time.Duration) *Circuit {
	return &Circuit{
		isLocked: false,
		health:   url + "/healthcheck",
		tick:     checkTick,
	}
}

func (c *Circuit) checkHealth() {
	c.mutex.Lock()
	c.isLocked = true
	for c.isLocked {
		for range time.Tick(c.tick) {
			log.Print("Attempting connection...")
			resp, _ := http.Get(c.health)
			if resp.StatusCode >= 200 && resp.StatusCode <= 299 {
				c.isLocked = false
			}
		}
	}
	log.Print("Unlocked breaker, good to go!")
	c.mutex.Unlock()
}

type Service struct {
	url     string
	circuit *Circuit
}

func (s *Service) Get(resource string) ([]byte, error) {
	if !s.circuit.isLocked {
		response, err := http.Get(s.url + "/" + resource)
		if err != nil || response.StatusCode >= 300 {
			log.Printf("%s", err)

			// Locked circuits will not be retried until
			// they are deemed healthy but the circuit loop.
			go s.circuit.checkHealth()
			log.Print("Connection Broken!.")

		} else {
			defer response.Body.Close()
			contents, err := ioutil.ReadAll(response.Body)
			if err != nil {
				log.Printf("%s", err)
			}
			return contents, nil
		}
	}
	return nil, errors.New("Circuit broken")
}

func NewService(root string) *Service {
	return &Service{
		url:     root,
		circuit: createCircuit(root, 3000*time.Millisecond),
	}
}

func main() {
	r := gin.Default()
	service1 := NewService("http://127.0.0.1:8081")

	r.GET("/ping", func(c *gin.Context) {
		contents, err := service1.Get("ping")

		// Failed, response with default
		if err != nil {
			log.Printf("%s", err)
			c.JSON(200, gin.H{
				"message": "default response",
			})
		} else {
			// Passed, response with contents
			c.JSON(200, gin.H{
				"message": string(contents),
			})
		}

	})
	r.Run() // listen and serve on 0.0.0.0:8080
}
