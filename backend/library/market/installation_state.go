package market

// IsServiceInstallationActive reports whether a service currently has a task
// that must skip physical package removal during uninstall.
func IsServiceInstallationActive(serviceID int64) bool {
	task, exists := GetInstallationManager().GetTaskStatus(serviceID)
	return exists && (task.Status == StatusPending || task.Status == StatusInstalling)
}
