package main

import (
	"errors"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type Circuit struct {
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
	c.isLocked = true
	for range time.Tick(c.tick) {
		log.Print("Retrying connection...")
		resp, err := http.Get(c.health)
		if err != nil {
			log.Printf("%s", err)
			continue
		} else if resp.StatusCode >= 200 && resp.StatusCode <= 299 {
			log.Printf("%d", resp.StatusCode)
			break
		}
	}
	log.Print("Unlocked breaker, good to go!")
	c.isLocked = false
}

type Service struct {
	url     string
	circuit *Circuit
}

func (s *Service) Get(resource string) ([]byte, error) {
	// If endpoint is not healthy the circuit will break
	// the endpoint will only be hit again when it is
	// deemed healthy
	if !s.circuit.isLocked {
		response, err := http.Get(s.url + "/" + resource)

		// If there is an error break the circuit.
		// We can also break the circuit when there is an invalid
		// response status code.
		if err != nil {
			log.Printf("%s", err)
			log.Print("Connection Broken!")
			go s.circuit.checkHealth()

		} else if response.StatusCode >= 504 {
			// log.Printf("%d", response.StatusCode)
			go s.circuit.checkHealth()

		} else {
			defer response.Body.Close()
			contents, err := ioutil.ReadAll(response.Body)
			if err != nil {
				log.Printf("%s", err)
			}
			return contents, nil
		}
	}
	return nil, errors.New("connection is not ready")
}

// NewService client
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

		// Bad response or broken Connection, respond with default
		// otherwise return content from upstream service
		if err != nil {
			log.Printf("%s", err)
			c.JSON(200, gin.H{
				"message": "default response",
			})

		} else {
			c.JSON(200, gin.H{
				"message": string(contents),
			})
		}

	})
	// listen and serve on 0.0.0.0:8080
	r.Run()
}
