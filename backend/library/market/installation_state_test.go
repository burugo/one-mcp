package market

import "testing"

func TestIsServiceInstallationActive(t *testing.T) {
	manager := GetInstallationManager()
	serviceID := int64(991101)
	manager.CleanupTask(serviceID)
	t.Cleanup(func() { manager.CleanupTask(serviceID) })

	if IsServiceInstallationActive(serviceID) {
		t.Fatal("service without an installation task must not be active")
	}

	manager.tasksMutex.Lock()
	manager.tasks[serviceID] = &InstallationTask{ServiceID: serviceID, Status: StatusPending}
	manager.tasksMutex.Unlock()
	if !IsServiceInstallationActive(serviceID) {
		t.Fatal("pending installation task must be active")
	}

	manager.tasksMutex.Lock()
	manager.tasks[serviceID].Status = StatusInstalling
	manager.tasksMutex.Unlock()
	if !IsServiceInstallationActive(serviceID) {
		t.Fatal("installing task must be active")
	}

	manager.tasksMutex.Lock()
	manager.tasks[serviceID].Status = StatusCompleted
	manager.tasksMutex.Unlock()
	if IsServiceInstallationActive(serviceID) {
		t.Fatal("completed installation task must not be active")
	}
}
