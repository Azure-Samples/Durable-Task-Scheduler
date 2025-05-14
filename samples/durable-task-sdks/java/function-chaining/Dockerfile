FROM eclipse-temurin:17-jdk-alpine AS build
WORKDIR /workspace/app

COPY . .
RUN chmod +x ./gradlew

# Run the specific Gradle task directly.
CMD ["./gradlew", "runChainingPattern"]
