#!/bin/bash
# Instala software soporte tarjeta joystick USB de Dragonrise 
# Autor junavarg version 1 11/05/2020 


wget https://raw.githubusercontent.com/junavarg/dragonrise/master/bin/rpi/31-dragonriseRPi.rules
sudo cp 31-dragonriseRPi.rules /etc/udev/rules.d/
sudo udevadm control --reload-rules

wget https://raw.githubusercontent.com/junavarg/dragonrise/master/bin/rpi/dragonrise

 
sudo cp dragonrise /usr/local/bin
sudo chmod 777 /usr/local/bin/dragonrise

# se crea el subdirectorio  para los ficheros de estado con propietario root y con permisos para todos
sudo mkdir /var/lib/dragonrise
sudo chmod 777 /var/lib/dragonrise
