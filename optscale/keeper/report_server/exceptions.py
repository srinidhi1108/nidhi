import enum


class Err(enum.Enum):
    OK0002 = [
        "Forbidden"
    ]
    OK0003 = [
        "Unauthorized"
    ]
    OK0004 = [
        "%s with id %s was not found"
    ]
    OK0005 = [
        "This resource requires authorization"
    ]
    OK0006 = [
        "Bad secret"
    ]
    OK0007 = [
        "Payload is malformed"
    ]
    OK0008 = [
        "Payload is not a valid json"
    ]
    OK0009 = [
        "Customers isn't provided correctly"
    ]
    OK0015 = [
        "Limit should be positive int"
    ]
    OK0018 = [
        "Event %s not ACK-able or already ACK-ed"
    ]
    OK0022 = [
        "%s method not allowed"
    ]
    OK0023 = [
        "Not found"
    ]
    OK0025 = [
        "Payload is required"
    ]
    OK0026 = [
        "Incorrect request body received"
    ]
    OK0027 = [
        "%s should be a list"
    ]
    OK0028 = [
        "Timestamp should be positive int"
    ]
    OK0030 = [
        "Unexpected arguments: %s"
    ]
    OK0032 = [
        "\"%s\" should be a string with valid JSON"
    ]
    OK0035 = [
        "%s is not provided"
    ]
    OK0036 = [
        "%s should be integer"
    ]
    OK0037 = [
        "Value of %s should be between %s and %s"
    ]
    OK0038 = [
        "Incorrect request body received"
    ]
    OK0040 = [
        "Unexpected parameter: %s"
    ]
    OK0041 = [
        "Parameter \"%s\" is immutable"
    ]
    OK0044 = [
        "Error validating fields: %s"
    ]
    OK0045 = [
        "Invalid %s"
    ]
    OK0046 = [
        "%s should be true or false"
    ]
