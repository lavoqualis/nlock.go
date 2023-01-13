// SPDX-FileCopyrightText: 2023 KåPI Tvätt AB <peter.magnusson@rikstvatt.se>
//
// SPDX-License-Identifier: MIT License

package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/lavoqualis/nlock.go"
	"github.com/nats-io/nats.go"
)

type Config struct {
	URL string

	User     string
	Password string
	Bucket   string
}

func getenv(key, defaultv string) string {
	v := os.Getenv(key)
	if v == "" {
		return defaultv
	}
	return v
}

func getConfig() Config {
	return Config{
		URL: getenv("KVCLUSTER_URL", nats.DefaultURL),

		User:     getenv("KVCLUSTER_USER", "admin"),
		Password: getenv("KVCLUSTER_PASSWORD", "secret"),
		Bucket:   getenv("KVCLUSTER_BUCKET", "process_locks"),
	}
}

func main() {
	cfg := getConfig()
	log.Printf("config %+v", cfg)
	nc, err := getConnection(&cfg)
	if err != nil {
		log.Fatalf("error connecting to nats: %v", err)
	}
	defer nc.Drain()

	log.Println("getting js")
	js, err := nc.JetStream()
	if err != nil {
		log.Fatalf("could not get JetStream context: %v", err)
	}

	log.Println("getting kv")

	kv, err := js.KeyValue(cfg.Bucket)
	// kv, err := js.CreateKeyValue(&nats.KeyValueConfig{
	// 	Bucket:       fmt.Sprintf(cfg.Bucket),
	// 	TTL:          time.Second * 30,
	// 	MaxValueSize: 1024,
	// })
	if err != nil {
		log.Fatalf("could not get bucket: %v", err)
	}
	host, _ := os.Hostname()
	id := fmt.Sprintf("%s-%d", host, os.Getpid())
	m, err := nlock.New(id, kv)
	if err != nil {
		log.Fatalf("could not create lock managager: %v", err)
	}
	l, err := m.Claim("lock.1")
	if err != nil {
		log.Fatalf("could not claim lock: %v", err)
	}
	ctrlc()
	l.Release()
	nc.Drain()

}

func ctrlc() {
	done := make(chan os.Signal, 1)
	signal.Notify(done, syscall.SIGINT, syscall.SIGTERM)
	fmt.Println("Blocking, press ctrl+c to continue...")
	<-done // Will block here until user hits ctrl+c
	log.Println("ctrl+c pressed")
}

func getConnection(cfg *Config) (*nats.Conn, error) {
	return nats.Connect(cfg.URL,
		nats.UserInfo(cfg.User, cfg.Password))
}
