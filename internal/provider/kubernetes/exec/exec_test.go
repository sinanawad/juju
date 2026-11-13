// Copyright 2019 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package exec_test

import (
	"bytes"
	"context"
	"io"
	"net/url"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/canonical/gomock/gomock"
	"github.com/juju/errors"
	"github.com/juju/tc"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
	k8sexec "k8s.io/client-go/util/exec"

	"github.com/juju/juju/internal/provider/kubernetes/exec"
	"github.com/juju/juju/internal/provider/kubernetes/exec/mocks"
	coretesting "github.com/juju/juju/internal/testing"
)

type execSuite struct {
	BaseSuite
}

func TestExecSuite(t *testing.T) {
	tc.Run(t, &execSuite{})
}

func (s *execSuite) TestExecParamsValidateCommandsAndPodName(c *tc.C) {
	ctrl := s.setupExecClient(c)
	defer ctrl.Finish()

	type testcase struct {
		Params  exec.ExecParams
		Err     string
		PodName string
	}

	for _, testCase := range []testcase{
		{
			Params: exec.ExecParams{},
			Err:    "empty commands not valid",
		},
		{
			Params: exec.ExecParams{
				Commands:    []string{"echo", "'hello world'"},
				Interactive: true,
			},
			Err: "commands with interactive shell not valid",
		},
		{
			Params: exec.ExecParams{
				Commands: []string{"echo", "'hello world'"},
				PodName:  "",
			},
			Err: `podName "" not valid`,
		},
		{
			Params: exec.ExecParams{
				Commands: []string{"echo", "'hello world'"},
				PodName:  "cm/gitlab-k8s-0",
			},
			Err: `podName "cm/gitlab-k8s-0" not valid`,
		},
		{
			Params: exec.ExecParams{
				Commands: []string{"echo", "'hello world'"},
				PodName:  "cm/",
			},
			Err: `podName "cm/" not valid`,
		},
		{
			Params: exec.ExecParams{
				Commands: []string{"echo", "'hello world'"},
				PodName:  "pod/",
			},
			Err: `podName "pod/" not valid`,
		},
	} {
		c.Check(testCase.Params.Validate(c.Context(), s.mockPodGetter), tc.ErrorMatches, testCase.Err)
	}

}

func (s *execSuite) TestProcessEnv(c *tc.C) {
	ctrl := s.setupExecClient(c)
	defer ctrl.Finish()

	res, err := exec.ProcessEnv(
		[]string{
			"AAA=1", "BBB=1 2", "CCC=1\n2", "DDD=1='2'", "EEE=1;2;\"foo\"",
		},
	)
	c.Assert(err, tc.ErrorIsNil)
	c.Assert(res, tc.Equals, "export AAA=1; export BBB='1 2'; export CCC='1\n2'; export DDD=1=\\'2\\'; export EEE=1\\;2\\;\\\"foo\\\"; ")
}

func (s *execSuite) TestSafeRunUsesExternalTerminalSizeQueue(c *tc.C) {
	ctrl := gomock.NewController(c)
	defer ctrl.Finish()

	executor := mocks.NewMockExecutor(ctrl)
	queue := &terminalSizeQueueStub{}
	var stdin, stdout bytes.Buffer
	executor.EXPECT().StreamWithContext(context.Background(), remotecommand.StreamOptions{
		Stdin:             &stdin,
		Stdout:            &stdout,
		Tty:               true,
		TerminalSizeQueue: queue,
	}).Return(nil)

	err := exec.SafeRun(context.Background(), exec.ExecParams{
		Stdin:             &stdin,
		Stdout:            &stdout,
		TTY:               true,
		TerminalSizeQueue: queue,
	}, executor)
	c.Assert(err, tc.ErrorIsNil)
}

type terminalSizeQueueStub struct{}

func (*terminalSizeQueueStub) Next() *exec.TerminalSize {
	return nil
}

func (s *execSuite) TestExecParamsValidatePodContainerExistence(c *tc.C) {
	ctrl := s.setupExecClient(c)
	defer ctrl.Finish()

	s.mockRemoteCMDGetter.EXPECT().RemoteCmdExecutorGetter(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(s.mockRemoteCmdExecutor, nil)

	// failed - completed pod.
	params := exec.ExecParams{
		Commands: []string{"echo", "'hello world'"},
		PodName:  "gitlab-k8s-uid",
	}
	pod := core.Pod{
		Spec: core.PodSpec{
			InitContainers: []core.Container{
				{Name: "gitlab-container"},
			},
		},
		Status: core.PodStatus{
			Phase: core.PodSucceeded,
		},
	}
	pod.SetUID("gitlab-k8s-uid")
	pod.SetName("gitlab-k8s-0")
	gomock.InOrder(
		s.mockPodGetter.EXPECT().Get(gomock.Any(), "gitlab-k8s-uid", metav1.GetOptions{}).
			Return(nil, s.k8sNotFoundError()),
		s.mockPodGetter.EXPECT().List(gomock.Any(), metav1.ListOptions{}).
			Return(&core.PodList{Items: []core.Pod{pod}}, nil),
	)
	c.Assert(params.Validate(c.Context(), s.mockPodGetter), tc.ErrorMatches, `cannot exec into a container within the "Succeeded" pod "gitlab-k8s-0"`)

	// failed - failed pod
	params = exec.ExecParams{
		Commands: []string{"echo", "'hello world'"},
		PodName:  "gitlab-k8s-uid",
	}
	pod = core.Pod{
		Spec: core.PodSpec{
			InitContainers: []core.Container{
				{Name: "gitlab-container"},
			},
		},
		Status: core.PodStatus{
			Phase: core.PodFailed,
		},
	}
	pod.SetUID("gitlab-k8s-uid")
	pod.SetName("gitlab-k8s-0")
	gomock.InOrder(
		s.mockPodGetter.EXPECT().Get(gomock.Any(), "gitlab-k8s-uid", metav1.GetOptions{}).
			Return(nil, s.k8sNotFoundError()),
		s.mockPodGetter.EXPECT().List(gomock.Any(), metav1.ListOptions{}).
			Return(&core.PodList{Items: []core.Pod{pod}}, nil),
	)
	c.Assert(params.Validate(c.Context(), s.mockPodGetter), tc.ErrorMatches, `cannot exec into a container within the "Failed" pod "gitlab-k8s-0"`)

	// failed - containerName not found
	params = exec.ExecParams{
		Commands:      []string{"echo", "'hello world'"},
		PodName:       "gitlab-k8s-uid",
		ContainerName: "non-existing-container-name",
	}
	pod = core.Pod{
		Spec: core.PodSpec{
			InitContainers: []core.Container{
				{Name: "gitlab-container"},
			},
		},
		Status: core.PodStatus{
			Phase: core.PodRunning,
		},
	}
	pod.SetUID("gitlab-k8s-uid")
	pod.SetName("gitlab-k8s-0")
	gomock.InOrder(
		s.mockPodGetter.EXPECT().Get(gomock.Any(), "gitlab-k8s-uid", metav1.GetOptions{}).
			Return(nil, s.k8sNotFoundError()),
		s.mockPodGetter.EXPECT().List(gomock.Any(), metav1.ListOptions{}).
			Return(&core.PodList{Items: []core.Pod{pod}}, nil),
	)
	c.Assert(params.Validate(c.Context(), s.mockPodGetter), tc.ErrorMatches, `container "non-existing-container-name" not found`)

	// all good - container name specified for init container
	params = exec.ExecParams{
		Commands:      []string{"echo", "'hello world'"},
		PodName:       "gitlab-k8s-uid",
		ContainerName: "gitlab-container",
	}
	pod = core.Pod{
		Spec: core.PodSpec{
			InitContainers: []core.Container{
				{Name: "gitlab-container"},
			},
		},
		Status: core.PodStatus{
			Phase: core.PodPending,
			InitContainerStatuses: []core.ContainerStatus{
				{Name: "gitlab-container", State: core.ContainerState{Running: &core.ContainerStateRunning{}}},
			},
		},
	}
	pod.SetUID("gitlab-k8s-uid")
	pod.SetName("gitlab-k8s-0")
	gomock.InOrder(
		s.mockPodGetter.EXPECT().Get(gomock.Any(), "gitlab-k8s-uid", metav1.GetOptions{}).
			Return(nil, s.k8sNotFoundError()),
		s.mockPodGetter.EXPECT().List(gomock.Any(), metav1.ListOptions{}).
			Return(&core.PodList{Items: []core.Pod{pod}}, nil),
	)
	c.Assert(params.Validate(c.Context(), s.mockPodGetter), tc.ErrorIsNil)

	// all good - container name specified.
	params = exec.ExecParams{
		Commands:      []string{"echo", "'hello world'"},
		PodName:       "gitlab-k8s-uid",
		ContainerName: "gitlab-container",
	}
	pod = core.Pod{
		Spec: core.PodSpec{
			Containers: []core.Container{
				{Name: "gitlab-container"},
			},
		},
		Status: core.PodStatus{
			Phase: core.PodRunning,
			ContainerStatuses: []core.ContainerStatus{
				{Name: "gitlab-container", State: core.ContainerState{Running: &core.ContainerStateRunning{}}},
			},
		},
	}
	pod.SetUID("gitlab-k8s-uid")
	pod.SetName("gitlab-k8s-0")
	gomock.InOrder(
		s.mockPodGetter.EXPECT().Get(gomock.Any(), "gitlab-k8s-uid", metav1.GetOptions{}).
			Return(nil, s.k8sNotFoundError()),
		s.mockPodGetter.EXPECT().List(gomock.Any(), metav1.ListOptions{}).
			Return(&core.PodList{Items: []core.Pod{pod}}, nil),
	)
	c.Assert(params.Validate(c.Context(), s.mockPodGetter), tc.ErrorIsNil)

	// non fatal error - container not running - container name specified.
	params = exec.ExecParams{
		Commands:      []string{"echo", "'hello world'"},
		PodName:       "gitlab-k8s-uid",
		ContainerName: "gitlab-container",
	}
	pod = core.Pod{
		Spec: core.PodSpec{
			Containers: []core.Container{
				{Name: "gitlab-container"},
			},
		},
		Status: core.PodStatus{
			Phase: core.PodRunning,
			ContainerStatuses: []core.ContainerStatus{
				{Name: "gitlab-container", State: core.ContainerState{Waiting: &core.ContainerStateWaiting{}}},
			},
		},
	}
	pod.SetUID("gitlab-k8s-uid")
	pod.SetName("gitlab-k8s-0")
	gomock.InOrder(
		s.mockPodGetter.EXPECT().Get(gomock.Any(), "gitlab-k8s-uid", metav1.GetOptions{}).Times(1).
			Return(nil, s.k8sNotFoundError()),
		s.mockPodGetter.EXPECT().List(gomock.Any(), metav1.ListOptions{}).Times(1).
			Return(&core.PodList{Items: []core.Pod{pod}}, nil),
	)
	c.Assert(params.Validate(c.Context(), s.mockPodGetter), tc.ErrorMatches, `container \"gitlab-container\" not running`)

	// all good - no container name specified, pick the 1st container.
	params = exec.ExecParams{
		Commands: []string{"echo", "'hello world'"},
		PodName:  "gitlab-k8s-uid",
	}
	c.Assert(params.ContainerName, tc.Equals, "")
	pod = core.Pod{
		Spec: core.PodSpec{
			Containers: []core.Container{
				{Name: "gitlab-container"},
			},
		},
		Status: core.PodStatus{
			Phase: core.PodRunning,
			ContainerStatuses: []core.ContainerStatus{
				{Name: "gitlab-container", State: core.ContainerState{Running: &core.ContainerStateRunning{}}},
			},
		},
	}
	pod.SetUID("gitlab-k8s-uid")
	pod.SetName("gitlab-k8s-0")
	gomock.InOrder(
		s.mockPodGetter.EXPECT().Get(gomock.Any(), "gitlab-k8s-uid", metav1.GetOptions{}).
			Return(nil, s.k8sNotFoundError()),
		s.mockPodGetter.EXPECT().List(gomock.Any(), metav1.ListOptions{}).
			Return(&core.PodList{Items: []core.Pod{pod}}, nil),
	)
	c.Assert(params.Validate(c.Context(), s.mockPodGetter), tc.ErrorIsNil)
	c.Assert(params.ContainerName, tc.Equals, "gitlab-container")
}

func (s *execSuite) TestExec(c *tc.C) {
	ctrl := s.setupExecClient(c)
	defer ctrl.Finish()

	s.mockRemoteCMDGetter.EXPECT().RemoteCmdExecutorGetter(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(s.mockRemoteCmdExecutor, nil)

	var stdin, stdout, stderr bytes.Buffer
	params := exec.ExecParams{
		Commands: []string{"echo", "'hello world'"},
		PodName:  "gitlab-k8s-uid",
		Stdout:   &stdout,
		Stderr:   &stderr,
		Stdin:    &stdin,
	}
	c.Assert(params.ContainerName, tc.Equals, "")
	pod := core.Pod{
		Spec: core.PodSpec{
			Containers: []core.Container{
				{Name: "gitlab-container"},
			},
		},
		Status: core.PodStatus{
			Phase: core.PodRunning,
			ContainerStatuses: []core.ContainerStatus{
				{Name: "gitlab-container", State: core.ContainerState{Running: &core.ContainerStateRunning{}}},
			},
		},
	}
	pod.SetUID("gitlab-k8s-uid")
	pod.SetName("gitlab-k8s-0")

	request := rest.NewRequestWithClient(
		&url.URL{Path: "/path/"},
		"",
		rest.ClientContentConfig{GroupVersion: core.SchemeGroupVersion},
		nil,
	).Resource("pods").Name("gitlab-k8s-0").Namespace("test").
		SubResource("exec").Param("container", "gitlab-container").VersionedParams(
		&core.PodExecOptions{
			Container: "gitlab-container",
			Command:   []string{""},
			Stdin:     true,
			Stdout:    true,
			Stderr:    true,
			TTY:       false,
		}, scheme.ParameterCodec)
	gomock.InOrder(
		s.mockPodGetter.EXPECT().Get(gomock.Any(), "gitlab-k8s-uid", metav1.GetOptions{}).
			Return(nil, s.k8sNotFoundError()),
		s.mockPodGetter.EXPECT().List(gomock.Any(), metav1.ListOptions{}).
			Return(&core.PodList{Items: []core.Pod{pod}}, nil),

		s.restClient.EXPECT().Post().Return(request),
		s.mockRemoteCmdExecutor.EXPECT().StreamWithContext(
			gomock.Any(),
			streamOptionsLike(remotecommand.StreamOptions{
				Stdin:  &stdin,
				Stdout: &stdout,
				Stderr: &stderr,
				Tty:    false,
			}),
		).Return(nil),
	)

	cancel := make(<-chan struct{}, 1)
	errChan := make(chan error, 1)
	go func() {
		errChan <- s.execClient.Exec(c.Context(), params, cancel)
	}()

	select {
	case err := <-errChan:
		c.Assert(err, tc.ErrorIsNil)
	case <-time.After(coretesting.ShortWait):
		c.Fatalf("timed out waiting for Exec return")
	}
}

func (s *execSuite) TestExecCancel(c *tc.C) {
	ctrl := s.setupExecClient(c)
	defer ctrl.Finish()

	s.PatchValue(exec.RandomString, func(n int, validRunes []rune) string {
		return "random"
	})

	var stdin, stdout, stderr bytes.Buffer
	params := exec.ExecParams{
		Commands: []string{"echo", "'hello world'"},
		PodName:  "gitlab-k8s-uid",
		Stdout:   &stdout,
		Stderr:   &stderr,
		Stdin:    &stdin,
	}
	c.Assert(params.ContainerName, tc.Equals, "")
	pod := core.Pod{
		Spec: core.PodSpec{
			Containers: []core.Container{
				{Name: "gitlab-container"},
			},
		},
		Status: core.PodStatus{
			Phase: core.PodRunning,
			ContainerStatuses: []core.ContainerStatus{
				{Name: "gitlab-container", State: core.ContainerState{Running: &core.ContainerStateRunning{}}},
			},
		},
	}
	pod.SetUID("gitlab-k8s-uid")
	pod.SetName("gitlab-k8s-0")

	cancel := make(chan struct{})
	wait := make(chan struct{})
	mut := sync.Mutex{}
	callNum := 0

	urls := []string{
		"/path/namespaces/test/pods/gitlab-k8s-0/exec?command=sh&command=-c&command=mkdir+-p+%2Ftmp%3B+echo+%24%24+%3E+%2Ftmp%2Frandom.pid%3B+exec+sh+-c+%27echo+%27%5C%27%27hello+world%27%5C%27%3B+&container=gitlab-container&container=gitlab-container&stderr=true&stdin=true&stdout=true",
		"/path/namespaces/test/pods/gitlab-k8s-0/exec?command=sh&command=-c&command=kill+-15+%24%28cat+%2Ftmp%2Frandom.pid%29&container=gitlab-container&container=gitlab-container&stderr=true&stdout=true",
		"/path/namespaces/test/pods/gitlab-k8s-0/exec?command=sh&command=-c&command=kill+-9+-%24%28cat+%2Ftmp%2Frandom.pid%29&container=gitlab-container&container=gitlab-container&stderr=true&stdout=true",
		"/path/namespaces/test/pods/gitlab-k8s-0/exec?command=sh&command=-c&command=kill+-9+-%24%28cat+%2Ftmp%2Frandom.pid%29&container=gitlab-container&container=gitlab-container&stderr=true&stdout=true",
	}
	waitTime := 11 * time.Second
	requests := []*rest.Request{}
	for range urls {
		request := rest.NewRequestWithClient(
			&url.URL{Path: "/path/"},
			"",
			rest.ClientContentConfig{GroupVersion: core.SchemeGroupVersion},
			nil,
		)
		requests = append(requests, request)
	}

	gomock.InOrder(
		s.mockPodGetter.EXPECT().Get(gomock.Any(), "gitlab-k8s-uid", metav1.GetOptions{}).
			Return(nil, s.k8sNotFoundError()),
		s.mockPodGetter.EXPECT().List(gomock.Any(), metav1.ListOptions{}).
			Return(&core.PodList{Items: []core.Pod{pod}}, nil),
	)

	s.restClient.EXPECT().Post().AnyTimes().DoAndReturn(func() *rest.Request {
		mut.Lock()
		defer mut.Unlock()
		c.Assert(callNum, tc.LessThan, len(requests))
		return requests[callNum]
	})
	s.mockRemoteCMDGetter.EXPECT().RemoteCmdExecutorGetter(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().
		DoAndReturn(func(config *rest.Config, method string, url *url.URL) (remotecommand.Executor, error) {
			mut.Lock()
			defer mut.Unlock()
			c.Assert(callNum, tc.LessThan, len(urls))
			c.Check(url.String(), tc.Equals, urls[callNum])
			return s.mockRemoteCmdExecutor, nil
		})
	s.mockRemoteCmdExecutor.EXPECT().StreamWithContext(gomock.Any(), gomock.Any()).AnyTimes().DoAndReturn(func(ctx context.Context, opts remotecommand.StreamOptions) error {
		mut.Lock()
		currentCall := callNum
		callNum++
		mut.Unlock()
		switch currentCall {
		case 0:
			close(cancel)
			err := s.clock.WaitAdvance(waitTime, coretesting.LongWait, 1)
			c.Assert(err, tc.ErrorIsNil)
		case 1:
		case 2:
		case 3:
			close(wait)
		}
		return nil
	})

	errChan := make(chan error, 1)
	go func() {
		errChan <- s.execClient.Exec(c.Context(), params, cancel)
	}()

	select {
	case err := <-errChan:
		c.Assert(err, tc.ErrorIsNil)
	case <-time.After(coretesting.ShortWait):
		c.Fatalf("timed out waiting for Exec return")
	}
}

func (s *execSuite) TestErrorHandling(c *tc.C) {
	err := exec.HandleContainerNotFoundError(errors.New(`unable to upgrade connection: container not found ("mariadb-k8s")`))
	c.Assert(err, tc.FitsTypeOf, &exec.ContainerNotRunningError{})
	err = exec.HandleContainerNotFoundError(errors.New(`wow`))
	c.Assert(err, tc.ErrorMatches, "wow")
}

const missingExecutableInPath = `Internal error occurred: error executing command in container: failed to exec in container: failed to start exec "abc": OCI runtime exec failed: exec failed: unable to start container process: exec: "sh": executable file not found in $PATH`

func (s *execSuite) TestExecShellSelection(c *tc.C) {
	missingSh := errors.New(missingExecutableInPath)
	missingBash := errors.New(`OCI runtime exec failed: exec failed: stat /usr/bin/bash: no such file or directory`)
	podNotFound := errors.New(`unable to upgrade connection: pods "does-not-exist" not found`)
	command := exec.ExecParams{
		Commands: []string{"echo", "SSH_OK"},
		PodName:  "gitlab-k8s-uid",
	}

	for _, test := range []struct {
		name       string
		params     exec.ExecParams
		streamErrs []error
		check      func(c *tc.C, urls []string, err error)
	}{{
		name:       "sh missing then bash",
		params:     command,
		streamErrs: []error{missingSh, nil},
		check: func(c *tc.C, urls []string, err error) {
			c.Assert(err, tc.ErrorIsNil)
			c.Assert(urls, tc.HasLen, 2)
			c.Check(strings.Contains(urls[0], "command=sh"), tc.IsTrue)
			c.Check(strings.Contains(urls[0], "mkdir+-p+%2Ftmp"), tc.IsTrue)
			c.Check(strings.Contains(urls[1], "command=bash"), tc.IsTrue)
			c.Check(strings.Contains(urls[1], "%2Fdev%2Fshm%2F"), tc.IsTrue)
			c.Check(strings.Contains(urls[1], "mkdir"), tc.IsFalse)
		},
	}, {
		name:       "neither shell",
		params:     command,
		streamErrs: []error{missingSh, missingBash},
		check: func(c *tc.C, urls []string, err error) {
			c.Assert(urls, tc.HasLen, 2)
			c.Check(err, tc.ErrorIs, exec.ErrNoShell)
			c.Check(strings.Contains(err.Error(), "couldn't find any shell"), tc.IsTrue)
			c.Check(strings.Contains(err.Error(), "executable file not found"), tc.IsFalse)
			c.Check(strings.Contains(err.Error(), "no such file or directory"), tc.IsFalse)
		},
	}, {
		name:       "pod not found",
		params:     command,
		streamErrs: []error{podNotFound},
		check: func(c *tc.C, urls []string, err error) {
			c.Assert(urls, tc.HasLen, 1)
			c.Check(err, tc.Not(tc.ErrorIs), exec.ErrNoShell)
			c.Check(err, tc.ErrorMatches, `.*pods "does-not-exist" not found.*`)
		},
	}, {
		name:       "exit code",
		params:     command,
		streamErrs: []error{k8sexec.CodeExitError{Err: errors.New("exit status 7"), Code: 7}},
		check: func(c *tc.C, urls []string, err error) {
			c.Assert(urls, tc.HasLen, 1)
			c.Check(err, tc.Not(tc.ErrorIs), exec.ErrNoShell)
			c.Check(err, tc.ErrorMatches, ".*exit status 7.*")
		},
	}, {
		name: "interactive non-login",
		params: exec.ExecParams{
			PodName:     "gitlab-k8s-uid",
			Interactive: true,
		},
		streamErrs: []error{nil},
		check: func(c *tc.C, urls []string, err error) {
			c.Assert(err, tc.ErrorIsNil)
			c.Assert(urls, tc.HasLen, 1)
			c.Check(strings.Contains(urls[0], "exec+sh%3B"), tc.IsTrue)
			c.Check(strings.Contains(urls[0], "exec+sh+-c"), tc.IsFalse)
		},
	}, {
		name: "interactive login",
		params: exec.ExecParams{
			PodName:     "gitlab-k8s-uid",
			Interactive: true,
			LoginShell:  true,
		},
		streamErrs: []error{nil},
		check: func(c *tc.C, urls []string, err error) {
			c.Assert(err, tc.ErrorIsNil)
			c.Assert(urls, tc.HasLen, 1)
			c.Check(strings.Contains(urls[0], "command=bash"), tc.IsTrue)
			c.Check(strings.Contains(urls[0], "exec+bash+--login"), tc.IsTrue)
		},
	}} {
		c.Logf("case %s", test.name)
		var streams []func(context.Context, remotecommand.StreamOptions) error
		for _, streamErr := range test.streamErrs {
			streams = append(streams, func(context.Context, remotecommand.StreamOptions) error {
				return streamErr
			})
		}
		urls, err := s.execRecordingStreams(c, test.params, streams)
		test.check(c, urls, err)
	}
}

func (s *execSuite) TestExecRetryKeepsStdin(c *tc.C) {
	var first io.Reader
	var got []byte
	streams := []func(context.Context, remotecommand.StreamOptions) error{
		func(_ context.Context, opts remotecommand.StreamOptions) error {
			// The failed attempt consumed part of the input before the
			// missing shell was reported. Streams run on another goroutine,
			// so they must not use c.Assert.
			first = opts.Stdin
			buf := make([]byte, 5)
			_, err := io.ReadFull(opts.Stdin, buf)
			c.Check(err, tc.ErrorIsNil)
			c.Check(string(buf), tc.Equals, "hello")
			return errors.New(missingExecutableInPath)
		},
		func(_ context.Context, opts remotecommand.StreamOptions) error {
			// The detached reader of the failed attempt must not take
			// input from the retry.
			n, err := first.Read(make([]byte, 1))
			c.Check(n, tc.Equals, 0)
			c.Check(err, tc.Equals, io.EOF)

			var readErr error
			got, readErr = io.ReadAll(opts.Stdin)
			c.Check(readErr, tc.ErrorIsNil)
			return nil
		},
	}
	params := exec.ExecParams{
		Commands: []string{"cat"},
		PodName:  "gitlab-k8s-uid",
		Stdin:    strings.NewReader("hello world"),
	}

	urls, err := s.execRecordingStreams(c, params, streams)
	c.Assert(err, tc.ErrorIsNil)
	c.Check(urls, tc.HasLen, 2)
	c.Check(string(got), tc.Equals, "hello world")
}

func (s *execSuite) TestExecRetryRefusedAfterLargeStdin(c *tc.C) {
	streams := []func(context.Context, remotecommand.StreamOptions) error{
		func(_ context.Context, opts remotecommand.StreamOptions) error {
			_, err := io.CopyN(io.Discard, opts.Stdin, exec.MaxStdinReplay+1)
			c.Check(err, tc.ErrorIsNil)
			return errors.New(missingExecutableInPath)
		},
	}
	params := exec.ExecParams{
		Commands: []string{"cat"},
		PodName:  "gitlab-k8s-uid",
		Stdin:    strings.NewReader(strings.Repeat("x", exec.MaxStdinReplay+1)),
	}

	urls, err := s.execRecordingStreams(c, params, streams)
	c.Check(urls, tc.HasLen, 1)
	c.Check(err, tc.Not(tc.ErrorIs), exec.ErrNoShell)
	c.Check(err, tc.ErrorMatches, "cannot retry with bash: .*")
}

func (s *execSuite) TestExecLoginShellFallback(c *tc.C) {
	missingBash := errors.New(`OCI runtime exec failed: exec failed: stat /usr/bin/bash: no such file or directory`)
	for _, fallbackErr := range []error{nil, errors.New(missingExecutableInPath)} {
		params := exec.ExecParams{
			PodName:     "gitlab-k8s-uid",
			Interactive: true,
			LoginShell:  true,
		}
		streams := []func(context.Context, remotecommand.StreamOptions) error{
			func(context.Context, remotecommand.StreamOptions) error { return missingBash },
			func(context.Context, remotecommand.StreamOptions) error { return fallbackErr },
		}
		urls, err := s.execRecordingStreams(c, params, streams)
		c.Assert(urls, tc.HasLen, 2)
		first, parseErr := url.Parse(urls[0])
		c.Assert(parseErr, tc.ErrorIsNil)
		second, parseErr := url.Parse(urls[1])
		c.Assert(parseErr, tc.ErrorIsNil)
		c.Check(first.Query()["command"], tc.DeepEquals, []string{
			"bash", "-c", "echo $$ > /dev/shm/random.pid; exec bash --login; ",
		})
		c.Check(second.Query()["command"], tc.DeepEquals, []string{
			"sh", "-c", "mkdir -p /tmp; echo $$ > /tmp/random.pid; exec sh; ",
		})
		if fallbackErr == nil {
			c.Assert(err, tc.ErrorIsNil)
		} else {
			c.Check(err, tc.ErrorIs, exec.ErrNoShell)
		}
	}
}

func (s *execSuite) TestExecPreservesNonShellErrors(c *tc.C) {
	exitErr := k8sexec.CodeExitError{
		Err:  errors.New(missingExecutableInPath),
		Code: 7,
	}
	for _, test := range []struct {
		name string
		err  error
	}{
		{name: "RBAC", err: errors.New(`pods "gitlab-k8s-0" is forbidden`)},
		{name: "network", err: errors.New("dial tcp: connection refused")},
		{name: "cancelled", err: context.Canceled},
		{name: "deadline", err: context.DeadlineExceeded},
		{name: "exit with misleading runtime text", err: exitErr},
		{name: "wrapped exit", err: errors.Annotate(exitErr, "stream failed")},
		{name: "container missing", err: errors.New(`unable to upgrade connection: container not found ("gitlab-container")`)},
		{name: "working directory missing", err: errors.New(`OCI runtime exec failed: exec failed: unable to start container process: chdir to cwd ("/work") set in config.json failed: no such file or directory`)},
		{name: "unrelated path missing", err: errors.New(`OCI runtime exec failed: exec failed: stat /etc/config: no such file or directory`)},
	} {
		for _, onFallback := range []bool{false, true} {
			c.Logf("%s, on fallback: %t", test.name, onFallback)
			var streams []func(context.Context, remotecommand.StreamOptions) error
			if onFallback {
				streams = append(streams, func(context.Context, remotecommand.StreamOptions) error {
					return errors.New(missingExecutableInPath)
				})
			}
			streams = append(streams, func(context.Context, remotecommand.StreamOptions) error {
				return test.err
			})
			params := exec.ExecParams{
				Commands: []string{"echo", "SSH_OK"},
				PodName:  "gitlab-k8s-uid",
			}
			urls, err := s.execRecordingStreams(c, params, streams)
			c.Check(urls, tc.HasLen, len(streams))
			c.Check(err, tc.Not(tc.ErrorIs), exec.ErrNoShell)
			c.Check(errors.Cause(err), tc.Equals, errors.Cause(test.err))
		}
	}
}

func (s *execSuite) TestExecMissingShellErrorShapes(c *tc.C) {
	for _, missing := range []string{
		missingExecutableInPath,
		`OCI runtime exec failed: exec failed: exec: "/bin/sh": stat /bin/sh: no such file or directory`,
		`OCI runtime exec failed: exec failed: stat /usr/bin/sh: no such file or directory`,
	} {
		c.Logf("missing shell: %s", missing)
		params := exec.ExecParams{
			Commands: []string{"echo", "SSH_OK"},
			PodName:  "gitlab-k8s-uid",
		}
		streams := []func(context.Context, remotecommand.StreamOptions) error{
			func(context.Context, remotecommand.StreamOptions) error {
				return errors.Annotate(errors.New(missing), "stream failed")
			},
			func(context.Context, remotecommand.StreamOptions) error { return nil },
		}
		urls, err := s.execRecordingStreams(c, params, streams)
		c.Assert(err, tc.ErrorIsNil)
		c.Check(urls, tc.HasLen, 2)
	}
}

func (s *execSuite) TestExecRetryPreservesTerminalSizes(c *tc.C) {
	source := newControlledSizeQueue(c.Context())
	initial := exec.TerminalSize{Width: 80, Height: 24}
	latest := exec.TerminalSize{Width: 120, Height: 40}
	resized := exec.TerminalSize{Width: 160, Height: 50}
	var failed, replacement exec.TerminalSizeQueue
	var pending <-chan *exec.TerminalSize
	streams := []func(context.Context, remotecommand.StreamOptions) error{
		func(_ context.Context, opts remotecommand.StreamOptions) error {
			c.Check(opts.Tty, tc.IsTrue)
			failed = opts.TerminalSizeQueue
			for _, size := range []exec.TerminalSize{initial, latest} {
				got := readSize(failed)
				<-source.entered
				source.sizes <- &size
				c.Check(<-got, tc.DeepEquals, &size)
			}
			// Client-go leaves this resize reader behind when sh fails.
			pending = readSize(failed)
			<-source.entered
			return errors.New(missingExecutableInPath)
		},
		func(_ context.Context, opts remotecommand.StreamOptions) error {
			c.Check(opts.Tty, tc.IsTrue)
			replacement = opts.TerminalSizeQueue
			c.Check(replacement.Next(), tc.DeepEquals, &latest)
			got := readSize(replacement)
			source.sizes <- &resized
			c.Check(<-pending, tc.IsNil)
			c.Check(<-got, tc.DeepEquals, &resized)
			c.Check(failed.Next(), tc.IsNil)
			return nil
		},
	}
	params := exec.ExecParams{
		PodName:           "gitlab-k8s-uid",
		Interactive:       true,
		TTY:               true,
		TerminalSizeQueue: source,
	}
	urls, err := s.execRecordingStreams(c, params, streams)
	c.Assert(err, tc.ErrorIsNil)
	c.Check(urls, tc.HasLen, 2)
	c.Check(replacement.Next(), tc.IsNil)
}

func (s *execSuite) TestExecBashFallbackForwardsSignals(c *tc.C) {
	for _, sig := range []syscall.Signal{syscall.SIGTERM, syscall.SIGKILL} {
		c.Logf("signal %d", sig)
		signals := make(chan syscall.Signal)
		handled := make(chan struct{})
		streams := []func(context.Context, remotecommand.StreamOptions) error{
			func(context.Context, remotecommand.StreamOptions) error {
				return errors.New(missingExecutableInPath)
			},
			func(context.Context, remotecommand.StreamOptions) error {
				signals <- sig
				<-handled
				return nil
			},
			func(context.Context, remotecommand.StreamOptions) error {
				close(handled)
				return nil
			},
		}
		params := exec.ExecParams{
			PodName:     "gitlab-k8s-uid",
			Interactive: true,
			Signal:      signals,
		}
		urls, err := s.execRecordingStreams(c, params, streams)
		c.Assert(err, tc.ErrorIsNil)
		c.Assert(urls, tc.HasLen, 3)
		signalURL, parseErr := url.Parse(urls[2])
		c.Assert(parseErr, tc.ErrorIsNil)
		want := "kill -15 $(</dev/shm/random.pid)"
		if sig == syscall.SIGKILL {
			want = "kill -9 -$(</dev/shm/random.pid)"
		}
		c.Check(signalURL.Query()["command"], tc.DeepEquals,
			[]string{"bash", "-c", want})
	}
}

func (s *execSuite) TestExecSignalFailureDoesNotRetryCommand(c *tc.C) {
	signals := make(chan syscall.Signal)
	primaryDone := make(chan struct{})
	streams := []func(context.Context, remotecommand.StreamOptions) error{
		func(ctx context.Context, _ remotecommand.StreamOptions) error {
			// The command has started. A later signal error must not cause
			// it to be executed a second time with another shell.
			signals <- syscall.SIGTERM
			<-ctx.Done()
			close(primaryDone)
			return ctx.Err()
		},
		func(context.Context, remotecommand.StreamOptions) error {
			return errors.New(missingExecutableInPath)
		},
	}
	params := exec.ExecParams{
		PodName:  "gitlab-k8s-uid",
		Commands: []string{"echo", "started"},
		Signal:   signals,
	}
	urls, err := s.execRecordingStreams(c, params, streams)
	c.Check(err, tc.ErrorMatches, "send signal 15 failed: .*")
	c.Check(err, tc.Not(tc.ErrorIs), exec.ErrNoShell)
	c.Check(urls, tc.HasLen, 2)
	// The failed signal request also cancels the primary exec stream.
	<-primaryDone
}

// execRecordingStreams runs Exec against a running pod and returns the exec
// URLs. Each callback handles an exec or signal stream and must honour its
// context if it blocks, so the caller can cancel and join a failed attempt.
func (s *execSuite) execRecordingStreams(c *tc.C, params exec.ExecParams, streams []func(context.Context, remotecommand.StreamOptions) error) ([]string, error) {
	ctrl := s.setupExecClient(c)
	defer ctrl.Finish()

	s.PatchValue(exec.RandomString, func(n int, validRunes []rune) string {
		return "random"
	})

	pod := core.Pod{
		Spec: core.PodSpec{
			Containers: []core.Container{{Name: "gitlab-container"}},
		},
		Status: core.PodStatus{
			Phase: core.PodRunning,
			ContainerStatuses: []core.ContainerStatus{
				{Name: "gitlab-container", State: core.ContainerState{Running: &core.ContainerStateRunning{}}},
			},
		},
	}
	pod.SetUID("gitlab-k8s-uid")
	pod.SetName("gitlab-k8s-0")
	gomock.InOrder(
		s.mockPodGetter.EXPECT().Get(gomock.Any(), "gitlab-k8s-uid", metav1.GetOptions{}).
			Return(nil, s.k8sNotFoundError()),
		s.mockPodGetter.EXPECT().List(gomock.Any(), metav1.ListOptions{}).
			Return(&core.PodList{Items: []core.Pod{pod}}, nil),
	)

	var urls []string
	var mu sync.Mutex
	s.restClient.EXPECT().Post().AnyTimes().DoAndReturn(func() *rest.Request {
		return rest.NewRequestWithClient(
			&url.URL{Path: "/path/"},
			"",
			rest.ClientContentConfig{GroupVersion: core.SchemeGroupVersion},
			nil,
		)
	})
	s.mockRemoteCMDGetter.EXPECT().RemoteCmdExecutorGetter(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().
		DoAndReturn(func(_ *rest.Config, _ string, u *url.URL) (remotecommand.Executor, error) {
			mu.Lock()
			urls = append(urls, u.String())
			mu.Unlock()
			return s.mockRemoteCmdExecutor, nil
		})
	call := 0
	s.mockRemoteCmdExecutor.EXPECT().StreamWithContext(gomock.Any(), gomock.Any()).Times(len(streams)).
		DoAndReturn(func(ctx context.Context, opts remotecommand.StreamOptions) error {
			mu.Lock()
			i := call
			call++
			mu.Unlock()
			return streams[i](ctx, opts)
		})

	err := s.execClient.Exec(c.Context(), params, nil)
	return urls, err
}
