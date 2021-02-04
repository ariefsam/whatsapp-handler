package main

import (
	"encoding/gob"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	qrcodeTerminal "github.com/Baozisoftware/qrcode-terminal-go"
	"github.com/Rhymen/go-whatsapp"
	"github.com/joho/godotenv"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	err := godotenv.Load()
	if err != nil {
		log.Println(err)
	}
	<-time.After(2 * time.Second)

	//create new WhatsApp connection
	wac, err := whatsapp.NewConn(5 * time.Second)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error creating connection: %v\n", err)
		return
	}
	wac.AddHandler(&waHandler{wac})
	err = login(wac, false)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error logging in: %v\n", err)
		return
	}

	for true {
		<-time.After(1000 * time.Second)
		msg := whatsapp.TextMessage{
			Info: whatsapp.MessageInfo{
				RemoteJid: "6285219132737-1558176315@g.us",
			},
			Text: "halo halo",
		}

		msgId, err := wac.Send(msg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error sending message: %v", err)
			log.Println(err)
		} else {
			fmt.Println("Message Sent -> ID : " + msgId)

		}
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c

	//Disconnect safe
	fmt.Println("Shutting down now.")
	session, err := wac.Disconnect()
	if err != nil {
		log.Fatalf("error disconnecting: %v\n", err)
	}
	if err := writeSession(session); err != nil {
		log.Fatalf("error saving session: %v", err)
	}

}

func getWA(queueUrl string) (Message, error) {
	resp, err := http.Get(queueUrl)
	if err != nil {
		return Message{}, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)

	var msg Message
	var pr PageResponse
	err = json.Unmarshal(body, &pr)
	if err != nil {
		return msg, fmt.Errorf("Failed decode ")
	}
	msg = pr.Data
	return msg, nil

}

func send(wac *whatsapp.Conn, destinationNum string, text string) {
	num := strings.ReplaceAll(destinationNum, "+", "")
	fmt.Println("Sending to ", num)
	toWA := num + "@s.whatsapp.net"

	msg := whatsapp.TextMessage{
		Info: whatsapp.MessageInfo{
			RemoteJid: toWA,
		},
		Text: text,
	}

	msgId, err := wac.Send(msg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error sending message: %v", err)
		log.Println(err)
	} else {
		fmt.Println("Message Sent -> ID : " + msgId)

	}
}

func login(wac *whatsapp.Conn, newSession bool) error {
	//load saved session
	session, err := readSession()
	if err == nil && newSession == false {
		//restore session
		session, err = wac.RestoreWithSession(session)
		if err != nil {
			return fmt.Errorf("restoring failed: %v\n", err)
		}
	} else {
		//no saved session -> regular login
		qr := make(chan string)
		go func() {
			terminal := qrcodeTerminal.New()
			terminal.Get(<-qr).Print()
		}()
		session, err = wac.Login(qr)
		if err != nil {
			return fmt.Errorf("error during login: %v\n", err)
		}
	}

	//save session
	err = writeSession(session)
	if err != nil {
		return fmt.Errorf("error saving session: %v\n", err)
	}
	return nil
}

func readSession() (whatsapp.Session, error) {
	session := whatsapp.Session{}
	file, err := os.Open("a.gob")
	if err != nil {
		return session, err
	}
	defer file.Close()
	decoder := gob.NewDecoder(file)
	err = decoder.Decode(&session)
	if err != nil {
		return session, err
	}
	return session, nil
}

func writeSession(session whatsapp.Session) error {
	file, err := os.Create("a.gob")
	if err != nil {
		return err
	}
	defer file.Close()
	encoder := gob.NewEncoder(file)
	err = encoder.Encode(session)
	if err != nil {
		return err
	}
	return nil
}
