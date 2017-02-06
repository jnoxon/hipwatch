package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
)

import (
	"github.com/tbruyelle/hipchat-go/hipchat"
)

type config struct {
	Token     string   `json:"token"`
	Notify    []string `json:"notify"`
	Statefile string   `json:"statefile"`
}

type Hipwatch struct {
	config config
	client *hipchat.Client
}

func (h Hipwatch) notify(s string) {
	msg := &hipchat.MessageRequest{Message: s}

	for _, u := range h.config.Notify {
		_, err := h.client.User.Message(u, msg)
		if err != nil {
			log.Printf("Unable to deliver notification to %v: %v", s, err)
		}
	}
}

func (h Hipwatch) fetchusers() (*[]hipchat.User, error) {
	var users []hipchat.User

	var opts hipchat.UserListOptions
	opts.MaxResults = 1000

	for {
		u, _, err := h.client.User.List(&opts)
		if err != nil {
			return nil, err
		}

		users = append(users, u...)

		if len(u) == opts.MaxResults {
			opts.StartIndex += opts.MaxResults
		} else {
			break
		}
	}

	return &users, nil
}

func NewHipwatch(configfile string) *Hipwatch {
	var h Hipwatch

	d, err := ioutil.ReadFile(configfile)
	if err != nil {
		log.Fatalf("couldn't read %v: %v", configfile, err)
	}

	err = json.Unmarshal(d, &h.config)
	if err != nil {
		log.Fatalf("couldn't understand %v: %v", configfile, err)
	}

	h.client = hipchat.NewClient(h.config.Token)

	return &h
}

func main() {

	var state []hipchat.User
	var configfile string

	flag.StringVar(&configfile, "c", "config.json", "configuration filename")
	flag.Parse()

	h := NewHipwatch(configfile)

	users, err := h.fetchusers()
	if err != nil {
		panic(err)
	}

	d, _ := ioutil.ReadFile(h.config.Statefile)
	err = json.Unmarshal(d, &state)
	if err != nil {
		log.Println("invalid state: resetting")
		state = *users
	}

	cur := make(map[int]*hipchat.User)
	prev := make(map[int]*hipchat.User)

	for i, u := range *users {
		cur[u.ID] = &(*users)[i]
	}

	for i, u := range state {
		prev[u.ID] = &state[i]
	}

	for k, v := range prev {
		if cur[k] == nil {
			h.notify(fmt.Sprintf("Goodbye to @%v (%v)", v.MentionName, v.Name))
		}
	}

	for k, v := range cur {
		if prev[k] == nil {
			h.notify(fmt.Sprintf("Say hello to @%v (%v)", v.MentionName, v.Name))
		}
	}

	j, err := json.MarshalIndent(users, "", "  ")
	if err == nil {
		err = ioutil.WriteFile(h.config.Statefile, j, 0644)
		if err != nil {
			log.Fatalln("Unable to write state file: %v", err)
		}
	} else {
		log.Fatalln("Unable to marshal json: %v", err)
	}

}
