connection "chaos6" {
    plugin = "chaos"
    regions = ["us-east-1", "us-west-2"]
    options "connection" {
      cache = true
      cache_ttl = 300
    }
}