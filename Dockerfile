FROM golang:1.9

COPY ./bin/tagcontroller /

CMD ["/tagcontroller", "-v=4", "-stderrthreshold=info"]
