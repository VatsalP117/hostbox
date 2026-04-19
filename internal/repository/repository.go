package repository

import "database/sql"

// Repositories holds all repository instances.
type Repositories struct {
	User          *UserRepository
	Session       *SessionRepository
	Project       *ProjectRepository
	Deployment    *DeploymentRepository
	SystemMetrics *SystemMetricRepository
	Domain        *DomainRepository
	EnvVar        *EnvVarRepository
	Notification  *NotificationRepository
	Activity      *ActivityRepository
	Settings      *SettingsRepository
}

// New creates all repository instances from a database connection.
func New(db *sql.DB) *Repositories {
	return &Repositories{
		User:          NewUserRepository(db),
		Session:       NewSessionRepository(db),
		Project:       NewProjectRepository(db),
		Deployment:    NewDeploymentRepository(db),
		SystemMetrics: NewSystemMetricRepository(db),
		Domain:        NewDomainRepository(db),
		EnvVar:        NewEnvVarRepository(db),
		Notification:  NewNotificationRepository(db),
		Activity:      NewActivityRepository(db),
		Settings:      NewSettingsRepository(db),
	}
}
