resource "aws_instance" "web1" {
  ami           = "ami-0abcdef1234567890"
  instance_type = "t3.micro"
}

resource "aws_s3_bucket" "data1" {
  bucket = "amalex-data-bucket"
}

variable "region1" {
  type    = string
  default = "us-east-1"
}
