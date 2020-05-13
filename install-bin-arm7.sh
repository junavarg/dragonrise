#!/bin/bash
# Instala software soporte tarjeta joystick USB de Dragonrise 
# desde fuentes
# Requiere compilador golang instalado
# Autor junavarg version 1 11/05/2020 


wget https://raw.githubusercontent.com/junavarg/dr/master/31-dragonriseRPi.rules
sudo cp 31-dragonriseRPi.rules /etc/udev/rules.d/
sudo udevadm control --reload-rules

wget https://raw.githubusercontent.com/junavarg/dr/master/dragonrise.go
go get github.com/eclipse/paho.mqtt.golang 
go build -ldflags "-s -w" dragonrise.go 
 
sudo cp dragonrise /usr/local/bin
sudo chmod 777 /usr/local/bin/dragonrise

# se crea el subdirectorio  para los ficheros de estado con propietario root y con permisos para todos
sudo mkdir /var/lib/dragonrise
sudo chmod 777 /var/lib/dragonrise