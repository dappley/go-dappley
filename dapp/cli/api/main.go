package main

import (
	"net/http"
	"os"
	"os/exec"

	"github.com/gin-gonic/gin"
)

var didSCAddress = ""

var sender = ""

type AddDIDParam struct {
	Doc string `form:"doc" json:"doc" binding:"required"`
}
type DeleteDIDParam struct {
	Sign string `form:"sign" json:"sign" binding:"required"`
	Msg  string `form:"message" json:"message" binding:"required"`
	Did  string `form:"did" json:"did" binding:"required"`
}
type ChangeDIDParam struct {
	Doc  string `form:"doc" json:"doc" binding:"required"`
	Sign string `form:"sign" json:"sign" binding:"required"`
	Msg  string `form:"message" json:"message" binding:"required"`
	Did  string `form:"did" json:"did" binding:"required"`
}
type CreateSchemaParam struct {
	Schema string `form:"schema" json:"schema" binding:"required"`
}
type AddVCParam struct {
	Msg  string `form:"message" json:"message" binding:"required"`
	Cred string `form:"credential" json:"credential" binding:"required"`
}
type UpdateVCParam struct {
	Msg  string `form:"message" json:"message" binding:"required"`
	Cred string `form:"credential" json:"credential" binding:"required"`
}
type DeleteVCParam struct {
	Sign   string `form:"sign" json:"sign" binding:"required"`
	Msg    string `form:"message" json:"message" binding:"required"`
	CredID string `form:"credID" json:"credID" binding:"required"`
}

func main() {
	args := os.Args[1:]
	if len(args) < 2 {
		println("Please provide contract address and sending account. Ex. \"./api contractaddress sendaccountaddress\"")
		return
	}
	didSCAddress = args[0]
	sender = args[1]

	r := gin.Default()

	r.POST("/addDID", addDID)
	r.POST("/deleteDID", deleteDID)
	r.POST("/changeDID", changeDID)
	r.POST("/createSchema", createSchema)
	r.POST("/addVC", addVC)
	r.POST("/updateVC", updateVC)
	r.POST("/deleteVC", deleteVC)
	r.Run(":8081")
}

func sendScript(command string) ([]byte, error) {
	cmd := exec.Command("./cli", "send", "-from", sender, "-to", didSCAddress, "-amount", "1", "-tip", "1", "-gasLimit", "100000", "-gasPrice", "1", "-data", command)
	return cmd.Output()
}

func addDID(c *gin.Context) {
	var input AddDIDParam
	if err := c.BindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	command := "{\"function\": \"addDID\", \"args\": [\"" + input.Doc + "\"]}"
	output, err := sendScript(command)
	if err != nil {
		c.String(http.StatusOK, err.Error())
		println(err.Error())
		return
	}
	println("error")
	c.String(http.StatusOK, string(output))
}

func deleteDID(c *gin.Context) {
	var input DeleteDIDParam
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	command := "{\"function\": \"deleteDID\", \"args\": [\"" + input.Did + "\",\"" + input.Msg + "\",\"" + input.Sign + "\"]}"
	output, err := sendScript(command)
	if err != nil {
		c.String(http.StatusOK, err.Error())
		println(err.Error())
		return
	}
	println("error")
	c.String(http.StatusOK, string(output))
}

func changeDID(c *gin.Context) {
	var input ChangeDIDParam
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	command := "{\"function\": \"updateDID\", \"args\": [\"" + input.Did + "\",\"" + input.Doc + "\",\"" + input.Msg + "\",\"" + input.Sign + "\"]}"
	output, err := sendScript(command)
	if err != nil {
		c.String(http.StatusOK, err.Error())
		println(err.Error())
		return
	}
	println("error")
	c.String(http.StatusOK, string(output))

}

func createSchema(c *gin.Context) {
	var input CreateSchemaParam
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	command := "{\"function\": \"createSchema\", \"args\": [\"" + input.Schema + "\"]}"
	if input.Schema == "" {
		c.String(http.StatusOK, "need schema")
		return
	}

	output, err := sendScript(command)
	if err != nil {
		c.String(http.StatusOK, err.Error())
		println(err.Error())
		return
	}
	println("error")
	c.String(http.StatusOK, string(output))

}

func addVC(c *gin.Context) {
	var input AddVCParam
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	command := "{\"function\": \"addVC\", \"args\": [\"" + input.Cred + "\",\"" + input.Msg + "\"]}"
	output, err := sendScript(command)
	if err != nil {
		c.String(http.StatusOK, err.Error())
		println(err.Error())
		return
	}
	println("error")
	c.String(http.StatusOK, string(output))
}

func updateVC(c *gin.Context) {
	var input UpdateVCParam
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	command := "{\"function\": \"updateVC\", \"args\": [\"" + input.Cred + "\",\"" + input.Msg + "\"]}"
	output, err := sendScript(command)
	if err != nil {
		c.String(http.StatusOK, err.Error())
		println(err.Error())
		return
	}
	println("error")
	c.String(http.StatusOK, string(output))
}

func deleteVC(c *gin.Context) {
	var input DeleteVCParam
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	command := "{\"function\": \"deleteVC\", \"args\": [\"" + input.CredID + "\",\"" + input.Sign + "\",\"" + input.Msg + "\"]}"
	output, err := sendScript(command)
	if err != nil {
		c.String(http.StatusOK, err.Error())
		println(err.Error())
		return
	}
	println("error")
	c.String(http.StatusOK, string(output))
}
