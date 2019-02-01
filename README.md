# vpcflow-diffd #

**A service which compares VPC flow log graphs.**

## Overview ##

AWS VPC Flow Logs are a data source by which a team can detect anomalies in
connection patterns, use of non-standard ports, or even view the interconnections of
systems. To assist in the consumption and analysis of these logs, vpcflow-diffd
provides APIs for fetching graph diffs of AWS VPC flow logs graphs that are generated
by vpcflow-diffd.

Graphs are DOT renditions of a digest, or fixed window of time, and are specified
with a `start` and `stop`. See
[vpcflow-grapherd](https://bitbucket.org/atlassian/vpcflow-grapherd/src) for more
information.

This project has two major components: an API to create and fetch diffs, and a worker
which performs the work for creating the diff This allows for multiple setups
depending on your use case. For example, for the simplest setup, this project can run
as a standalone service if `STREAM_APPLIANCE_ENDPOINT` is set to `127.0.0.1:<PORT>`.
Another, more asynchronous setup would involve running vpcflow-diffd as two services,
with the API component producing to some event bus, and configuring the event bus to
POST into the worker component.

## Modules ##

The service struct in the diffd package contains the modules used by this
application. If none of these modules are configured, the built-in modules will be
used.

```
func main() {
    ...

    // Service created with default modules
    service := &diffd.Service{
        Middleware: middleware,
    }

    ...
}
```

### Storage ###

This module is responsible for storing and retrieving the diff graphs. The built-in
storage module uses S3 as the store and can be configured with the
`DIFF_STORAGE_BUCKET` and `DIFF_STORAGE_BUCKET_REGION` environment variables. To
use a custom storage module, implement the `domain.Storage` interface and set the
Storage attribute on the `diffd.Service` struct in your `main.go`.

### Marker ###

As previously described, the project components can be configured to run
asynchronously. The Marker module is used to mark when a graph is in progress of being
created and when a graph is complete. The built-in Marker uses S3 as its backend and
can be configured with the `DIFF_PROGRESS_BUCKET` and `DIFF_PROGRESS_BUCKET_REGION`
environment variables. To use a custom marker module, implement the `domain.Marker`
interface and set the Marker attribute on the `diffd.Service` struct in your
`main.go`.

### Queuer ###

This module is responsible for queuing diff jobs which will eventually be consumed
by the Produce handler. The built-in Queuer POSTs to an HTTP endpoint. It can be
configured with the `STREAM_APPLIANCE_ENDPOINT` environment variable. This project
can be configured to run asynchronously if the queuer POSTs to some event bus and
returns immediately, so long as a 200 response from that event bus indicates that
the diff job will eventually be POSTed to the worker component of the project. To
use a custom queuer module, implement the `domain.Queuer` interface and set the Queuer
attribute on the `diffd.Service` struct in your `main.go`.

### Grapher ###

This module is responsible for creating and fetching VPC Flow Log graphs. The
built-in Grapher can be configured by configuring `GRAPHER_ENDPOINT` to point to a
running intance of
[vpcflow-grapherd](https://bitbucket.org/atlassian/vpcflow-grapherd/src). It will
create two graphs and poll the grapher on an interval specified by
`GRAPHER_POLLING_INTERVAL`, and will continue to poll until
`GRAPHER_POLLING_TIMEOUT` is reached.

### HTTP Clients ###

There are two clients used in this project. One is the client to be used with the
default Queuer module. The other is used with the default Grapher module. If no
clients are provided, a default will be used. This project makes use of the
[transport](https://bitbucket.org/atlassian/transport) library which provides a thin
layer of configuration on top of the `http.Client` from the standard lib. While the
HTTP client that is built-in to this project will be sufficient for most uses cases,
a custom one can be provided by setting the QueuerHTTPClient and GrapherHTTPClient
attributes on the `diffd.Service` struct in your `main.go`.


### Logging ###

This project uses [logevent](https://bitbucket.org/atlassian/logevent) as its logging
interface. Structured logs that this project emits can be found in the `logs`
package. This project comes with a couple of default logging implementations that can
be found in the plugins package. These loggers are injected via HTTP middleware on
the request context.

```
func main() {
    router := chi.NewRouter()
    middleware := []func(http.Handler) http.Handler{
        plugins.DefaultLogMiddleware(), // injects a logger which sends to os.Stdout
    }
    service := &diffd.Service{Middleware: middleware}
    if err := service.BindRoutes(router); err != nil {
        panic(err.Error())
    }
}
```
Please note that this project will not run without some sort of logger being
installed. While it's not recommended, if you wish to omit logging, use the
`NopLogMiddleware`.

### Stats ###

This project uses [xstats](https://github.com/rs/xstats) as the stats client. It
supports a decent range of backends. The default stats backend for the project is
statsd using the datadog tagging extensions. The default backend will send stats to
"localhost:8126". To change the destination or the backend install the
`CustomStatMiddleware` with your own xstats client.

### ExitSignals ###

Exit signals in this project are used to signal the service to perform a graceful
shutdown. The built-in exit signal listens for SIGTERM and SIGINT and signals to the
main routine to shutdown the service.

## Setup ##

* configure and deploy
  [vpcflow-grapherd](https://bitbucket.org/atlassian/vpcflow-grapherd/src)
* create a bucket in AWS to store the created diffs
* create a bucket in AWS to store progress states for queued diffs
* setup environment variables

| Name                        | Required | Description                                                                                                                     | Example                                         |
|-----------------------------|:--------:|---------------------------------------------------------------------------------------------------------------------------------|-------------------------------------------------|
| PORT                        |    No   | HTTP Port for application (defaults to 8080)                                                                                     | 8080                                            |
| DIFF\_STORAGE\_BUCKET       |    Yes   | The name of the S3 bucket used to store graphs                                                                                 | vpc-flow-diffs                                |
| DIFF\_STORAGE\_BUCKET\_REGION       |    Yes   | The region of the S3 bucket used to store graphs                                                                                 | us-west-2                                |
| DIFF\_PROGRESS\_BUCKET      |    Yes   | The name of the S3 bucket used to store graph progress states                                                                  | vpc-flow-diffs-progress                       |
| DIFF\_PROGRESS\_TIMEOUT      |    Yes   | The time in milliseconds after which a progress marker is considered invalid        | 100000                       |
| DIFF\_PROGRESS\_BUCKET\_REGION      |    Yes   | The region of the S3 bucket used to store graph progress states            | us-west-2                       |
| GRAPHER\_ENDPOINT               |    Yes   | Endpoint to vpcflow-grapherd api                                                                                     | http://ec2-grapherd.us-west-2.compute.amazonaws.com                                            |
| GRAPHER\_POLLING\_INTERVAL               |    Yes   | Amount of time to wait in between poll attempts in milliseconds                                                                                     | 1000                                            |
| GRAPHER\_POLLING\_TIMEOUT               |    Yes   | Amount of total time to continue polling the grapher in milliseconds. If you wish to poll indefinitely, set to -1.                                                                      | 10000                                            |
| STREAM\_APPLIANCE\_ENDPOINT   |    Yes   | Endpoint for the service which queues graphs to be created. | http://ec2-event-bus.us-west-2.compute.amazonaws.com |
| USE\_IAM                     |    Yes   | true or false. Set this flag to true if your application will be assuming an IAM role to read and write to the S3 buckets. This is recommended if you are deploying your application to an ec2 instance.       | true                                            |
| AWS\_CREDENTIALS\_FILE        |    No    | If not using IAM, use this to specify a credential file                                                                         | ~/.aws/credentials                              |
| AWS\_CREDENTIALS\_PROFILE     |    No    | If not using IAM, use this to specify the credentials profile to use                                                            | default                                         |
| AWS\_ACCESS\_KEY\_ID           |    No    | If not using IAM, use this to specify an AWS access key ID                                                                      |                                                 |
| AWS\_SECRET\_ACCESS\_KEY       |    No    | If not using IAM, use this to specify an AWS secret key                                                                         |                                                 |




## Contributing ##

### License ###

This project is licensed under Apache 2.0. See LICENSE.txt for details.

### Contributing Agreement ###

Atlassian requires signing a contributor's agreement before we can accept a patch. If
you are an individual you can fill out the [individual
CLA](https://na2.docusign.net/Member/PowerFormSigning.aspx?PowerFormId=3f94fbdc-2fbe-46ac-b14c-5d152700ae5d).
If you are contributing on behalf of your company then please fill out the [corporate
CLA](https://na2.docusign.net/Member/PowerFormSigning.aspx?PowerFormId=e1c17c66-ca4d-4aab-a953-2c231af4a20b).
