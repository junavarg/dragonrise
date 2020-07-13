#!/bin/bash
# Instala software soporte tarjeta joystick USB de Dragonrise 
# Autor junavarg version 1 11/05/2020 

#Se para el servicio dragonrise.service y se retirra de arranque automatico
sudo systemctl stop dragonrise.service
sudo systemctl disable dragonrise.service

#Se elimina el fichero de definición del servicio y previamente el enlace 
sudo rm /etc/systemd/system/dragonrise.service
sudo rm  /lib/systemd/system/dragonrise.service

#Se elimina el script del servicio
sudo rm  /usr/local/sbin/dragonrise_service.sh

# se elimina el subdirectorio de los ficheros de estado
sudo rm -R /var/lib/dragonrise

# se elimina el programa dragonrise
sudo rm  /usr/local/bin/dragonrise

# se eliminan la reglas UDEV
sudo rm  /etc/udev/rules.d/31-dragonriseRPi.rules
sudo udevadm control --reload-rules


# se elimina ficehro de desintalaciónV
sudo rm  /usr/local/bin/uninstall-dragonrise.sh





