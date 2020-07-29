
### Bluetooth Sonos Controller
#### Control your Sonos speakers with cheap bluetooth controller buttons

#### What is this?
Surprisingly there's not many hardware remotes for Sonos, and those that do exist are either [more expensive](https://www.sonos.com/en-gb/shop/senic-nuimo-control-starter-kit.html) than the speakers themselves , or are a little bit _too_ [cheap and cheerful](https://www.ikea.com/gb/en/p/symfonisk-sound-remote-white-60370480/). Fortunately there's a plethora of off-brand Bluetooth Media Control Buttons on [Amazon](https://www.amazon.co.uk/s?k=bluetooth%20media%20button&ref=nb_sb_noss_2) and [eBay](https://www.ebay.co.uk/p/2239568803) ranging between $8 and $30, depending on how much you want to pay.

I have a [Satechi Media Button](https://www.amazon.co.uk/Satechi-Bluetooth-Button-compatible-Samsung-Media/dp/B00RM75NL0/) which I purchase for about $25, just because it seemed to be the most credible and had plenty of positive reviews. For this reason, the project is setup to use the Satechi button, however there's no reason why it shouldn't work with just about any other bluetooth input device.

The purpose of this project is to provide a small, customisable bluetooth listener which can use to interact with your Sonos system (everything from play/pause and volume to playing favourites).

#### Acknowledgements
Originally when I bought my Satechi Button, I intended to use it with [SvenSommer's BlueSonosButton](https://github.com/SvenSommer/BlueSonosButton). However, there was more customisation I wanted to do, and since I'm better at Go than I am at Python, this project is written in Go. Much of the configuration I learned directly from SvenSommer.

#### Caveats
This is a bit of a quick-and-dirty project to meet a goal. It's not perfect. That said, I've used to every day for a month and it never failed me. Pull requests and issues are welcome to tidy and formalise and add new features!

BLE devices go into a deep sleep when they haven't been used for a while. This means that after a few minutes, you may have to press the button and wait a second for it to respond. The listener is setup to mute and unmute the speakers for a second to indicate that the remote is reconnected.

In an ideal world this wouldn't happen, but I noticed the IKEA Symfonisk remote has the same issue, so I guess that's just how it is. You do get used to it.

* **This project requires `evdev`, so it currently only works on linux.**

#### Prerequisites
You will need two things:

 - Something to run the listener on. I recommend a Raspberry Pi or something similar (I'm using a [Pi Zero](https://thepihut.com/products/raspberry-pi-zero-wh-with-pre-soldered-header)).
 - A bluetooth media button. I'm using a Satechi Media Button.

For the purposes of this readme, we'll assume you're using the same.

#### Setup
We will do three things:
1. Clone this repo to your device and build the listener.
2. Install Go on your device.
3. Install Jishi's amazing [node-sonos-http-api](https://github.com/jishi/node-sonos-http-api).
4. Configure the button to work with your Sonos. 
5. Configure the listener and sonos api to start on boot.

We'll assume that all of this will happen on your Raspberry Pi (or other device you are using).

##### Clone the Repo
	
	git clone https://github.com/mochi-co/bluetooth-sonos-controller.git

Build the listener: 
	
	cd bluetooth-sonos-controller
	go build -o bluetooth-sonos-controller

##### Install Go

	sudo apt-get install golang

##### Install Node-Sonos-HTTP-Api
	git clone https://github.com/jishi/node-sonos-http-api.git
	cd node-sonos-http-api
	npm install --production
	npm start

I recommend you run `npm start` in a new console tab since it won't be backgrounded. You can go to https://github.com/jishi/node-sonos-http-api to read more about the Sonos API. There is also a Docker version you can use if that's your thing and your device is powerful enough.

#####  Connect to your Media Button
First you need to pair the bluetooth button to your device. We can do this with `bluetoothctl`:

```
$ bluetoothctl
agent on
scan on
```
This will then scan for nearby bluetooth devices. If you are lucky, you will see your Media Button show up. Note the device ID and also the name (for later). Once you have seen the device ID, trust and pair it:
```
trust DC:2C:26:BD:DD:9B
pair DC:2C:26:BD:DD:9B
connect DC:2C:26:BD:DD:9B
```
Once it says `Connected`, you can type  `exit` to leave `bluetoothctl`.

##### Configure the listener
In this repo you will find an example `config.yml` which contains everything you need to map your new media button to various Sonos features. However, in order to listen to the button, you'll need to tell the listener what the button is called.

Like many bluetooth devices, the Media Button may expose more than one `/dev/input` address. Each of these have slightly different purposes and different names, and you need to know which one to listen on. For Satechi Media button, it's `Satechi Media Button Consumer Control`, however if you need to check, you can use `evtest`:

	sudo apt-get install evtest
	evtest

	Available devices:
	/dev/input/event0:  Satechi Media Button Keyboard
	/dev/input/event1:  Satechi Media Button Consumer Control
	/dev/input/event2:  Satechi Media Button System Control

Since the button uses BLE, you'll need to press it a few times before running `evtest`. You can then enter the index id (0-2, etc) of the input you wish to test. If you press volume+ and it works, and the others don't, that's probably your device.

Once you have the name of your input device, open up your `config.yml`, and set the `deviceName` property, eg:
```
zone: "MySonosZone"
sonosGateway: "http://localhost:5005/"
deviceName: "Satechi Media Button Consumer Control"
```

##### Map the buttons to Sonos events
The `config.yml` contains a basic, default configuration which should work with all standard buttons (play/pause, next, prev, vol up, vol down). You can do a lot more with this controller, which is described further down the readme, but for now just familiarise yourself with the options.

##### Configure the listener and Sonos API to start on boot
This is the important one. I looked through lots of different options, but, in the end, the simplest and most reliable was using the `@reboot` parameter in `crontab`.

You can add them by running the crontab editor on your pi:
```
sudo crontab -e
```

And then add the following lines, substituting the paths for those on your system:
```
@reboot sleep 60 && sudo npm start --prefix /home/pi/Dev/node-sonos-http-api &
@reboot sleep 60 && sudo /home/pi/Dev/bluetooth-sonos/bluetooth-sonos-controller -config /home/pi/Dev/bluetooth-sonos/config.yaml &
```

This will allow the listener and API to start on boot.

#### Customising your Button
If everything works, you should be able to enter your `bluetooth-sonos-controller` folder and run the listener. 

First, set the `debug` parameter in your config file to `true`. This will output logs for what the listener is doing. You might also want to set the `refreshConfig` down to `5` seconds if you are planning on doing active config development, but remember to set it back to a reasonable value when you are finished. The listener uses a cheap hot-reload function which simply re-reads the config file. You can start the listener like so:

	./bluetooth-sonos-controller -config config.yaml

This will start outputting an event logs. Press play/pause and you will see the event in the console.

In another window, open up your `config.yml`. You can now edit your config and the listener will reload every (n) seconds. When you are satisfied, you can reboot, or run the command you added to the crontab (eg `sudo /home/pi/Dev/bluetooth-sonos/bluetooth-sonos-controller -config /home/pi/Dev/bluetooth-sonos/config.yaml &`)

### Understanding config.yml
 The command in`connected` and `bindings` are API paths affixed to the `sonosGateway` address, so if the command is `pause`, then it will call `http://localhost:5005/pause`. 
 
A full list of API paths can be found on https://github.com/jishi/node-sonos-http-api, and you can use any of them, so get creative!

````yaml

# zone is the Sonos zone you wish to control with the remote.
zone: "JonBedroom"

# sonosGateway is the address of the Node Sonos API. 
# Normally you will run this on the same device, but if not you 
# can change it here.
sonosGateway: "http://localhost:5005/"

# deviceName is the name of the input device you found using evtest.
deviceName: "Satechi Media Button Consumer Control"

# debug turns logging on and off.
debug: true

# refreshConfig is the number of seconds between hot-reloads of the 
# config file.
refreshConfig: 30

# connected is the event sequence that occurs when the device 
# reconnects. In this case, it pauses for a moment, then plays. 
connected:
  - "pause"
  - "play"

# bindings is the meat of the config and maps key events (refer to the 
# event codes output when selecting a device in evtest, or pressing 
# buttons when logging).
bindings:

  # A basic single path binding
  115: # Volume Up keycode
    - path: 
	    - "volume/+2" # >> http://localhost:5005/volume/+2

  # A binding can have multiple paths which are triggered depending
  # on how long the button was held. Multiple bindings must be listed
  # in order of longest to shortest (default len being 0). The len
  # properties represents the number of keypress events that are 
  # triggered during the hold. On the Satechi button, len 20 is 
  # about 3 seconds. You will have experiment to find your preferred
  # values.
  # This binding has two paths, an instant -2, or if the button is
  # held for a second, then the volume is dropped by -5 instead.
  114: # Volume Down keycode
    - path:
	- "volume/-5" 
      len: 10 
    - path: 
	- "volume/-2"

  # Multi-path bindings can also be used to change the currently 
  # playing track. In this case, holding the next button for 3 seconds
  # will play the Deezer Classical station which has been added to my
  # Sonos as a favourite. Otherwise it will just skip track.
  163: # Next keycode
    - path: 
	    - "favourite/Classical"
      len: 20
    - path: 
      - "next"

  # As seen in Connected, you can have multiple steps in a single path.
  # These steps will be executed in sequence.
  # The below example will announce the playlist change in an Australian
  # accent, then temporarily enable shuffle mode when the favourite 
  # playlist is triggered using long play so it always starts on a random
  # track.
  165: # Prev keycode
    - path: 
	    - "say/calm piano/en-au/10"
	    - "shuffle/on"
	- "favourite/Calm Piano"
	- "shuffle/off"
      len: 20
    - path: 
      - "previous"

````

There are two special syntax features available in the listener. These are `pipe` syntax, which creates a list which will be randomly selected from, and `len: -1`, which is a bit more experimental. Let's look at how to use them:

```yaml
  # pipe syntax allows you to provide a list of different commands which
  # will be selected at random when a path is triggered. This allows you 
  # to create some natural variations in your commands. 
  # In the below example, it used to randomly select a different Sonos 
  # favourite when the play-pause button is held.
  # To enable pipe mode, the first character must be a '|' pipe, and each
  # command must be separated by a | pipe.
  164: # Play/Pause keycode
    - path: 
        - "|favourite/Ocean Rain Sounds|favourite/The Sound of Rain|favourite/Woodland Rain"
      len: 20
    - path: 
      - "playpause"

  # Setting len: -1 sets a binding into continuous mode. All the time the
  # button is held down, it will continuously fire an API request. In this
  # example it's used to lower the volume as the button is held. You can only
  # have 1 path if you are using len: -1 (no long press). 
  114: # Volume Down keycode
    - path: 
        - "volume/-2"
      len: -1
```

### Tips
* If you're a Deezer user, you can access your "My Flow" by favouriting it and then referencing `favourite/Flow,%20your%20personal%20mix`.


### Contributions
Contributions and feedback are both welcomed and encouraged! Open an [issue](https://github.com/mochi-co/bluetooth-sonos-controller/issues) to report a bug, ask a question, or make a feature request.

