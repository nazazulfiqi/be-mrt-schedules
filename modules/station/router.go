package station

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/nazazulfiqi/be-mrt-schedules/common/response"
)

func Initiate(router *gin.RouterGroup) {
	stationService := NewService()

	station := router.Group("/stations")
	station.GET("", func(c *gin.Context) {
		GetAllStation(c, stationService)
	})
	station.POST("/check", func(c *gin.Context) {
		CheckSchedulesByStation(c, stationService)
	})
}

func GetAllStation(c *gin.Context, service Service) {
	datas, err := service.GetAllStation()
	if err != nil {
		c.JSON(http.StatusBadRequest, response.APIResponse{Success: false, Message: err.Error(), Data: nil})
		return
	}
	c.JSON(http.StatusOK, response.APIResponse{Success: true, Message: "Successfully get all station", Data: datas})
}

func CheckSchedulesByStation(c *gin.Context, service Service) {
	var req RouteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.APIResponse{Success: false, Message: err.Error(), Data: nil})
		return
	}

	datas, err := service.CheckSchedulesByStation(req)
	if err != nil {
		c.JSON(http.StatusBadRequest, response.APIResponse{Success: false, Message: err.Error(), Data: nil})
		return
	}

	c.JSON(http.StatusOK, response.APIResponse{Success: true, Message: "Successfully get schedules by station", Data: datas})
}
