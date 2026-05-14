FROM eclipse-temurin:21-jdk AS build
WORKDIR /project

COPY gradlew gradlew
COPY gradle/ gradle/
COPY build.gradle build.gradle
COPY settings.gradle settings.gradle
COPY shared/ shared/
COPY shipper-worker/ shipper-worker/

RUN chmod +x gradlew && ./gradlew :shipper-worker:installDist --no-daemon

FROM eclipse-temurin:21-jre
WORKDIR /app
COPY --from=build /project/shipper-worker/build/install/shipper-worker .

ENTRYPOINT ["bin/shipper-worker"]
