package main

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strings"

	dc "github.com/samalba/dockerclient"

	"net/http"
)

func randString(n int) string {
	const alphanum = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	var bytes = make([]byte, n)
	rand.Read(bytes)
	for i, b := range bytes {
		a := alphanum[b%byte(len(alphanum))]
		fmt.Println(a)
		bytes[i] = a
	}
	return string(bytes)
}

func main() {
	fmt.Println("Devices runner")

	// Init the client
	docker, err := dc.NewDockerClient("unix:///var/run/docker.sock", nil)
	if err != nil {
		panic(err)
	}

	containers, err := docker.ListContainers(true)
	if err != nil {
		panic(err)
	}

	for _, c := range containers {
		fmt.Println("container: ", c.Names, " status: ", c.Status)
	}

	// let's create an API for starting devices

	http.HandleFunc("/device", func(w http.ResponseWriter, r *http.Request) { devicesHandler(w, r, docker) })
	http.HandleFunc("/device/", func(w http.ResponseWriter, r *http.Request) { deviceHandler(w, r, docker) })

	http.ListenAndServe(":8080", nil)
}
func deviceHandler(w http.ResponseWriter, r *http.Request, docker *dc.DockerClient) {
	path := r.URL.Path

	switch r.Method {
	case "DELETE":
		fmt.Println("DELETE")
		deviceName := path[len("/devices"):]
		fmt.Println("deletion of", deviceName, "requested")
		err := stopAndDeleteDevice(deviceName, docker)
		if err != nil {
			fmt.Println("Error deleting the container:", err)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
		} else {
			w.WriteHeader(http.StatusNoContent)
		}
	}
}

func devicesHandler(w http.ResponseWriter, r *http.Request, docker *dc.DockerClient) {
	path := r.URL.Path

	switch r.Method {
	case "POST":
		fmt.Println("POST")
		// request of a new device
		var deviceRq struct {
			Type   string
			SN     string
			Server string
		}

		defer r.Body.Close()
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			panic(err)
		}

		err = json.Unmarshal(body, &deviceRq)
		if err != nil {
			w.Write([]byte(err.Error()))
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if deviceRq.SN == "" {
			deviceRq.SN = randString(6)
		}
		fmt.Println("Start the device", deviceRq.Type, "with S/N", deviceRq.SN, "Versus server", deviceRq.Server)

		err = startDevice(deviceRq.Type, deviceRq.SN, deviceRq.Server, docker)
		if err != nil {
			fmt.Println("Error starting the container:", err)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
		} else {
			w.WriteHeader(http.StatusCreated)
			w.Write([]byte(deviceRq.SN))
		}
	case "GET":
		fmt.Println("GET")
		devices := listDevices(docker)
		body, err := json.Marshal(&devices)
		if err != nil {
			panic(err)
		}
		w.Write(body)
	case "DELETE":
		fmt.Println("DELETE")
		deviceName := path[len("/device"):]
		fmt.Println("deletion of", deviceName, "requested")
		err := stopAndDeleteDevice(deviceName, docker)
		if err != nil {
			fmt.Println("Error deleting the container:", err)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
		} else {
			w.WriteHeader(http.StatusNoContent)
		}
	}
	fmt.Println("start device API: ", path)
}

func startDevice(devType string, sn string, server string, docker *dc.DockerClient) error {
	// try to start the associated docker container
	var cfg dc.ContainerConfig
	cfg.Image = "device-" + devType

	cfg.Env = []string{"DEVICE_SN=" + sn, "DEVICE_SERVER=" + server}

	cfg.Tty = true

	cfg.AttachStdout = true
	cfg.AttachStdin = false
	cfg.AttachStderr = true

	id, err := docker.CreateContainer(&cfg, "device"+sn)

	if err != nil {
		return err
	}

	var host dc.HostConfig

	err = docker.StartContainer(id, &host)
	return err
}

func stopAndDeleteDevice(deviceSn string, docker *dc.DockerClient) error {
	container := "device" + deviceSn
	err := docker.KillContainer(container)
	if err != nil {
		return err
	}

	err = docker.RemoveContainer(container)
	return nil
}

func listDevices(docker *dc.DockerClient) []string {
	containers, err := docker.ListContainers(false)
	if err != nil {
		panic(err)
	}

	res := make([]string, 0, 10)

	for _, c := range containers {
		name := arrayToStr(c.Names)
		if strings.HasPrefix(name, "/device") {
			// found a device container
			res = append(res, name[len("/device"):])
		}
	}

	return res
}

func arrayToStr(src []string) string {
	res := ""
	for _, s := range src {
		res += s
	}
	return res
}
