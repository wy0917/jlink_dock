# jlink_dock

When a USB device is plugged in, `jlink_dock` automatically starts a Docker
container and initiates a service inside the container. This service enables
the user to flash the connected JLink board by uploading an ELF file and a
script to a REST API. `jlink_dock` then flashes the board with the ELF file and
runs the uploaded script after flashing.
