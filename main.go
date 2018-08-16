package main

import (
	"errors"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ecs"

	"github.com/aws/aws-sdk-go/aws"
	// "github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
)

const (
	memWithinBounds         = "Will not scale in. MemoryReservation is within bounds."
	clusterSizeWithinBounds = "Will not scale in. Cluster size within bounds."
)

var (
	// AWS Session
	sess = session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))
)

func main() {
	lambda.Start(Handler)
}

// Handler wraps ScaleInECS for Lambda
func Handler() {
	scaleInECS(getEnv("CLUSTER", "dev"), getEnv("DESIRED_MEMORY_RESERVATION", "80"), getEnv("MIN_CLUSTER_SIZE", "1"))
}

func scaleInECS(cluster string, maxMemory string, minClusterSize string) (s string, err error) {

	// init the ECS Service
	svc := ecs.New(sess)

	maxMemoryInt, err := strconv.Atoi(getEnv("DESIRED_MEMORY_RESERVATION", maxMemory))
	if err != nil {
		return
	}

	minClusterSizeInt, err := strconv.Atoi(getEnv("MIN_CLUSTER_SIZE", minClusterSize))
	if err != nil {
		return
	}

	currentMem, err := getCurrentMemoryReservation(cluster)
	if err != nil {
		return
	}

	log.Printf(`Reviewing %v ECS Cluster's size. 
      Current memory reservation: %v
      Desired memory reservation: %v`, cluster, currentMem, maxMemoryInt)

	// check if min cluster memory reservation
	if currentMem > maxMemoryInt {
		log.Print(memWithinBounds)
		return
	}

	// check if min cluster memory reservation
	if currentMem > (maxMemoryInt - 10) {
		log.Print(memWithinBounds)
		return
	}

	containerInstanceList, err := svc.ListContainerInstances(&ecs.ListContainerInstancesInput{
		Cluster: aws.String(cluster),
	})
	if err != nil {
		return
	}

	if len(containerInstanceList.ContainerInstanceArns) <= minClusterSizeInt {
		log.Print(clusterSizeWithinBounds)
		return
	}

	o, err := svc.DescribeContainerInstances(&ecs.DescribeContainerInstancesInput{
		Cluster:            aws.String(cluster),
		ContainerInstances: containerInstanceList.ContainerInstanceArns,
	})

	if err != nil {
		return
	}

	var ec2InstanceIds []string

	for _, instance := range o.ContainerInstances {
		log.Printf("Container Instance %v has %v pending tasks and %v running tasks", *instance.Ec2InstanceId, *instance.PendingTasksCount, *instance.RunningTasksCount)
		if *instance.PendingTasksCount == 0 && *instance.RunningTasksCount == 0 {
			ec2InstanceIds = append(ec2InstanceIds, *instance.Ec2InstanceId)

		}
	}

	if len(ec2InstanceIds) > 0 {
		terminateContainerInstances(ec2InstanceIds)
	}

	return
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func getCurrentMemoryReservation(cluster string) (i int, err error) {

	// init the CloudWatch Service
	svc := cloudwatch.New(sess)
	endTime := time.Now()
	startTime := endTime.Add(time.Duration(-10 * time.Minute))
	data, err := svc.GetMetricData(&cloudwatch.GetMetricDataInput{
		StartTime:     aws.Time(startTime),
		EndTime:       aws.Time(endTime),
		MaxDatapoints: aws.Int64(1),
		MetricDataQueries: []*cloudwatch.MetricDataQuery{
			&cloudwatch.MetricDataQuery{
				Id:         aws.String("memres"),
				ReturnData: aws.Bool(true),
				MetricStat: &cloudwatch.MetricStat{
					Period: aws.Int64(300),
					Stat:   aws.String("Average"),
					Metric: &cloudwatch.Metric{
						Namespace:  aws.String("AWS/ECS"),
						MetricName: aws.String("MemoryReservation"),
						Dimensions: []*cloudwatch.Dimension{
							&cloudwatch.Dimension{
								Name:  aws.String("ClusterName"),
								Value: aws.String(cluster),
							},
						},
					},
				},
			},
		},
	})

	if err != nil {
		return
	}

	if len(data.MetricDataResults) > 0 {
		i = int(*data.MetricDataResults[0].Values[0])
	} else {
		err = errors.New("unable to get current memory reservation")
	}
	return
}

func terminateContainerInstances(instanceIds []string) {
	// init the EC2 Service
	svc := ec2.New(sess)

	// Terminate half of the instances with nothing running on them
	// Each run reduces unused instances by 50% until they're gone
	half := (len(instanceIds) + 1) / 2
	log.Printf("Terminating Container Instances %v", instanceIds[0:half])

	_, err := svc.TerminateInstances(&ec2.TerminateInstancesInput{
		InstanceIds: aws.StringSlice(instanceIds[0:half]),
	})
	if err != nil {
		log.Fatal(err)
	}
}
