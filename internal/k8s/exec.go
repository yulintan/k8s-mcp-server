package k8s

import (
	"bytes"
	"context"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
)

// ExecOptions specifies a single pod exec invocation.
type ExecOptions struct {
	Namespace string
	PodName   string
	Container string
	Command   []string
}

// ExecOutput holds captured stdout/stderr from a single exec.
type ExecOutput struct {
	Stdout string
	Stderr string
}

// ExecInPod runs a command in a pod and returns captured stdout/stderr.
func ExecInPod(ctx context.Context, cfg *rest.Config, client kubernetes.Interface, opts ExecOptions) (ExecOutput, error) {
	req := client.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(opts.PodName).
		Namespace(opts.Namespace).
		SubResource("exec").
		VersionedParams(&corev1.PodExecOptions{
			Container: opts.Container,
			Command:   opts.Command,
			Stdout:    true,
			Stderr:    true,
			Stdin:     false,
			TTY:       false,
		}, scheme.ParameterCodec)

	exec, err := remotecommand.NewSPDYExecutor(cfg, "POST", req.URL())
	if err != nil {
		return ExecOutput{}, err
	}

	var stdout, stderr bytes.Buffer
	err = exec.StreamWithContext(ctx, remotecommand.StreamOptions{
		Stdout: &stdout,
		Stderr: &stderr,
	})
	return ExecOutput{Stdout: stdout.String(), Stderr: stderr.String()}, err
}
