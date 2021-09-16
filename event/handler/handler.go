package handler

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"log"
	"os"
	"sd-for-vm-telemetry/event"
)

const (
	fileFullPath = "/data/file_sd/staticConfigurations.json"
)

type Handler interface {
	Start()
}

type handler struct {
	eventChan <-chan event.Event
	namespace string
}

func NewHandler(eventChan <-chan event.Event) Handler {
	return &handler{
		eventChan: eventChan,
	}
}

func (h *handler) Start() {
	go h.handleEvent()
}

func (h *handler) handleEvent() {
event:
	for e := range h.eventChan {
		config, err := getCurConfig()
		if err != nil {
			log.Printf("Read config file error, event not handle, type: %v, target: %s, err: %v\n", e.Type, e.Address, err)
			continue
		}

		switch e.Type {
		case event.TypeDelete:
			log.Printf("handle deleted workload %s", e.Address)
			toDelete := 0
		outer:
			for i, target := range config {
				for _, ip := range target["targets"] {
					if ip == e.Address {
						toDelete = i
						break outer
					}
				}
			}
			config = append(config[:toDelete], config[toDelete+1:]...)
			log.Printf("Deleted VM workload %s\n", e.Name)
		default:
			// Remove duplicates from the node IPs.
			existsDupEP := isDuplicate(config, e.Address)
			if existsDupEP {
				log.Printf("VM workload %s exists\n", e.Name)
				continue event
			}
			log.Printf("handle update workload %s", e.Address)
			newTarget := make(map[string][]string)
			newTarget["targets"] = append(newTarget["targets"], e.Address)
			config = append(config, newTarget)
			log.Printf("Registered VM workload %s \n", e.Name)
		}

		err = setConfig(config)
		if err != nil {
			log.Printf("Write config error, err: %v\n", err)
		}
	}
}

func getCurConfig() ([]map[string][]string, error) {
	bs, err := ioutil.ReadFile(fileFullPath)
	if err != nil && err != io.EOF {
		return nil, err
	}

	config := make([]map[string][]string, 0)
	err = json.Unmarshal(bs, &config)
	if err != nil {
		return nil, err
	}
	return config, nil
}

func isDuplicate(existing []map[string][]string, newTarget string) bool {
	for _, target := range existing {
		for _, ip := range target["targets"] {
			if ip == newTarget {
				return true
			}
		}
	}
	return false
}

func setConfig(config []map[string][]string) error {
	bs, _ := json.Marshal(config)
	return os.WriteFile(fileFullPath, bs, 0644)
}
