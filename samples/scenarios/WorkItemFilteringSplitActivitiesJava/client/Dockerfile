FROM eclipse-temurin:21-jdk AS build
WORKDIR /project

COPY gradlew gradlew
COPY gradle/ gradle/
COPY build.gradle build.gradle
COPY settings.gradle settings.gradle
COPY shared/ shared/
COPY client/ client/

RUN chmod +x gradlew && ./gradlew :client:installDist --no-daemon

FROM eclipse-temurin:21-jre
WORKDIR /app
COPY --from=build /project/client/build/install/client .

ENTRYPOINT ["bin/client"]
