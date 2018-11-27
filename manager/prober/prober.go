package prober

import (
	"fmt"

	"k8s.io/api/core/v1"
	"k8s.io/kubernetes/pkg/kubelet/util/format"
	httprobe "k8s.io/kubernetes/pkg/probe/http"
	tcprobe "k8s.io/kubernetes/pkg/probe/tcp"
)

type prober struct {
	http httprobe.Prober
	tcp  tcprobe.Prober
}

func newProber() *prober {
	return &prober{
		http: httprobe.New(),
		tcp:  tcprobe.New(),
	}
}

// probe probes the container.
func (pb *prober) probe(namespace, podName, containerName, podIP string, container *v1.Container) (bool, error) {
	ctrName := fmt.Sprintf("%s:%s", format.Pod(pod), container.Name)
	if probeSpec == nil {
		glog.Warningf("%s probe for %s is nil", probeType, ctrName)
		return results.Success, nil
	}

	result, output, err := pb.runProbeWithRetries(probeType, probeSpec, pod, status, container, containerID, maxProbeRetries)
	if err != nil || result != probe.Success {
		// Probe failed in one way or another.
		ref, hasRef := pb.refManager.GetRef(containerID)
		if !hasRef {
			glog.Warningf("No ref for container %q (%s)", containerID.String(), ctrName)
		}
		if err != nil {
			glog.V(1).Infof("%s probe for %q errored: %v", probeType, ctrName, err)
			if hasRef {
				pb.recorder.Eventf(ref, v1.EventTypeWarning, events.ContainerUnhealthy, "%s probe errored: %v", probeType, err)
			}
		} else { // result != probe.Success
			glog.V(1).Infof("%s probe for %q failed (%v): %s", probeType, ctrName, result, output)
			if hasRef {
				pb.recorder.Eventf(ref, v1.EventTypeWarning, events.ContainerUnhealthy, "%s probe failed: %s", probeType, output)
			}
		}
		return results.Failure, err
	}
	glog.V(3).Infof("%s probe for %q succeeded", probeType, ctrName)
	return results.Success, nil
}

// runProbeWithRetries tries to probe the container in a finite loop, it returns the last result
// if it never succeeds.
func (pb *prober) runProbeWithRetries(probeType probeType, p *v1.Probe, pod *v1.Pod, status v1.PodStatus, container v1.Container, containerID kubecontainer.ContainerID) (probe.Result, string, error) {
	var err error
	var result probe.Result
	var output string
	for i := 0; i < 3; i++ {
		result, output, err = pb.runProbe(probeType, p, pod, status, container, containerID)
		if err == nil {
			return result, output, nil
		}
	}
	return result, output, err
}

// buildHeaderMap takes a list of HTTPHeader <name, value> string
// pairs and returns a populated string->[]string http.Header map.
func buildHeader(headerList []v1.HTTPHeader) http.Header {
	headers := make(http.Header)
	for _, header := range headerList {
		headers[header.Name] = append(headers[header.Name], header.Value)
	}
	return headers
}

func (pb *prober) runProbe(p *v1.Probe, pod *v1.Pod, status v1.PodStatus, container v1.Container, containerID kubecontainer.ContainerID) (probe.Result, string, error) {
	timeout := time.Duration(p.TimeoutSeconds) * time.Second
	if p.Exec != nil {
		glog.V(4).Infof("Exec-Probe Pod: %v, Container: %v, Command: %v", pod, container, p.Exec.Command)
		command := kubecontainer.ExpandContainerCommandOnlyStatic(p.Exec.Command, container.Env)
		return pb.exec.Probe(pb.newExecInContainer(container, containerID, command, timeout))
	}
	if p.HTTPGet != nil {
		scheme := strings.ToLower(string(p.HTTPGet.Scheme))
		host := p.HTTPGet.Host
		if host == "" {
			host = status.PodIP
		}
		port, err := extractPort(p.HTTPGet.Port, container)
		if err != nil {
			return probe.Unknown, "", err
		}
		path := p.HTTPGet.Path
		glog.V(4).Infof("HTTP-Probe Host: %v://%v, Port: %v, Path: %v", scheme, host, port, path)
		url := formatURL(scheme, host, port, path)
		headers := buildHeader(p.HTTPGet.HTTPHeaders)
		glog.V(4).Infof("HTTP-Probe Headers: %v", headers)
		return pb.http.Probe(url, headers, timeout)
	}
	if p.TCPSocket != nil {
		port, err := extractPort(p.TCPSocket.Port, container)
		if err != nil {
			return probe.Unknown, "", err
		}
		host := p.TCPSocket.Host
		if host == "" {
			host = status.PodIP
		}
		glog.V(4).Infof("TCP-Probe Host: %v, Port: %v, Timeout: %v", host, port, timeout)
		return pb.tcp.Probe(host, port, timeout)
	}
	glog.Warningf("Failed to find probe builder for container: %v", container)
	return probe.Unknown, "", fmt.Errorf("Missing probe handler for %s:%s", format.Pod(pod), container.Name)
}

func extractPort(param intstr.IntOrString, container v1.Container) (int, error) {
	port := -1
	var err error
	switch param.Type {
	case intstr.Int:
		port = param.IntValue()
	case intstr.String:
		if port, err = findPortByName(container, param.StrVal); err != nil {
			// Last ditch effort - maybe it was an int stored as string?
			if port, err = strconv.Atoi(param.StrVal); err != nil {
				return port, err
			}
		}
	default:
		return port, fmt.Errorf("IntOrString had no kind: %+v", param)
	}
	if port > 0 && port < 65536 {
		return port, nil
	}
	return port, fmt.Errorf("invalid port number: %v", port)
}

// findPortByName is a helper function to look up a port in a container by name.
func findPortByName(container v1.Container, portName string) (int, error) {
	for _, port := range container.Ports {
		if port.Name == portName {
			return int(port.ContainerPort), nil
		}
	}
	return 0, fmt.Errorf("port %s not found", portName)
}

// formatURL formats a URL from args.  For testability.
func formatURL(scheme string, host string, port int, path string) *url.URL {
	u, err := url.Parse(path)
	// Something is busted with the path, but it's too late to reject it. Pass it along as is.
	if err != nil {
		u = &url.URL{
			Path: path,
		}
	}
	u.Scheme = scheme
	u.Host = net.JoinHostPort(host, strconv.Itoa(port))
	return u
}