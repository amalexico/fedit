resource "aws_instance" "web" {
  ami           = "ami-0abcdef1234567890"
  instance_type = "t3.micro"
}

resource "aws_s3_bucket" "data" {
  bucket = "amalex-data-bucket"
}

variable "region" {
  type    = string
  default = "us-east-1"
}
