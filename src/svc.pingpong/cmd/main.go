package main

import (
	"bufio"
	protos "common/svcprotos/gen"
	"fmt"
	"log"
	"os"
	"strconv"
	"svc.pingpong/client"
)

const (
	SendPing = 1
	SendPong = 2
	Exit     = 3
	Invalid  = 4
)

var (
	reader        = bufio.NewReader(os.Stdin)
	serverAddress string
	port          int
	resp          *protos.PingPongResponse
)

func displayMenu() int {
	fmt.Println("Choose an option from below:")
	fmt.Println("[1]. Send ping")
	fmt.Println("[2]. Send pong")
	fmt.Println("[3]. Exit")
	fmt.Print("Enter your option: ")

	text, err := reader.ReadString('\n')
	if err != nil {
		log.Fatalf("Failed reading input: %+v", err)
	}

	if len(text) > 0 {
		text = text[:len(text)-1]
	}

	option, err := strconv.Atoi(text)
	if err != nil {
		log.Printf("Invalid input: %s\n", text)
		return Invalid
	}

	if option < 0 || option > 3 {
		return Invalid
	}

	return option
}

func main() {
	serverAddress = "localhost"
	port = 9090

	fmt.Println("Welcome to Ping-Pong!")

	pingpongClient, err := client.NewPingPongClient(serverAddress, port)
	if err != nil {
		log.Fatalf("Failed starting %s[%d]", serverAddress, port)
	}

	for {
		resp = nil
		option := displayMenu()

		switch option {
		case SendPing:
			resp, err = pingpongClient.SendPing("Hi!")
			if err != nil {
				fmt.Printf("Error sending ping: %+v", err)
			}
		case SendPong:
			resp, err = pingpongClient.SendPong("Hi!")
			if err != nil {
				fmt.Printf("Error sending pong: %+v", err)
			}
		case Invalid:
			fmt.Printf("Please try again!\n")
		case Exit:
			fmt.Printf("Bye-bye!\n")
			return
		}

		if resp != nil {
			fmt.Printf("Received: %s\n\n", string(resp.Message))
		}
	}
}
