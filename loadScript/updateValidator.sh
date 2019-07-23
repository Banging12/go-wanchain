#!/bin/bash
# set -x
echo ''
echo ''
echo ''
echo ''
echo '=========================================='
echo '+     Welcome to Validator Update        +'
echo ''
echo 'If you have deployed your validator with deployValidator.sh, you can update with this script'
echo 'Please Enter your validator Name:'
read YOUR_NODE_NAME
echo 'Please Enter your validator Address'
read addrNew
echo 'Please Enter your password of Validator account:'
read -s PASSWD
echo ''
echo ''
echo ''
echo ''
echo ''

DOCKERIMG=wanchain/client-go:2.1.0-beta
NETWORK=--testnet
NETWORKPATH=testnet

DOCKERID=$(docker ps|grep gwan|awk '{print $1}')

sudo docker pull ${DOCKERIMG}
if [ $? -ne 0 ]; then
    echo "Docker Pull failed. Please verify your Access of docker command."
    echo "You can add yourself into docker group by this command, and re-login:"
    echo "sudo usermod -aG docker ${USER}"
    exit 1
else
    echo "docker pull succeed"
fi

sudo docker stop ${DOCKERID}

sudo docker rm ${DOCKERID} >/dev/null 2>&1

sudo docker rm gwan >/dev/null 2>&1

echo ${PASSWD} | sudo tee -a /home/${USER}/.wanchain/pw.txt > /dev/null
if [ $? -ne 0 ]; then
    echo "Write pw.txt failed"
    exit 1
fi

sudo docker run -d --name gwan -p 17717:17717 -p 17717:17717/udp -v /home/${USER}/.wanchain:/root/.wanchain ${DOCKERIMG} /bin/gwan ${NETWORK} --etherbase ${addrNew} --unlock ${addrNew} --password /root/.wanchain/pw.txt --mine --minerthreads=1 --wanstats ${YOUR_NODE_NAME}:admin@54.193.4.239:80

if [ $? -ne 0 ]; then
    echo "docker run failed"
    exit 1
fi

echo 'Please wait a few seconds...'

sleep 5

sudo rm /home/${USER}/.wanchain/pw.txt

if [ $? -ne 0 ]; then
    echo "rm pw.txt failed"
    exit 1
fi

echo ''
echo ''
echo ''
echo ''

if [ $(ps -ef | grep -c "gwan") -gt 1 ]; 
then 
    echo "Validator Start Success";
else
    echo "Validator Start Failed";
    echo "Please use command 'sudo docker logs gwan' to check reason." 
fi

