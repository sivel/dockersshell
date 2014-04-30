// Copyright 2014 Matt Martz <matt@sivel.net>
// All Rights Reserved.
//
//    Licensed under the Apache License, Version 2.0 (the "License"); you may
//    not use this file except in compliance with the License. You may obtain
//    a copy of the License at
//
//         http://www.apache.org/licenses/LICENSE-2.0
//
//    Unless required by applicable law or agreed to in writing, software
//    distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
//    WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the
//    License for the specific language governing permissions and limitations
//    under the License.

package main

import (
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/fsouza/go-dockerclient"
	"launchpad.net/goyaml"
)

type Config struct {
	Endpoints []string `yaml:"endpoints,omitempty"`
	Image     string   `yaml:"image,omitempty"`
	User      string   `yaml:"user,omitempty"`
}

func getconfig() *Config {
	var config Config

	defaults := []byte("endpoints: ['http://127.0.0.1:4243']\nimage: ssh\nuser: ubuntu")

	text, err := ioutil.ReadFile("/etc/dockersshell.yaml")
	if err != nil {
		goyaml.Unmarshal([]byte(defaults), &config)
	} else {
		goyaml.Unmarshal(text, &config)
	}

	return &config
}

func connect(user string, host string, port string) {
	cmd := exec.Command("ssh", "-q", "-p", port, "-l", user, host)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		fmt.Printf("Unable to initiate ssh connection: %s\n", err)
		os.Exit(2)
	}
}

func main() {
	var Endpoint string
	Smallest := 1024
	user := os.Getenv("USER")
	os.Setenv("DSSHUSER", user)
	stamp := strconv.FormatInt(time.Now().Unix(), 10)
	name := fmt.Sprintf("%s-%s", user, stamp)

	config := getconfig()

	listOptions := docker.ListContainersOptions{
		All:    false,
		Size:   false,
		Limit:  -1,
		Since:  "",
		Before: "",
	}
	for _, endpoint := range config.Endpoints {
		client, err := docker.NewClient(endpoint)
		if err != nil {
			continue
		}

		containers, err := client.ListContainers(listOptions)
		if err != nil {
			continue
		}

		if len(containers) == 0 {
			Endpoint = endpoint
			break
		} else if len(containers) < Smallest {
			Endpoint = endpoint
			Smallest = len(containers)
		}
	}

	if Endpoint == "" {
		fmt.Println("No acceptable endpoints found")
		os.Exit(2)
	}

	Url, err := url.Parse(Endpoint)
	if err != nil {
		fmt.Printf("Unable to parse endpoint URL: %s\n", err)
		os.Exit(2)
	} else if Url.Host == "" {
		fmt.Println("No host found in endpoint")
	}

	hostPort := strings.SplitN(Url.Host, ":", 2)

	client, err := docker.NewClient(Endpoint)
	if err != nil {
		fmt.Printf("Unable to communicate: %s\n", err)
		os.Exit(2)
	}

	dockerConfig := docker.Config{Image: config.Image}
	opts := docker.CreateContainerOptions{Name: name, Config: &dockerConfig}
	container, err := client.CreateContainer(opts)
	if err != nil {
		fmt.Printf("Unable to create container: %s\n", err)
		os.Exit(2)
	}

	host := docker.HostConfig{PublishAllPorts: true}
	if client.StartContainer(container.ID, &host) != nil {
		fmt.Printf("Unable to start container: %s\n", err)
		os.Exit(2)
	}

	inspect, err := client.InspectContainer(container.ID)
	if err != nil {
		fmt.Printf("Unable to get port information for container: %s\n", err)
	}
	port := inspect.NetworkSettings.Ports["22/tcp"][0].HostPort

	connect(config.User, hostPort[0], port)

	if client.StopContainer(container.ID, 0) != nil {
		fmt.Printf("Unable to stop container: %s\n", err)
		os.Exit(2)
	}

	remove := docker.RemoveContainerOptions{ID: container.ID, RemoveVolumes: false}
	if client.RemoveContainer(remove) != nil {
		fmt.Printf("Unable to remove container: %s\n", err)
		os.Exit(2)
	}

	os.Exit(0)
}
