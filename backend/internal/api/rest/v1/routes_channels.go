package v1

import "github.com/gin-gonic/gin"

func registerChannelRoutes(rg *gin.RouterGroup, svc *Services) {
	channelHandler := NewChannelHandler(svc.Channel, svc.Ticket)
	channels := rg.Group("/channels")
	{
		channels.GET("", channelHandler.ListChannels)
		channels.POST("", channelHandler.CreateChannel)
		channels.GET("/unread", channelHandler.GetUnreadCounts)
		channels.GET("/:id", channelHandler.GetChannel)
		channels.PUT("/:id", channelHandler.UpdateChannel)
		channels.POST("/:id/archive", channelHandler.ArchiveChannel)
		channels.POST("/:id/unarchive", channelHandler.UnarchiveChannel)
		channels.GET("/:id/messages", channelHandler.ListMessages)
		channels.POST("/:id/messages", channelHandler.SendMessage)
		channels.PUT("/:id/messages/:msg_id", channelHandler.EditMessage)
		channels.DELETE("/:id/messages/:msg_id", channelHandler.DeleteMessage)
		channels.POST("/:id/read", channelHandler.MarkRead)
		channels.POST("/:id/mute", channelHandler.MuteChannel)
		channels.GET("/:id/members", channelHandler.ListMembers)
		channels.GET("/:id/document", channelHandler.GetDocument)
		channels.PUT("/:id/document", channelHandler.UpdateDocument)
		channels.GET("/:id/pods", channelHandler.ListChannelPods)
		channels.POST("/:id/pods", channelHandler.JoinPod)
		channels.DELETE("/:id/pods/:pod_key", channelHandler.LeavePod)
	}
}

func registerMessageRoutes(rg *gin.RouterGroup, svc *Services) {
	if svc.Message == nil {
		return
	}
	messageHandler := NewMessageHandler(svc.Message)
	messages := rg.Group("/messages")
	{
		messages.POST("", messageHandler.SendMessage)
		messages.GET("", messageHandler.GetMessages)
		messages.GET("/unread-count", messageHandler.GetUnreadCount)
		messages.GET("/sent", messageHandler.GetSentMessages)
		messages.POST("/mark-read", messageHandler.MarkRead)
		messages.POST("/mark-all-read", messageHandler.MarkAllRead)
		messages.GET("/conversation/:correlation_id", messageHandler.GetConversation)
		messages.GET("/dlq", messageHandler.GetDeadLetters)
		messages.POST("/dlq/:id/replay", messageHandler.ReplayDeadLetter)
		messages.GET("/:id", messageHandler.GetMessage)
	}
}
