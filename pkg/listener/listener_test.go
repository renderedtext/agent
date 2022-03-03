package listener

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"runtime"
	"testing"
	"time"

	"github.com/semaphoreci/agent/pkg/api"
	"github.com/semaphoreci/agent/pkg/config"
	"github.com/semaphoreci/agent/pkg/eventlogger"
	testsupport "github.com/semaphoreci/agent/test/support"
	"github.com/stretchr/testify/assert"
)

func Test__Register(t *testing.T) {
	testsupport.SetupTestLogs()

	loghubMockServer := testsupport.NewLoghubMockServer()
	loghubMockServer.Init()

	hubMockServer := testsupport.NewHubMockServer()
	hubMockServer.Init()
	hubMockServer.UseLogsURL(loghubMockServer.URL())

	config := Config{
		ExitOnShutdown:     false,
		Endpoint:           hubMockServer.Host(),
		Token:              "token",
		RegisterRetryLimit: 5,
		Scheme:             "http",
		EnvVars:            []config.HostEnvVar{},
		FileInjections:     []config.FileInjection{},
		AgentVersion:       "0.0.7",
	}

	listener, err := Start(http.DefaultClient, config)
	assert.Nil(t, err)

	if assert.Nil(t, hubMockServer.WaitUntilRegistered()) {
		registerRequest := hubMockServer.GetRegisterRequest()
		assert.NotEmpty(t, registerRequest.Arch)
		assert.NotEmpty(t, registerRequest.Hostname)
		assert.NotEmpty(t, registerRequest.Name)
		assert.NotEmpty(t, registerRequest.OS)
		assert.NotZero(t, registerRequest.PID)
		assert.Equal(t, registerRequest.Version, "0.0.7")
	}

	listener.Stop()
	hubMockServer.Close()
	loghubMockServer.Close()
}

func Test__RegisterRequestIsRetried(t *testing.T) {
	testsupport.SetupTestLogs()

	loghubMockServer := testsupport.NewLoghubMockServer()
	loghubMockServer.Init()

	hubMockServer := testsupport.NewHubMockServer()
	hubMockServer.Init()
	hubMockServer.UseLogsURL(loghubMockServer.URL())
	hubMockServer.RejectRegisterAttempts(3)

	config := Config{
		ExitOnShutdown:     false,
		Endpoint:           hubMockServer.Host(),
		Token:              "token",
		RegisterRetryLimit: 5,
		Scheme:             "http",
		EnvVars:            []config.HostEnvVar{},
		FileInjections:     []config.FileInjection{},
		AgentVersion:       "0.0.7",
	}

	listener, err := Start(http.DefaultClient, config)
	assert.Nil(t, err)

	if assert.Nil(t, hubMockServer.WaitUntilRegistered()) {
		assert.Equal(t, 3, hubMockServer.RegisterAttempts)
		registerRequest := hubMockServer.GetRegisterRequest()
		assert.NotEmpty(t, registerRequest.Arch)
		assert.NotEmpty(t, registerRequest.Hostname)
		assert.NotEmpty(t, registerRequest.Name)
		assert.NotEmpty(t, registerRequest.OS)
		assert.NotZero(t, registerRequest.PID)
		assert.Equal(t, registerRequest.Version, "0.0.7")
	}

	listener.Stop()
	hubMockServer.Close()
	loghubMockServer.Close()
}

func Test__RegistrationFails(t *testing.T) {
	testsupport.SetupTestLogs()

	loghubMockServer := testsupport.NewLoghubMockServer()
	loghubMockServer.Init()

	hubMockServer := testsupport.NewHubMockServer()
	hubMockServer.Init()
	hubMockServer.UseLogsURL(loghubMockServer.URL())
	hubMockServer.RejectRegisterAttempts(10)

	config := Config{
		ExitOnShutdown:     false,
		Endpoint:           hubMockServer.Host(),
		Token:              "token",
		RegisterRetryLimit: 5,
		Scheme:             "http",
		EnvVars:            []config.HostEnvVar{},
		FileInjections:     []config.FileInjection{},
		AgentVersion:       "0.0.7",
	}

	_, err := Start(http.DefaultClient, config)
	assert.NotNil(t, err)
	assert.Equal(t, 4, hubMockServer.RegisterAttempts)

	hubMockServer.Close()
	loghubMockServer.Close()
}

func Test__ShutdownHookIsExecuted(t *testing.T) {
	testsupport.SetupTestLogs()

	loghubMockServer := testsupport.NewLoghubMockServer()
	loghubMockServer.Init()

	hubMockServer := testsupport.NewHubMockServer()
	hubMockServer.Init()
	hubMockServer.UseLogsURL(loghubMockServer.URL())

	hook, err := tempFileWithExtension()
	assert.Nil(t, err)

	/*
	 * To assert that the shutdown hook was executed,
	 * we make it create a file with the same name + .done suffix.
	 * If that file exists after the listener stopped,
	 * it means the shutdown hook was executed.
	 */
	destination := fmt.Sprintf("%s.done", hook)
	err = ioutil.WriteFile(hook, []byte(testsupport.CopyFile(hook, destination)), 0777)
	assert.Nil(t, err)

	config := Config{
		ExitOnShutdown:     false,
		Endpoint:           hubMockServer.Host(),
		Token:              "token",
		RegisterRetryLimit: 5,
		Scheme:             "http",
		EnvVars:            []config.HostEnvVar{},
		FileInjections:     []config.FileInjection{},
		AgentVersion:       "0.0.7",
		ShutdownHookPath:   hook,
	}

	listener, err := Start(http.DefaultClient, config)
	assert.Nil(t, err)

	// listener has not been stopped yet, so file created by shutdown hook does not exist yet
	assert.NoFileExists(t, destination)

	time.Sleep(time.Second)
	listener.Stop()

	// listener has been stopped, so file created by shutdown hook should exist
	assert.FileExists(t, destination)

	os.Remove(hook)
	os.Remove(destination)
	hubMockServer.Close()
	loghubMockServer.Close()
}

func Test__ShutdownHookCanSeeShutdownReason(t *testing.T) {
	testsupport.SetupTestLogs()

	loghubMockServer := testsupport.NewLoghubMockServer()
	loghubMockServer.Init()

	hubMockServer := testsupport.NewHubMockServer()
	hubMockServer.Init()
	hubMockServer.UseLogsURL(loghubMockServer.URL())

	hook, err := tempFileWithExtension()
	assert.Nil(t, err)

	/*
	 * To assert that the shutdown hook has access to the SEMAPHORE_AGENT_SHUTDOWN_REASON
	 * variable, we tell the shutdown hook script to write its value on a new file.
	 */
	destination := fmt.Sprintf("%s.done", hook)
	err = ioutil.WriteFile(hook, []byte(testsupport.EchoEnvVarToFile("SEMAPHORE_AGENT_SHUTDOWN_REASON", destination)), 0777)
	assert.Nil(t, err)

	config := Config{
		ExitOnShutdown:     false,
		Endpoint:           hubMockServer.Host(),
		Token:              "token",
		RegisterRetryLimit: 5,
		Scheme:             "http",
		EnvVars:            []config.HostEnvVar{},
		FileInjections:     []config.FileInjection{},
		AgentVersion:       "0.0.7",
		ShutdownHookPath:   hook,
	}

	listener, err := Start(http.DefaultClient, config)
	assert.Nil(t, err)

	// listener has not been stopped yet, so file created by shutdown hook does not exist yet
	assert.NoFileExists(t, destination)

	time.Sleep(time.Second)
	listener.Stop()

	// listener has been stopped, so file created by shutdown hook should exist
	assert.FileExists(t, destination)

	bytes, err := ioutil.ReadFile(destination)
	assert.Nil(t, err)
	assert.Equal(t, ShutdownReasonRequested.String(), string(bytes))

	os.Remove(hook)
	os.Remove(destination)
	hubMockServer.Close()
	loghubMockServer.Close()
}

func Test__ShutdownAfterJobFinished(t *testing.T) {
	testsupport.SetupTestLogs()

	loghubMockServer := testsupport.NewLoghubMockServer()
	loghubMockServer.Init()

	hubMockServer := testsupport.NewHubMockServer()
	hubMockServer.Init()
	hubMockServer.UseLogsURL(loghubMockServer.URL())

	config := Config{
		ExitOnShutdown:     false,
		DisconnectAfterJob: true,
		Endpoint:           hubMockServer.Host(),
		Token:              "token",
		RegisterRetryLimit: 5,
		Scheme:             "http",
		EnvVars:            []config.HostEnvVar{},
		FileInjections:     []config.FileInjection{},
		AgentVersion:       "0.0.7",
	}

	listener, err := Start(http.DefaultClient, config)
	assert.Nil(t, err)

	hubMockServer.AssignJob(&api.JobRequest{
		ID: "Test__ShutdownAfterJobFinished",
		Commands: []api.Command{
			{Directive: testsupport.Output("hello world")},
		},
		Callbacks: api.Callbacks{
			Finished:         "https://httpbin.org/status/200",
			TeardownFinished: "https://httpbin.org/status/200",
		},
		Logger: api.Logger{
			Method: eventlogger.LoggerMethodPush,
			URL:    loghubMockServer.URL(),
			Token:  "doesnotmatter",
		},
	})

	assert.Nil(t, hubMockServer.WaitUntilDisconnected(30, 2*time.Second))
	assert.Equal(t, listener.JobProcessor.ShutdownReason, ShutdownReasonJobFinished)

	hubMockServer.Close()
	loghubMockServer.Close()
}

func Test__ShutdownAfterIdleTimeout(t *testing.T) {
	testsupport.SetupTestLogs()

	loghubMockServer := testsupport.NewLoghubMockServer()
	loghubMockServer.Init()

	hubMockServer := testsupport.NewHubMockServer()
	hubMockServer.Init()
	hubMockServer.UseLogsURL(loghubMockServer.URL())

	config := Config{
		ExitOnShutdown:             false,
		DisconnectAfterIdleTimeout: 15 * time.Second,
		Endpoint:                   hubMockServer.Host(),
		Token:                      "token",
		RegisterRetryLimit:         5,
		Scheme:                     "http",
		EnvVars:                    []config.HostEnvVar{},
		FileInjections:             []config.FileInjection{},
		AgentVersion:               "0.0.7",
	}

	listener, err := Start(http.DefaultClient, config)
	assert.Nil(t, err)
	assert.Nil(t, hubMockServer.WaitUntilDisconnected(15, 2*time.Second))
	assert.Equal(t, listener.JobProcessor.ShutdownReason, ShutdownReasonIdle)

	hubMockServer.Close()
	loghubMockServer.Close()
}

func Test__ShutdownFromUpstreamWhileWaiting(t *testing.T) {
	testsupport.SetupTestLogs()

	loghubMockServer := testsupport.NewLoghubMockServer()
	loghubMockServer.Init()

	hubMockServer := testsupport.NewHubMockServer()
	hubMockServer.Init()
	hubMockServer.UseLogsURL(loghubMockServer.URL())

	config := Config{
		ExitOnShutdown:     false,
		Endpoint:           hubMockServer.Host(),
		Token:              "token",
		RegisterRetryLimit: 5,
		Scheme:             "http",
		EnvVars:            []config.HostEnvVar{},
		FileInjections:     []config.FileInjection{},
		AgentVersion:       "0.0.7",
	}

	listener, err := Start(http.DefaultClient, config)
	assert.Nil(t, err)

	time.Sleep(time.Second)
	hubMockServer.ScheduleShutdown()

	assert.Nil(t, hubMockServer.WaitUntilDisconnected(5, 2*time.Second))
	assert.Equal(t, listener.JobProcessor.ShutdownReason, ShutdownReasonRequested)

	hubMockServer.Close()
	loghubMockServer.Close()
}

func Test__ShutdownFromUpstreamWhileRunningJob(t *testing.T) {
	testsupport.SetupTestLogs()

	loghubMockServer := testsupport.NewLoghubMockServer()
	loghubMockServer.Init()

	hubMockServer := testsupport.NewHubMockServer()
	hubMockServer.Init()
	hubMockServer.UseLogsURL(loghubMockServer.URL())

	config := Config{
		ExitOnShutdown:     false,
		Endpoint:           hubMockServer.Host(),
		Token:              "token",
		RegisterRetryLimit: 5,
		Scheme:             "http",
		EnvVars:            []config.HostEnvVar{},
		FileInjections:     []config.FileInjection{},
		AgentVersion:       "0.0.7",
	}

	listener, err := Start(http.DefaultClient, config)
	assert.Nil(t, err)

	hubMockServer.AssignJob(&api.JobRequest{
		ID: "Test__ShutdownFromUpstreamWhileRunningJob",
		Commands: []api.Command{
			{Directive: "sleep 300"},
		},
		Callbacks: api.Callbacks{
			Finished:         "https://httpbin.org/status/200",
			TeardownFinished: "https://httpbin.org/status/200",
		},
		Logger: api.Logger{
			Method: eventlogger.LoggerMethodPush,
			URL:    loghubMockServer.URL(),
			Token:  "doesnotmatter",
		},
	})

	assert.Nil(t, hubMockServer.WaitUntilRunningJob(5, 2*time.Second))
	hubMockServer.ScheduleShutdown()

	assert.Nil(t, hubMockServer.WaitUntilDisconnected(10, 2*time.Second))
	assert.Equal(t, listener.JobProcessor.ShutdownReason, ShutdownReasonRequested)

	hubMockServer.Close()
	loghubMockServer.Close()
}

func tempFileWithExtension() (string, error) {
	tmpFile, err := ioutil.TempFile("", fmt.Sprintf("file*.%s", extension()))
	if err != nil {
		return "", err
	}

	tmpFile.Close()
	return tmpFile.Name(), nil
}

func extension() string {
	if runtime.GOOS == "windows" {
		return "ps1"
	}

	return "sh"
}
