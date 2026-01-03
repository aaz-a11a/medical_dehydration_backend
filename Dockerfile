FROM golang:1.23-alpine

WORKDIR /app

# Копируем файлы модулей для кэширования зависимостей
COPY go.mod go.sum ./
RUN go mod download

# Копируем ВСЕ исходные файлы
COPY . .

# Создаем необходимые директории
RUN mkdir -p resources templates

# Собираем приложение из правильной папки
RUN go build -o main ./cmd/awesomeProject

EXPOSE 8080

CMD ["./main"]