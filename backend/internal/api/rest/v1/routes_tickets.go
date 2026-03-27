package v1

import "github.com/gin-gonic/gin"

func registerTicketRoutes(rg *gin.RouterGroup, svc *Services) {
	ticketHandler := NewTicketHandler(svc.Ticket)
	meshHandler := NewMeshHandler(svc.Mesh, svc.Ticket)
	tickets := rg.Group("/tickets")
	{
		tickets.GET("", ticketHandler.ListTickets)
		tickets.POST("", ticketHandler.CreateTicket)
		tickets.GET("/active", ticketHandler.GetActiveTickets)
		tickets.GET("/board", ticketHandler.GetBoard)
		tickets.POST("/batch-pods", meshHandler.BatchGetTicketPods)
		tickets.GET("/:ticket_slug", ticketHandler.GetTicket)
		tickets.PUT("/:ticket_slug", ticketHandler.UpdateTicket)
		tickets.DELETE("/:ticket_slug", ticketHandler.DeleteTicket)
		tickets.PATCH("/:ticket_slug/status", ticketHandler.UpdateTicketStatus)
		tickets.POST("/:ticket_slug/assignees", ticketHandler.AddAssignee)
		tickets.DELETE("/:ticket_slug/assignees/:user_id", ticketHandler.RemoveAssignee)
		tickets.POST("/:ticket_slug/labels", ticketHandler.AddLabel)
		tickets.DELETE("/:ticket_slug/labels/:label_id", ticketHandler.RemoveLabel)
		tickets.GET("/:ticket_slug/merge-requests", ticketHandler.ListMergeRequests)
		tickets.GET("/:ticket_slug/sub-tickets", ticketHandler.GetSubTickets)
		tickets.GET("/:ticket_slug/relations", ticketHandler.ListRelations)
		tickets.POST("/:ticket_slug/relations", ticketHandler.CreateRelation)
		tickets.DELETE("/:ticket_slug/relations/:relation_id", ticketHandler.DeleteRelation)
		tickets.GET("/:ticket_slug/commits", ticketHandler.ListCommits)
		tickets.POST("/:ticket_slug/commits", ticketHandler.LinkCommit)
		tickets.DELETE("/:ticket_slug/commits/:commit_id", ticketHandler.UnlinkCommit)
		tickets.GET("/:ticket_slug/comments", ticketHandler.ListComments)
		tickets.POST("/:ticket_slug/comments", ticketHandler.CreateComment)
		tickets.PUT("/:ticket_slug/comments/:id", ticketHandler.UpdateComment)
		tickets.DELETE("/:ticket_slug/comments/:id", ticketHandler.DeleteComment)
		tickets.GET("/:ticket_slug/pods", meshHandler.GetTicketPods)
		tickets.POST("/:ticket_slug/pods", meshHandler.CreatePodForTicket)
	}

	labels := rg.Group("/labels")
	{
		labels.GET("", ticketHandler.ListLabels)
		labels.POST("", ticketHandler.CreateLabel)
		labels.PUT("/:id", ticketHandler.UpdateLabel)
		labels.DELETE("/:id", ticketHandler.DeleteLabel)
	}

	meshGroup := rg.Group("/mesh")
	{
		meshGroup.GET("/topology", meshHandler.GetTopology)
	}
}
