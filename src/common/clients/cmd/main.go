package main

import (
	"bufio"
	"common/clients/echo"
	"common/clients/pingpong"
	"fmt"
	"log"
	"os"
	"os/user"
	"reflect"
	"strconv"
)

const (
	SendPing = 1
	SendPong = 2
	SendEcho = 3
	Exit     = 4
	Invalid  = 5
)

var (
	reader                                   = bufio.NewReader(os.Stdin)
	pingPongServerAddress, echoServerAddress = "localhost", "localhost"
	pingPongPort, echoPort                   = 9090, 10010
)

func displayMenu() int {
	fmt.Println("Choose an option from below:")
	fmt.Println("[1]. Send ping")
	fmt.Println("[2]. Send pong")
	fmt.Println("[3]. Send echo")
	fmt.Println("[4]. Exit")
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

	if option < 0 || option >= Exit {
		return Invalid
	}

	return option
}

func main() {
	currentUser, err := user.Current()
	if err != nil {
		log.Fatalf("Failed retrieving current logged user! ERR-%+v", err)
	}
	userName := "<current user>"
	if currentUser != nil {
		userName = currentUser.Name
	}

	fmt.Printf("Welcome %s!\n", userName)

	echoClient, err := echo.NewClient(echoServerAddress, echoPort)
	if err != nil {
		log.Fatalf("Failed starting ping-pong client %s[%d]! ERR=%+v", echoServerAddress, echoPort, err)
	}

	pingpongClient, err := pingpong.NewClient(pingPongServerAddress, pingPongPort)
	if err != nil {
		log.Fatalf("Failed starting ping-pong client %s[%d]! ERR=%+v", pingPongServerAddress, pingPongPort, err)
	}

	for {
		var response interface{}
		var option = displayMenu()

		switch option {
		case SendPing:
			response, err = pingpongClient.SendPing("Hi!")
			if err != nil {
				fmt.Printf("Error sending ping: %+v", err)
			}
		case SendPong:
			response, err = pingpongClient.SendPong("Hi!")
			if err != nil {
				fmt.Printf("Error sending pong: %+v", err)
			}
		case SendEcho:
			response, err = echoClient.SendEcho(fmt.Sprintf("Hi %s!", userName))
			if err != nil {
				fmt.Printf("Error sending echo: %+v", err)
			}
		case Invalid:
			fmt.Printf("Please try again!\n")
		case Exit:
			fmt.Printf("Bye-bye!\n")
			return
		}

		if response != nil {
			r := reflect.ValueOf(response)
			f := reflect.Indirect(r).FieldByName("Message")
			message := f.String()

			if len(message) > 0 {
				fmt.Printf("\n\tReceived: %s\n\n", message)
			}
		}
	}
}
