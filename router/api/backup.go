package api

import (
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	cfenv "github.com/cloudfoundry-community/go-cfenv"
	echo "github.com/labstack/echo/v4"
	"gitlab.swisscloud.io/appc-cf-core/appcloud-backman-app/util"
)

func (h *Handler) ListBackups(c echo.Context) error {
	var services []cfenv.Service

	// get list of services to display backups for
	serviceType := c.QueryParam("service_type")
	serviceName := c.QueryParam("service_name")
	if len(serviceName) > 0 {
		// list backups only for a specific service binding
		service, err := h.App.Services.WithName(serviceName)
		if err != nil {
			return echo.NewHTTPError(http.StatusNotFound, err.Error())
		}
		services = append(services, *service)
	} else if len(serviceType) > 0 {
		// list backups only for a specific service type
		var err error
		services, err = h.App.Services.WithLabel(serviceType)
		if err != nil {
			return echo.NewHTTPError(http.StatusNotFound, err.Error())
		}
	} else {
		// list backups for all services
		for label, s := range h.App.Services {
			if util.IsValidServiceType(label) {
				services = append(services, s...)
			}
		}
	}

	type File struct {
		Key          string
		Filepath     string
		Filename     string
		Size         int64
		LastModified time.Time
	}
	type Backup struct {
		ServiceType string
		ServiceName string
		Files       []File
	}
	backups := make([]Backup, 0)
	for _, service := range services {
		folderPath := fmt.Sprintf("%s/%s/", service.Label, service.Name)
		objects, err := h.S3.ListBackups(folderPath)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}

		// collect backup files
		files := make([]File, 0)
		for _, obj := range objects {
			// exclude "directories"
			if obj.Key != folderPath && !strings.Contains(folderPath, filepath.Base(obj.Key)) {
				files = append(files, File{
					Key:          obj.Key,
					Filepath:     filepath.Dir(obj.Key),
					Filename:     filepath.Base(obj.Key),
					Size:         obj.Size,
					LastModified: obj.LastModified,
				})
			}
		}

		backups = append(backups, Backup{
			ServiceType: service.Label,
			ServiceName: service.Name,
			Files:       files,
		})
	}

	return c.JSON(http.StatusOK, backups)
}