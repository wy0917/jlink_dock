#!/bin/bash -e
LOG_FILE=/udev.log
echo "$0 $@" >> $LOG_FILE
ACTION=$1; shift;
DEV_NAME=$1; shift;
SERIAL=$1; shift;
BOARD_TYPE=$(sqlite /path/to/board.db "select type from stm32 where serial=${SERIAL}") || exit 3
CID_FILE="/var/run/$(basename ${DEV_NAME}).dockerid"
TTY_PREFIX='ttyACM'
TTY_DEV=$(basename "${DEV_NAME}")
TTY_NUM=${TTY_DEV##$TTY_PREFIX}
re='^[0-9]+$'
[[ $TTY_NUM =~ $re ]] || exit 4
echo "BOARD_TYPE=${BOARD_TYPE}" >> $LOG_FILE
echo "docker run -d --restart=always -v /dev:/dev -e SERIAL=${SERIAL} -e TYPE=${BOARD_TYPE} IMAGE" >> $LOG_FILE

case "$1" in
    add)
        $0 remove "${DEV_NAME}" "${SERIAL}"
        docker run -d --cidfile "${CID_FILE}" --privileged --restart=always \
                -v /:/data \
                -v /dev:/dev \
                -v ./jlink_dock:/jlink_dock \
                -v /JLink_Linux_V766b_x86_64:/JLink_Linux_V766b_x86_64 \
                -v /gcc-arm-none-eabi:/gcc-arm-none-eabi-gdb \
                -e SERIAL="${SERIAL}" -e TYPE="${BOARD_TYPE}" -e DEV_NAME="${DEV_NAME}" -e ACTION=${ACTION} \
                -p $((2430+${TTY_NUM})):2331 -p $((8030+${TTY_NUM})):80 \
                ubuntu:jammy /bin/bash -c 'echo "${SERIAL} ${TYPE} ${DEV_NAME} ${ACTION}" >> /data/$LOG_FILE; /jlink_dock -tty ${DEV_NAME} -serial ${SERIAL} -type ${TYPE}'
        ;;
    remove)
        if [ -f "${CID_FILE}" ]; then
            RUNNING_CID=$(cat "${CID_FILE}")
            docker kill "${RUNNING_CID}" || :
            docker rm "${RUNNING_CID}" || :
            rm -vf "${CID_FILE}"
            echo "removed ${CID_FILE}" >> $LOG_FILE
        fi
        ;;
    change)
        $0 remove "${DEV_NAME}" "${SERIAL}"
        $0 add "${DEV_NAME}" "${SERIAL}"
        ;;
esac
