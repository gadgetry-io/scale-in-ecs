resource "aws_iam_role" "ecs_autoscale_in" {
  name = "ecs-autoscale-in"

  assume_role_policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Action": "sts:AssumeRole",
      "Principal": {
        "Service": "lambda.amazonaws.com"
      },
      "Effect": "Allow",
      "Sid": ""
    }
  ]
}
EOF
}

resource "aws_iam_role_policy" "ecs_autoscale_in" {
  name = "ecs_autoscale_in"
  role = "${aws_iam_role.ecs_autoscale_in.id}"

  policy = <<EOF
{
   "Version":"2012-10-17",
   "Statement":[
      {
         "Sid":"",
         "Effect":"Allow",
         "Action":[
            "cloudwatch:GetMetricData",
            "ec2:TerminateInstances",
            "ec2:TerminateInstances",
            "ecs:ListContainerInstances",
            "ecs:DescribeContainerInstances"
         ],
         "Resource":"*"
      }
   ]
}
EOF
}
