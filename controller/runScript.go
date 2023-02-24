package controller

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/wy0917/jlink_dock/model"
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
	runFiles := model.RunFiles{}

	if ctx.ShouldBind(&runFiles) != nil {
		ctx.String(http.StatusBadRequest, "Try again")
		return
	}

	// Check if elf file is a valid arm elf file, and cache it to local storage
	elfFile, err := ctx.FormFile("elf")
	if err != nil {
		ctx.String(http.StatusBadRequest, err.Error())
		return
	}
	log.Println(elfFile.Filename)

	err = checkElf(elfFile)
	if err != nil {
		ctx.String(http.StatusBadRequest, err.Error())
		return
	}

	err = ctx.SaveUploadedFile(elfFile, fmt.Sprintf("./bin.elf"))
	if err != nil {
		log.Println("Unable to save uploaded file")
		ctx.String(http.StatusBadRequest, fmt.Sprintf("error: %s", err.Error()))
		return
	}

	// Check zip file, cache it, and unzip to /script
	scriptZip, err := ctx.FormFile("script")
	if err != nil {
		ctx.String(http.StatusBadRequest, err.Error())
		return
	}
	log.Println(scriptZip.Filename)

	err = extractZipFile(scriptZip, "./script/")
	if err != nil {
		ctx.String(http.StatusBadRequest, err.Error())
	}

	err = ctx.SaveUploadedFile(scriptZip, scriptZip.Filename)
	if err != nil {
		log.Println("Unable to save uploaded file to", scriptZip.Filename)
		ctx.String(http.StatusBadRequest, err.Error())
		return
	}

	//go execKPI(gdbPath, kpiElfFile, scriptZipFile), id.String()
	ctx.String(http.StatusOK, "OK")
}
