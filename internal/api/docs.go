package api

// @title QCAT API
// @version 1.0
// @description Quantitative Contract Automated Trading System API
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.url http://www.swagger.io/support
// @contact.email support@swagger.io

// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html

// @host localhost:8082
// @BasePath /api/v1

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and JWT token.

// @tag.name Optimizer
// @tag.description Strategy parameter optimization operations

// @tag.name Strategy
// @tag.description Trading strategy management operations

// @tag.name Portfolio
// @tag.description Portfolio and position management operations

// @tag.name Risk
// @tag.description Risk management and limits operations

// @tag.name Hotlist
// @tag.description Hot market symbols management operations

// @tag.name Metrics
// @tag.description Performance metrics and monitoring operations

// @tag.name Audit
// @tag.description Audit logs and decision chain operations

// @tag.name WebSocket
// @tag.description Real-time data streaming operations
