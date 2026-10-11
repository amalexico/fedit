resource "test" "alpha" {
  value = "}"
  other = "{{"
}

resource "test" "beta" {
  value = "}}}"
}

resource "test" "gamma" {
  value = "x"
}
