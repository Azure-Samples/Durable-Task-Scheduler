FROM eclipse-temurin:17-jdk

# Install dos2unix to fix line endings
RUN apt-get update && apt-get install -y dos2unix && rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Copy all files
COPY . .

# Fix line endings for gradlew
RUN dos2unix gradlew && chmod +x gradlew

# Run the gradle task
CMD ["./gradlew", "runChainingPattern"]