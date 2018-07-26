// create schedule
resource "aws_cloudwatch_event_rule" "ecs_autoscale_in" {
  name                = "${aws_lambda_function.ecs_autoscale_in.function_name}"
  schedule_expression = "rate(5 minutes)"
}
