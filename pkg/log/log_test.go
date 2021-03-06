package log

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"sync"
	"testing"

	"github.com/go-kit/kit/log/level"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConcurrentLogging(t *testing.T) {
	// Validate that the logging level can be changed while multiple
	// goroutines are logging without races.
	l := NewLogger(ioutil.Discard)
	var wg sync.WaitGroup

	// Write via multiple goroutines
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(i int) {
			for j := 0; j < 10; j++ {
				level.Info(l).Log(i, j)
			}
			wg.Done()
		}(i)
	}

	// Change log level in separate goroutine
	wg.Add(1)
	go func() {
		for i := 0; i < 10; i++ {
			l.AllowDebug()
			l.AllowInfo()
		}
		wg.Done()
	}()

	wg.Wait()
}

func TestCaller(t *testing.T) {
	// Ensure the correct caller is returned with all the crazy wrapping
	// of loggers
	buf := &bytes.Buffer{}
	l := NewLogger(buf)

	var parsedLog struct {
		Caller string `json:"caller"`
	}

	level.Info(l).Log("foo", "bar")

	err := json.Unmarshal(buf.Bytes(), &parsedLog)
	require.Nil(t, err)
	// This line could fail if the filename for these tests changes from
	// "log_test"
	assert.Contains(t, parsedLog.Caller, "log_test.go:")
}

func TestExtractOsqueryCaller(t *testing.T) {
	testCases := []struct {
		log      string
		expected string
	}{
		{
			`I1101 19:21:40.292618 84815872 distributed.cpp:133] Executing distributed query: kolide:populate:practices:1: SELECT COUNT(*) AS result FROM (select * from time);`,
			`distributed.cpp:133`,
		},
		{
			`E1201 08:21:54.254618 84815872 foobar.m:47] Penguin`,
			`foobar.m:47`,
		},
		{
			`E1201 08:21:54.254618 84815872 unknown] Penguin`,
			``,
		},
		{
			`Just plain bad`,
			``,
		},
	}

	for _, tt := range testCases {
		t.Run("", func(t *testing.T) {
			assert.Equal(t, tt.expected, extractOsqueryCaller(tt.log))
		})
	}
}
