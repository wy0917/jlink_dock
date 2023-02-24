# jlink\_dock

This is a framework for serving connected JLink board as service. (Currently
only designed for Linux platform)

When a USB device is plugged in, `jlink_dock` automatically starts a Docker
container and initiates a service inside the container. This service enables
the user to flash the connected JLink board by uploading an ELF file and a
script to a REST API. `jlink_dock` then flashes the board with the ELF file and
runs the uploaded script after flashing.

## Usage:

To use this tool there are some preparation needed:

1. Run `udevadm monitor -p` on host, and connect jlink board to it, and note
down the `ID_SERIAL_SHORT` and the type of the board.

2. Create a sqlite database using the following command, then update
`/path/to/jlink\_dock.sh:6` for correct path to database:

   ```bash
   sqlite3 /path/to/board.db  "create table stm32(serial NCHAR(12) primary key, type NCHAR(11));"
   ```

3. Insert the SERIAL and Type mapping to the database created in prior step.

4. Put `51-jlink_dock.rules` into `/etc/udev/rules.d`, and update
`/path/to/jlink_dock.sh` to the correct value.

5. Make the udev rules to take effort, for example using the following command.
```bash
   udevadm control --reload   # Reload udev rules
   udevadm trigger            # Re-trigger the udev events
   ```

Now for each of jlink board connected to the host, there will be a container
started to it.

