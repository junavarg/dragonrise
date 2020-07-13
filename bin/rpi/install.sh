#!/bin/bash
# Instala software soporte tarjeta joystick USB de Dragonrise 
# Autor junavarg version 1 11/05/2020 

mkdir ~/tmp-dragonrise
cd ~/tmp-dragonrise
rm *

wget https://raw.githubusercontent.com/junavarg/dragonrise/master/bin/rpi/31-dragonriseRPi.rules
sudo cp 31-dragonriseRPi.rules /etc/udev/rules.d/
sudo udevadm control --reload-rules

wget https://raw.githubusercontent.com/junavarg/dragonrise/master/bin/rpi/dragonrise
sudo cp dragonrise /usr/local/bin
sudo chmod 777 /usr/local/bin/dragonrise

# se crea el subdirectorio  para los ficheros de estado con propietario root y con permisos para todos
sudo mkdir /var/lib/dragonrise
sudo chmod 777 /var/lib/dragonrise

wget https://raw.githubusercontent.com/junavarg/dragonrise/master/bin/rpi/dragonrise_service.sh
sudo cp dragonrise_service.sh /usr/local/sbin/
sudo chmod +x /usr/local/sbin/ dragonrise_service.sh

wget https://raw.githubusercontent.com/junavarg/dragonrise/master/bin/rpi/dragonrise.service
sudo cp dragonrise.service /lib/systemd/system/
sudo ln -s /lib/systemd/system/dragonrise.service /etc/systemd/system/


wget https://raw.githubusercontent.com/junavarg/dragonrise/master/bin/rpi/uninstall-dragonrise.sh
sudo cp uninstall-dragonrise.sh /usr/local/bin
sudo chmod 777 /usr/local/bin/uninstall-dragonrise.sh

cd ~
rm -r ~/tmp-dragonrise
