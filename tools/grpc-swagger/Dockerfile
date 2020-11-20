FROM openjdk:8

EXPOSE 8080

RUN wget https://github.com/grpc-swagger/grpc-swagger/releases/download/v0.1.8/grpc-swagger.jar

ENTRYPOINT java -jar grpc-swagger.jar
