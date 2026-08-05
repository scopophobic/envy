package middleware

import (
	"net/http"

	"github.com/envo/backend/internal/models"
	"github.com/envo/backend/internal/services"
	"github.com/envo/backend/internal/utils"
	"github.com/gin-gonic/gin"
)

const (
	agentContextKey           = "agent"
	agentCredentialContextKey = "agent_credential"
)

// AgentAuthMiddleware accepts only Envo agent credentials. It is deliberately
// separate from human JWT authentication so an agent token cannot reach user
// or organization management routes.
func AgentAuthMiddleware(agentService *services.AgentService) gin.HandlerFunc {
	return func(c *gin.Context) {
		raw, err := utils.ExtractToken(c.GetHeader("Authorization"))
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Agent authorization required"})
			c.Abort()
			return
		}
		agent, credential, err := agentService.AuthenticateToken(c.Request.Context(), raw)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired agent token"})
			c.Abort()
			return
		}
		c.Set(agentContextKey, agent)
		c.Set(agentCredentialContextKey, credential)
		c.Next()
	}
}

func GetCurrentAgent(c *gin.Context) (*models.AgentIdentity, *models.AgentCredential, bool) {
	agentValue, agentOK := c.Get(agentContextKey)
	credentialValue, credentialOK := c.Get(agentCredentialContextKey)
	agent, agentTypeOK := agentValue.(*models.AgentIdentity)
	credential, credentialTypeOK := credentialValue.(*models.AgentCredential)
	return agent, credential, agentOK && credentialOK && agentTypeOK && credentialTypeOK
}

func AgentRateLimitKey(c *gin.Context) string {
	if agent, credential, ok := GetCurrentAgent(c); ok {
		return agent.ID.String() + ":" + credential.ID.String()
	}
	return ClientIPRateLimitKey(c)
}
