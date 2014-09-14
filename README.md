# PiGlow Ambient
With this simple program written in Go you can have an awesome ambient light when it gets dark. The program will fade the PiGlow in around sunset and will fade it out around sunrise. This is still an alpha alpha version and not sure what direction I want to take it. Any ideas are welcome.

**Please let me know by email or issues or whatever if you are using this and want features!!!**

## Installation
Just build it with Go and if you are using a debian based installation you can use the include debian.sh for init script. Just copy debian.sh to /etc/init.d/piglow-ambient and the go binary to /usr/local/bin/piglow-ambient.

## TODO
- Config file (/etc/piglow-ambient) in JSON or custom
- Geolocation, so perhaps make it possible to just configure your address in config
- SIGTERM for correctly stopping
- SIGHUP for reloading config file
- It currently runs the whole night perhaps you want motion detection or something?
- Web interface for configuration and control so you can enable or disable from other computers??
- Ping checking if other computer(s) are on if not stop the ambient lighting

## About me
Author: Kevin Valk <kevin@kevinvalk.nl> 
