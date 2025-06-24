FROM debian:buster-slim
# Official Debian and Ubuntu images automatically run apt-get clean, so explicit invocation is not required.
ARG PROJECT
RUN set -xe && apt-get update && apt-get install -y curl
RUN cp /usr/share/zoneinfo/Europe/Moscow /etc/localtime
RUN echo "Europe/Moscow" > /etc/timezone
COPY builds/${PROJECT}/${PROJECT} /usr/dev/bin/executed
COPY config/${PROJECT}/config.yaml /usr/dev/bin/config.yaml
RUN chmod +x /usr/dev/bin/executed
ENV BIN_PATH=/usr/dev/bin/executed
WORKDIR /usr/dev/bin
# CMD должен использовать строковый литерал или JSON-формат
CMD ["sh", "-c", "${BIN_PATH}"]
EXPOSE 10001