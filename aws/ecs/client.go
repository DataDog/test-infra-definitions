package ecs

import (
	"context"
	"errors"
	"fmt"

	awsConfig "github.com/aws/aws-sdk-go-v2/config"
	awsECS "github.com/aws/aws-sdk-go-v2/service/ecs"
)

type Client struct {
	awsECS.Client

	ctx context.Context
}

func NewECSClient(ctx context.Context, region string) (*Client, error) {
	cfg, err := awsConfig.LoadDefaultConfig(ctx,
		awsConfig.WithRegion(region),
	)
	if err != nil {
		return nil, err
	}
	return &Client{Client: *awsECS.NewFromConfig(cfg), ctx: ctx}, nil
}

func (c *Client) GetTaskPrivateIP(clusterArn, serviceName string) (string, error) {
	taskArn, err := c.getTaskArn(clusterArn, serviceName)
	if err != nil {
		return "", err
	}
	return c.getTaskPrivateIP(clusterArn, taskArn)
}

func (c *Client) getTaskArn(clusterArn, serviceName string) (string, error) {
	taskList, err := c.ListTasks(c.ctx, &awsECS.ListTasksInput{
		Cluster:     &clusterArn,
		ServiceName: &serviceName,
	})
	if err != nil {
		return "", err
	}
	if len(taskList.TaskArns) < 1 {
		return "", errors.New("no task arn found")
	}
	return taskList.TaskArns[0], nil
}

func (c *Client) getTaskPrivateIP(clusterArn string, taskArn string) (string, error) {
	taskOutput, err := c.DescribeTasks(c.ctx, &awsECS.DescribeTasksInput{
		Cluster: &clusterArn,
		Tasks:   []string{taskArn},
	})
	if err != nil {
		return "", err
	}
	if len(taskOutput.Tasks) < 1 {
		return "", fmt.Errorf("no task found on cluster %s with arn %s", clusterArn, taskArn)
	}
	if len(taskOutput.Tasks[0].Containers) < 1 {
		return "", fmt.Errorf("no container found on cluster %s with arn %s", clusterArn, taskArn)
	}
	if len(taskOutput.Tasks[0].Containers[0].NetworkInterfaces) < 1 {
		return "", fmt.Errorf("no network interface found on cluster %s with arn %s", clusterArn, taskArn)
	}
	return *taskOutput.Tasks[0].Containers[0].NetworkInterfaces[0].PrivateIpv4Address, nil
}
