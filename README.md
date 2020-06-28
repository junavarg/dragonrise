![alt text](image014.jpg?raw=true "Title")

The dragonrise program allows:
1) Monitor the status of the 12 switches (and 2 three position switches) of the popular “Generic USB joystick” controller card from DragonRise Inc.
2) Publish the status of these switches to a server or MQTT broker every time a change occurs in any of them.
It is written in Golang so it is a direct executable on the processor (it does not require a runtime) and it can work in the background as a daemon or service. 
The same process allows multiple controller cards to be monitored which is ideal for applications requiring a high number of switches.
It allows transporting the MQTT protocol over TCP or over Websockets, including the encrypted version over TLS. In the latter case, the inclusion of the certificates of the MQTT server trust chain is not required.
Oriented to run on single-board computers (SBC), being written in Golang, it can run on Linux or even, with the necessary modifications, on Windows.
Along with the program, a UDEV rules file is included to ensure the order and correct identification of the controller cards according to the USB port where they are plugged in.




To install from sources: 
Install golang compiler
$ sudo apt-get install golang

wget -O - https://raw.githubusercontent.com/junavarg/dragonrise/master/install-from-sources.sh | sudo bash

