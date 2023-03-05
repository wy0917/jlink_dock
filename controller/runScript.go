package controller

import (
	"bytes"
	"fmt"
	expect "github.com/google/goexpect"
	"github.com/sirupsen/logrus"
	"net/http"
	"os/exec"
	"regexp"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/wy0917/jlink_dock/model"
)

var (
	gdbServerRE = regexp.MustCompile("Waiting for GDB connection...")
	wg          = sync.WaitGroup{}
)

// RunScript godoc
//
//			@Summary		Flash elf to the board, and unzip the script.zip file and run script according to autorun.sh
//			@Description	Accept two files from formData, and cache them to the file system. Then flashing the elf onto the board, then run the autorun.sh
//			@Tags			script
//			@Accept			multipart/form-data
//			@Produce		text/plain
//		    @Param          elf  formData  file  true  "elf file for flashing onto the board"
//	        @Param          script  formData  file  true  "zipped script file with an autorun.sh in the root directory"
//			@Success		200
//			@Failure		500	{object}	model.APIError
//			@Router			/script [post]
func (c *Controller) RunScript(ctx *gin.Context) {
	log_ := ctx.MustGet("logger").(*logrus.Logger)
	config := ctx.MustGet("config").(*model.Config)
	files := model.UploadedFiles{}

	if ctx.ShouldBind(&files) != nil {
		ctx.String(http.StatusBadRequest, "Try again")
		return
	}

	// Check if elf file is a valid arm elf file, and cache it to local storage
	elfFile, err := ctx.FormFile("elf")
	if err != nil {
		ctx.String(http.StatusBadRequest, err.Error())
		return
	}

	err = checkElf(elfFile)
	if err != nil {
		ctx.String(http.StatusBadRequest, err.Error())
		return
	}

	err = ctx.SaveUploadedFile(elfFile, fmt.Sprintf("./bin.elf"))
	if err != nil {
		log_.Println("Unable to save uploaded file")
		ctx.String(http.StatusBadRequest, fmt.Sprintf("error: %s", err.Error()))
		return
	}

	done := make(chan int)
	defer close(done)
	serverReady := make(chan int)
	defer close(serverReady)
	wg.Add(1)
	go func() {
		err := gdbServerSpawn(ctx, log_, config.GDB.ServerPath, config.Type, config.Serial, serverReady, done)
		if err != nil {
			log_.Println(fmt.Sprintf("error: %s", err.Error()))
			ctx.String(http.StatusInternalServerError, fmt.Sprintf("error: %s", err.Error()))
		}
	}()
	_ = <-serverReady
	defer func() { done <- 1 }()

	flashDone := make(chan int)
	defer close(flashDone)
	flashReady := make(chan int)
	defer close(flashReady)
	wg.Add(1)
	go func() {
		err := flashBoardAndRun(ctx, log_, "./bin.elf", flashReady, flashDone)
		if err != nil {
			log_.Println(fmt.Sprintf("error: %s", err.Error()))
			ctx.String(http.StatusInternalServerError, fmt.Sprintf("error: %s", err.Error()))
		}
	}()
	_ = <-flashReady
	defer func() { flashDone <- 1 }()

	// Check zip file, cache it, and unzip to /script
	scriptZip, err := ctx.FormFile("script")
	if err != nil {
		ctx.String(http.StatusBadRequest, err.Error())
		return
	}

	err = extractZipFile(scriptZip, "./script/")
	if err != nil {
		ctx.String(http.StatusBadRequest, err.Error())
	}
	wg.Add(1)
	go func() {
		err := runScript(ctx, log_, "./script/")
		if err != nil {
			log_.Println(fmt.Sprintf("error: %s", err.Error()))
			ctx.String(http.StatusInternalServerError, fmt.Sprintf("error: %s", err.Error()))
		}
	}()

	//err = ctx.SaveUploadedFile(scriptZip, scriptZip.Filename)
	//if err != nil {
	//	log_.Println("Unable to save uploaded file to", scriptZip.Filename)
	//	ctx.String(http.StatusBadRequest, err.Error())
	//	return
	//}
	wg.Wait()

	ctx.String(http.StatusOK, "OK")
}

type LogrusEntryWriter struct {
	Entry *logrus.Entry
}

func (lew *LogrusEntryWriter) Write(p []byte) (n int, err error) {
	lew.Entry.Println(string(bytes.TrimSpace(p)))
	return len(p), nil
}

func flashBoardAndRun(ctx *gin.Context, log_ *logrus.Logger, elfPath string, ready, done chan int) error {
	defer wg.Done()
	config := ctx.MustGet("config").(*model.Config)
	entry := log_.WithField("component", "flash")
	entry.Println("Spin up GDB client for flashing")

	gdb, _, err := expect.Spawn(config.GDB.EXEPath, -1)
	if err != nil {
		entry.Error(err)
		ctx.String(http.StatusInternalServerError, "%v", err)
		return err
	}
	defer gdb.Close()

	output, _, _ := gdb.Expect(regexp.MustCompile("\\(gdb\\) "), 1*time.Second)
	entry.Println(output)
	gdb.Send("target extended-remote localhost:2331 \n")
	output, _, _ = gdb.Expect(regexp.MustCompile("\\(gdb\\) "), 1*time.Second)
	entry.Println(output)
	gdb.Send(fmt.Sprintf("file %s \n", elfPath))
	output, _, err = gdb.Expect(regexp.MustCompile("Are you sure you want to change the file? \\(y or n\\)"), 1*time.Second)
	if err != nil {
		if output == "" {
			fmt.Println("Timed out waiting")
		} else {
			entry.Error(err)
			ctx.String(http.StatusInternalServerError, "%v", err)
			return err
		}
	} else {
		entry.Println(output)
		gdb.Send("y\n")
	}
	output, _, err = gdb.Expect(regexp.MustCompile("\\(gdb\\) "), 1*time.Second)
	entry.Println(output)
	gdb.Send("load\n")
	output, _, err = gdb.Expect(regexp.MustCompile("\\(gdb\\) "), 1*time.Second)
	entry.Println(output)
	gdb.Send("r\n")
	output, _, err = gdb.Expect(regexp.MustCompile("Start it from the beginning? \\(y or n\\)"), 1*time.Second)
	if err != nil {
		if output == "" {
			fmt.Println("Timed out waiting")
		} else {
			entry.Error(err)
			ctx.String(http.StatusInternalServerError, "%v", err)
			return err
		}
	} else {
		entry.Println(output)
		gdb.Send("y\n")
	}

	ready <- 1

	for {
		select {
		case <-done:
			// If we receive a message on the control channel, stop the goroutine.
			entry.Println("Stopping command.")
			return nil
		default:
			// If we don't receive a message on the control channel, read output from the command.
			text, _, err := gdb.Expect(regexp.MustCompile(""), 100*time.Millisecond)
			if err != nil {
				return err
			}
			entry.Println(text)
		}
	}
}

// , timeout time.Duration, opts ...expect.Option
func gdbServerSpawn(ctx *gin.Context, log_ *logrus.Logger, path, device, serial string, ready, done chan int) error {
	defer wg.Done()
	entry := log_.WithField("component", "gdbServer")
	entry.Println("Starting GDB Server")

	gdbServerCmdLine := fmt.Sprintf("%s/JLinkGDBServerCLExe -device %s -select usb=%s", path, device, serial)

	entry.Println("Spawn gdb server up: %s", gdbServerCmdLine)
	spawn, _, err := expect.Spawn(gdbServerCmdLine, -1)
	if err != nil {
		ctx.String(http.StatusInternalServerError, "%v", err)
		return err
	}
	defer spawn.Close()
	ready <- 1

	for {
		select {
		case <-done:
			// If we receive a message on the control channel, stop the goroutine.
			entry.Println("Stopping command.")
			return nil
		default:
			// If we don't receive a message on the control channel, read output from the command.
			text, _, err := spawn.Expect(regexp.MustCompile(""), 100*time.Millisecond)
			if err == nil {
				entry.Println(text)
			} else {
				return err
			}
		}
	}
}

func runScript(ctx *gin.Context, log_ *logrus.Logger, path string) error {
	defer wg.Done()
	entry := log_.WithField("component", "runScript")
	entry.Println("Starting GDB Server")

	lew := &LogrusEntryWriter{Entry: entry}
	cmd := exec.Command("%s/autorun.sh", path)
	cmd.Stdout = lew
	cmd.Stderr = lew
	if err := cmd.Run(); err != nil {
		entry.Println(fmt.Sprintf("error: %s", err.Error()))
		ctx.String(http.StatusInternalServerError, fmt.Sprintf("error: %s", err.Error()))
		return err
	}
	return nil
}
