package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/ecs"
)

var (
	// AWS Session
	sess = session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))
)

func main() {
	if isLambda() {
		lambda.Start(Handler)
	} else {
		Handler()
	}
}

// Handler wraps ScaleInECS for Lambda
func Handler() {
	scaleInECS(getEnv("CLUSTER", "dev"))
}

func scaleInECS(cluster string) (s string, err error) {

	// init the ECS Service
	svc := ecs.New(sess)

	containerInstanceList, err := svc.ListContainerInstances(&ecs.ListContainerInstancesInput{
		Cluster: aws.String(cluster),
	})

	if err != nil {
		return
	}

	o, err := svc.DescribeContainerInstances(&ecs.DescribeContainerInstancesInput{
		Cluster:            aws.String(cluster),
		ContainerInstances: containerInstanceList.ContainerInstanceArns,
	})

	if err != nil {
		return
	}

	for _, instance := range o.ContainerInstances {
		log.Printf("Container Instance %v has %v pending tasks and %v running tasks", *instance.Ec2InstanceId, *instance.PendingTasksCount, *instance.RunningTasksCount)
		if *instance.PendingTasksCount == 0 && *instance.RunningTasksCount == 0 {
			setContainerInstanceScaleInProtection(instance, false)
		} else {
			setContainerInstanceScaleInProtection(instance, true)
		}
	}

	return
}

func setContainerInstanceScaleInProtection(instance *ecs.ContainerInstance, protectionEnabled bool) {

	svc := autoscaling.New(sess)
	res, err := svc.DescribeAutoScalingInstances(&autoscaling.DescribeAutoScalingInstancesInput{
		InstanceIds: []*string{instance.Ec2InstanceId},
	})
	if err != nil {
		fmt.Println(err)
		return
	}
	for _, autoscalingInstance := range res.AutoScalingInstances {
		if *autoscalingInstance.ProtectedFromScaleIn != protectionEnabled {
			fmt.Printf("Setting protection for Container Instance %s: %t\n", *instance.Ec2InstanceId, protectionEnabled)
			_, err := svc.SetInstanceProtection(&autoscaling.SetInstanceProtectionInput{
				AutoScalingGroupName: autoscalingInstance.AutoScalingGroupName,
				InstanceIds:          []*string{instance.Ec2InstanceId},
				ProtectedFromScaleIn: aws.Bool(protectionEnabled),
			})
			if err != nil {
				fmt.Println(err)
			}
		}
	}
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func isLambda() bool {
	executionEnvironment := getEnv("AWS_EXECUTION_ENV", "false")
	return strings.Contains(executionEnvironment, "AWS_Lambda_")
}
