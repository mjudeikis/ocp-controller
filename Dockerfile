FROM golang:1.9

COPY ./bin/deploymentconfig-operator /

ENTRYPOINT /deploymentconfig-operator
