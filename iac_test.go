package main

import (
	"strings"
	"testing"
)

// ── getHCLBlocks ──────────────────────────────────────────────

var hclSample = []string{
	`resource "aws_instance" "web" {`,
	`  ami           = "ami-12345"`,
	`  instance_type = "t2.micro"`,
	`}`,
	``,
	`resource "aws_s3_bucket" "data" {`,
	`  bucket = "my-bucket"`,
	`}`,
	``,
	`module "vpc" {`,
	`  source = "./modules/vpc"`,
	`}`,
	``,
	`variable "instance_type" {`,
	`  default = "t2.micro"`,
	`}`,
	``,
	`output "instance_ip" {`,
	`  value = aws_instance.web.public_ip`,
	`}`,
	``,
	`provider "aws" {`,
	`  region = "us-east-1"`,
	`}`,
	``,
	`locals {`,
	`  env = "prod"`,
	`}`,
	``,
	`data "aws_ami" "ubuntu" {`,
	`  most_recent = true`,
	`}`,
	``,
	`terraform {`,
	`  required_version = ">= 1.0"`,
	`}`,
}

func TestGetHCLBlocks_ResourceCount(t *testing.T) {
	entries := getHCLBlocks(hclSample)
	if len(entries) != 9 {
		t.Errorf("expected 9 blocks, got %d: %v", len(entries), blockNames(entries))
	}
}

func TestGetHCLBlocks_ResourceNames(t *testing.T) {
	entries := getHCLBlocks(hclSample)
	names := blockNames(entries)
	for _, want := range []string{
		`resource "aws_instance" "web"`,
		`resource "aws_s3_bucket" "data"`,
		`module "vpc"`,
		`variable "instance_type"`,
		`output "instance_ip"`,
		`provider "aws"`,
		`locals`,
		`data "aws_ami" "ubuntu"`,
		`terraform`,
	} {
		found := false
		for _, n := range names {
			if n == want {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("block %q not found in %v", want, names)
		}
	}
}

func TestGetHCLBlocks_LineRanges(t *testing.T) {
	entries := getHCLBlocks(hclSample)
	// First resource: lines 1-4
	if entries[0].start != 1 || entries[0].end != 4 {
		t.Errorf("resource aws_instance: start=%d end=%d, want 1-4", entries[0].start, entries[0].end)
	}
	// Second resource: lines 6-8
	if entries[1].start != 6 || entries[1].end != 8 {
		t.Errorf("resource aws_s3_bucket: start=%d end=%d, want 6-8", entries[1].start, entries[1].end)
	}
}

func TestGetHCLBlocks_KeyExtraction(t *testing.T) {
	entries := getHCLBlocks(hclSample)
	// resource "aws_instance" "web" → key should contain aws_instance and web
	key := entries[0].key
	if !strings.Contains(key, "aws_instance") {
		t.Errorf("key %q should contain 'aws_instance'", key)
	}
}

func TestGetHCLBlocks_SingleLineLocals(t *testing.T) {
	src := []string{
		`locals {}`,
		``,
		`resource "aws_vpc" "main" {`,
		`  cidr_block = "10.0.0.0/16"`,
		`}`,
	}
	entries := getHCLBlocks(src)
	if len(entries) != 2 {
		t.Errorf("expected 2 blocks (locals + resource), got %d", len(entries))
	}
	if entries[0].key != "locals" {
		t.Errorf("first block key = %q, want 'locals'", entries[0].key)
	}
}

func TestGetHCLBlocks_IgnoresNestedBlocks(t *testing.T) {
	src := []string{
		`resource "aws_security_group" "web" {`,
		`  ingress {`,
		`    from_port = 80`,
		`    to_port   = 80`,
		`  }`,
		`}`,
	}
	entries := getHCLBlocks(src)
	// Only the top-level resource, not the nested ingress block
	if len(entries) != 1 {
		t.Errorf("expected 1 top-level block, got %d", len(entries))
	}
	if entries[0].start != 1 || entries[0].end != 6 {
		t.Errorf("resource range: start=%d end=%d, want 1-6", entries[0].start, entries[0].end)
	}
}

func TestGetHCLBlocks_IgnoresComments(t *testing.T) {
	src := []string{
		`# This is a comment`,
		`// Also a comment`,
		`resource "aws_instance" "web" {`,
		`  # nested comment`,
		`  ami = "ami-123"`,
		`}`,
	}
	entries := getHCLBlocks(src)
	if len(entries) != 1 {
		t.Errorf("expected 1 block, got %d", len(entries))
	}
	if entries[0].start != 3 {
		t.Errorf("start = %d, want 3 (after comments)", entries[0].start)
	}
}

func TestGetHCLBlocks_Empty(t *testing.T) {
	entries := getHCLBlocks([]string{})
	if len(entries) != 0 {
		t.Errorf("expected 0 entries for empty input")
	}
}

func TestGetHCLBlocks_OnlyComments(t *testing.T) {
	entries := getHCLBlocks([]string{"# just a comment", "", "// another"})
	if len(entries) != 0 {
		t.Errorf("expected 0 entries for comment-only input")
	}
}

func TestGetHCLBlocks_MovedBlock(t *testing.T) {
	src := []string{
		`moved {`,
		`  from = aws_instance.web`,
		`  to   = aws_instance.app`,
		`}`,
	}
	entries := getHCLBlocks(src)
	if len(entries) != 1 || entries[0].key != "moved" {
		t.Errorf("moved block: %v", entries)
	}
}

func TestGetHCLBlocks_DataSource(t *testing.T) {
	src := []string{
		`data "aws_ami" "ubuntu" {`,
		`  most_recent = true`,
		`  owners      = ["099720109477"]`,
		`}`,
	}
	entries := getHCLBlocks(src)
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if !strings.Contains(entries[0].name, "data") {
		t.Errorf("name = %q, want to contain 'data'", entries[0].name)
	}
}

// ── resolveBlock with HCL ─────────────────────────────────────

func TestResolveBlock_HCL_Resource(t *testing.T) {
	s, e, err := resolveBlock(hclSample, "hcl", "aws_instance")
	if err != nil {
		t.Fatal(err)
	}
	if s != 1 || e != 4 {
		t.Errorf("aws_instance: start=%d end=%d, want 1-4", s, e)
	}
}

func TestResolveBlock_HCL_Module(t *testing.T) {
	s, _, err := resolveBlock(hclSample, "hcl", "vpc")
	if err != nil {
		t.Fatal(err)
	}
	if s != 10 {
		t.Errorf("module vpc start = %d, want 10", s)
	}
}

func TestResolveBlock_HCL_Ambiguous(t *testing.T) {
	// "aws" matches both provider "aws" and data "aws_ami" "ubuntu"
	_, _, err := resolveBlock(hclSample, "hcl", "aws")
	if err == nil {
		t.Error("expected ambiguity error for 'aws'")
	}
	if !strings.Contains(err.Error(), "matched") {
		t.Errorf("error = %q, want 'matched'", err.Error())
	}
}

func TestResolveBlock_TFAlias(t *testing.T) {
	// "tf" is an alias for "hcl"
	s, _, err := resolveBlock(hclSample, "tf", "aws_instance")
	if err != nil {
		t.Fatal(err)
	}
	if s != 1 {
		t.Errorf("tf alias: start=%d, want 1", s)
	}
}

func TestResolveBlock_TerraformAlias(t *testing.T) {
	s, _, err := resolveBlock(hclSample, "terraform", "aws_s3_bucket")
	if err != nil {
		t.Fatal(err)
	}
	if s != 6 {
		t.Errorf("terraform alias: start=%d, want 6", s)
	}
}

// ── getNixBlocks ──────────────────────────────────────────────

var nixSample = []string{
	`programs.git = {`,
	`  enable = true;`,
	`  userName = "Alice";`,
	`};`,
	``,
	`programs.ssh = {`,
	`  enable = true;`,
	`};`,
	``,
	`environment.systemPackages = [`,
	`  pkgs.vim`,
	`  pkgs.git`,
	`];`,
	``,
	`services.nginx = {`,
	`  enable = true;`,
	`  virtualHosts."example.com" = {`,
	`    root = "/var/www";`,
	`  };`,
	`};`,
}

func TestGetNixBlocks_Basic(t *testing.T) {
	entries := getNixBlocks(nixSample)
	if len(entries) < 3 {
		t.Errorf("expected at least 3 blocks, got %d: %v", len(entries), blockNames(entries))
	}
}

func TestGetNixBlocks_ProgramsGit(t *testing.T) {
	entries := getNixBlocks(nixSample)
	found := false
	for _, e := range entries {
		if e.key == "programs.git" {
			found = true
			if e.start != 1 {
				t.Errorf("programs.git start = %d, want 1", e.start)
			}
			break
		}
	}
	if !found {
		t.Errorf("programs.git not found in %v", blockNames(entries))
	}
}

func TestGetNixBlocks_ListBinding(t *testing.T) {
	// environment.systemPackages = [...] should be detected
	entries := getNixBlocks(nixSample)
	found := false
	for _, e := range entries {
		if strings.Contains(e.key, "systemPackages") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("systemPackages list binding not found in %v", blockNames(entries))
	}
}

func TestGetNixBlocks_Empty(t *testing.T) {
	entries := getNixBlocks([]string{})
	if len(entries) != 0 {
		t.Errorf("expected 0 entries for empty input")
	}
}

func TestGetNixBlocks_IgnoresIndented(t *testing.T) {
	src := []string{
		`top = {`,
		`  nested = {`,
		`    deep = true;`,
		`  };`,
		`};`,
	}
	entries := getNixBlocks(src)
	// Only "top" should be detected
	if len(entries) != 1 {
		t.Errorf("expected 1 top-level block, got %d: %v", len(entries), blockNames(entries))
	}
	if entries[0].key != "top" {
		t.Errorf("key = %q, want 'top'", entries[0].key)
	}
}

func TestGetNixBlocks_IgnoresComments(t *testing.T) {
	src := []string{
		`# This is a comment`,
		`programs.vim = {`,
		`  enable = true;`,
		`};`,
	}
	entries := getNixBlocks(src)
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].start != 2 {
		t.Errorf("start = %d, want 2", entries[0].start)
	}
}

// ── resolveBlock with Nix ─────────────────────────────────────

func TestResolveBlock_Nix_ProgramsGit(t *testing.T) {
	s, _, err := resolveBlock(nixSample, "nix", "programs.git")
	if err != nil {
		t.Fatal(err)
	}
	if s != 1 {
		t.Errorf("programs.git start = %d, want 1", s)
	}
}

func TestResolveBlock_Nix_NoMatch(t *testing.T) {
	_, _, err := resolveBlock(nixSample, "nix", "nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent block")
	}
}

// ── integration: move HCL resource ───────────────────────────

func TestHCLMove_ResourceReorder(t *testing.T) {
	src := []string{
		`resource "aws_instance" "web" {`,
		`  ami = "ami-123"`,
		`}`,
		``,
		`resource "aws_s3_bucket" "data" {`,
		`  bucket = "my-data"`,
		`}`,
	}
	// Move aws_s3_bucket before aws_instance
	srcStart, srcEnd, err := resolveBlock(src, "hcl", "aws_s3_bucket")
	if err != nil {
		t.Fatal(err)
	}
	destStart, _, err := resolveBlock(src, "hcl", "aws_instance")
	if err != nil {
		t.Fatal(err)
	}
	result, _, err := execMove(src, srcStart, srcEnd, destStart-1, destStart, 1)
	if err != nil {
		t.Fatal(err)
	}
	// aws_s3_bucket should now come first
	if !strings.Contains(result[0], "aws_s3_bucket") {
		t.Errorf("result[0] = %q, want aws_s3_bucket", result[0])
	}
}

func TestHCLCopy_DuplicateResource(t *testing.T) {
	src := []string{
		`resource "aws_instance" "web" {`,
		`  ami = "ami-123"`,
		`}`,
	}
	srcStart, srcEnd, err := resolveBlock(src, "hcl", "aws_instance")
	if err != nil {
		t.Fatal(err)
	}
	result, _, err := execCopy(src, srcStart, srcEnd, srcEnd, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 6 {
		t.Errorf("expected 6 lines (3+3), got %d", len(result))
	}
}

// ── getTopLevelBlocks router ──────────────────────────────────

func TestGetTopLevelBlocks_HCLRouted(t *testing.T) {
	_, err := getTopLevelBlocks(hclSample, "hcl")
	if err != nil {
		t.Errorf("hcl: unexpected error: %v", err)
	}
}

func TestGetTopLevelBlocks_TFRouted(t *testing.T) {
	_, err := getTopLevelBlocks(hclSample, "tf")
	if err != nil {
		t.Errorf("tf: unexpected error: %v", err)
	}
}

func TestGetTopLevelBlocks_TerraformRouted(t *testing.T) {
	_, err := getTopLevelBlocks(hclSample, "terraform")
	if err != nil {
		t.Errorf("terraform: unexpected error: %v", err)
	}
}

func TestGetTopLevelBlocks_NixRouted(t *testing.T) {
	_, err := getTopLevelBlocks(nixSample, "nix")
	if err != nil {
		t.Errorf("nix: unexpected error: %v", err)
	}
}

// ── helpers ───────────────────────────────────────────────────

func blockNames(entries []blockEntry) []string {
	names := make([]string, len(entries))
	for i, e := range entries {
		names[i] = e.name
	}
	return names
}
