package shell

import (
	"encoding/base64"
	"io/ioutil"
	"os"
	"runtime"
	"testing"

	"github.com/semaphoreci/agent/pkg/api"
	"github.com/semaphoreci/agent/pkg/config"
	"github.com/stretchr/testify/assert"
)

func Test__CreateEnvironment(t *testing.T) {
	t.Run("vars from job request are base64 decoded", func(t *testing.T) {
		varsFromRequest := []api.EnvVar{
			{Name: "A", Value: base64.StdEncoding.EncodeToString([]byte("AAA"))},
			{Name: "B", Value: base64.StdEncoding.EncodeToString([]byte("BBB"))},
		}

		env, err := CreateEnvironment(varsFromRequest, []config.HostEnvVar{})
		assert.Nil(t, err)
		assert.NotNil(t, env)

		assertValueExists(t, env, "A", "AAA")
		assertValueExists(t, env, "B", "BBB")
	})

	t.Run("vars from host are not base64 decoded", func(t *testing.T) {
		varsFromHost := []config.HostEnvVar{
			{Name: "A", Value: "AAA"},
			{Name: "B", Value: "BBB"},
		}

		env, err := CreateEnvironment([]api.EnvVar{}, varsFromHost)
		assert.Nil(t, err)
		assert.NotNil(t, env)
		assertValueExists(t, env, "A", "AAA")
		assertValueExists(t, env, "B", "BBB")
	})

	t.Run("var from job request not properly encoded => error", func(t *testing.T) {
		varsFromRequest := []api.EnvVar{
			{Name: "A", Value: "AAA"},
		}

		env, err := CreateEnvironment(varsFromRequest, []config.HostEnvVar{})
		assert.NotNil(t, err)
		assert.Nil(t, env)
	})

	t.Run("var is overwritten by subsequent var in request", func(t *testing.T) {
		varsFromRequest := []api.EnvVar{
			{Name: "FOO", Value: base64.StdEncoding.EncodeToString([]byte("FOO"))},
			{Name: "FOO", Value: base64.StdEncoding.EncodeToString([]byte("BAR"))},
		}

		env, err := CreateEnvironment(varsFromRequest, []config.HostEnvVar{})
		assert.Nil(t, err)
		assertValueExists(t, env, "FOO", "BAR")
	})

	t.Run("var is overwritten by subsequent host var", func(t *testing.T) {
		varsFromRequest := []api.EnvVar{
			{Name: "FOO", Value: base64.StdEncoding.EncodeToString([]byte("FOO"))},
			{Name: "FOO", Value: base64.StdEncoding.EncodeToString([]byte("BAR"))},
		}

		varsFromHost := []config.HostEnvVar{
			{Name: "FOO", Value: "AAA"},
		}

		env, err := CreateEnvironment(varsFromRequest, varsFromHost)
		assert.Nil(t, err)
		assertValueExists(t, env, "FOO", "AAA")
	})
}

func Test__CreateEnvironmentFromFile(t *testing.T) {
	file, err := ioutil.TempFile("", "environment-dump")
	assert.Nil(t, err)

	content := `
VAR_A=AAA
VAR_B=BBB
VAR_C=CCC
	`

	_ = ioutil.WriteFile(file.Name(), []byte(content), 0644)
	env, err := CreateEnvironmentFromFile(file.Name())
	assert.Nil(t, err)
	assert.NotNil(t, env)

	assert.Equal(t, env.Keys(), []string{"VAR_A", "VAR_B", "VAR_C"})
	assertValueExists(t, env, "VAR_A", "AAA")
	assertValueExists(t, env, "VAR_B", "BBB")
	assertValueExists(t, env, "VAR_C", "CCC")

	file.Close()
	os.Remove(file.Name())
}

func Test__EnvironmentToFile(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Environment.ToFile() is only used in non-windows")
	}

	vars := []api.EnvVar{
		{Name: "Z", Value: base64.StdEncoding.EncodeToString([]byte("ZZZ"))},
		{Name: "O", Value: base64.StdEncoding.EncodeToString([]byte("OOO"))},
		{Name: "QUOTED", Value: base64.StdEncoding.EncodeToString([]byte("This is going to get quoted"))},
	}

	env, err := CreateEnvironment(vars, []config.HostEnvVar{})
	assert.Nil(t, err)
	assert.NotNil(t, env)

	file, err := ioutil.TempFile("", ".env")
	assert.Nil(t, err)

	err = env.ToFile(file.Name(), nil)
	assert.Nil(t, err)

	content, err := ioutil.ReadFile(file.Name())
	assert.Nil(t, err)
	assert.Equal(t, string(content), "export O=OOO\nexport QUOTED='This is going to get quoted'\nexport Z=ZZZ\n")

	file.Close()
	os.Remove(file.Name())
}

func Test__EnvironmentToSlice(t *testing.T) {
	varsFromRequest := []api.EnvVar{
		{Name: "A", Value: base64.StdEncoding.EncodeToString([]byte("AAA"))},
		{Name: "B", Value: base64.StdEncoding.EncodeToString([]byte("BBB"))},
		{Name: "C", Value: base64.StdEncoding.EncodeToString([]byte("CCC"))},
	}

	env, err := CreateEnvironment(varsFromRequest, []config.HostEnvVar{})
	assert.Nil(t, err)
	assert.Contains(t, env.ToSlice(), "A=AAA")
	assert.Contains(t, env.ToSlice(), "B=BBB")
	assert.Contains(t, env.ToSlice(), "C=CCC")
}

func Test__EnvironmentAppend(t *testing.T) {
	vars := []api.EnvVar{
		{Name: "C", Value: base64.StdEncoding.EncodeToString([]byte("CCC"))},
		{Name: "D", Value: base64.StdEncoding.EncodeToString([]byte("DDD"))},
		{Name: "A", Value: base64.StdEncoding.EncodeToString([]byte("AAA"))},
	}

	other, _ := CreateEnvironment(vars, []config.HostEnvVar{})
	appended := []string{}

	first := Environment{}
	first.Append(other, func(name, value string) {
		appended = append(appended, name)
	})

	assert.Equal(t, appended, []string{"A", "C", "D"})
	assertValueExists(t, &first, "A", "AAA")
	assertValueExists(t, &first, "C", "CCC")
	assertValueExists(t, &first, "D", "DDD")
}

func assertValueExists(t *testing.T, env *Environment, key, expectedValue string) {
	value, ok := env.Get(key)
	assert.True(t, ok)
	assert.Equal(t, value, expectedValue)
}
