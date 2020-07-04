![alt text](image014.jpg?raw=true "Title")
<h1>Introduction</h1>
The dragonrise program allows:
<li>Monitor and returns by stdout a JSON structure with the status of the 12 switches (and 2 three position switches) of the popular “Generic USB joystick” controller card from DragonRise Inc.</li>
<li>Publish the status of these switches to a server or MQTT broker every time a change occurs in any of them.</li>

It is written in Golang so it is a direct executable on the processor (it does not require a runtime) and it can work in the background as a daemon or service. 
The same process allows multiple controller cards to be monitored which is ideal for applications requiring a high number of switches.

It allows transporting the MQTT protocol over TCP or over Websockets, including the encrypted version over TLS. In the latter case, the inclusion of the certificates of the MQTT server trust chain is not required.

Oriented to run on single-board computers (SBC), being written in Golang, it can run on Linux or even, with the necessary modifications, on Windows.
Along with the program, a UDEV rules file is included to ensure the order and correct identification of the controller cards according to the USB port where they are plugged in.

More info: https://junavarg.github.io/dragonrise/
<h1>Instalation</h1>
<h2>Install binary in rpi</h2>
<p>Download and execute the install script
<p><i>$ wget -O - https://raw.githubusercontent.com/junavarg/dragonrise/master/bin/rpi/install.sh | sudo bash</i>
<p>This script: 
<li>  Download de executable, copy it in /usr/local/bin, and change permissions to 777</li>
<li>  Create directory for state files in /var/lib/dragonrise, with owner root and change permissions to 777</li>
<li>  Download a UDEV rule file with name 31-dragonriseRPi.rules, copy it to /etc/udev/rules.d/ and reload the rules with udevadm command.</li>
<p> NOTE: If a USB HUB is used, it will be necesary change accordingly the rules.
<h2>Install binary in Linux intel x64/amd64</h2>
<p> 1) Download the binary-executable 
  <p><i>wget https://raw.githubusercontent.com/junavarg/dragonrise/master/bin/linux-amd64/dragonrise</i>
<p> 2) Copy it in /usr/local/bin, and change permissions to 777
<p> 3) Create directory for state files in /var/lib/dragonrise, with owner root and change permissions to 777
<p> 4) Create a UDEV rule file (for example 31-dragonrise.rules) in /etc/udev/rules.d/ and reload the rules with udevadm command.
<h2>Install from sources</h2>
<p>Install golang compiler
<p>$ sudo apt-get install golang
<p>$ wget -O - https://raw.githubusercontent.com/junavarg/dragonrise/master/install-from-sources.sh | sudo bash

<h1>Use</h1>
<p>Use: dragonrise [options] [device_file1] [device_file2]… 
<p>Options:
<p>&nbsp&nbsp -mqpub <url>
<p>&nbsp&nbsp   Specify the MQTT broker URL and the root of a topic (basetopic) where to post the status every time an event occurs. The url format is
<p>&nbsp&nbsp   protocol://[user[:password]@]host.domain.tld:port/base_topic
<p>&nbsp&nbsp   Options for protocol: tcp, ssl, ws, wss
<p>&nbsp&nbsp   Examples:
<p>&nbsp&nbsp&nbsp&nbsp     -mqpub = tcp://host.domain.dom:1883/base_topic
<p>&nbsp&nbsp&nbsp&nbsp     -mqpub = ssl://pepe@host.domain.dom:8883/base_topic
<p>&nbsp&nbsp&nbsp&nbsp     -mqpub = ws://host.domain.dom:80/base_topic
<p>&nbsp&nbsp&nbsp&nbsp     -mqpub = wss://pepe:p2ssw0d@host.domain.dom:443/base_topic
  
<p>&nbsp&nbsp  -mqpub2 <url>
<p>&nbsp&nbsp  -mqpub3 <url>
<p>&nbsp&nbsp&nbsp&nbsp    Two additional brokers to which the program sends events
  
The messages are published in 'clean session' with qos 0 and with 'retained flag' so that on each new connection the subscriber receives a message with the current status.

<h2>Examples</h2>

<i>$ dragonrise</i>

Try to read switch status and events from a USB joystick card in /dev/input/js0 (default).
In a Raspberry Pi, /dev/input/js0 normally will correspond to the first USB joystick card conected to any USB port.
Program is alwais running until is killed (e.g., with ctrl-c).
If device file is not present, program reintent open it every 1 second.
Status is printed in stdout in JSON, for example:
Info and errors are printed in stderr.

<i>$ dragonrise /dev/input/js0 /dev/input/js1  2>/dev/null</i>

Try to read switch status and events from /dev/input/js0 and /dev/input/js1. Program does not show info or errors
Without the proper UDEV rule there is no way to warrant which USB card correspond to which device file (/dev/input/js0 or /dev/input/js1).

<i>$ dragonrise /dev/dragonrise_3</i>

Try to read switch status and events from a USB joystick card connected to USB number 3 (/dev/dragonrise_3) and no other, thanks to a UDEV rule that makes the correct mapping. In a Raspebrry Pi USB 3 corresponds to external USB port number 2 (USB number 1 correspond to ethernet NIC).

<i>$ dragonrise -mqpub=tcp://test.mosquitto.org:1883/base_topic /dev/dragonrise_3</i>

Publish in topic /base_topic/dragon_rise/event of MQTT broquer in url test.mosquito.org using transport tcp in clear with port 1883 (standard)

<i>$ nohup dragonrise -mqpub=tcp://test.mosquitto.org:1883/base_topic /dev/dragonrise_3 2>/dev/null 1>/dev/null &</i>

Execution in background (as a daemon).
System return PID.
