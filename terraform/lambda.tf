resource "aws_lambda_function" "ecs_autoscale_in" {
  function_name    = "${terraform.workspace}-scale-in-ecs"
  s3_bucket        = "${var.s3_bucket}"
  s3_key           = "${var.s3_key}"
  role             = "${aws_iam_role.ecs_autoscale_in.arn}"
  handler          = "scale-in-ecs"
  source_code_hash = "${var.source_code_hash}"
  runtime          = "go1.x"

  lifecycle {
    ignore_changes = ["last_modified"]
  }
}

variable "s3_bucket" {
  type = "string"
}

variable "s3_key" {
  type = "string"
}

variable "source_code_hash" {
  type    = "string"
  default = ""
}
