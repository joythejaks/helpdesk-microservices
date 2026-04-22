@echo off
echo Starting Helpdesk Microservices Backend...
cd backend
docker-compose up -d
echo Backend services are starting...
echo API Gateway: http://localhost:8080
echo Auth Service: http://localhost:8081
echo Ticket Service: http://localhost:8082
echo Notification Service: http://localhost:8083
echo RabbitMQ Management: http://localhost:15672
pause