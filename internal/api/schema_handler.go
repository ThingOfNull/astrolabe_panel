package api

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"astrolabe/internal/store"
)

// registerSchemaRoutes exposes widget metadata that was historically duplicated
// between the Go store package and the Vue registry. The frontend pulls it on
// startup; the Go declarations remain the single source of truth.
func registerSchemaRoutes(r *gin.Engine) {
	r.GET("/api/schema/widgets", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"types":           knownWidgetTypes(),
			"accepted_shapes": store.AcceptedShapesByType,
			"icon_types":      []string{store.IconTypeInternal, store.IconTypeRemote, store.IconTypeIconify},
		})
	})
}

// knownWidgetTypes mirrors store.WidgetType* constants. Keeping the list here
// rather than in store avoids exposing every internal identifier as public API.
func knownWidgetTypes() []string {
	return []string{
		store.WidgetTypeLink,
		store.WidgetTypeSearch,
		store.WidgetTypeGauge,
		store.WidgetTypeBigNumber,
		store.WidgetTypeLine,
		store.WidgetTypeBar,
		store.WidgetTypeGrid,
		store.WidgetTypeText,
		store.WidgetTypeDivider,
		store.WidgetTypeWeather,
		store.WidgetTypeClock,
		store.WidgetTypeLiquid,
		store.WidgetTypeRadial3D,
		store.WidgetTypeHeatmap,
		store.WidgetTypeSparkline,
		store.WidgetTypeBullet,
		store.WidgetTypeProgressRing,
		store.WidgetTypeTimeline,
	}
}
