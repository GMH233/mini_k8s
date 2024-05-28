package main

import (
	"context"
	"fmt"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
)

func main() {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return
	}
	defer cli.Close()
	summaries, err := cli.ImageList(context.Background(), image.ListOptions{})
	if err != nil {
		return
	}
	for _, summary := range summaries {
		fmt.Println("ID:", summary.ID)
		for _, tag := range summary.RepoTags {
			fmt.Println(tag)
		}
	}
}
