package dashboardserver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"syscall"
	"time"

	"github.com/shirou/gopsutil/process"
	"github.com/turbot/steampipe/constants"
	"github.com/turbot/steampipe/dashboard/dashboardassets"
	"github.com/turbot/steampipe/filepaths"
	"github.com/turbot/steampipe/utils"
)

type ServiceState string

const (
	ServiceStateRunning ServiceState = "running"
	ServiceStateError   ServiceState = "running"
)

type DashboardServiceState struct {
	State      ServiceState
	Error      string
	Pid        int
	Port       int
	ListenType string
	Listen     []string
}

func GetDashboardServiceState() (*DashboardServiceState, error) {
	state, err := loadServiceStateFile()
	if err != nil {
		return nil, err
	}
	if state == nil {
		return nil, nil
	}
	pidExists, err := utils.PidExists(state.Pid)
	if err != nil {
		return nil, err
	}
	if !pidExists {
		return nil, os.Remove(filepaths.DashboardServiceStateFilePath())
	}
	return state, nil
}

func StopDashboardService(ctx context.Context) error {
	state, err := GetDashboardServiceState()
	if err != nil {
		return err
	}
	if state == nil {
		return nil
	}
	pidExists, err := utils.PidExists(state.Pid)
	if err != nil {
		return err
	}
	if !pidExists {
		return nil
	}
	process, err := process.NewProcessWithContext(ctx, int32(state.Pid))
	if err != nil {
		return err
	}
	err = process.SendSignalWithContext(ctx, syscall.SIGINT)
	if err != nil {
		return err
	}
	return os.Remove(filepaths.DashboardServiceStateFilePath())
}

func RunForService(ctx context.Context, serverListen ListenType, serverPort ListenPort) error {
	self, err := os.Executable()
	if err != nil {
		return err
	}

	// remove the state file (if any)
	os.Remove(filepaths.DashboardServiceStateFilePath())

	err = dashboardassets.Ensure(ctx)
	if err != nil {
		return err
	}

	utils.FailOnError(serverPort.IsValid())
	utils.FailOnError(serverListen.IsValid())

	cmd := exec.Command(
		self,
		"dashboard",
		fmt.Sprintf("--%s=%s", constants.ArgDashboardListen, string(serverListen)),
		fmt.Sprintf("--%s=%d", constants.ArgDashboardPort, serverPort),
		fmt.Sprintf("--%s=%s", constants.ArgInstallDir, filepaths.SteampipeDir),
		fmt.Sprintf("--%s=true", constants.ArgServiceMode),
	)
	cmd.Env = os.Environ()

	// set group pgid attributes on the command to ensure the process is not shutdown when its parent terminates
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid:    true,
		Foreground: false,
	}

	err = cmd.Start()
	if err != nil {
		return err
	}

	return waitForDashboardService(ctx)
}

// when started as a service, 'steampipe dashboard' always writes a
// state file in 'internal' with the outcome - even on failures
// this function polls for the file and loads up the error, if any
func waitForDashboardService(ctx context.Context) error {
	utils.LogTime("db.waitForDashboardServerStartup start")
	defer utils.LogTime("db.waitForDashboardServerStartup end")

	pingTimer := time.NewTicker(constants.ServicePingInterval)
	timeoutAt := time.After(constants.ServiceStartTimeout)
	defer pingTimer.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-pingTimer.C:
			// poll for the state file.
			// when it comes up, return it
			state, err := loadServiceStateFile()
			if err != nil {
				if os.IsNotExist(err) {
					// if the file hasn't been generated yet, that means 'dashboard' is still booting up
					continue
				}
				// there was an unexpected error
				return err
			}

			if state == nil {
				// no state file yet
				continue
			}

			// check the state file for an error
			if len(state.Error) > 0 {
				// there was an error during start up
				// remove the state file, since we don't need it anymore
				os.Remove(filepaths.DashboardServiceStateFilePath())
				// and return the error from the state file
				return errors.New(state.Error)
			}

			// we loaded the state and there was no error
			return nil
		case <-timeoutAt:
			return fmt.Errorf("dashboard server startup timed out")
		}
	}
}

func WriteServiceStateFile(state *DashboardServiceState) error {
	stateBytes, err := json.MarshalIndent(state, "", " ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepaths.DashboardServiceStateFilePath(), stateBytes, 0666)
}

func loadServiceStateFile() (*DashboardServiceState, error) {
	state := new(DashboardServiceState)
	stateBytes, err := os.ReadFile(filepaths.DashboardServiceStateFilePath())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	err = json.Unmarshal(stateBytes, state)
	return state, err
}
