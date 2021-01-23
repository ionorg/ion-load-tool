#!/bin/bash
# please run this script with sudo!
ulimit -c unlimited
ulimit -SHn 1000000
sysctl -w net.ipv4.tcp_keepalive_time=60
sysctl -w net.ipv4.tcp_timestamps=0
sysctl -w net.ipv4.tcp_tw_reuse=1
#sysctl -w net.ipv4.tcp_tw_recycle=0
sysctl -w net.core.somaxconn=65535
sysctl -w net.ipv4.tcp_max_syn_backlog=65535
sysctl -w net.ipv4.tcp_syncookies=1

# please run this script with sudo!
./ion-load-tool -file ./djrm480p.webm -clients 10 -role pubsub -addr "127.0.0.1:8000" -session 'conference' -log debug -cycle 1000
