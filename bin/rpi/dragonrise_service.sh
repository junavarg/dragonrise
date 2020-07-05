#!/bin/bash
# script de control dl programa dragonrise como servicio systemd 
# Autor junavarg version 1 04/07/2020 
DATE=`date '+%Y-%m-%d %H:%M:%S'`
echo "dragonrise service service started at ${DATE}" | systemd-cat -p info

# se ejecuta dragonrise contra los 4 puertos USB externos de la Raspberry Pi
dragonrise -mqpub=wss://mqttws.vigilanet.com:443/honduras6 /dev/dragonrise_2 /dev/dragonrise_3 /dev/dragonrise_4 /dev/dragonrise_5
