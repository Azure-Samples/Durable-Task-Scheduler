FROM eclipse-temurin:21-jdk AS build
WORKDIR /project

COPY gradlew gradlew
COPY gradle/ gradle/
COPY build.gradle build.gradle
COPY settings.gradle settings.gradle
COPY shared/ shared/
COPY validator-worker/ validator-worker/

RUN chmod +x gradlew && ./gradlew :validator-worker:installDist --no-daemon

FROM eclipse-temurin:21-jre
WORKDIR /app
COPY --from=build /project/validator-worker/build/install/validator-worker .

ENTRYPOINT ["bin/validator-worker"]
