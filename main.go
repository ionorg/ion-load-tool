package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/pion/ion-load-tool/ion"
	"github.com/pion/ion-load-tool/webm"
	"github.com/spf13/viper"
)

func run(room, sfu, input string, produce, consume bool, n, duration int, stagger time.Duration) {
	var clients []*ion.LoadClient
	timer := time.NewTimer(time.Duration(duration) * time.Second)

	for i := 0; i < n; i++ {
		client := ion.NewLoadClient(fmt.Sprintf("client_%s_%d", room, i), room, sfu, input)

		if produce {
			// Validate type
			if input != "" {
				ext, ok := webm.ValidateVPFile(input)
				log.Println(ext)
				if !ok {
					panic("Only IVF and WEBM containers are supported.")
				}
			}

			client.Publish()
		} else {
			panic("unsupported configuration. must produce or consume")
		}

		clients = append(clients, client)

		time.Sleep(stagger)
	}

	select {
	case <-timer.C:
	}

	for i, a := range clients {
		// Signal shutdown
		a.Close()
		// Staggered shutdown.
		if len(clients) > 1 && i < len(clients)-1 {
			time.Sleep(stagger)
		}
	}
}

type SessionConfig struct {
	Rooms    int     `mapstructure:"rooms"`    // Rooms to join.
	Clients  int     `mapstructure:"clients"`  // Number of clients to start.
	Input    string  `mapstructure:"input"`    // path to the input media.
	SFU      string  `mapstructure:"sfu"`      // ion-sfu grpc url.
	Stagger  float64 `mapstructure:"stagger"`  // Number of seconds to stagger client start and stop.
	Duration int     `mapstructure:"duration"` // Number of seconds to run test for
	Audio    bool    `mapstructure:"audio"`    // Publish audio stream from webm file
	Produce  bool    `mapstructure:"produce"`
	Consume  bool    `mapstructure:"consume"` // Run subscribe to all streams and consume data
}

type Config struct {
	SessionConfig `mapstructure:"session"`
}

var (
	conf = Config{}
	file string
)

func loadConfig() bool {
	_, err := os.Stat(file)
	if err != nil {
		return false
	}

	viper.SetConfigFile(file)
	viper.SetConfigType("toml")

	err = viper.ReadInConfig()
	if err != nil {
		fmt.Printf("config file %s read failed. %v\n", file, err)
		return false
	}
	err = viper.GetViper().Unmarshal(&conf)
	if err != nil {
		fmt.Printf("ion-load-tool: loading config file %s failed. %v\n", file, err)
		return false
	}
	fmt.Printf("config loaded! %v\n", conf)
	return true
}

func main() {
	flag.StringVar(&file, "c", "", "config file")
	flag.Parse()

	if loaded := loadConfig(); loaded {
		log.Println("configuration successfully loaded!")
	} else {
		log.Println("using default settings")
	}

	staggerDur := time.Duration(conf.Stagger*1000) * time.Millisecond

	for i := 0; i < conf.Rooms; i++ {
		addr := ":" + conf.SFU
		go run(fmt.Sprintf("room %d", i), addr, conf.Input, conf.Produce, conf.Consume, conf.Clients, conf.Duration, staggerDur)
	}

	// Setup shutdown
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-sigs:
	}

	log.Println("Wait for client shutdown")
	log.Println("All clients shut down")
}
