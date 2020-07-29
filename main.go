package main

import (
	"flag"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"strings"
	"time"

	evdev "github.com/gvalkov/golang-evdev"
	yaml "gopkg.in/yaml.v2"
)

// Config contains the parameters for the listener.
type Config struct {
	Zone          string          `yaml:"zone"`
	Gateway       string          `yaml:"sonosGateway"`
	DeviceName    string          `yaml:"deviceName"`
	Debug         bool            `yaml:"debug"`
	Bindings      map[int][]Press `yaml:"bindings"`
	Connected     []string        `yaml:"connected"`
	RefreshConfig int             `yaml:"refreshConfig"`
}

// Press is a binding for a keypress.
type Press struct {
	Length int      `yaml:"len"`  // Duration of press before activation
	Path   []string `yaml:"path"` // Sonos Gateway path
}

var (
	// Conf holds the parsed config values for the listener.
	Conf *Config

	// configPath is the filepath of the config file.
	configPath *string
)

func init() {
	configPath = flag.String("config", "config.yaml", "path of YAML config file")
	flag.Parse()
	loadConfig(true)

	// Poor man's hot reload. Reload the config file after every (n) seconds.
	if Conf.RefreshConfig > 0 {
		go func() {
			for range time.Tick(time.Second * time.Duration(Conf.RefreshConfig)) {
				loadConfig(false)
				logOut("CONFIG_RELOADED", Conf)
			}
		}()
	}
}

// loadConfig loads the configuration parameters from the named config file.
func loadConfig(first bool) {
	d, err := ioutil.ReadFile(*configPath)
	if err != nil {
		log.Fatal(err)
	}

	err = yaml.Unmarshal([]byte(d), &Conf)
	if err != nil {
		if first == true {
			log.Fatal(err)
		}
		log.Println("CONFIG err", err)
	}
}

func main() {
	log.Println("Started Bluetooth-Sonos Controller")

	for {
	RESET:

		// Try to find our bluetooth keyboard/button in the system's /dev/input.
		logOut("SCANNING", "for device: "+Conf.DeviceName)
		devices, err := evdev.ListInputDevices()
		if err != nil {
			logOut("LIST_ERR", err.Error())
		}

		// There may be a bunch of devices, so let's select it by name.
		for _, device := range devices {
			if device.Name == Conf.DeviceName {
				logOut("FOUND", device.Name+", "+device.Fn)

				// If we have the device, we need to open it.
				keyboard, err := evdev.Open(device.Fn)
				if err != nil {
					goto RESET
				}

				// After a long period of inactivity, the BLE device will
				// sleep. Indicate when the device is ready to take events
				// again by briefly muting the track.
				for i := 0; i < len(Conf.Connected); i++ {
					logOut("CONN_API_EVENT", Conf.Gateway+Conf.Zone+"/"+Conf.Connected[i])
					http.Get(Conf.Gateway + Conf.Zone + "/" + Conf.Connected[i])
					time.Sleep(time.Millisecond * 250)
				}

				// Listen for key events from the bluetooth button.
				var pressLength int // the length of events that have happened for a single press.
				for {
					keyEvent, err := keyboard.ReadOne()
					if err != nil {
						logOut("KEY_EVENT_ERR", err.Error())
						break
					}

					// Take control of the bluetooth keyboard exclusively.
					keyboard.Grab()
					defer keyboard.Release()

					logOut("KEY_EVENT", keyEvent)
					logOut("PRESS_LEN", pressLength)

					// We're only interested keypresses.
					if keyEvent.Type == evdev.EV_KEY {
						if keyEvent.Value == 0 { // 0 == Released

							// If we have a binding for the key, see if there are
							// and paths which have a matching press length.
							if binding, ok := Conf.Bindings[int(keyEvent.Code)]; ok {
								for i := 0; i < len(binding); i++ {
									if binding[i].Length <= pressLength {

										// If we found a path, loop through all the commands
										// for the path and send them to the Sonos API gateway.
										for j := 0; j < len(binding[i].Path); j++ {
											path := binding[i].Path[j]

											// If the path starts with a special pipe, pick a random
											// path from the list. eg. |opt1|opt2|opt3
											if path[0] == '|' {
												rand.Seed(time.Now().UnixNano())
												opts := strings.Split(path, "|")[1:]
												path = opts[rand.Intn(len(opts))]
											}

											logOut("API_EVENT", Conf.Gateway+Conf.Zone+"/"+path)
											_, err := http.Get(Conf.Gateway + Conf.Zone + "/" + path)
											if err != nil {
												logOut("API_EVENT_ERR", err.Error())
											}
										}
										break // only take the topmost length
									}
								}
							}

							// If the button is being held, we can optionally send
							// incremental updates for every other tick, such as volume up or down.
							// Set the length of the path to -1 to trigger this behaviour.
						} else if keyEvent.Value == 2 { // 2 == Held
							if binding, ok := Conf.Bindings[int(keyEvent.Code)]; ok && len(binding) == 1 && binding[0].Length == -1 { //&& pressLength%2 == 0 {
								logOut("CONT_API_EVENT", Conf.Gateway+Conf.Zone+"/"+binding[0].Path[0])
								_, err := http.Get(Conf.Gateway + Conf.Zone + "/" + binding[0].Path[0])
								if err != nil {
									logOut("CONT_API_EVENT_ERR", err.Error())
									break
								}
							}
							pressLength++
						} else if keyEvent.Value == 1 { // 1 = Pressed
							pressLength = 0
							logOut(" --- ", "")
						}
					}
				}

				logOut("LOST_DEVICE", Conf.DeviceName)
				goto RESET // If we lose the device, go back to searching for devices.
			}
		}

		time.Sleep(time.Second) // Wait a moment between searches.
	}

}

// logOut prints log information if Conf.Debug = true.
func logOut(pl string, str interface{}) {
	if Conf.Debug {
		log.Println(pl, str)
	}
}
