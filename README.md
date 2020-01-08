# Screenshot As A Service

Screenshot as a service is a simple service powered by the Chromium Project and Go. It enables capturing screenshots through a REST interface.

## Installation

The service can be used as a stand alone binary or as a docker image.

#### Binary
Download the latest binary from the releases section. You would need to download `Chromium` separately for the service to work.
#### Docker
`docker pull ajitemsahasrabuddhe/screenshot-as-a-service:<version>`

While you can use the latest tag as a version, it is recommended to pin to a version like `v1.1.2` or `v1.1.3` for guaranteed stability. The docker image comes with all pre-requisites installed out of the box.

## Usage

Capturing a screenshot is as simple as:

`$ curl http://localhost:3000/?url=http://google.com > screenshot.png`

The complete API is as follows:

```
# Take a screenshot
GET /?url=http://google.com
## Returns a 1920x1080 PNG screenshot of google.com

# Custom dimensions
GET /?url=http://google.com&width=1366&height=768
## Returns a 1366x768 PNG screenshot of google.com

# Take a full page screenshot
GET /?url=http://google.com&fullpage=true
## Returns a full page PNG screenshot of google.com

# Take a clipped screenshot
GET /?url=http://google.com&top=572&left=50&width=800&height=600
## Returns a 800x600 PNG screenshot of google.com clipped at {"top": 572, "left": 50, "width": 800, "height": 600}

# Custom useragent
GET /?url=http://google.com&useragent=abc
## Returns a screenshot using customized useragent

# Delay
GET /?url=http://google.com&delay=5000
## Returns a 1920x1080 PNG screenshot of google.com after a delay of 5 seconds after the site is loaded

# HTTP Authentication (Basic)
GET /?url=http://www.example.com&username=user&password=pass
## Returns a screenshot of a website that requires HTTP (Basic) Authentication
```