FROM eclipse-temurin:21-jdk AS build
WORKDIR /project

# Copy Gradle wrapper and build files first (for layer caching)
COPY gradlew gradlew
COPY gradle/ gradle/
COPY build.gradle build.gradle
COPY settings.gradle settings.gradle
COPY shared/ shared/
COPY orchestrator-worker/ orchestrator-worker/

RUN chmod +x gradlew && ./gradlew :orchestrator-worker:installDist --no-daemon

FROM eclipse-temurin:21-jre
WORKDIR /app
COPY --from=build /project/orchestrator-worker/build/install/orchestrator-worker .

ENTRYPOINT ["bin/orchestrator-worker"]
