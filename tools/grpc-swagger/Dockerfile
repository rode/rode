FROM openjdk:8

EXPOSE 8080

RUN wget https://github.com/grpc-swagger/grpc-swagger/releases/latest/download/grpc-swagger.jar

ENTRYPOINT java -jar grpc-swagger.jar
